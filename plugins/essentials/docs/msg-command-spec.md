# /msg 命令规范

> **版本**: v1.0
> **目标读者**: Java 插件开发团队
> **Proto 包**: `frontleaves.essentials.v1`

---

## 1. 命令定义

| 字段 | 值 |
|------|-----|
| Command | `/msg <player>` |
| Aliases | `/message`, `/tell`, `/whisper`, `/t` |
| Permission | `essentials.msg` |
| Usage | `/msg <player>` 进入私聊模式，`/exit` 或 `/q` 退出 |

---

## 2. 行为规范

IRC /query 风格的持续私聊模式：

1. 玩家执行 `/msg <player>` 后进入与目标玩家的私聊会话。
2. 后续所有普通聊天消息自动路由到目标玩家，直到执行 `/exit` 或 `/q`。
3. 支持目标玩家名的 Tab 补全。
4. 禁止自聊（发送者与接收者相同时提示错误）。
5. 目标玩家不在线时提示错误。
6. 无参数执行时显示用法提示。

---

## 3. gRPC 交互

### 3.1 发送私聊（Plugin → Go 后端）

当玩家在私聊模式下发送消息，插件构造并发送 protobuf：

**Stream**: `PlayerEventService.PlayerEventStream`

**Event Type**: `PLAYER_EVENT_TYPE_PRIVATE_CHAT` (enum 值 = 8)

**Message**: `PlayerPrivateChatEvent`（oneof 字段名: `private_chat_event`，字段编号 18）

```protobuf
message PlayerPrivateChatEvent {
  string sender_uuid = 1;   // 发送者 UUID
  string sender_name = 2;   // 发送者用户名
  string receiver_name = 3; // 接收者用户名
  string message = 4;       // 消息内容
}
```

完整的 `PlayerEventStreamRequest` 构造示例：

```protobuf
PlayerEventStreamRequest {
  event_type: PLAYER_EVENT_TYPE_PRIVATE_CHAT,
  private_chat_event: PlayerPrivateChatEvent {
    sender_uuid: "uuid-string",
    sender_name: "PlayerName",
    receiver_name: "TargetName",
    message: "Hello!"
  }
}
```

### 3.2 接收私聊（Go 后端 → Plugin）

Go 后端将私聊消息推送给目标玩家所在的插件实例。

**Stream**: `PlayerMessageService.PlayerMessageStream`（Server-Streaming）

**Push Type**: `PLAYER_MESSAGE_PUSH_TYPE_PRIVATE` (enum 值 = 3)

**Payload**: `PrivateMessagePush`（oneof 字段名: `private_push`，字段编号 13）

```protobuf
message PrivateMessagePush {
  string sender_name = 1;  // 发送者用户名
  string message = 2;      // 消息内容
  string sender_uuid = 3;  // 发送者 UUID
}
```

完整的 `PlayerMessagePushResponse` 结构：

```protobuf
PlayerMessagePushResponse {
  base_response: BaseResponse { ... },
  push_type: PLAYER_MESSAGE_PUSH_TYPE_PRIVATE,
  private_push: PrivateMessagePush {
    sender_name: "SenderName",
    message: "Hello!",
    sender_uuid: "uuid-string"
  }
}
```

插件收到后，根据 `sender_uuid` / `sender_name` 定位当前服务器上的目标玩家，展示格式化的私聊消息。

---

## 4. i18n 消息键

| Key | 用途 | 示例值 |
|-----|------|--------|
| `essentials.msg.entered` | 进入私聊模式提示 | `You are now in private chat with {player}. Type /exit to leave.` |
| `essentials.msg.exited` | 退出私聊模式提示 | `You have left private chat mode.` |
| `essentials.msg.self` | 禁止自聊 | `You cannot send a message to yourself.` |
| `essentials.msg.offline` | 目标玩家不在线 | `Player {player} is not online.` |
| `essentials.msg.format.send` | 发送方消息格式 | `[You -> {player}] {message}` |
| `essentials.msg.format.receive` | 接收方消息格式 | `[{player} -> You] {message}` |
| `essentials.msg.no-target` | 缺少目标参数 | `Usage: /msg <player>` |
| `essentials.msg.no-permission` | 无权限 | `You do not have permission to use this command.` |

---

## 5. 实现备注

### 5.1 命令注册

- 命令类继承 `BasicCommand`（项目惯例）。
- 通过 `CommandInitializer` 注册。
- 在 `plugin.yml` 中声明命令及别名，绑定权限节点 `essentials.msg`。

### 5.2 会话状态管理

- 使用 `HashMap<UUID, String>` 维护每个玩家的当前私聊目标（玩家名）。
- 玩家退出服务器时清理对应条目（监听 `PlayerQuitEvent`）。
- 切换目标：重复执行 `/msg <other>` 会替换当前目标。
- 无目标时，普通聊天走正常公屏逻辑。

### 5.3 消息路由逻辑

```
玩家发送聊天消息
  → 检查是否处于私聊模式（HashMap 中有记录）
    → 是：构造 PlayerPrivateChatEvent，通过 gRPC 上报
    → 否：走正常 PlayerChatEvent 公屏流程
```

### 5.4 消息展示

收到 `PrivateMessagePush` 时：
- 对发送者：使用 `essentials.msg.format.send` 格式展示。
- 对接收者：使用 `essentials.msg.format.receive` 格式展示。

### 5.5 Proto 引用文件

| 文件 | 路径 |
|------|------|
| PlayerEventStream | `proto/essentials/v1/player_event.proto` |
| PlayerPrivateChatEvent | `proto/essentials/v1/player_event.proto` |
| PlayerMessageStream | `proto/essentials/v1/player_message.proto` |
| PrivateMessagePush | `proto/essentials/v1/player_message.proto` |
