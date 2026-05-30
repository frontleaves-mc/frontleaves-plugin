# Economy gRPC 对接文档（Java 插件）

> 面向 Minecraft 插件开发者（Java/Bukkit），说明如何对接 FrontLeaves 经济审计系统的 gRPC 服务。

---

## 1. 概述

FrontLeaves 经济审计系统提供两个 gRPC 流：

| RPC | 模式 | 方向 | 用途 |
|-----|------|------|------|
| `RecordTransactionStream` | Client Streaming | MC → Go | 上报交易流水日志 |
| `BalanceStream` | Bidi Streaming | MC ⇄ Go | 接收余额查询、返回余额数据 |

**架构约束**：Go 服务**永远是 Server**，MC 插件**永远是 Client**。MC 不需要启动任何 gRPC Server。

---

## 2. 连接配置

### 2.1 服务地址

| 项目 | 值 |
|------|-----|
| **gRPC 服务地址** | `localhost:5600`（默认） |
| **通信协议** | gRPC（Plaintext，内网直连无需 TLS） |
| **Proto 包** | `frontleaves.economy.v1` |
| **Proto 文件** | `proto/economy/v1/economy.proto` |

### 2.2 认证

**所有 RPC 调用必须携带 gRPC Metadata**：

| Metadata Key | 类型 | 必填 | 说明 |
|-------------|------|------|------|
| `plugin-secret-key` | String | ✅ 是 | 插件密钥，由管理员配发，Go 端验证合法性 |
| `plugin-name` | String | ❌ 否 | 插件名称，仅用于日志标识 |

> **密钥获取**：通过网页管理面板 `/admin/plugin-credentials` 生成，每个插件持有独立密钥。

### 2.3 Java 连接代码

```java
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.stub.MetadataUtils;

// 1. 创建 Channel（复用，全局单例）
ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 5600)
    .usePlaintext()          // 内网直连
    .keepAliveTime(30, TimeUnit.SECONDS)  // 长连接保活
    .keepAliveTimeout(10, TimeUnit.SECONDS)
    .build();

// 2. 附加认证 Metadata
Metadata authHeader = new Metadata();
authHeader.put(
    Metadata.Key.of("plugin-secret-key", Metadata.ASCII_STRING_MARSHALLER),
    "your-secret-key-here"
);
authHeader.put(
    Metadata.Key.of("plugin-name", Metadata.ASCII_STRING_MARSHALLER),
    "my-economy-plugin"
);

// 3. 创建 Stub（复用）
TransactionLogServiceGrpc.TransactionLogServiceStub stub = TransactionLogServiceGrpc
    .newStub(channel)
    .withInterceptors(MetadataUtils.newAttachHeadersInterceptor(authHeader));
```

---

## 3. 交易流水上报（Client Streaming）

### 3.1 接口签名

```protobuf
rpc RecordTransactionStream(stream RecordTransactionRequest)
    returns (RecordTransactionResponse);
```

- **流模式**：Client Streaming — MC 连续发送多条日志，EOF 后 Go 批量写入
- **幂等性**：Go 侧通过 `idempotency_key` 防重，重复提交自动跳过
- **批量写入**：流中的所有有效记录在一个事务中写入 PostgreSQL

### 3.2 请求消息

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `player_uuid` | `string` | ✅ | 玩家 UUID（标准格式，如 `550e8400-e29b-41d4-a716-446655440000`） |
| `player_name` | `string` | ✅ | 玩家名称（快照值，用于审计） |
| `amount` | `int64` | ✅ | 交易金额，**单位：分（fen）**。正数为收入，负数为支出 |
| `type` | `TransactionType` | ✅ | 交易类型：`TRANSFER=1`（转账）、`ADMIN=2`（管理员操作） |
| `counterparty_uuid` | `string` | ❌ | 对方 UUID（转账时填写） |
| `counterparty_name` | `string` | ❌ | 对方名称（转账时填写） |
| `operator_uuid` | `string` | ❌ | 操作者 UUID（管理员操作时填写） |
| `operator_name` | `string` | ❌ | 操作者名称（管理员操作时填写） |
| `comment` | `string` | ✅ | 备注信息（可为空字符串） |
| `idempotency_key` | `string` | ✅ | **幂等键**，同一笔交易多次上报只记录一次。建议格式：`{插件名}:{交易唯一ID}` |
| `timestamp` | `int64` | ✅ | 交易时间戳（**Unix 毫秒**） |

