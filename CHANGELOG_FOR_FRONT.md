# CHANGELOG_FOR_FRONT.md

前端 API 对接变更日志 — 新增消息模块 6 个 RESTful 接口 + 1 个 SSE 实时流。

---

## 接口总览

| # | 方法 | 路径 | 权限 | 说明 |
|---|------|------|------|------|
| 1 | GET | `/api/v1/admin/messages/chat` | Admin | 查询所有聊天记录 |
| 2 | GET | `/api/v1/admin/messages/commands` | Admin | 查询所有指令记录 |
| 3 | GET | `/api/v1/user/messages/chat` | Player | 查询我的聊天记录 |
| 4 | GET | `/api/v1/user/messages/commands` | Player | 查询我的指令记录 |
| 5 | GET | `/api/v1/user/messages/chat/stream` | Player | SSE 实时聊天流 |
| 6 | POST | `/api/v1/user/messages/chat` | Player | 发送聊天消息 |

---

## 管理端接口

### 1. 查询所有聊天记录

```
GET /api/v1/admin/messages/chat
Authorization: Bearer <token>
```

**Query 参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `page` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 20 | 每页数量（最大 100） |
| `player_uuid` | string(uuid) | 否 | - | 按玩家 UUID 筛选 |
| `server_name` | string | 否 | - | 按服务器名称筛选 |
| `source` | int | 否 | - | 按来源筛选：1=Game, 2=Web |

**响应体：**

```json
{
  "output": "Success",
  "message": "查询成功",
  "data": {
    "list": [
      {
        "id": "1234567890",
        "player_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "player_name": "Steve",
        "server_name": "survival",
        "world_name": "world",
        "message": "Hello World",
        "source": 1,
        "sender_id": null
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

**`source` 字段含义：**

| 值 | 含义 | 说明 |
|----|------|------|
| 1 | Game | 游戏内发送 |
| 2 | Web | 网页端发送 |

Web 来源消息的 `player_uuid` 可能为空字符串（`""`），此时 `sender_id` 字段有值。

---

### 2. 查询所有指令记录

```
GET /api/v1/admin/messages/commands
Authorization: Bearer <token>
```

**Query 参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `page` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 20 | 每页数量（最大 100） |
| `player_uuid` | string(uuid) | 否 | - | 按玩家 UUID 筛选 |
| `server_name` | string | 否 | - | 按服务器名称筛选 |

**响应体：**

```json
{
  "output": "Success",
  "message": "查询成功",
  "data": {
    "list": [
      {
        "id": "1234567891",
        "player_uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "player_name": "Steve",
        "server_name": "survival",
        "world_name": "world",
        "command": "/gamemode creative"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

---

## 用户端接口

### 3. 查询我的聊天记录

```
GET /api/v1/user/messages/chat
Authorization: Bearer <token>
```

返回当前登录用户关联的所有游戏角色的聊天记录（通过 GameProfile 自动关联）。

**Query 参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `page` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 20 | 每页数量（最大 100） |

**响应体：** 同管理端聊天记录接口格式。

**注意：** 若用户未绑定任何游戏角色，`list` 为空数组，`total` 为 0。

---

### 4. 查询我的指令记录

```
GET /api/v1/user/messages/commands
Authorization: Bearer <token>
```

返回当前登录用户关联的所有游戏角色的指令使用记录。

**Query 参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `page` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 20 | 每页数量（最大 100） |

**响应体：** 同管理端指令记录接口格式。

---

### 5. SSE 实时聊天流

```
GET /api/v1/user/messages/chat/stream
Authorization: Bearer <token>
```

通过 Server-Sent Events 建立实时消息流连接。

**响应头：**

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

**事件类型：**

#### `init` — 连接建立时推送最近 50 条消息

```
event: init
data: [{"id":"123","player_name":"Steve","server_name":"survival","message":"Hello","source":1}]
```

`init` 数据为 `ChatLogResponse[]` JSON 数组，按时间正序排列。

#### `chat` — 实时聊天消息推送

```
event: chat
data: {"player_name":"Steve","server_name":"survival","message":"Hello World","source":1}
```

**SSEChatMessage 结构：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `player_name` | string | 发送者名称 |
| `server_name` | string | 服务器名称（Web 消息为空） |
| `message` | string | 消息内容 |
| `source` | int | 1=Game, 2=Web |

**前端实现要求：**

1. 使用 `EventSource` 或 `fetch` + `ReadableStream` 连接
2. 监听 `init` 事件初始化消息列表
3. 监听 `chat` 事件追加新消息
4. 连接断开时自动重连（建议 3-5 秒延迟）
5. Bearer Token 通过 URL 参数 `?token=xxx` 或在连接时通过 `Authorization` header 传递（取决于 SSE 实现方式）

> **注意：** 标准 `EventSource` 不支持自定义 Header。如使用 `EventSource`，需通过反向代理将 `query.token` 映射为 `Authorization` header，或改用 `fetch` API 手动处理 SSE 流。

---

### 6. 发送聊天消息

```
POST /api/v1/user/messages/chat
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

```json
{
  "message": "Hello from web!"
}
```

| 字段 | 类型 | 必填 | 约束 | 说明 |
|------|------|------|------|------|
| `message` | string | 是 | 1-500 字符 | 消息内容 |

**响应体：**

```json
{
  "output": "Success",
  "message": "发送成功",
  "data": null
}
```

**行为说明：**
- 消息记录到数据库（source=2, sender_id=当前用户ID）
- 消息推送到 MC 游戏内（通过 gRPC PlayerMessageStream）
- 消息广播到所有已连接的 SSE 客户端
- 即使 MC 服务器不在线，消息仍会记录并广播到 Web 端

---

## 数据模型参考

### ChatLogResponse

```typescript
interface ChatLogResponse {
  id: string
  player_uuid: string       // Web 消息可能为空字符串
  player_name: string
  server_name: string
  world_name: string
  message: string
  source: 1 | 2             // 1=Game, 2=Web
  sender_id?: string        // Web 消息才有值
}
```

### CommandLogResponse

```typescript
interface CommandLogResponse {
  id: string
  player_uuid: string
  player_name: string
  server_name: string
  world_name: string
  command: string
}
```

### PaginatedResponse\<T\>

```typescript
interface PaginatedResponse<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}
```

### SSEChatMessage

```typescript
interface SSEChatMessage {
  player_name: string
  server_name: string
  message: string
  source: 1 | 2
}
```
