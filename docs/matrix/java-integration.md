# Matrix Telemetry Java 插件对接教程

> 从零开始，手把手完成 Java 插件与 Matrix 遥测系统的对接。

## 这篇教程适合谁

你是 Minecraft 服务器的 Java 插件开发者（Bukkit/Paper），需要把玩家行为数据上报到 FrontLeaves 的 Matrix 遥测后端。这篇文档会带你走完全程：从创建项目到数据成功到达 Go 服务。

---

## 1 对接概述

Matrix 遥测系统分两半：**Go 后端**（本仓库 `frontleaves-plugin`）和 **Java 插件**（你写的部分）。两者之间用 gRPC Client Stream 通信。

```
Java 插件                              Go 后端
┌─────────────────┐                   ┌──────────────────────┐
│ Bukkit Event    │                   │ TelemetryStream      │
│   ↓             │    gRPC           │   ↓ dispatch         │
│ buildRequest()  │ ──── Stream ────→ │ PlayerSession        │
│   ↓             │                   │   ├ 统计聚合 → PG     │
│ stream.onNext() │                   │   └ 反作弊检测 → Redis│
└─────────────────┘                   └──────────────────────┘
```

Java 端做的事很纯粹：监听 Bukkit 事件，转成 Protobuf 消息，通过 Stream 发出去。所有业务逻辑都在 Go 端处理。

---

## 2 前置条件

动手之前，确认以下东西准备好了。

### 2.1 环境要求

| 条件 | 最低版本 |
|------|---------|
| Java | 21+ |
| Paper API | 1.21.1-R0.1-SNAPSHOT |
| `frontleaves-lib` | 1.0.0+ |
| gRPC | 1.62.2 |
| Protobuf | 3.25.3 |

`frontleaves-lib` 负责 gRPC Channel 管理、认证注入和连接监控。如果你的插件体系里已经有了它，直接用就行。

### 2.2 添加 Maven 依赖

```xml
<properties>
    <grpc.version>1.62.2</grpc.version>
    <protobuf.version>3.25.3</protobuf.version>
</properties>

<dependencies>
    <!-- Paper API -->
    <dependency>
        <groupId>io.papermc.paper</groupId>
        <artifactId>paper-api</artifactId>
        <version>1.21.1-R0.1-SNAPSHOT</version>
        <scope>provided</scope>
    </dependency>

    <!-- frontleaves-lib（提供 gRPC Channel 和认证） -->
    <dependency>
        <groupId>com.frontleaves.plugins</groupId>
        <artifactId>frontleaves-lib</artifactId>
        <version>1.0.0</version>
        <scope>provided</scope>
    </dependency>

    <!-- gRPC -->
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-netty-shaded</artifactId>
        <version>${grpc.version}</version>
        <scope>provided</scope>
    </dependency>
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-protobuf</artifactId>
        <version>${grpc.version}</version>
        <scope>provided</scope>
    </dependency>
    <dependency>
        <groupId>io.grpc</groupId>
        <artifactId>grpc-stub</artifactId>
        <version>${grpc.version}</version>
        <scope>provided</scope>
    </dependency>

    <!-- Protobuf -->
    <dependency>
        <groupId>com.google.protobuf</groupId>
        <artifactId>protobuf-java</artifactId>
        <version>${protobuf.version}</version>
        <scope>provided</scope>
    </dependency>
</dependencies>
```

### 2.3 配置 Protobuf 编译插件

在 `pom.xml` 的 `<build>` 中加上 protobuf-maven-plugin，这样 `mvn compile` 会自动把 `.proto` 文件编译成 Java 类。

