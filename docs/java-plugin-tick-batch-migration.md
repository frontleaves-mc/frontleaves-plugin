# Java 插件遥测数据批量上报迁移指南

> 适用版本：Matrix 遥测服务 proto 重构后（`oneof` → `repeated`）
> 目标读者：Java/Bukkit 插件开发者

---

## 1. 变更概述

### 1.1 变更目的

重构前，Java 插件每 tick 为每位在线玩家各发送一条 `MatrixTelemetryRequest` gRPC 消息。当服务器在线人数较多时，这会产生大量细小消息，带来不必要的网络开销和序列化成本。

重构后，Java 插件改为**每 tick 只发送一条消息**，该消息内部通过多个 `repeated` 字段承载同一 tick 内所有玩家的事件数据。这样显著降低了消息数量和传输开销。

### 1.2 核心变化

| 维度 | 旧格式 | 新格式 |
|------|--------|--------|
| 消息粒度 | 每 tick 每玩家一条 | 每 tick 每服务器一条 |
| payload 结构 | `oneof payload { ... }` | 16 个独立的 `repeated` 字段 |
| 顶层字段 | 仅 `server_name` | 新增 `tick_number`、`timestamp` |
| 连接事件 | 放在 `oneof` 内 | 提升为独立单次字段（保持不变） |

### 1.3 影响范围

- Java 插件端的 `TelemetryStream` 发送逻辑需要重写
- 事件采集逻辑需要从“即时发送”改为“tick 末批量发送”
- `PlayerJoinEvent` / `PlayerQuitEvent` 保持原有即时发送逻辑，不受影响

---

## 2. 旧格式 vs 新格式

### 2.1 Proto 结构对比

**旧格式（`oneof`）**

```protobuf
message MatrixTelemetryRequest {
  string server_name = 1;

  oneof payload {
    PlayerJoinEvent     player_join     = 11;
    PlayerQuitEvent     player_quit     = 12;
    TelemetryTick       telemetry_tick  = 13;
    BlockBreakEvent     block_break     = 14;
    BlockPlaceEvent     block_place     = 15;
    EntityKillEvent     entity_kill     = 16;
    // ... 更多事件
  }
}
```

**新格式（`repeated`）**

```protobuf
message MatrixTelemetryRequest {
  string server_name = 1;
  int64  tick_number = 2;
  int64  timestamp   = 3;

  // 生命周期事件 — 单次设置，不使用 repeated
  PlayerJoinEvent     player_join     = 11;
  PlayerQuitEvent     player_quit     = 12;

  // 所有其他事件 — repeated，支持同 tick 多玩家批量上报
  repeated TelemetryTick       telemetry_ticks    = 13;
  repeated BlockBreakEvent     block_breaks       = 14;
  repeated BlockPlaceEvent     block_places       = 15;
  repeated EntityKillEvent     entity_kills       = 16;
  // ... 共 16 个 repeated 字段
}
```

### 2.2 Java 构造代码对比

**旧写法（每 tick 每玩家循环发送）**

```java
// 每 tick 遍历所有玩家，每人发一条消息
for (Player player : Bukkit.getOnlinePlayers()) {
    MatrixTelemetryRequest req = MatrixTelemetryRequest.newBuilder()
        .setServerName(serverName)
        .setTelemetryTick(TelemetryTick.newBuilder()
            .setPlayerUuid(player.getUniqueId().toString())
            .setPlayerName(player.getName())
            .setPosX(player.getLocation().getX())
            .setPosY(player.getLocation().getY())
            .setPosZ(player.getLocation().getZ())
            // ... 更多字段
            .build())
        .build();

    streamObserver.onNext(req);
}
```

**新写法（每 tick 一条消息，内含所有玩家数据）**

```java
// 每 tick 只构建一条消息
MatrixTelemetryRequest.Builder builder = MatrixTelemetryRequest.newBuilder()
    .setServerName(serverName)
    .setTickNumber(currentTick)           // 自增 tick 计数器
    .setTimestamp(System.currentTimeMillis());

for (Player player : Bukkit.getOnlinePlayers()) {
    TelemetryTick tick = TelemetryTick.newBuilder()
        .setPlayerUuid(player.getUniqueId().toString())
        .setPlayerName(player.getName())
        .setPosX(player.getLocation().getX())
        .setPosY(player.getLocation().getY())
        .setPosZ(player.getLocation().getZ())
        // ... 更多字段
        .build();

    builder.addTelemetryTicks(tick);      // add 而非 set
}

streamObserver.onNext(builder.build());
```

> **关键区别**：旧代码在循环体内部调用 `streamObserver.onNext()`，新代码在循环结束后只调用一次。

---

## 3. 16 种事件类型对照表

下表列出了所有从 `oneof` 迁移到 `repeated` 的事件类型，以及对应的 Java API 变化。

