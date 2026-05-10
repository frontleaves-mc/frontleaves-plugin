# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### ⚠️ Breaking Changes — gRPC Proto

#### `HeartbeatEvent` 字段变更

- **Proto**: `frontleaves.status.v1.HeartbeatEvent` 新增 `int32 online_player = 3` 和 `repeated PlayerStatus players = 4` 字段
- **影响范围**: 所有使用 `ServerEventStream` RPC 发送 `HeartbeatEvent` 的 Java 插件
- **迁移路径**:
  - 在 `HeartbeatEvent` 构建中添加 `online_player`（在线玩家数）和 `players`（`PlayerStatus` 列表）字段填充
  - `online_player` 和 `players` 为可选字段，不填充不影响服务端正常运作
  - 填充后 Go 服务端将使用心跳携带的玩家列表进行定期校准/对账，增强在线状态可靠性
- **旧版 Java 插件兼容性**: ✅ **向后兼容** — 新增字段使用编号 3 和 4，旧版插件不发送这两个字段，Go 服务端按零值/空列表处理，跳过对账逻辑

> **Java 插件开发者行动项**: 建议在心跳中填充 `online_player` 和 `players` 字段以启用服务端玩家对账，防止 Redis 缓存过期导致玩家离线误判

---

### Added

#### fp_server_player 玩家在线快照持久化

- 新增 `fp_server_player` 数据表，持久化玩家在线状态快照（Snowflake ID, Gene=47）
  - 外键关联 `fp_server` 表（`server_id → id`）
  - 唯一索引 `uk_server_player`（`server_id + player_uuid`）
  - 字段：`player_uuid`, `player_name`, `world_name`, `online`, `last_seen`
- 新增 `ServerPlayerRepository` 数据访问层（`internal/repository/server_player_repo.go`）:
  - `UpsertOnline` — 创建或更新玩家在线状态（`FirstOrCreate + Assign` 模式）
  - `MarkOffline` — 标记指定玩家离线
  - `MarkAllOfflineByServer` — 标记某服务器所有在线玩家离线
  - `GetOnlineByServer` — 查询服务器所有在线玩家
  - `GetOnlineByServerAndUUIDs` — 查询指定 UUID 中在线的玩家
  - `GetOnlinePlayerUUIDsByServer` — 查询服务器在线玩家 UUID 列表
- 新增 `ServerPlayerLogic` 业务逻辑层（`internal/logic/server_player_logic.go`）:
  - `ReconcilePlayers` — 心跳对账：比较心跳玩家列表与 DB 状态，新增玩家 `UpsertOnline`，缺失玩家 `MarkOffline`，空列表跳过对账
  - `PlayerJoined` / `PlayerLeft` / `ServerOffline` — 事件驱动的 DB 状态同步
  - `GetOnlinePlayers` — 查询服务器在线玩家列表

#### gRPC 心跳玩家对账

- `HeartbeatEvent` 处理新增玩家列表解析与 DB 对账：
  - 解析 `players` 字段，将 proto `PlayerStatus` 转换为 `PlayerInfo`
  - 调用 `ServerPlayerLogic.ReconcilePlayers` 与 DB 状态 diff 同步
  - 仅当 `players` 非空时触发对账，空列表心跳安全跳过（不会误标记全部离线）
- `PlayerJoinEvent` 处理新增 DB 同步：Redis 更新后调用 `ServerPlayerLogic.PlayerJoined`
- `PlayerQuitEvent` 处理新增 DB 同步：Redis 更新后调用 `ServerPlayerLogic.PlayerLeft`
- `cleanupServerStatus` 新增 DB 清理：Redis 清理后调用 `ServerPlayerLogic.ServerOffline` 标记所有玩家离线
- 所有 DB 同步错误仅 `log.Warn`，不阻断 Redis 主流程

#### HTTP API DB 降级读取

- `GET /servers/status` 和 `POST /servers/:name/refresh` 新增 DB 降级：
  - 当 Redis 缓存过期（`HGetAll` 返回空）时，查询 `fp_server_player` 表获取在线玩家
  - 降级数据包含：玩家 UUID、名称、所在世界
  - 降级数据不含：TPS、最后心跳时间（Redis 过期后无法获取，保持为 0）
  - DB 查询失败时优雅降级为空结果，不返回错误