```xml
<build>
    <extensions>
        <extension>
            <groupId>kr.motd.maven</groupId>
            <artifactId>os-maven-plugin</artifactId>
            <version>1.7.1</version>
        </extension>
    </extensions>
    <plugins>
        <plugin>
            <groupId>org.xolstice.maven.plugins</groupId>
            <artifactId>protobuf-maven-plugin</artifactId>
            <version>0.6.1</version>
            <configuration>
                <protocArtifact>
                    com.google.protobuf:protoc:${protobuf.version}:exe:${os.detected.classifier}
                </protocArtifact>
                <pluginId>grpc-java</pluginId>
                <pluginArtifact>
                    io.grpc:protoc-gen-grpc-java:${grpc.version}:exe:${os.detected.classifier}
                </pluginArtifact>
            </configuration>
            <executions>
                <execution>
                    <goals>
                        <goal>compile</goal>
                        <goal>compile-custom</goal>
                    </goals>
                </execution>
            </executions>
        </plugin>
    </plugins>
</build>
```

### 2.4 放置 Proto 文件

从 `frontleaves-plugin` 仓库复制两个 proto 文件到你的项目中：

```
src/main/proto/
├── link/
│   └── base.proto                           # 基础依赖
└── matrix/
    └── v1/
        └── matrix_telemetry.proto           # 遥测消息定义
```

复制命令：

```bash
cp frontleaves-plugin/proto/link/base.proto src/main/proto/link/
cp frontleaves-plugin/proto/matrix/v1/matrix_telemetry.proto src/main/proto/matrix/v1/
```

运行 `mvn compile`，生成的 Java 代码在 `target/generated-sources/protobuf/` 下。你会得到两个关键的类：

- `MatrixTelemetryServiceGrpc.java` — gRPC Stub
- `MatrixTelemetryProto.java` — 所有消息类型

---

## 3 快速开始：最小可运行示例

先把最核心的流程跑通。下面的代码展示了如何建立 gRPC Client Stream 并发送一条 PlayerJoin 事件。

### Step 1：建立 Stream 连接

```java
// 通过 frontleaves-lib 获取 Channel
FrontleavesLib lib = FrontleavesLib.getInstance()
        .orElseThrow(() -> new IllegalStateException("frontleaves-lib 未加载"));

ManagedChannel channel = lib.createChannel("matrix-telemetry");

// 创建异步 Stub
MatrixTelemetryServiceGrpc.MatrixTelemetryServiceStub asyncStub =
        MatrixTelemetryServiceGrpc.newStub(channel);

// 建立 Client Stream
StreamObserver<MatrixTelemetryProto.MatrixTelemetryRequest> requestObserver =
        asyncStub.telemetryStream(new StreamObserver<>() {
            @Override
            public void onNext(MatrixTelemetryProto.MatrixTelemetryResponse value) {
                // 服务端的最终响应，一般不需要处理
            }

            @Override
            public void onError(Throwable t) {
                // 连接出错，后续会讲重连
            }

            @Override
            public void onCompleted() {
                // 服务端主动关闭了流
            }
        });
```

### Step 2：发送一条事件

```java
// 玩家加入时
requestObserver.onNext(
    MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
        .setServerName("survival")
        .setPlayerJoin(MatrixTelemetryProto.PlayerJoinEvent.newBuilder()
            .setPlayerUuid(player.getUniqueId().toString())
            .setPlayerName(player.getName())
            .setWorldName(player.getWorld().getName())
            .setIpAddress(player.getAddress().getAddress().getHostAddress())
            .setTimestamp(System.currentTimeMillis())
            .build())
        .build()
);
```

到这里，你已经完成了最基础的对接。接下来的内容会逐步完善：重连、心跳、全部事件类型。

---

## 4 事件类型速查表

Proto 文件里定义了 18 种事件，通过 `oneof payload` 区分。下表列出了每种事件对应的 Proto 消息、字段编号和触发它的 Bukkit Paper API 事件。

### 连接事件（必需）

| 事件 | Proto 字段 | 编号 | Bukkit 事件 | 说明 |
|------|-----------|------|------------|------|
| 玩家加入 | `player_join` | 11 | `PlayerJoinEvent` | 必须是会话第一个事件 |
| 玩家退出 | `player_quit` | 12 | `PlayerQuitEvent` | 必须是会话最后一个事件 |

### 心跳快照

| 事件 | Proto 字段 | 编号 | 建议频率 |
|------|-----------|------|---------|
| 遥测心跳 | `telemetry_tick` | 13 | 每 20-40 tick (1-2 秒) |