| 事件类型 | 旧 API（`oneof`） | 新 API（`repeated`） |
|----------|------------------|---------------------|
| 心跳快照 | `setTelemetryTick(tick)` | `addTelemetryTicks(tick)` |
| 方块破坏 | `setBlockBreak(evt)` | `addBlockBreaks(evt)` |
| 方块放置 | `setBlockPlace(evt)` | `addBlockPlaces(evt)` |
| 实体击杀 | `setEntityKill(evt)` | `addEntityKills(evt)` |
| 实体伤害 | `setEntityDamage(evt)` | `addEntityDamages(evt)` |
| 玩家受伤 | `setPlayerDamage(evt)` | `addPlayerDamages(evt)` |
| 玩家死亡 | `setPlayerDeath(evt)` | `addPlayerDeaths(evt)` |
| 物品丢弃 | `setItemDrop(evt)` | `addItemDrops(evt)` |
| 物品拾取 | `setItemPickup(evt)` | `addItemPickups(evt)` |
| 背包操作 | `setInventoryAction(evt)` | `addInventoryActions(evt)` |
| 聊天消息 | `setPlayerChat(evt)` | `addPlayerChats(evt)` |
| 命令执行 | `setPlayerCommand(evt)` | `addPlayerCommands(evt)` |
| 状态切换 | `setPlayerToggle(evt)` | `addPlayerToggles(evt)` |
| 玩家传送 | `setTeleport(evt)` | `addTeleports(evt)` |
| 玩家重生 | `setRespawn(evt)` | `addRespawns(evt)` |
| 模式切换 | `setGameModeChange(evt)` | `addGameModeChanges(evt)` |

**保持不变的事件（仍为单次设置）**

| 事件类型 | API（新旧相同） |
|----------|----------------|
| 玩家加入 | `setPlayerJoin(evt)` |
| 玩家退出 | `setPlayerQuit(evt)` |

> `PlayerJoinEvent` 和 `PlayerQuitEvent` 不属于 repeated 字段，仍然与旧版本一样使用 `setXxx()` 单次赋值。这类事件在触发时应立即发送，不需要等待 tick 结束。

---

## 4. Java 插件端迁移步骤

### Step 1：更新 proto 文件并重新生成 Java 代码

1. 将新的 `matrix_telemetry.proto` 替换到插件项目的 `src/main/proto/` 目录
2. 执行 protobuf 代码生成（Maven 示例）

```bash
mvn protobuf:compile
```

或 Gradle：

```bash
./gradlew generateProto
```

3. 确认生成的 `MatrixTelemetryRequest` 类中已出现 `addTelemetryTicks()`、`addBlockBreaks()` 等 repeated 字段的 `add` 方法

### Step 2：重构 TelemetryTick 发送逻辑

找到原有的 tick 任务（通常是 Bukkit Scheduler 的同步任务），将“循环内逐人发送”改为“循环内收集，循环外统一发送”。

**改造前：**

```java
Bukkit.getScheduler().runTaskTimer(plugin, () -> {
    for (Player player : Bukkit.getOnlinePlayers()) {
        MatrixTelemetryRequest req = MatrixTelemetryRequest.newBuilder()
            .setServerName(serverName)
            .setTelemetryTick(buildTick(player))
            .build();
        streamObserver.onNext(req);   // 每人发一条
    }
}, 0L, 1L);
```

**改造后：**

```java
final AtomicLong tickCounter = new AtomicLong(0);

Bukkit.getScheduler().runTaskTimer(plugin, () -> {
    MatrixTelemetryRequest.Builder builder = MatrixTelemetryRequest.newBuilder()
        .setServerName(serverName)
        .setTickNumber(tickCounter.getAndIncrement())
        .setTimestamp(System.currentTimeMillis());

    for (Player player : Bukkit.getOnlinePlayers()) {
        builder.addTelemetryTicks(buildTick(player));   // 收集到 builder
    }

    streamObserver.onNext(builder.build());             // 只发一次
}, 0L, 1L);
```

### Step 3：事件采集改为先收集到 List，tick 结束时批量构造

对于非 Tick 类事件（如方块破坏、聊天消息等），如果旧代码是事件监听后直接发送，需要改为先缓存到内存队列，在 tick 结束时统一打包。

**简易实现示例：**

```java
public class TelemetryEventBuffer {
    private final List<BlockBreakEvent> blockBreaks = new ArrayList<>();
    private final List<PlayerChatEvent> playerChats = new ArrayList<>();
    // ... 其他事件列表

    public synchronized void addBlockBreak(BlockBreakEvent evt) {
        blockBreaks.add(evt);
    }

    public synchronized void drainTo(MatrixTelemetryRequest.Builder builder) {
        blockBreaks.forEach(builder::addBlockBreaks);
        playerChats.forEach(builder::addPlayerChats);
        // ...
        blockBreaks.clear();
        playerChats.clear();
    }
}
```

然后在 tick 任务中调用 `drainTo(builder)`，将本 tick 内收集的所有事件一次性附加到请求中。

### Step 4：PlayerJoin / PlayerQuit 保持原有逻辑

玩家加入和退出事件仍然采用即时发送，不经过 tick 批量缓冲。

