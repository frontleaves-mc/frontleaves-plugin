# Matrix Telemetry — Java 插件对接文档

> **版本**: 1.0.0
> **适用插件**: 需要上报玩家行为遥测数据的 Minecraft Bukkit/Paper 插件
> **依赖**: `frontleaves-lib` ≥ 1.0.0, gRPC ≥ 1.62.2, Protobuf ≥ 3.25.3

---

## 目录

1. [概述](#1-概述)
2. [接入准备](#2-接入准备)
3. [Proto 文件配置](#3-proto-文件配置)
4. [服务架构](#4-服务架构)
5. [事件类型速查](#5-事件类型速查)
6. [代码实现](#6-代码实现)
7. [事件上报指南](#7-事件上报指南)
8. [注意事项](#8-注意事项)

---

## 1. 概述

Matrix Telemetry 是 FrontLeaves 的玩家行为遥测子系统。Java 插件通过 **gRPC Client Stream** 持续上报玩家行为数据，Go 后端接收后进行：

- **统计聚合** — 方块破坏/放置、实体击杀、死亡计数等批量聚合写入数据库
- **反作弊检测** — 实时检测 Reach（攻击距离异常）和 Speed（移动速度异常）
- **实时监控** — 反作弊风险分数写入 Redis，供网页端实时查询

### 数据流

```
┌──────────────────────────────────────────────────────┐
│  Minecraft 插件 (Java/Bukkit)                         │
│                                                       │
│  onPlayerJoin ──→ sendEvent(PlayerJoinEvent)          │
│  onPlayerQuit ──→ sendEvent(PlayerQuitEvent)          │
│  每个 tick ─────→ sendEvent(TelemetryTick)            │
│  各种事件 ──────→ sendEvent(BlockBreak/Place/...)     │
│                                                       │
│          ║ gRPC Client Stream (protobuf)              │
└──────────╫───────────────────────────────────────────┘
           ╠════════════════════════════════════════════
           ║
┌──────────╫───────────────────────────────────────────┐
│  frontleaves-plugin (Go)                              │
│                                                       │
│  TelemetryStream ──→ dispatch ──→ PlayerSession       │
│                                       ├── StatisticsSub│
│                                       └── AntiCheatSub│
│                                                       │
│                    ┌── PostgreSQL (统计/警告)          │
│                    └── Redis (实时监控)                │
└──────────────────────────────────────────────────────┘
```

---

## 2. 接入准备

### 2.1 前置条件

| 条件 | 说明 |
|------|------|
| `frontleaves-lib` | 已安装且正常加载，负责 gRPC Channel 管理和认证 |
| gRPC 依赖 | `grpc-netty-shaded`, `grpc-protobuf`, `grpc-stub` |
| Protobuf 依赖 | `protobuf-java`, `protoc-gen-grpc-java` |
| Java 版本 | ≥ 21 |

### 2.2 pom.xml 依赖配置

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

    <!-- FrontleavesLib -->
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

### 2.3 Maven Protobuf 编译插件

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
                <protocArtifact>com.google.protobuf:protoc:${protobuf.version}:exe:${os.detected.classifier}</protocArtifact>
                <pluginId>grpc-java</pluginId>
                <pluginArtifact>io.grpc:protoc-gen-grpc-java:${grpc.version}:exe:${os.detected.classifier}</pluginArtifact>
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

---

## 3. Proto 文件配置

### 3.1 目录结构

在插件项目中创建以下目录结构：

```
src/main/proto/
├── link/
│   └── base.proto          # 从 frontleaves-plugin/proto/link/base.proto 复制
└── matrix/
    └── v1/
        └── matrix_telemetry.proto  # 从 frontleaves-plugin/proto/matrix/v1/ 复制
```

> ⚠️ `link/base.proto` 是基础依赖，必须与 Go 后端保持一致。

### 3.2 获取 Proto 文件

```bash
# 从 frontleaves-plugin 仓库复制
cp frontleaves-plugin/proto/link/base.proto src/main/proto/link/
cp frontleaves-plugin/proto/matrix/v1/matrix_telemetry.proto src/main/proto/matrix/v1/
```

### 3.3 编译生成

```bash
mvn compile
```

生成的代码位于 `target/generated-sources/protobuf/`：
- `grpc-java/.../MatrixTelemetryServiceGrpc.java` — gRPC Stub
- `java/.../MatrixTelemetryProto.java` — 消息类

---

## 4. 服务架构

### 4.1 RPC 方法

| 方法 | 类型 | 说明 |
|------|------|------|
| `TelemetryStream` | Client Stream | 插件持续发送遥测事件，服务端返回一个最终响应 |

### 4.2 认证方式

通过 gRPC Metadata 传递，由 `frontleaves-lib` 的 `ClientAuthInterceptor` 自动注入：

| Key | 说明 |
|-----|------|
| `plugin-name` | 插件标识名（如 `"matrix-telemetry"`） |
| `plugin-secret-key` | 共享密钥，由 lib 统一管理 |

### 4.3 生命周期

```
插件 onEnable
    │
    ├── 创建 ManagedChannel (via frontleaves-lib)
    ├── 创建 AsyncStub
    ├── 建立 Client Stream
    │
    ├── [持续运行]
    │   ├── PlayerJoinEvent → 上报 PlayerJoin
    │   ├── 定时任务 → 上报 TelemetryTick
    │   ├── 各种 Bukkit Event → 上报对应事件
    │   └── PlayerQuitEvent → 上报 PlayerQuit
    │
插件 onDisable
    │
    └── 关闭 Stream → 释放资源
```

**关键规则**：
- `PlayerJoin` **必须**是某玩家会话的第一个事件（后端用它创建 Session）
- `PlayerQuit` **必须**是某玩家会话的最后一个事件（后端用它触发 Drain 排水）
- 其他事件在 Join 和 Quit 之间发送，后端通过 `playerUUID` 路由到对应 Session
- 在 Join 之前或 Quit 之后发送的事件会被**静默丢弃**

---

## 5. 事件类型速查

### 5.1 连接事件（生命周期必需）

| 事件 | oneof 字段 | 触发时机 | 优先级 |
|------|-----------|---------|--------|
| `PlayerJoinEvent` | `player_join` (11) | 玩家加入服务器 | **必须** |
| `PlayerQuitEvent` | `player_quit` (12) | 玩家离开服务器 | **必须** |

### 5.2 心跳快照（定期上报）

| 事件 | oneof 字段 | 建议频率 |
|------|-----------|---------|
| `TelemetryTick` | `telemetry_tick` (13) | 每 20-40 tick (1-2秒) |

### 5.3 方块事件

| 事件 | oneof 字段 | 触发时机 |
|------|-----------|---------|
| `BlockBreakEvent` | `block_break` (14) | 玩家破坏方块 |
| `BlockPlaceEvent` | `block_place` (15) | 玩家放置方块 |

### 5.4 实体/战斗事件

| 事件 | oneof 字段 | 触发时机 |
|------|-----------|---------|
| `EntityKillEvent` | `entity_kill` (16) | 玩家击杀实体 |
| `EntityDamageEvent` | `entity_damage` (17) | 玩家对实体造成伤害 |
| `PlayerDamageEvent` | `player_damage` (18) | 玩家受到伤害 |
| `PlayerDeathEvent` | `player_death` (19) | 玩家死亡 |

### 5.5 物品/背包事件

| 事件 | oneof 字段 | 触发时机 |
|------|-----------|---------|
| `ItemDropEvent` | `item_drop` (20) | 玩家丢弃物品 |
| `ItemPickupEvent` | `item_pickup` (21) | 玩家拾取物品 |
| `InventoryActionEvent` | `inventory_action` (22) | 背包操作 |

### 5.6 玩家行为事件

| 事件 | oneof 字段 | 触发时机 |
|------|-----------|---------|
| `PlayerChatEvent` | `player_chat` (23) | 玩家发送聊天消息 |
| `PlayerCommandEvent` | `player_command` (24) | 玩家执行命令 |
| `PlayerToggleEvent` | `player_toggle` (25) | 玩家切换状态（潜行/飞行等） |
| `PlayerTeleportEvent` | `teleport` (26) | 玩家传送 |
| `PlayerRespawnEvent` | `respawn` (27) | 玩家重生 |
| `GameModeChangeEvent` | `game_mode_change` (28) | 玩家游戏模式变更 |

---

## 6. 代码实现

### 6.1 Stream Handler（核心类）

参考 `server-status` 插件的 `ServerEventStreamHandler` 模式：

```java
package com.frontleaves.plugins.matrix.grpc;

import com.frontleaves.plugins.matrix.grpc.generated.MatrixTelemetryProto;
import com.frontleaves.plugins.matrix.grpc.generated.MatrixTelemetryServiceGrpc;
import io.grpc.stub.StreamObserver;
import org.bukkit.plugin.java.JavaPlugin;
import org.jetbrains.annotations.NotNull;

import java.util.Optional;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * Matrix Telemetry Client Stream 处理器。
 * <p>
 * 通过一次 Stream 连接持续发送玩家行为遥测事件，
 * 支持自动重连（指数退避），使用 synchronized 保证线程安全。
 *
 * @author xiao_lfeng
 * @version 1.0.0
 */
public class TelemetryStreamHandler {

    private static final long INITIAL_RETRY_DELAY_MS = 5000;
    private static final long MAX_RETRY_DELAY_MS = 60000;

    private final JavaPlugin plugin;
    private final MatrixTelemetryServiceGrpc.MatrixTelemetryServiceStub asyncStub;
    private final ScheduledExecutorService retryExecutor;

    private volatile StreamObserver<MatrixTelemetryProto.MatrixTelemetryRequest> requestObserver;
    private volatile boolean running = false;
    private volatile long generation = 0;
    private long retryDelayMs = INITIAL_RETRY_DELAY_MS;

    public TelemetryStreamHandler(
            @NotNull JavaPlugin plugin,
            @NotNull MatrixTelemetryServiceGrpc.MatrixTelemetryServiceStub asyncStub
    ) {
        this.plugin = plugin;
        this.asyncStub = asyncStub;
        this.retryExecutor = Executors.newSingleThreadScheduledExecutor(r -> {
            Thread t = new Thread(r, "MatrixTelemetry-Retry");
            t.setDaemon(true);
            return t;
        });
    }

    /**
     * 启动遥测流连接，失败时自动重连。
     */
    public void startWithRetry() {
        retryExecutor.submit(this::connect);
    }

    /**
     * 建立 Client Stream 连接。
     */
    public void connect() {
        running = true;
        final long currentGeneration = ++generation;

        StreamObserver<MatrixTelemetryProto.MatrixTelemetryResponse> responseObserver =
                new StreamObserver<>() {
            @Override
            public void onNext(MatrixTelemetryProto.MatrixTelemetryResponse value) {
                // 服务端最终响应，通常忽略
            }

            @Override
            public void onError(Throwable t) {
                if (currentGeneration != generation) return;
                plugin.getLogger().warning("[Matrix] 流错误: "
                        + Optional.ofNullable(t.getMessage()).orElse(t.getClass().getSimpleName()));
                synchronized (TelemetryStreamHandler.this) {
                    requestObserver = null;
                }
                if (running) scheduleReconnect();
            }

            @Override
            public void onCompleted() {
                if (currentGeneration != generation) return;
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
        retryDelayMs = INITIAL_RETRY_DELAY_MS;
        plugin.getLogger().info("[Matrix] Client Stream 已建立 [generation=" + currentGeneration + "]");
    }

    private void scheduleReconnect() {
        plugin.getLogger().info("[Matrix] 将在 " + (retryDelayMs / 1000) + " 秒后重连...");
        retryExecutor.schedule(() -> {
            if (running) connect();
        }, retryDelayMs, TimeUnit.MILLISECONDS);
        retryDelayMs = Math.min(retryDelayMs * 2, MAX_RETRY_DELAY_MS);
    }

    /**
     * 关闭流连接和重连调度器。
     */
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

    /**
     * 发送遥测事件。线程安全，流断开时静默丢弃。
     *
     * @param event 遥测请求
     */
    public void sendEvent(@NotNull MatrixTelemetryProto.MatrixTelemetryRequest event) {
        StreamObserver<MatrixTelemetryProto.MatrixTelemetryRequest> observer;
        synchronized (this) {
            observer = requestObserver;
        }
        if (observer == null) return;
        try {
            observer.onNext(event);
        } catch (Exception e) {
            plugin.getLogger().warning("[Matrix] 发送事件失败: "
                    + Optional.ofNullable(e.getMessage()).orElse(e.getClass().getSimpleName()));
        }
    }
}
```

### 6.2 插件主类

```java
package com.frontleaves.plugins.matrix;

import com.frontleaves.plugins.lib.FrontleavesLib;
import com.frontleaves.plugins.lib.grpc.ChannelReloadListener;
import com.frontleaves.plugins.lib.grpc.ConnectivityMonitor;
import com.frontleaves.plugins.lib.message.Message;
import com.frontleaves.plugins.matrix.grpc.TelemetryStreamHandler;
import com.frontleaves.plugins.matrix.grpc.generated.MatrixTelemetryProto;
import com.frontleaves.plugins.matrix.grpc.generated.MatrixTelemetryServiceGrpc;
import com.frontleaves.plugins.matrix.listener.PlayerEventListener;
import com.frontleaves.plugins.matrix.listener.WorldEventListener;
import io.grpc.ManagedChannel;
import org.bukkit.Bukkit;
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.scheduler.BukkitTask;

/**
 * Matrix Telemetry 插件主类。
 * <p>
 * 通过 gRPC Client Stream 上报玩家行为遥测数据到 Go 后端。
 *
 * @author xiao_lfeng
 * @version 1.0.0
 */
public final class MatrixTelemetry extends JavaPlugin {

    private ManagedChannel channel;
    private TelemetryStreamHandler streamHandler;
    private BukkitTask tickTask;
    private String serverName;

    @Override
    public void onEnable() {
        this.saveDefaultConfig();
        serverName = this.getConfig().getString("grpc.server-name", "survival");
        int tickInterval = this.getConfig().getInt("telemetry.tick-interval-ticks", 20);

        // 1. 通过 frontleaves-lib 获取 gRPC Channel
        FrontleavesLib lib = FrontleavesLib.getInstance()
                .orElseThrow(() -> new IllegalStateException("FrontleavesLib 未加载"));

        channel = lib.createChannel("matrix-telemetry");

        // 2. 创建 AsyncStub 并建立 Stream
        var asyncStub = MatrixTelemetryServiceGrpc.newStub(channel);
        streamHandler = new TelemetryStreamHandler(this, asyncStub);
        streamHandler.startWithRetry();

        // 3. 注册 Bukkit 事件监听器
        Bukkit.getPluginManager().registerEvents(new PlayerEventListener(this), this);
        Bukkit.getPluginManager().registerEvents(new WorldEventListener(this), this);

        // 4. 定时上报 TelemetryTick（心跳快照）
        tickTask = Bukkit.getScheduler().runTaskTimerAsynchronously(this, () -> {
            for (var player : Bukkit.getOnlinePlayers()) {
                streamHandler.sendEvent(buildTelemetryTick(player));
            }
        }, tickInterval, tickInterval);

        // 5. 连接状态监控
        ConnectivityMonitor monitor = lib.createConnectivityMonitor(this, channel, "matrix-telemetry");
        monitor.onReady(() -> {
            Message.of(this, "Matrix").console().info("gRPC 连接已恢复");
        }).onFailure(() -> {
            Message.of(this, "Matrix").console().warning("gRPC 连接异常，遥测暂停");
        }).startMonitoring();

        // 6. 注册通道重载回调
        lib.registerPlugin("matrix-telemetry", newChannel -> {
            channel = newChannel;
            if (streamHandler != null) streamHandler.shutdown();

            var newStub = MatrixTelemetryServiceGrpc.newStub(newChannel);
            streamHandler = new TelemetryStreamHandler(this, newStub);
            streamHandler.startWithRetry();

            Message.of(this, "Matrix").console().info("gRPC 通道已重建，流处理器已重启");
        });

        Message.of(this, "Matrix").console().info("Matrix Telemetry 初始化完成");
    }

    @Override
    public void onDisable() {
        FrontleavesLib.getInstance().ifPresent(lib -> lib.unregisterPlugin("matrix-telemetry"));

        if (tickTask != null) tickTask.cancel();
        if (streamHandler != null) streamHandler.shutdown();
        // Channel 由 frontleaves-lib 管理，不在此关闭

        Message.of(this, "Matrix").console().warning("Matrix Telemetry 已停止");
    }

    // === 公共方法 ===

    /**
     * 获取 Stream Handler（供 Listener 使用）。
     */
    public TelemetryStreamHandler getStreamHandler() {
        return streamHandler;
    }

    /**
     * 获取服务器名称标识。
     */
    public String getServerName() {
        return serverName;
    }

    // === TelemetryTick 构建 ===

    private MatrixTelemetryProto.MatrixTelemetryRequest buildTelemetryTick(
            @NotNull org.bukkit.entity.Player player
    ) {
        var loc = player.getLocation();
        var vel = player.getVelocity();
        var builder = MatrixTelemetryProto.TelemetryTick.newBuilder()
                .setPlayerUuid(player.getUniqueId().toString())
                .setPlayerName(player.getName())
                .setWorldName(loc.getWorld().getName())
                .setPosX(loc.getX()).setPosY(loc.getY()).setPosZ(loc.getZ())
                .setYaw(loc.getYaw()).setPitch(loc.getPitch())
                .setHealth((float) player.getHealth())
                .setMaxHealth((float) player.getMaxHealth())
                .setFoodLevel(player.getFoodLevel())
                .setSaturation(player.getSaturation())
                .setExpLevel(player.getLevel())
                .setExpProgress(player.getExp())
                .setMainHandItem(player.getInventory().getItemInMainHand().getType().name())
                .setOffHandItem(player.getInventory().getItemInOffHand().getType().name())
                .setIsSneaking(player.isSneaking())
                .setIsSprinting(player.isSprinting())
                .setIsFlying(player.isFlying())
                .setIsOnGround(player.isOnGround())
                .setIsInWater(player.isInWater())
                .setIsInLava(player.getLocation().getBlock().isLiquid()
                        && !player.getLocation().getBlock().getType().name().contains("WATER"))
                .setIsBlocking(player.isBlocking())
                .setIsClimbing(player.getLocation().getBlock().getType().name().contains("LADDER")
                        || player.getLocation().getBlock().getType().name().contains("VINE"))
                .setVelocityX(vel.getX()).setVelocityY(vel.getY()).setVelocityZ(vel.getZ())
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
}
```

### 6.3 config.yml

```yaml
# Matrix Telemetry 配置

grpc:
  # 服务器名称标识（用于后端区分不同 MC 服务器）
  server-name: "survival"

telemetry:
  # TelemetryTick 心跳快照上报间隔（单位：tick，20 tick = 1 秒）
  # 建议 20-40 tick（1-2 秒）
  tick-interval-ticks: 20
```

---

## 7. 事件上报指南

### 7.1 玩家事件监听器示例

```java
package com.frontleaves.plugins.matrix.listener;

import com.frontleaves.plugins.matrix.MatrixTelemetry;
import com.frontleaves.plugins.matrix.grpc.generated.MatrixTelemetryProto;
import org.bukkit.event.EventHandler;
import org.bukkit.event.EventPriority;
import org.bukkit.event.Listener;
import org.bukkit.event.player.*;
import org.jetbrains.annotations.NotNull;

public class PlayerEventListener implements Listener {

    private final MatrixTelemetry plugin;

    public PlayerEventListener(@NotNull MatrixTelemetry plugin) {
        this.plugin = plugin;
    }

    // ========== 连接事件（必需） ==========

    @EventHandler(priority = EventPriority.MONITOR)
    public void onPlayerJoin(@NotNull PlayerJoinEvent event) {
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

    @EventHandler(priority = EventPriority.MONITOR)
    public void onPlayerQuit(@NotNull PlayerQuitEvent event) {
        var player = event.getPlayer();
        plugin.getStreamHandler().sendEvent(
                MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                        .setServerName(plugin.getServerName())
                        .setPlayerQuit(MatrixTelemetryProto.PlayerQuitEvent.newBuilder()
                                .setPlayerUuid(player.getUniqueId().toString())
                                .setPlayerName(player.getName())
                                // Paper 1.21+ 可用 event.getReason().name()
                                // 兼容写法留空或使用 Bukkit 的 PlayerQuitEvent 无 getReason
                                .setQuitReason("")
                                .setTimestamp(System.currentTimeMillis())
                                .build())
                        .build()
        );
    }

    // ========== 方块事件 ==========

    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onBlockBreak(@NotNull org.bukkit.event.block.BlockBreakEvent event) {
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
                                .setIsInstaBreak(false) // Paper 可用 event.isDropItems()
                                .setToolUsed(player.getInventory().getItemInMainHand().getType().name())
                                .setTimestamp(System.currentTimeMillis())
                                .build())
                        .build()
        );
    }

    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onBlockPlace(@NotNull org.bukkit.event.block.BlockPlaceEvent event) {
        var player = event.getPlayer();
        var block = event.getBlock();
        plugin.getStreamHandler().sendEvent(
                MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                        .setServerName(plugin.getServerName())
                        .setBlockPlace(MatrixTelemetryProto.BlockPlaceEvent.newBuilder()
                                .setPlayerUuid(player.getUniqueId().toString())
                                .setPlayerName(player.getName())
                                .setMaterial(block.getType().name())
                                .setWorldName(block.getWorld().getName())
                                .setX(block.getX()).setY(block.getY()).setZ(block.getZ())
                                .setHand(event.getHand().name())
                                .setReplaced(event.getBlockReplacedState().getType().name())
                                .setTimestamp(System.currentTimeMillis())
                                .build())
                        .build()
        );
    }

    // ========== 战斗/实体事件 ==========

    @EventHandler(priority = EventPriority.MONITOR)
    public void onEntityKill(@NotNull org.bukkit.event.entity.EntityDeathEvent event) {
        var killer = event.getEntity().getKiller();
        if (killer == null) return; // 非玩家击杀，忽略

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

    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onEntityDamage(@NotNull org.bukkit.event.entity.EntityDamageByEntityEvent event) {
        if (!(event.getDamager() instanceof org.bukkit.entity.Player player)) return;

        var entity = event.getEntity();
        var entityLoc = entity.getLocation();
        var playerLoc = player.getLocation();
        plugin.getStreamHandler().sendEvent(
                MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                        .setServerName(plugin.getServerName())
                        .setEntityDamage(MatrixTelemetryProto.EntityDamageEvent.newBuilder()
                                .setPlayerUuid(player.getUniqueId().toString())
                                .setPlayerName(player.getName())
                                .setEntityType(entity.getType().name())
                                .setDamage(event.getDamage())
                                .setFinalDamage(event.getFinalDamage())
                                .setDamageCause(event.getCause().name())
                                .setEntityX(entityLoc.getX()).setEntityY(entityLoc.getY()).setEntityZ(entityLoc.getZ())
                                .setPlayerX(playerLoc.getX()).setPlayerY(playerLoc.getY()).setPlayerZ(playerLoc.getZ())
                                .setPlayerYaw(playerLoc.getYaw()).setPlayerPitch(playerLoc.getPitch())
                                .setTimestamp(System.currentTimeMillis())
                                .build())
                        .build()
        );
    }

    @EventHandler(priority = EventPriority.MONITOR)
    public void onPlayerDeath(@NotNull org.bukkit.event.entity.PlayerDeathEvent event) {
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
                                .setKillerName(player.getKiller() != null ? player.getKiller().getName() : "")
                                .setPosX(loc.getX()).setPosY(loc.getY()).setPosZ(loc.getZ())
                                .setKeepInventory(event.getKeepInventory())
                                .setDroppedExp(event.getDroppedExp())
                                .setTimestamp(System.currentTimeMillis())
                                .build())
                        .build()
        );
    }

    // ========== 聊天/命令事件 ==========

    @EventHandler(priority = EventPriority.MONITOR)
    public void onPlayerChat(@NotNull org.bukkit.event.player.AsyncChatEvent event) {
        plugin.getStreamHandler().sendEvent(
                MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                        .setServerName(plugin.getServerName())
                        .setPlayerChat(MatrixTelemetryProto.PlayerChatEvent.newBuilder()
                                .setPlayerUuid(event.getPlayer().getUniqueId().toString())
                                .setPlayerName(event.getPlayer().getName())
                                .setMessage(org.bukkit.ChatColor.stripColor(
                                        net.kyori.adventure.text.ComponentSerializer.serialize(
                                                event.message())))
                                .setTimestamp(System.currentTimeMillis())
                                .build())
                        .build()
        );
    }

    @EventHandler(priority = EventPriority.MONITOR, ignoreCancelled = true)
    public void onPlayerCommand(@NotNull PlayerCommandPreprocessEvent event) {
        plugin.getStreamHandler().sendEvent(
                MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
                        .setServerName(plugin.getServerName())
                        .setPlayerCommand(MatrixTelemetryProto.PlayerCommandEvent.newBuilder()
                                .setPlayerUuid(event.getPlayer().getUniqueId().toString())
                                .setPlayerName(event.getPlayer().getName())
                                .setCommand(event.getMessage())
                                .setTimestamp(System.currentTimeMillis())
                                .build())
                        .build()
        );
    }
}
```

### 7.2 通用构建辅助方法

建议在插件主类或工具类中封装通用构建器，减少重复代码：

```java
/**
 * 创建带 serverName 的请求构建器。
 */
private MatrixTelemetryProto.MatrixTelemetryRequest.Builder newRequest() {
    return MatrixTelemetryProto.MatrixTelemetryRequest.newBuilder()
            .setServerName(serverName);
}
```

---

## 8. 注意事项

### 8.1 必须遵守的规则

| 规则 | 原因 |
|------|------|
| **PlayerJoin 必须先于其他事件** | 后端用 PlayerJoin 创建 PlayerSession，后续事件依赖 Session 存在 |
| **PlayerQuit 必须是最后一个事件** | 后端用 PlayerQuit 触发 Drain（排水）将缓冲数据刷盘 |
| **playerUUID 必须是标准格式** | 后端使用 `uuid.Parse()` 解析，格式错误会被静默丢弃 |
| **timestamp 使用毫秒级 Unix 时间戳** | `System.currentTimeMillis()` |
| **EventPriority 使用 MONITOR** | 确保在其他插件处理完毕后才上报，数据更准确 |
| **ignoreCancelled = true** | 已取消的事件不应上报（非方块事件除外） |

### 8.2 性能建议

| 建议 | 说明 |
|------|------|
| TelemetryTick 间隔 ≥ 20 tick | 过于频繁会增加后端 Redis 负载 |
| 使用 `runTaskTimerAsynchronously` | 上报逻辑在异步线程执行，避免阻塞主线程 |
| 流断开时静默丢弃 | 不要缓存未发送的事件，重连后从当前状态继续即可 |
| 物品/背包事件可选择性上报 | 高频操作（如快速整理背包）可能产生大量事件 |

### 8.3 反作弊检测说明

后端当前实现了两个反作弊检测规则：

| 规则 | 检测字段 | 阈值 |
|------|---------|------|
| **Reach（攻击距离）** | `EntityDamageEvent` 中玩家坐标与实体坐标的欧氏距离 | > 3.5 格 |
| **Speed（移动速度）** | `TelemetryTick` 中相邻两帧的位移量（跳过飞行状态） | > 0.6 格/tick |

触发时后端会：
1. 写入 `fp_matrix_player_warning` 表
2. 更新 Redis monitor（risk_score += 20，上限 100）

插件端**无需额外处理**，检测完全在后端完成。

### 8.4 重连机制

`TelemetryStreamHandler` 内置指数退避重连：
- 初始延迟：5 秒
- 最大延迟：60 秒
- 每次失败延迟翻倍：5s → 10s → 20s → 40s → 60s → 60s...

重连时 `generation` 递增，防止旧流的延迟回调干扰新流。

### 8.5 错误处理

| 场景 | 行为 |
|------|------|
| Go 后端未启动 | 流建立失败 → 指数退避重连 |
| 网络中断 | `onError` 回调 → 触发重连 |
| UUID 格式错误 | 后端静默丢弃该事件，打印 WARN 日志 |
| PlayerJoin 前发送其他事件 | 后端找不到 Session → 静默丢弃 |
| 重复 PlayerJoin（不断线重连） | 后端 GetOrCreate 保证幂等 |