TelemetryTick 不是由某个 Bukkit 事件触发的，而是通过 BukkitScheduler 定时采集。

### 方块事件

| 事件 | Proto 字段 | 编号 | Bukkit 事件 |
|------|-----------|------|------------|
| 方块破坏 | `block_break` | 14 | `BlockBreakEvent` |
| 方块放置 | `block_place` | 15 | `BlockPlaceEvent` |

### 实体/战斗事件

| 事件 | Proto 字段 | 编号 | Bukkit 事件 |
|------|-----------|------|------------|
| 实体击杀 | `entity_kill` | 16 | `EntityDeathEvent` |
| 实体伤害 | `entity_damage` | 17 | `EntityDamageByEntityEvent` |
| 玩家受伤 | `player_damage` | 18 | `EntityDamageByEntityEvent`（攻击者是实体） |
| 玩家死亡 | `player_death` | 19 | `PlayerDeathEvent` |

### 物品/背包事件

| 事件 | Proto 字段 | 编号 | Bukkit 事件 |
|------|-----------|------|------------|
| 物品丢弃 | `item_drop` | 20 | `PlayerDropItemEvent` |
| 物品拾取 | `item_pickup` | 21 | `EntityPickupItemEvent` |
| 背包操作 | `inventory_action` | 22 | `InventoryClickEvent` |

### 玩家行为事件

| 事件 | Proto 字段 | 编号 | Bukkit 事件 |
|------|-----------|------|------------|
| 聊天消息 | `player_chat` | 23 | `AsyncChatEvent` |
| 命令执行 | `player_command` | 24 | `PlayerCommandPreprocessEvent` |
| 状态切换 | `player_toggle` | 25 | `PlayerToggleSneakEvent` 等 |
| 传送 | `teleport` | 26 | `PlayerTeleportEvent` |
| 重生 | `respawn` | 27 | `PlayerRespawnEvent` |
| 游戏模式变更 | `game_mode_change` | 28 | `PlayerGameModeChangeEvent` |

---

## 5 心跳快照实现

TelemetryTick 是 Matrix 的核心心跳数据。它不是事件驱动的，而是定时采集每个在线玩家的完整状态快照：位置、血量、速度、手上的物品、药水效果，等等。

Go 后端用它做两件事：反作弊的速度检测，以及玩家状态的实时画像。

下面是完整的实现。

### 5.1 BukkitScheduler 定时任务

在插件主类的 `onEnable` 中注册异步定时任务：

```java
@Override
public void onEnable() {
    // ... 前面的初始化代码 ...

    int tickInterval = getConfig().getInt("telemetry.tick-interval-ticks", 20);

    // 异步定时采集所有在线玩家的状态快照
    tickTask = Bukkit.getScheduler().runTaskTimerAsynchronously(
        this,
        () -> {
            for (var player : Bukkit.getOnlinePlayers()) {
                streamHandler.sendEvent(buildTelemetryTick(player));
            }
        },
        tickInterval,  // 首次延迟
        tickInterval   // 后续间隔
    );
}
```

### 5.2 构建心跳快照

