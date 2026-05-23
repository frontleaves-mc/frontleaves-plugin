# CHANGELOG_FOR_MC.md

Essentials 插件对接变更日志 — 后端新增 gRPC 能力，MC 插件需配合修改。

---

## 新增：PlayerCommandEvent 玩家指令事件上报

`player_event.proto` 新增事件类型 `PLAYER_EVENT_TYPE_PLAYER_COMMAND = 7`。

### 触发时机

当玩家在游戏中执行任意指令时，插件应通过 `PlayerEventStream` 上报该事件。

### Proto 定义

```protobuf
enum PlayerEventType {
  // ... 已有类型 1-6 ...
  PLAYER_EVENT_TYPE_PLAYER_COMMAND = 7;  // 新增
}

message PlayerEventStreamRequest {
  PlayerEventType event_type = 1;
  oneof event {
    // ... 已有字段 11-16 ...
    PlayerCommandEvent player_command_event = 17;  // 新增
  }
}

message PlayerCommandEvent {
  string player_uuid = 1;   // 执行指令的玩家 UUID
  string player_name = 2;   // 玩家用户名
  string server_name = 3;   // 服务器名称
  string world_name = 4;    // 世界名称
  string command = 5;       // 完整指令字符串（包含 / 前缀）
}
```

### 插件侧实现要求

1. 监听 `PlayerCommandPreprocessEvent` 或类似事件
2. 构造 `PlayerEventStreamRequest`，设置 `event_type = PLAYER_EVENT_TYPE_PLAYER_COMMAND`
3. 填充 `player_command_event` 字段
4. 通过已有的 `PlayerEventStream` 客户端流发送

---

## 新增：PlayerMessageService 消息推送接收

新 gRPC 服务 `PlayerMessageService`，用于后端主动向 MC 服务器推送消息。

### 服务定义

```protobuf
service PlayerMessageService {
  rpc PlayerMessageStream(google.protobuf.Empty) returns (stream PlayerMessagePushResponse);
}
```

### 连接方式

- 与 `PlayerEventStream` 相同的 gRPC 通道
- 需在 metadata 中携带 `plugin-name` 用于身份验证
- 调用 `PlayerMessageStream(Empty)` 建立服务端流，持续监听推送消息

### 推送消息类型

```protobuf
enum PlayerMessagePushType {
  PLAYER_MESSAGE_PUSH_TYPE_UNSPECIFIED = 0;
  PLAYER_MESSAGE_PUSH_TYPE_CHAT = 1;     // Web 聊天消息转发
  PLAYER_MESSAGE_PUSH_TYPE_SYSTEM = 2;   // 系统消息
}
```

#### CHAT 类型 (`PlayerChatPush`)

| 字段 | 类型 | 说明 |
|------|------|------|
| `sender_name` | string | Web 端发送者用户名 |
| `message` | string | 消息内容 |
| `source` | string | 固定值 `"web"` |

插件收到后应将消息以 `[Web] <sender_name>: <message>` 格式在游戏内广播。

#### SYSTEM 类型 (`SystemMessagePush`)

| 字段 | 类型 | 说明 |
|------|------|------|
| `title` | string | 系统消息标题 |
| `content` | string | 系统消息内容 |

插件收到后应以标题+内容格式展示给所有在线玩家（ActionBar / Chat / Title 均可）。

### 消息结构

```protobuf
message PlayerMessagePushResponse {
  xBase.BaseResponse base_response = 1;
  PlayerMessagePushType push_type = 2;
  oneof payload {
    PlayerChatPush chat_push = 11;
    SystemMessagePush system_push = 12;
  }
}
```

### 插件侧实现要求

1. 在插件启动时，建立 `PlayerMessageStream` 服务端流连接
2. 在独立线程中持续 `readNext()` 接收推送
3. 根据 `push_type` 分发处理：
   - `CHAT` → 解析 `chat_push`，游戏内广播聊天消息
   - `SYSTEM` → 解析 `system_push`，游戏内展示系统消息
4. 连接断开时自动重连（建议指数退避策略）

---

## 数据库变更说明

无需 MC 插件侧关注，所有数据库操作由后端服务处理。

新增表：
- `fp_player_command_log` — 玩家指令日志

新增字段（`fp_player_chat_log` 表）：
- `source` (smallint, default 1) — 消息来源：1=Game, 2=Web
- `sender_id` (bigint, nullable) — Web 发送者用户 ID
- `player_uuid` 改为 nullable — Web 端消息无关联玩家