```java
@EventHandler
public void onPlayerJoin(PlayerJoinEvent event) {
    Player player = event.getPlayer();
    MatrixTelemetryRequest req = MatrixTelemetryRequest.newBuilder()
        .setServerName(serverName)
        .setPlayerJoin(PlayerJoinEvent.newBuilder()
            .setPlayerUuid(player.getUniqueId().toString())
            .setPlayerName(player.getName())
            .setWorldName(player.getWorld().getName())
            .setIpAddress(player.getAddress().getAddress().getHostAddress())
            .setTimestamp(System.currentTimeMillis())
            .build())
        .build();

    streamObserver.onNext(req);
}
```

### Step 5：编译验证与联调测试

1. 编译 Java 插件，确保所有旧的 `setXxx()` 调用已替换为正确的 `addXxxs()`
2. 在测试服启动插件，检查 gRPC 流是否正常建立
3. 观察 Go 服务端日志，确认每秒接收的消息数从 `N * 玩家数` 降到 `N * 服务器数`
4. 验证各事件类型在 Go 端是否正确解析和入库

---

## 5. 注意事项

### 5.1 空数组行为

repeated 字段如果没有调用任何 `add` 方法，序列化后默认为空数组，不会导致解析错误。Java 插件不需要为“本 tick 无某类事件”做特殊填充，直接不 `add` 即可。

### 5.2 tick_number 的语义

- `tick_number` 是一个**自增计数器**，建议从 `0` 开始，每 tick 递增 `1`
- Go 端使用它进行消息排序和去重，不要求全局唯一，只要求在单服务器流内单调递增
- 如果插件重启，从 `0` 重新开始计数是可接受的

### 5.3 timestamp 的语义

- `timestamp` 是消息构造时刻的**毫秒级 Unix 时间戳**（`System.currentTimeMillis()`）
- 用于 Go 端判断数据延迟和做时间窗口分析
- 不需要与 `tick_number` 严格对应，只要求能反映真实时间

### 5.4 PlayerJoin 与 PlayerQuit 的即时性

- `PlayerJoinEvent` 和 `PlayerQuitEvent` 仍然保持**事件触发时立即发送**
- 它们不需要等待 tick 结束，也不应该被缓冲到 tick 批量消息中
- 这样能保证 Go 端第一时间感知到玩家上下线状态变化

### 5.5 gRPC 流连接

- 本次重构不涉及 gRPC 连接层变更
- Java 端仍然保持一条长连接 `streamObserver`，不需要逐条重建
- 如果连接断开，按原有重连逻辑处理即可

### 5.6 事件顺序

- 同一条 `MatrixTelemetryRequest` 内，各 repeated 列表中的事件顺序**不影响**业务正确性
- Go 端会按 `player_uuid` 重新路由和聚合，不依赖发送顺序

---

## 6. FAQ

**Q：旧版 Java 插件能与新版 Go 服务端互通吗？**

A：不能。这是一次 breaking change。proto 的 `oneof` 和 `repeated` 字段编号虽然部分重叠，但 wire 格式不兼容。Java 插件和 Go 服务端必须同步升级，否则会出现解析失败或字段丢失。

**Q：同 tick 内的事件顺序重要吗？**

A：不重要。Go 端消费时会按 `player_uuid` 维度重新分组和排序，同 tick 内的相对顺序不会影响最终处理结果。插件端只要保证事件被正确放入对应的 repeated 列表即可。

**Q：`tick_number` 可以为 0 吗？**

A：可以。建议从 `0` 开始自增。Go 端只要求它在单条流内单调递增，用于检测丢包和乱序，对起始值没有限制。

**Q：如果某 tick 内没有任何事件，还需要发送空消息吗？**

A：建议仍然发送。即使所有 repeated 字段都为空，携带 `tick_number` 和 `timestamp` 的空消息对 Go 端也有意义，它可以用来确认服务器仍然存活、以及衡量空 tick 的频率。如果担心网络开销，可在插件端配置“空 tick 抑制”策略，但默认建议发送。

**Q：一个 tick 消息过大会有问题吗？**

A：protobuf 的 repeated 字段本身没有硬性数量限制。如果服务器在线人数极多（如 500+），单条消息可能变大，但相比旧方案的“500 条独立消息”，总字节量通常会减少（因为头部字段只发一次）。如有极端场景，可联系 Go 端评估是否需要流控或分片。

---

## 7. 快速检查清单

迁移完成后，对照以下清单确认无遗漏：

- [ ] proto 文件已更新到最新版本
- [ ] Java protobuf 代码已重新生成
- [ ] `TelemetryTick` 发送逻辑从逐人发送改为 tick 批量发送
- [ ] 所有事件类型的 API 已从 `setXxx()` 改为 `addXxxs()`（PlayerJoin/Quit 除外）
- [ ] `tick_number` 自增逻辑已添加
- [ ] `timestamp` 已设置为 `System.currentTimeMillis()`
- [ ] `PlayerJoin` / `PlayerQuit` 仍为即时发送，未被误放入 tick 缓冲
- [ ] 编译通过，无遗留的 `setTelemetryTick` 等旧 API 调用
- [ ] 测试服联调通过，Go 端能正确接收和解析数据

---

*文档版本：v1.0 | 对应 proto commit：`d856f82`*