```java
private MatrixTelemetryProto.MatrixTelemetryRequest buildTelemetryTick(
        org.bukkit.entity.Player player
) {
    var loc = player.getLocation();
    var vel = player.getVelocity();

    var builder = MatrixTelemetryProto.TelemetryTick.newBuilder()
            .setPlayerUuid(player.getUniqueId().toString())
            .setPlayerName(player.getName())
            .setWorldName(loc.getWorld().getName())
            // 坐标与朝向
            .setPosX(loc.getX()).setPosY(loc.getY()).setPosZ(loc.getZ())
            .setYaw(loc.getYaw()).setPitch(loc.getPitch())
            // 生命值
            .setHealth((float) player.getHealth())
            .setMaxHealth((float) player.getMaxHealth())
            // 饱食度
            .setFoodLevel(player.getFoodLevel())
            .setSaturation(player.getSaturation())
            // 经验
            .setExpLevel(player.getLevel())
            .setExpProgress(player.getExp())
            // 手持物品
            .setMainHandItem(player.getInventory().getItemInMainHand().getType().name())
            .setOffHandItem(player.getInventory().getItemInOffHand().getType().name())
            // 状态标志
            .setIsSneaking(player.isSneaking())
            .setIsSprinting(player.isSprinting())
            .setIsFlying(player.isFlying())
            .setIsOnGround(player.isOnGround())
            .setIsInWater(player.isInWater())
            .setIsBlocking(player.isBlocking())
            // 速度向量
            .setVelocityX(vel.getX())
            .setVelocityY(vel.getY())
            .setVelocityZ(vel.getZ())
            // 时间戳
            .setTickTime(player.getWorld().getFullTime())
            .setTimestamp(System.currentTimeMillis());

    // 药水效果列表
    for (var effect : player.getActivePotionEffects()) {
        builder.addActiveEffects(effect.getType().getName() + ":" + effect.getAmplifier());
    }

    return MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
            .setServerName(serverName)
            .setTelemetryTick(builder)
            .build();
}
```

### 5.3 config.yml 配置

```yaml
grpc:
  server-name: "survival"

telemetry:
  # 心跳间隔，单位 tick（20 tick = 1 秒）
  # 建议 20 到 40 之间
  tick-interval-ticks: 20
```

---

## 6 事件监听器示例

每个事件对应一个 Bukkit `@EventHandler`，代码结构基本一致：取字段，构建 Protobuf，调用 `sendEvent`。

```java
public class PlayerEventListener implements Listener {

    private final MatrixTelemetry plugin;

    public PlayerEventListener(MatrixTelemetry plugin) {
        this.plugin = plugin;
    }

    // ===== 玩家加入（必需） =====

    @EventHandler(priority = EventPriority.MONITOR)
    public void onPlayerJoin(PlayerJoinEvent event) {
        var player = event.getPlayer();
        plugin.getStreamHandler().sendEvent(
            MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                .setServerName(plugin.getServerName())
                .setPlayerJoin(MatrixTelemetryProto.PlayerJoinEvent.newBuilder()
                    .setPlayerUuid(player.getUniqueId().toString())
                    .setPlayerName(player.getName())
                    .setWorldName(player.getWorld().getName())
                    .setIpAddress(player.getAddress().getAddress().getHostAddress())
                    .setTimestamp(System.currentTimeMillis())
                    .build())
                .build()
        );
    }

    // ===== 玩家退出（必需） =====

    @EventHandler(priority = EventPriority.MONITOR)
    public void onPlayerQuit(PlayerQuitEvent event) {
        var player = event.getPlayer();
        plugin.getStreamHandler().sendEvent(
            MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                .setServerName(plugin.getServerName())
                .setPlayerQuit(MatrixTelemetryProto.PlayerQuitEvent.newBuilder()
                    .setPlayerUuid(player.getUniqueId().toString())
                    .setPlayerName(player.getName())
                    .setQuitReason("")
                    .setTimestamp(System.currentTimeMillis())
                    .build())
                .build()
        );
    }

    // ===== 方块破坏 =====

    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onBlockBreak(org.bukkit.event.block.BlockBreakEvent event) {
        var player = event.getPlayer();
        var block = event.getBlock();
        plugin.getStreamHandler().sendEvent(
            MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                .setServerName(plugin.getServerName())
                .setBlockBreak(MatrixTelemetryProto.BlockBreakEvent.newBuilder()
                    .setPlayerUuid(player.getUniqueId().toString())
                    .setPlayerName(player.getName())
                    .setMaterial(block.getType().name())
                    .setWorldName(block.getWorld().getName())
                    .setX(block.getX()).setY(block.getY()).setZ(block.getZ())
                    .setIsInstaBreak(false)
                    .setToolUsed(player.getInventory().getItemInMainHand().getType().name())
                    .setTimestamp(System.currentTimeMillis())
                    .build())
                .build()
        );
    }

    // ===== 实体击杀 =====

    @EventHandler(priority = EventPriority.MONITOR)
    public void onEntityKill(org.bukkit.event.entity.EntityDeathEvent event) {
        var killer = event.getEntity().getKiller();
        if (killer == null) return; // 非玩家击杀，跳过

        var entity = event.getEntity();
        var loc = entity.getLocation();
        plugin.getStreamHandler().sendEvent(
            MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                .setServerName(plugin.getServerName())
                .setEntityKill(MatrixTelemetryProto.EntityKillEvent.newBuilder()
                    .setPlayerUuid(killer.getUniqueId().toString())
                    .setPlayerName(killer.getName())
                    .setEntityType(entity.getType().name())
                    .setEntityName(entity.getName())
                    .setWorldName(entity.getWorld().getName())
                    .setEntityX(loc.getX()).setEntityY(loc.getY()).setEntityZ(loc.getZ())
                    .setExpDropped(event.getDroppedExp())
                    .setTimestamp(System.currentTimeMillis())
                    .build())
                .build()
        );
    }

    // ===== 玩家死亡 =====

    @EventHandler(priority = EventPriority.MONITOR)
    public void onPlayerDeath(org.bukkit.event.entity.PlayerDeathEvent event) {
        var player = event.getEntity();
        var loc = player.getLocation();
        plugin.getStreamHandler().sendEvent(
            MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                .setServerName(plugin.getServerName())
                .setPlayerDeath(MatrixTelemetryProto.PlayerDeathEvent.newBuilder()
                    .setPlayerUuid(player.getUniqueId().toString())
                    .setPlayerName(player.getName())
                    .setDeathMessage(event.getDeathMessage())
                    .setDeathCause(player.getLastDamageCause() != null
                        ? player.getLastDamageCause().getCause().name() : "UNKNOWN")
                    .setKillerName(player.getKiller() != null
                        ? player.getKiller().getName() : "")
                    .setPosX(loc.getX()).setPosY(loc.getY()).setPosZ(loc.getZ())
                    .setKeepInventory(event.getKeepInventory())
                    .setDroppedExp(event.getDroppedExp())
                    .setTimestamp(System.currentTimeMillis())
                    .build())
                .build()
        );
    }
}
```

