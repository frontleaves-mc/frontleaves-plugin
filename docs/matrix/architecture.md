# Matrix 系统架构概览

> **版本**: 1.0.0
> **适用读者**: Java 插件开发者、Go 后端维护者
> **前置阅读**: 本文档聚焦 Go 后端架构；Java 插件对接请参阅 [Java Integration](./java-integration.md)

---

## 目录

1. [系统概述](#1-系统概述)
2. [架构全景图](#2-架构全景图)
3. [核心组件](#3-核心组件)
4. [数据流详解](#4-数据流详解)
5. [Redis 缓冲机制](#5-redis-缓冲机制)
6. [配置参数](#6-配置参数)
7. [REST API 端点](#7-rest-api-端点)
8. [目录结构](#8-目录结构)

---

## 1. 系统概述

Matrix 是 frontleaves-plugin 的玩家行为遥测子系统。它的核心任务是从 Minecraft 服务端（Java/Paper 插件）实时接收玩家行为数据，在 Go 后端完成两层处理：

- **统计聚合** — 方块破坏/放置、实体击杀、死亡计数等数据按玩家维度聚合后写入 PostgreSQL
- **反作弊检测** — 实时分析攻击距离（Reach）和移动速度（Speed）异常，触发警告并写入数据库

### 为什么要外部检测？

传统的 Minecraft 反作弊方案运行在服务端主线程上，受制于 Java 单线程模型。高精度检测会拖慢 TPS，低精度又容易被绕过。Matrix 把计算卸载到独立的 Go 进程：

- MC 服务端仅负责**数据采集和上报**，几乎零性能开销
- Go 后端利用协程并发处理，天然适合高吞吐的小消息场景
- 检测规则可以在不重启 MC 服务器的情况下更新（只重启 Go 服务）

### 数据规模

以 100 个在线玩家为例：

- 心跳快照：每 1-2 秒/人，约 200 字节/条
- 事件数据：离散触发，约 100-200 字节/条
- 峰值约 200+ 条/秒原始数据流入 Go 后端
- 原始数据不落盘，仅在 Redis 中缓冲后被消费丢弃

---

## 2. 架构全景图

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Minecraft 服务端 (Java/Paper)                      │
│                                                                      │
│  PlayerJoin ─┐                                                      │
│  PlayerQuit ─┤                                                      │
│  TelemetryTick ────→  TelemetryStreamHandler ──→ gRPC Client Stream │
│  BlockBreak ──┤      (一个 Java 实例一条 Stream)                     │
│  EntityDamage ┤                                                      │
│  ...各类事件 ─┘                                                      │
└──────────────────────────┬──────────────────────────────────────────┘
                           │ gRPC Client-Streaming (protobuf)
                           │
┌──────────────────────────▼──────────────────────────────────────────┐
│                  frontleaves-plugin (Go)                             │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ MatrixTelemetryHandler                                       │   │
│  │  TelemetryStream() — Recv Loop (per server connection)       │   │
│  │    ├── handlePlayerJoin() → SessionManager.GetOrCreate()     │   │
│  │    ├── handlePlayerQuit()  → session.MarkOffline() + Stop()  │   │
│  │    └── handleGenericEvent() → session.Send()                 │   │
│  └──────────────────────────────┬───────────────────────────────┘   │
│                                 │                                    │
│  ┌──────────────────────────────▼───────────────────────────────┐   │
│  │ MatrixSessionManager (全局单例)                                │   │
│  │  sessions map["{server}:{uuid}"] → *PlayerSession            │   │
│  └──────────────────────────────┬───────────────────────────────┘   │
│                                 │ per-player                         │
│  ┌──────────────────────────────▼───────────────────────────────┐   │
│  │ PlayerSession                                                │   │
│  │  ┌──────────────┐   ┌───────────────────────────────────┐   │   │
│  │  │ base 协程     │   │ manage 协程                         │   │   │
│  │  │ inputCh ────→│   │ 定时 LPOP batch(100) ──→ syncBroadcast │
│  │  │ RPUSH+LTRIM  │   │                    │                │   │   │
│  │  │  → Redis List│   │         ┌──────────┼──────────┐    │   │   │
│  │  └──────────────┘   │         ▼          ▼          │    │   │   │
│  │                     │  StatisticsSub  AntiCheatSub   │    │   │   │
│  │                     │  (聚合统计→DB)  (行为检测→DB+Redis)│   │   │
│  │                     └───────────────────────────────────┘   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│              ┌─────────────┬──────────────┐                          │
│              ▼             ▼              ▼                          │
│         PostgreSQL     Redis List    Redis Hash                     │
│         (统计/警告)    (原始缓冲)    (监控快照)                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. 核心组件

### 3.1 MatrixTelemetryHandler — gRPC 接收层

**位置**: `internal/grpc/handler/matrix_telemetry.go`

这是整个 Matrix 的入口。每个 MC 服务器实例建立一条 Client Stream，Handler 在 `TelemetryStream()` 中循环 `Recv()`，按 `oneof payload` 类型分发消息：

| 分支 | 处理方式 |
|------|---------|
| `PlayerJoin` | 调用 `SessionManager.GetOrCreate()` 创建/获取会话，再 `session.Send()` |
| `PlayerQuit` | 获取现有会话，发送 Quit 事件后调用 `session.MarkOffline()` + `session.Stop()` |
| 其他事件 | 解析 `playerUUID`，通过 `sessionManager.Get()` 查找会话，`session.Send()` 投递 |

Stream 断连时（`Recv()` 返回 error），该服务器关联的所有活跃会话会因 context 取消进入排水模式。

**认证**: 通过 gRPC metadata 的 `plugin-secret-key` 字段验证，由 `StreamPluginVerify` 拦截器处理。

### 3.2 MatrixSessionManager — 全局会话管理器

**位置**: `internal/logic/matrix/session_manager.go`

全局单例，维护 `map[string]*PlayerSession`，key 格式为 `{serverName}:{playerUUID}`。

核心方法：

| 方法 | 作用 |
|------|------|
| `GetOrCreate(ctx, serverName, uuid, name)` | 获取已有会话或创建新的，创建时自动注入 Sub 实例并启动协程 |
| `Get(serverName, uuid)` | 读锁查找，返回 `nil` 表示无活跃会话 |
| `Remove(sessionKey)` | 从 map 中移除已结束的会话 |
| `ShutdownAll()` | 遍历所有会话调用 `Stop()`，服务关闭时使用 |

每个新会话创建时会实例化独立的 `StatisticsSub` 和 `AntiCheatSub`，它们持有 per-player 状态。

### 3.3 PlayerSession — Per-Player 协程架构

**位置**: `internal/logic/matrix/player_session.go`

每个在线玩家（准确说是每个 `serverName + playerUUID` 组合）对应一个 `PlayerSession`，内部运行两个协程：

#### base 协程

负责**写入**。从 `inputCh`（带缓冲 channel，容量 5000）读取消息，序列化后通过 Redis Pipeline 写入 List：

```
RPUSH bufferKey <data>
LTRIM bufferKey 0 4999
```

`inputCh` 关闭时（玩家离线触发 `MarkOffline()`），base 协程自然退出。

#### manage 协程

负责**消费和处理**。每 500ms 触发一次：

1. `LPOP count=100` 批量弹出 Redis List 中的消息
2. 反序列化为 protobuf 对象
3. `syncBroadcast()` — 用 `sync.WaitGroup` 并行调用所有 Sub 的 `Process()`，等待全部完成后再进入下一轮

manage 协程还负责排水逻辑：context 取消后，循环消费直到 Redis List 为空，最后调用每个 Sub 的 `Drain()` 做收尾。

#### 生命周期

```
上线 (PlayerJoin)
  → GetOrCreate() → NewPlayerSession() → Start()
    → base goroutine: inputCh → RPUSH + LTRIM
    → manage goroutine: ticker → LPOP → syncBroadcast

运行中
  → Send(msg): 非阻塞写入 inputCh
  → manage 每 500ms 消费一批 → subs 处理

离线 (PlayerQuit)
  → MarkOffline(): isOnline=false, isDraining=true, close(inputCh)
  → base 退出（channel 关闭）
  → manage 进入 drainAll(): 循环 LPOP 直到 LLEN=0 → subs.Drain()
  → 清理 Redis buffer + monitor key
  → Stop(): cancel context, WaitGroup 等待（5s 超时）
  → SessionManager.Remove()
```

### 3.4 MatrixSub 接口 — 插件式注册

**位置**: `internal/logic/matrix/sub.go`

```go
type MatrixSub interface {
    Name() string
    Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error
    Drain(ctx context.Context) error
}
```

新增 Sub 只需实现这个接口，然后在 `SessionManager.GetOrCreate()` 中注册。manage 协程的 `syncBroadcast` 会自动调用新 Sub，无需修改管理逻辑。

#### StatisticsSub

**位置**: `internal/logic/matrix/sub_statistics.go`

维护内存中的计数器（方块破坏/放置、实体击杀、死亡），按材质和死因分类。每处理 10 条消息或收到 `PlayerQuit` 时 `flushToDB()`，将累积的统计数据写入 `tpl_matrix_player_statistic` 表。`Drain()` 时强制刷盘。

#### AntiCheatSub

**位置**: `internal/logic/matrix/sub_anti_cheat.go`

当前实现两个检测规则：

| 规则 | 数据来源 | 阈值 |
|------|---------|------|
| Reach（攻击距离） | `EntityDamageEvent` 中玩家坐标与实体坐标的欧氏距离 | > 3.5 格 |
| Speed（移动速度） | `TelemetryTick` 中相邻两帧位移量（跳过飞行状态） | > 12.0 格/秒 |

触发警告时：
1. 写入 `tpl_matrix_player_warning` 表
2. 累加 `riskScore`（每次 +20，上限 100）
3. 更新 Redis Hash 监控快照

传送、重生、游戏模式变更时会重置位置追踪状态，避免误报。

---

## 4. 数据流详解

以一个 `BlockBreakEvent` 从产生到写入数据库的完整路径为例：

```
1. MC 服务端触发 BlockBreakEvent (Bukkit API)
       │
2. Java 插件 TelemetryStreamHandler.sendEvent()
   → 封装为 MatrixTelemetryRequest { server_name, oneof=block_break }
   → gRPC Client Stream onNext()
       │
3. Go 后端 MatrixTelemetryHandler.TelemetryStream() Recv Loop
   → dispatchTelemetry() → handleGenericEvent()
   → 解析 playerUUID → sessionManager.Get(serverName, uuid)
   → session.Send(req)  // 非阻塞写入 inputCh
       │
4. PlayerSession.base 协程
   ← 从 inputCh 读取
   → protojson.Marshal()
   → Redis Pipeline: RPUSH + LTRIM(bufferKey, 0, 4999)
       │
5. PlayerSession.manage 协程（500ms ticker）
   → LPOP(bufferKey, 100) 批量弹出
   → protojson.Unmarshal() 反序列化
       │
6. syncBroadcast: 并行调用所有 Sub
   ├─ StatisticsSub.Process()
   │  → blocksBreak["STONE"]++ , totalBlocksBroken++
   │  → batchCount++, 每 10 条 flushToDB()
   │
   └─ AntiCheatSub.Process()
      → BlockBreak 不是关注的事件类型，跳过
       │
7. StatisticsSub.flushToDB()
   → 构建 entity.MatrixPlayerStatistic
   → statRepo.Upsert(ctx, stat)  // INSERT ON CONFLICT UPDATE
   → 重置批次计数器
```

---

## 5. Redis 缓冲机制

Matrix 在原始数据和处理器之间引入了 Redis List 作为缓冲层，核心目的是**解耦写入和消费速度**。

### 5.1 写入端（base 协程）

```go
pipe := rdb.Pipeline()
pipe.RPush(ctx, bufferKey, data)      // 追加到尾部
pipe.LTrim(ctx, bufferKey, 0, 4999)   // 滑动窗口，最多保留 5000 条
pipe.Exec(ctx)
```

滑动窗口保证即使消费速度跟不上，内存也不会无限增长。超出 5000 条的旧数据被截断丢弃。

### 5.2 消费端（manage 协程）

```go
results, _ := rdb.LPopCount(ctx, bufferKey, 100).Result()
```

每 500ms 弹出最多 100 条，反序列化后广播给 Sub 处理。

### 5.3 排水（Drain）

玩家离线后 manage 协程不再等待 ticker，直接循环消费：

```go
for {
    length, _ := rdb.LLen(ctx, bufferKey).Result()
    if length == 0 { break }
    batch := popBatch()
    syncBroadcast(batch)
}
```

排水完成后 `DEL bufferKey` 和 `DEL monitorKey` 清理 Redis。

### 5.4 Redis Key 命名

| Key 模式 | 类型 | 用途 | TTL |
|----------|------|------|-----|
| `tpl:matrix:session:{sessionKey}:buffer` | List | 原始数据缓冲 | 无（排水后 DEL） |
| `tpl:matrix:session:{sessionKey}:monitor` | Hash | 反作弊监控快照 | 5 分钟 |

`sessionKey` 格式：`{serverName}:{playerUUID}`，如 `survival:550e8400-e29b-41d4-a716-446655440000`

### 5.5 监控快照字段

Redis Hash `monitorKey` 中存储的字段：

| 字段 | 说明 | 写入者 |
|------|------|--------|
| `risk_score` | 风险分数 (0-100) | AntiCheatSub |
| `warning_count_session` | 本次会话累计警告数 | AntiCheatSub |
| `last_tick_processed` | 最后处理心跳的 Unix 毫秒时间戳 | manage 协程 |

---

## 6. 配置参数

### 6.1 Go 后端参数

以下参数定义在 `internal/logic/matrix/player_session.go` 中：

| 参数 | 值 | 可配置 | 说明 |
|------|-----|--------|------|
| `inputChSize` | 5000 | 否 | inputCh 缓冲区大小，控制内存中最多缓存的消息数 |
| `manageInterval` | 500ms | 否 | manage 协程消费间隔，越小消费越快但 Redis 压力越大 |
| `popBatchSize` | 100 | 否 | 每次 LPOP 的最大条数 |
| `drainTimeout` | 5s | 否 | 排水超时时间，超时后强制退出 |

反作弊阈值（定义在 `sub_anti_cheat.go`）：

| 参数 | 值 | 说明 |
|------|-----|------|
| `reachThreshold` | 3.5 | Reach 攻击距离阈值（blocks） |
| `speedThreshold` | 12.0 | Speed 移动速度阈值（blocks/second） |
| `riskScorePerHit` | 20 | 每次触发警告增加的风险分数 |
| `maxRiskScore` | 100 | 风险分数上限 |

统计刷盘间隔（定义在 `sub_statistics.go`）：

| 参数 | 值 | 说明 |
|------|-----|------|
| `flushInterval` | 10 | 每处理 N 条消息触发一次 DB 写入 |

### 6.2 Java 插件端参数

在 `config.yml` 中配置：

```yaml
grpc:
  server-name: "survival"          # 服务器标识，影响 sessionKey 隔离

telemetry:
  tick-interval-ticks: 20          # TelemetryTick 心跳间隔（20 tick = 1 秒）
```

---

## 7. REST API 端点

Matrix 的 REST API 分为管理端和玩家端，均需要登录认证。

### 7.1 统计查询

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `GET` | `/api/v1/admin/matrix/statistics/:uuid` | 管理员 | 按 UUID 查询指定玩家的统计数据 |
| `GET` | `/api/v1/matrix/statistics/me` | 玩家 | 查询当前登录玩家的统计数据 |

### 7.2 警告查询

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| `GET` | `/api/v1/admin/matrix/warnings` | 管理员 | 查询警告列表（支持分页/筛选） |
| `GET` | `/api/v1/admin/matrix/warnings/:id` | 管理员 | 查询单条警告详情 |

### 7.3 gRPC 接口

| 方法 | 类型 | 说明 |
|------|------|------|
| `TelemetryStream` | Client Stream | MC 插件持续发送遥测事件，服务端返回一个最终响应 |

---

## 8. 目录结构

Matrix 相关源码文件在项目中的位置：

```
proto/
└── matrix/v1/
    └── matrix_telemetry.proto              # Proto 消息契约

internal/
├── constant/
│   ├── cache.go                            # Redis key 定义 (CacheMatrixPlayerBuffer, CacheMatrixPlayerMonitor)
│   ├── gene_number.go                      # Snowflake Gene 编号 (50=Statistic, 51=Warning)
│   └── matrix_event_type.go                # 事件类型枚举
│
├── entity/
│   ├── matrix_player_statistic.go          # 统计实体 (1:1 UUID)
│   └── matrix_player_warning.go            # 警告日志实体
│
├── grpc/
│   ├── handler/
│   │   ├── handler.go                      # matrixService 定义 + newMatrixService 构造
│   │   └── matrix_telemetry.go             # TelemetryStream 接收层 + 事件分发
│   ├── middleware/plugin_verify.go         # gRPC plugin-secret-key 认证
│   ├── register/register.go               # RegisterMatrixTelemetryService 注册
│   └── gen/matrix/v1/                      # buf 自动生成（勿手动编辑）
│
├── logic/matrix/
│   ├── sub.go                              # MatrixSub 接口 + MatrixSubRegistry
│   ├── session_manager.go                  # MatrixSessionManager 全局单例
│   ├── player_session.go                   # PlayerSession (baseLoop + manageLoop)
│   ├── sub_statistics.go                   # StatisticsSub 统计聚合实现
│   └── sub_anti_cheat.go                   # AntiCheatSub 反作弊检测实现
│
├── handler/
│   └── matrix_*.go                         # HTTP handler（统计/警告查询）
│
├── repository/
│   ├── matrix_statistic.go                 # 统计数据 DB 操作
│   ├── matrix_warning.go                   # 警告数据 DB 操作
│   └── cache/matrix_monitor.go             # Redis 监控快照缓存管理
│
└── app/
    ├── route/
    │   ├── route_matrix_statistic.go       # 统计路由（admin + player）
    │   └── route_matrix_warning.go         # 警告路由（admin only）
    └── startup/startup_database.go         # Migration 注册 Matrix 实体
```

---

## 附录: 数据库表

### tpl_matrix_player_statistic

玩家统计表，与 UUID 一对一关系。使用 `jsonb` 字段存储按材质/类型/死因分类的计数：

| 字段 | 类型 | 说明 |
|------|------|------|
| `player_uuid` | UUID | 玩家唯一标识 |
| `blocks_break` | JSONB | 方块破坏统计 `{material: count}` |
| `blocks_place` | JSONB | 方块放置统计 |
| `entities_kill` | JSONB | 实体击杀统计 |
| `deaths` | JSONB | 死因统计 `{cause: count}` |
| `total_blocks_broken` | INT | 总破坏方块数 |
| `total_blocks_placed` | INT | 总放置方块数 |
| `total_entities_killed` | INT | 总击杀实体数 |
| `total_deaths` | INT | 总死亡次数 |
| `total_play_time_ms` | INT | 总游戏时长 |
| `total_sessions` | INT | 总会话数 |

Gene 编号：50

### tpl_matrix_player_warning

玩家警告日志表，每条警告一行记录：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | Snowflake | 警告 ID |
| `player_uuid` | UUID | 玩家唯一标识 |
| `server_name` | VARCHAR(64) | 服务器名称 |
| `warning_type` | VARCHAR(32) | 警告类型（REACH / SPEED / ...） |
| `description` | TEXT | 警告描述 |
| `risk_score` | INT | 触发时的风险分数 |
| `context_data` | JSONB | 上下文数据快照 |

Gene 编号：51