### 3.3 交易类型枚举

| 枚举值 | 编号 | 含义 | 使用场景 |
|--------|------|------|----------|
| `TRANSACTION_TYPE_TRANSFER` | 1 | 玩家间转账 | `/pay` 命令 |
| `TRANSACTION_TYPE_ADMIN` | 2 | 管理员操作 | `/eco set/add/remove/reset` 命令 |

> ⚠️ **不记录**：`/eco giveall`、`/eco takeall` 等全局操作不记入审计日志。

### 3.4 幂等键格式

```text
建议格式：{插件名}:{事件ID}:{玩家UUID}:{交易序号}

示例：
  vault:economy-transfer:550e8400-e29b-41d4-a716-446655440000:1704067200000
  vault:economy-admin:550e8400-e29b-41d4-a716-446655440000:1704067200001
```

### 3.5 Java 示例

```java
import io.grpc.stub.StreamObserver;
import frontleaves.economy.v1.Economy.*;

public void reportTransactions(List<Transaction> transactions) {
    // 创建 Client Streaming Observer
    StreamObserver<RecordTransactionRequest> stream = stub.recordTransactionStream(
        new StreamObserver<RecordTransactionResponse>() {
            @Override
            public void onNext(RecordTransactionResponse response) {
                plugin.getLogger().info("交易流水上报完成: " + response.getMessage());
            }

            @Override
            public void onError(Throwable t) {
                plugin.getLogger().severe("交易流水上报失败: " + t.getMessage());
                // 建议：缓存失败日志，定时重试
            }

            @Override
            public void onCompleted() {
                // 流正常关闭
            }
        }
    );

    // 逐条发送交易记录
    for (Transaction tx : transactions) {
        RecordTransactionRequest req = RecordTransactionRequest.newBuilder()
            .setPlayerUuid(tx.getPlayerUUID().toString())
            .setPlayerName(tx.getPlayerName())
            .setAmount(tx.getAmountFen())        // int64，单位：分
            .setType(tx.getTransactionType())     // TRANSFER=1 / ADMIN=2
            .setCounterpartyUuid(tx.getCounterpartyUUID())  // 可选
            .setCounterpartyName(tx.getCounterpartyName())  // 可选
            .setOperatorUuid(tx.getOperatorUUID())          // 可选
            .setOperatorName(tx.getOperatorName())          // 可选
            .setComment(tx.getComment())
            .setIdempotencyKey(tx.getIdempotencyKey())
            .setTimestamp(tx.getTimestampMs())    // Unix 毫秒
            .build();

        stream.onNext(req);
    }

    // 发送完毕，关闭流 → Go 端批量写入
    stream.onCompleted();
}
```

### 3.6 注意事项

1. **金额单位**：全部使用 **分（fen）**（`int64`），杜绝浮点精度问题。前端展示时由 Go 转换为 `X.XX` 格式
2. **幂等键唯一性**：使用 `UNIQUE INDEX` 约束，重复的 `idempotency_key` 会被静默跳过
3. **错误处理**：单条记录的 `player_uuid` 无效或 `idempotency_key` 为空时 **只跳过不中断** 整个流
4. **批量最佳实践**：建议每 50 条或每 5 秒 flush 一次，避免长时间不 `onCompleted()` 导致内存积压

---

## 4. 余额查询（Bidi Streaming）

### 4.1 接口签名

```protobuf
rpc BalanceStream(stream BalanceResult) returns (stream BalanceQuery);
```

- **流模式**：Bidi Streaming — MC 接收 Go 的查询请求，通过同一流返回余额数据
- **多路复用**：Go 通过 `request_id` 匹配请求与响应，支持多个 Web 请求并发查询
- **长连接**：MC 启动时建立，断开后 Go 自动标记余额查询不可用，等待重连

### 4.2 消息格式

**BalanceQuery（Go → MC）**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `request_id` | `uint64` | 请求 ID（单调递增），用于 MC 匹配响应 |
| `player_uuid` | `string` | 待查询的玩家 UUID |