所有事件监听器都遵循同样的模式，这里只列出了几个代表性的。完整实现请参考项目中的 Listener 类。

---

## 7 重连机制：指数退避

gRPC Stream 不是铁打的。网络波动、Go 端重启、服务器迁移，都会导致断连。你的插件需要自动重连。

### 7.1 核心思路

用 `generation` 计数器区分新旧连接。每次重连 generation 递增，旧连接的回调检查到 generation 不匹配就退出，不会干扰新连接。

### 7.2 完整的 Stream Handler

```java
public class TelemetryStreamHandler {

    private static final long INITIAL_RETRY_MS = 5000;   // 初始 5 秒
    private static final long MAX_RETRY_MS = 60000;      // 上限 60 秒

    private final JavaPlugin plugin;
    private final MatrixTelemetryServiceGrpc.MatrixTelemetryServiceStub asyncStub;
    private final ScheduledExecutorService retryExecutor;

    private volatile StreamObserver<MatrixTelemetryProto.MatrixTelemetryRequest> requestObserver;
    private volatile boolean running = false;
    private volatile long generation = 0;
    private long retryDelayMs = INITIAL_RETRY_MS;

    public TelemetryStreamHandler(
            JavaPlugin plugin,
            MatrixTelemetryServiceGrpc.MatrixTelemetryServiceStub asyncStub
    ) {
        this.plugin = plugin;
        this.asyncStub = asyncStub;
        this.retryExecutor = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "MatrixTelemetry-Retry");
            t.setDaemon(true);
            return t;
        });
    }

    /** 启动连接，失败自动重连 */
    public void startWithRetry() {
        retryExecutor.submit(this::connect);
    }

    /** 建立新的 Client Stream */
    public void connect() {
        running = true;
        final long currentGen = ++generation;

        StreamObserver<MatrixTelemetryProto.MatrixTelemetryResponse> responseObserver =
            new StreamObserver<>() {
                @Override
                public void onNext(MatrixTelemetryProto.MatrixTelemetryResponse value) { }

                @Override
                public void onError(Throwable t) {
                    if (currentGen != generation) return; // 旧连接，忽略
                    plugin.getLogger().warning("[Matrix] 流错误: "
                        + Optional.ofNullable(t.getMessage())
                                  .orElse(t.getClass().getSimpleName()));
                    synchronized (TelemetryStreamHandler.this) {
                        requestObserver = null;
                    }
                    if (running) scheduleReconnect();
                }

                @Override
                public void onCompleted() {
                    if (currentGen != generation) return;
                    plugin.getLogger().info("[Matrix] 遥测流已关闭");
                    synchronized (TelemetryStreamHandler.this) {
                        requestObserver = null;
                    }
                    if (running) scheduleReconnect();
                }
            };

        synchronized (this) {
            requestObserver = asyncStub.telemetryStream(responseObserver);
        }
        retryDelayMs = INITIAL_RETRY_MS; // 连接成功，重置退避时间
        plugin.getLogger().info("[Matrix] Stream 已建立 [gen=" + currentGen + "]");
    }

    /** 指数退避调度重连 */
    private void scheduleReconnect() {
        plugin.getLogger().info("[Matrix] " + (retryDelayMs / 1000) + "s 后重连...");
        retryExecutor.schedule(() -> {
            if (running) connect();
        }, retryDelayMs, TimeUnit.MILLISECONDS);
        retryDelayMs = Math.min(retryDelayMs * 2, MAX_RETRY_MS);
    }

    /** 优雅关闭 */
    public void shutdown() {
        running = false;
        retryExecutor.shutdownNow();
        synchronized (this) {
            if (requestObserver != null) {
                requestObserver.onCompleted();
                requestObserver = null;
            }
        }
    }

    /** 发送事件，线程安全，断连时静默丢弃 */
    public void sendEvent(MatrixTelemetryProto.MatrixTelemetryRequest event) {
        StreamObserver<MatrixTelemetryProto.MatrixTelemetryRequest> observer;
        synchronized (this) {
            observer = requestObserver;
        }
        if (observer == null) return; // 没连接，丢弃
        try {
            observer.onNext(event);
        } catch (Exception e) {
            plugin.getLogger().warning("[Matrix] 发送失败: " + e.getMessage());
        }
    }
}
```

