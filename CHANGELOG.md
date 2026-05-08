# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### ⚠️ Breaking Changes — gRPC Proto

#### `HeartbeatEvent.online_players` 字段弃用 (reserved)

- **Proto**: `frontleaves.status.v1.HeartbeatEvent` 中的 `int32 online_players = 2` 字段已被标记为 `reserved`
- **影响范围**: 所有使用 `ServerEventStream` RPC 发送 `HeartbeatEvent` 的 Java 插件
- **迁移路径**:
  - **不再需要**在心跳中发送 `online_players` 字段
  - 在线人数现在由 Go 服务端自动计算：`PlayerJoinEvent` 时 `SAdd` 到 Redis set，`PlayerQuitEvent` 时 `SRem`，服务端通过 `SCard` 获取实时在线人数
  - Java 插件只需正确上报 `PlayerJoinEvent` 和 `PlayerQuitEvent` 即可
- **字段编号 2 已永久保留**，不可复用于其他字段
- **旧版 Java 插件兼容性**: 旧版插件继续发送 `online_players` 字段不会报错，但该值会被 Go 服务端忽略

> **Java 插件开发者行动项**: 移除 `HeartbeatEvent` 构建中的 `setOnlinePlayers()` 调用，并确保 `PlayerJoinEvent` / `PlayerQuitEvent` 正确上报

---

### Added

#### fp_server 服务器管理系统

- 新增 `fp_server` 数据表，支持服务器注册与管理（Snowflake ID, Gene=46）
- 新增管理员 CRUD 接口（`/api/v1/admin/servers`，需 `LoginAuth` + `SuperAdmin` 中间件）:
  - `POST /admin/servers` — 创建服务器
  - `GET /admin/servers` — 列表查询
  - `GET /admin/servers/:id` — 单个查询
  - `PUT /admin/servers/:id` — 更新服务器信息
  - `DELETE /admin/servers/:id` — 删除服务器
  - `PUT /admin/servers/:id/public` — 设置公开可见性
  - `PUT /admin/servers/:id/enabled` — 设置启用/禁用状态
- 新增 gRPC 心跳被动创建：当收到未注册服务器的心跳时，自动创建 `fp_server` 记录（默认 `IsPublic=false, IsEnabled=true`）
- 新增用户查询接口（`/api/v1/servers`，需 `LoginAuth` 中间件）:
  - `GET /servers/game-profiles/online/mine` — 查询当前用户在线游戏账号
  - `GET /servers/players/online/check?uuid=&username=` — 按 UUID 或用户名检查玩家在线状态

#### gRPC 心跳逻辑调整

- `HeartbeatEvent` 处理新增被动服务器注册（调用 `ServerLogic.GetOrCreateByName`）
- `HeartbeatEvent` 处理新增禁用服务器过滤（`IsEnabled=false` 的服务器跳过状态更新）
- `PlayerJoinEvent` / `PlayerQuitEvent` 处理新增在线人数自动更新（通过 Redis set `SCard` 计算）

### Changed

#### 服务器状态接口过滤

- `GET /servers/status` 不再使用 `rdb.Keys` 扫描全部 Redis key，改为查询 `fp_server` 表中 `IsPublic=true AND IsEnabled=true` 的服务器
- `POST /servers/:name/refresh` 新增公开性检查，非公开或禁用的服务器返回 `404`
- 服务器状态响应新增 `server_display_name` 字段，来自 `fp_server.DisplayName`

### Fixed

- 修复 `SetServerPublicRequest` / `SetServerEnabledRequest` 中 `bool` 字段的 `binding:"required"` 标签导致 `false` 值被拒绝的 400 错误