**BalanceResult（MC → Go）**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `request_id` | `uint64` | **必须与请求的 request_id 一致** |
| `player_uuid` | `string` | 玩家 UUID |
| `balance` | `int64` | 余额（**单位：分 fen**）。`success=false` 时为 0 |
| `currency` | `string` | 货币类型（如 `"CNY"`） |
| `success` | `bool` | 是否查询成功 |
| `error_message` | `string` | 错误描述（`success=false` 时有值） |

### 4.3 数据流

```
MC（Client）                            Go（Server）
   │                                       │
   │── 建立 Bidi Stream ─────────────────→│  BalanceStream 开始
   │                                       │
   │←─ recvLoop 启动，等待查询 ←───────────│  Go 等待 Web 请求
   │                                       │
   │←─ BalanceQuery {request_id:1, uuid} ──│  Web 用户查询余额
   │                                       │
   │── 查 Vault，获取余额 ──→               │
   │                                       │
   │── BalanceResult {request_id:1,        │
   │       balance:100000, success:true} ──│  Go 匹配 request_id，返回给 Web
   │                                       │
   │←─ BalanceQuery {request_id:2, uuid} ──│  另一个 Web 用户查询
   │── BalanceResult {request_id:2, ...} ──│
   │                                       │
```

### 4.4 Java 示例

```java
import io.grpc.stub.StreamObserver;
import frontleaves.economy.v1.Economy.*;
import java.util.concurrent.ConcurrentHashMap;

public class BalanceStreamHandler {
    private volatile StreamObserver<BalanceResult> balanceStream;
    private final ConcurrentHashMap<Long, CompletableFuture<Long>> pendingQueries = new ConcurrentHashMap<>();

    /**
     * 启动余额查询 Bidi Stream（插件启动时调用一次）。
     * 这个方法会阻塞直到流断开，建议在独立线程中运行。
     */
    public void startBalanceStream() {
        while (!Thread.currentThread().isInterrupted()) {
            try {
                CountDownLatch latch = new CountDownLatch(1);

                balanceStream = stub.balanceStream(new StreamObserver<BalanceQuery>() {
                    @Override
                    public void onNext(BalanceQuery query) {
                        // 收到查询请求
                        long requestId = query.getRequestId();
                        String playerUUID = query.getPlayerUuid();

                        // 从 Vault 查余额
                        Player player = Bukkit.getPlayer(UUID.fromString(playerUUID));
                        if (player == null) {
                            // 玩家不在线，返回无玩家 UUID（后端会做异常处理）
                            plugin.getLogger().warning("玩家不在线，无法查询余额: " + playerUUID);
                        }

                        Economy economy = plugin.getEconomy();
                        double balanceDollars = economy.getBalance(player);
                        long balanceFen = (long) (balanceDollars * 100);  // 元 → 分

                        // 构造并发送响应
                        BalanceResult result = BalanceResult.newBuilder()
                            .setRequestId(requestId)
                            .setPlayerUuid(playerUUID)
                            .setBalance(balanceFen)
                            .setCurrency("CNY")
                            .setSuccess(true)
                            .build();

                        balanceStream.onNext(result);
                    }

                    @Override
                    public void onError(Throwable t) {
                        plugin.getLogger().severe("余额流错误: " + t.getMessage());
                        latch.countDown();
                    }

                    @Override
                    public void onCompleted() {
                        plugin.getLogger().info("余额流正常关闭");
                        latch.countDown();
                    }
                });

                // 等待流结束
                latch.await();

            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                break;
            } catch (Exception e) {
                plugin.getLogger().severe("余额流异常: " + e.getMessage());
            }

            // 断开后等待 5 秒重连
            if (!Thread.currentThread().isInterrupted()) {
                try {
                    TimeUnit.SECONDS.sleep(5);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    break;
                }
            }
        }
    }
}
```

### 4.5 注意事项

1. **Vault 金额转换**：Vault 使用 `double`（元），转换为 gRPC 的 `int64`（分）时需 `*100` 并转为 `long`
2. **request_id 匹配**：`BalanceResult.request_id` 必须与 `BalanceQuery.request_id` 一致，否则 Go 无法匹配请求
3. **流重连**：流断开后需要重新建立（Go 侧不缓存查询请求，流断开期间余额接口返回 503）
4. **离线玩家**：如果玩家不在线，返回 `success=false` 并附带 `error_message`
5. **并发安全**：Go 侧 5s 超时，MC 侧查 Vault 应尽快返回（< 1s）