### 7.3 退避时间线

```
失败 1 次 → 等 5s
失败 2 次 → 等 10s
失败 3 次 → 等 20s
失败 4 次 → 等 40s
失败 5 次+ → 等 60s（封顶）
```

连接成功后，`retryDelayMs` 重置为 5 秒。

---

## 8 Session 恢复说明

这里有个好消息：Java 插件端**不需要做任何 Session 恢复的工作**。

当 Go 后端重启时（部署更新、故障恢复等），所有 PlayerSession 会丢失。但 Go 端启动后会自动执行以下流程：

1. 等待 10 秒，让 MC 插件侧的 gRPC Stream 重新建立
2. 调用 `QueryServerStatus` 查询每台已连接服务器上的在线玩家列表
3. 对每个玩家调用 `GetOrCreate`，重建 PlayerSession

```
Go 后端重启
    ↓ 等待 10s（gRPC 流重建）
    ↓
遍历已连接服务器
    ↓ QueryServerStatus(serverName)
    ↓
拿到在线玩家列表
    ↓
逐个 GetOrCreate → 重建 Session
```

整个过程对 Java 插件完全透明。你的插件只需要保证 Stream 重连机制正常工作就行。

唯一需要注意的是：Go 端重启后的短暂窗口内（约 10 秒），上报的事件可能会被丢弃（因为 Session 还没恢复）。这是预期行为，不需要特殊处理。

---

## 9 性能建议

### 9.1 间隔控制

TelemetryTick 的频率不要低于 20 tick。1 秒一次快照已经足够做反作弊检测和状态画像，更频繁只会增加 Redis 负载。

### 9.2 异步执行

所有上报操作必须走异步线程。`runTaskTimerAsynchronously` 保证了心跳采集不会阻塞主线程。Stream 的 `onNext` 本身是异步的，不会卡 Bukkit 主线程。

### 9.3 断连不缓存

Stream 断开时，`sendEvent` 静默丢弃事件。不要尝试在内存中缓存未发送的事件。重连后从当前状态继续采集就好，历史快照对后端来说没有意义。

### 9.4 高频事件的取舍

物品丢弃、背包操作这类事件在玩家快速整理背包时可能产生大量消息。如果发现流量过大，可以加一个简单的节流：用 `HashMap<UUID, Long>` 记录上次发送时间，同一玩家 500ms 内只上报一次。

### 9.5 内存控制

每条 Protobuf 消息在发送完就丢弃，不要持有引用。TelemetryTick 的 `activeEffects` 列表通常很短（0-5 个），不会有内存问题。

---

## 10 常见问题 FAQ

### Q：Go 后端没启动，插件会崩溃吗？

不会。Stream 建立失败后进入指数退避重连。插件正常运行，只是遥测数据发不出去。

### Q：PlayerJoin 必须在其他事件之前吗？

是的。后端用 PlayerJoin 创建 PlayerSession。在 PlayerJoin 之前发送的其他事件会被静默丢弃。同理，PlayerQuit 必须是最后一个事件，它触发 Drain 将缓冲数据刷到数据库。

### Q：玩家 UUID 格式有要求吗？

必须是标准的 `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` 格式。后端用 `uuid.Parse()` 解析，格式不对的事件会被丢弃并打 WARN 日志。

### Q：事件监听器为什么用 `EventPriority.MONITOR`？

MONITOR 优先级最低，确保在其他插件处理完事件之后才上报。这样你拿到的数据是最终状态（比如其他插件可能取消了事件、修改了掉落物）。

### Q：方块事件为什么要 `ignoreCancelled = true`？

被其他插件取消的方块事件（比如保护区域内的破坏）不应该上报。已经取消的事件说明没有实际发生。

### Q：反作弊检测需要我在插件端做什么？

什么都不用做。Go 后端通过 `EntityDamageEvent` 中的玩家和实体坐标计算攻击距离（Reach），通过 `TelemetryTick` 中的位移计算移动速度（Speed）。检测结果写入数据库和 Redis，供网页端查询。

### Q：多台 MC 服务器连同一个 Go 后端可以吗？

可以。每台服务器用不同的 `server_name` 标识。后端通过 `server_name` 字段路由事件到对应的 Session。所有服务器的数据独立隔离。

### Q：Go 端重启期间的数据会丢失吗？

会有一小部分丢失。Go 端重启后等待 10 秒进行 Session 恢复，这期间上报的事件无法处理。恢复完成后一切正常。

### Q：timestamp 用什么格式？

毫秒级 Unix 时间戳，`System.currentTimeMillis()` 返回的就是。

---

## 附录：生命周期全景

```
插件 onEnable()
    │
    ├─ frontleaves-lib 获取 ManagedChannel
    ├─ 创建 AsyncStub
    ├─ TelemetryStreamHandler.startWithRetry()
    │       └─ connect() → 建立 Client Stream
    │
    ├─ 注册 Bukkit EventListener
    │       └─ PlayerJoin → sendEvent(player_join)
    │       └─ BlockBreak → sendEvent(block_break)
    │       └─ EntityKill → sendEvent(entity_kill)
    │       └─ ...更多事件
    │       └─ PlayerQuit → sendEvent(player_quit)
    │
    ├─ BukkitScheduler.runTaskTimerAsynchronously
    │       └─ 每 N tick → 遍历在线玩家 → sendEvent(telemetry_tick)
    │
    ├─ ConnectivityMonitor 监控连接状态
    │
    └─ 注册 ChannelReloadListener（通道重建时自动重启 Stream）

插件 onDisable()
    │
    ├─ tickTask.cancel()
    ├─ streamHandler.shutdown()
    └─ Channel 由 frontleaves-lib 管理，不用手动关
```