---

## 5. 运行清单

### 5.1 插件启动

```java
@Override
public void onEnable() {
    // 1. 创建 gRPC Channel（复用）
    this.channel = createGrpcChannel();

    // 2. 创建认证 Stub（复用）
    this.stub = createAuthenticatedStub();

    // 3. 启动余额查询 Bidi Stream（独立线程）
    this.balanceStreamHandler = new BalanceStreamHandler(this, stub);
    new Thread(balanceStreamHandler::startBalanceStream, "economy-balance-stream").start();
}

@Override
public void onDisable() {
    // 清理资源
    if (channel != null) {
        channel.shutdownNow();
    }
}
```

### 5.2 交易上报时机

| 事件 | RPC | idempotency_key 示例 |
|------|-----|---------------------|
| `/pay` 转账成功 | `RecordTransactionStream` | `vault:transfer:{txID}` |
| `/eco set` 管理员设余额 | `RecordTransactionStream` | `vault:admin_set:{txID}` |
| `/eco add` 管理员加余额 | `RecordTransactionStream` | `vault:admin_add:{txID}` |
| `/eco remove` 管理员扣余额 | `RecordTransactionStream` | `vault:admin_remove:{txID}` |
| `/eco reset` 管理员重置余额 | `RecordTransactionStream` | `vault:admin_reset:{txID}` |
| `/eco giveall` 全局赠送 | **不上报** | — |
| `/eco takeall` 全局扣除 | **不上报** | — |

### 5.3 配置清单

```yaml
# plugin.yml 或 config.yml
grpc:
  host: localhost
  port: 5600
  secret-key: "your-secret-key-from-admin-panel"
```

---

## 6. 错误处理

| 场景 | gRPC Status | 插件处理建议 |
|------|-------------|-------------|
| 缺少 `plugin-secret-key` | `UNAUTHENTICATED` | 检查配置，确保 Metadata 正确 |
| 密钥无效 | `UNAUTHENTICATED` | 联系管理员更换密钥 |
| 流中断 | `INTERNAL` | 日志记录 + 自动重连 |
| 单条记录校验失败 | 静默跳过 | 校验 `player_uuid` 和 `idempotency_key` |
| 批量写入失败 | `INTERNAL` | 流中所有记录回滚，插件缓存后重试 |

---

## 7. 附录：完整 Proto 定义

```protobuf
syntax = "proto3";
package frontleaves.economy.v1;
option go_package = "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/economy/v1;economypb";
import "link/base.proto";

enum TransactionType {
  TRANSACTION_TYPE_UNSPECIFIED = 0;
  TRANSACTION_TYPE_TRANSFER = 1;   // 玩家间转账
  TRANSACTION_TYPE_ADMIN = 2;      // 管理员操作
}

service TransactionLogService {
  // Client Streaming：MC 上报交易流水
  rpc RecordTransactionStream(stream RecordTransactionRequest) returns (RecordTransactionResponse);

  // Bidi Streaming：Go 查询余额 ← → MC 返回数据
  rpc BalanceStream(stream BalanceResult) returns (stream BalanceQuery);
}

message RecordTransactionRequest {
  string player_uuid = 1;
  string player_name = 2;
  int64 amount = 3;                     // 单位：分
  TransactionType type = 4;
  optional string counterparty_uuid = 5;
  optional string counterparty_name = 6;
  optional string operator_uuid = 7;
  optional string operator_name = 8;
  string comment = 9;
  string idempotency_key = 10;
  int64 timestamp = 11;                 // Unix 毫秒
}

message RecordTransactionResponse {
  xBase.BaseResponse base_response = 1;
  bool success = 2;
  string message = 3;
}

message BalanceQuery {
  uint64 request_id = 1;
  string player_uuid = 2;
}

message BalanceResult {
  xBase.BaseResponse base_response = 1;
  uint64 request_id = 2;
  string player_uuid = 3;
  int64 balance = 4;                    // 单位：分
  string currency = 5;
  bool success = 6;
  string error_message = 7;
}
```

---

> **文档版本**: v2.0 | **最后更新**: 2026-05-31 | **架构**: Go=Server, MC=Client（Bidi Stream）
