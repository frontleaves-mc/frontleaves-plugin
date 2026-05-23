# Java 插件开发者指南：服务器负载指标采集

本文档面向 Java/Bukkit 插件开发者，介绍 server-status 服务的负载数据持久化功能，以及如何在心跳上报中正确填充负载指标字段。

---

## 1. 背景说明

server-status 服务经过精简，当前职责明确为两部分：

- **心跳检测**：接收 Minecraft 服务器定时上报的心跳事件，判断服务器在线状态。
- **负载采集**：从心跳事件中提取 TPS、CPU、内存、JVM 等运行指标，进行持久化存储。

玩家管理（上线/下线/查询）已由 essentials 服务独立处理，server-status 不再涉及玩家逻辑。

本次新增的负载数据持久化功能会将每分钟聚合的 TPS、CPU、RAM、JVM 指标写入数据库，供管理面板或其他系统查询历史负载趋势。

---

## 2. 本次改造不影响 Java 插件

gRPC proto 定义和 handler 层均未做任何修改，Java 插件不需要改动代码。

需要确认的是：插件在上报心跳时，正确填充了 `cpu_info`、`memory_info`、`jvm_info` 等负载指标字段。如果这些字段为空，服务端在聚合时将缺少对应指标，导致历史数据不完整。

---

## 3. 负载指标采集说明

以下字段会被服务端采集并持久化：

| 字段路径 | 类型 | 说明 |
|----------|------|------|
| `tps` | `double` | 服务器 TPS（Ticks Per Second），建议保留两位小数 |
| `cpu_info.cores` | `int32` | CPU 核心数 |
| `cpu_info.usage_percent` | `double` | CPU 使用率百分比（0.0 ~ 100.0） |
| `memory_info.total_bytes` | `int64` | 系统总物理内存（字节） |
| `memory_info.used_bytes` | `int64` | 系统已用物理内存（字节） |
| `memory_info.free_bytes` | `int64` | 系统空闲物理内存（字节） |
| `jvm_info.max_memory_bytes` | `int64` | JVM 最大可用内存（字节），对应 `-Xmx` |
| `jvm_info.used_memory_bytes` | `int64` | JVM 当前已用内存（字节） |

所有内存相关字段统一使用字节（bytes）为单位，不要使用 KB、MB 等换算后的值。

---

## 4. 数据存储说明

负载数据的存储策略如下：

- **聚合周期**：Go 服务端每分钟聚合一次，取该分钟内最后一次上报的采样值。
- **保留时长**：数据保留 30 天，超期自动清理。
- **存储格式**：每分钟一个采样点，以 JSON 数组形式按时间序列存储。
- **查询方式**：管理面板通过 RESTful API 查询指定服务器、指定时间范围的负载数据。

插件端无需关心存储细节，只需按时上报即可。

---

## 5. 注意事项

- **心跳频率**：建议不低于每 30 秒一次。频率过低会导致聚合窗口内缺少采样点，数据出现断档。
- **时区**：服务端统一使用 `Asia/Shanghai`（UTC+8），时间戳以毫秒级 Unix timestamp 传递即可。
- **字段缺失**：如果 `cpu_info`、`memory_info`、`jvm_info` 字段未填充（为 null），该分钟采样将缺少对应指标。建议尽量填充所有指标。
- **不持久化的字段**：`disk_info`、`version_info`、`worlds` 等字段会被写入 Redis 缓存用于实时展示，但不会进行数据库持久化。
- **server_name**：必须与后台注册的服务器名称完全一致，否则心跳会被忽略。

---

## 6. HeartbeatEvent Proto 定义

以下是 `proto/status/v1/status.proto` 中的完整 HeartbeatEvent 定义，供 Java 端生成 protobuf 代码时参考。

```protobuf
syntax = "proto3";

package frontleaves.status.v1;

option go_package = "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/status/v1;statuspb";

import "link/base.proto";

// ServerStatusService 服务器状态服务
//
// 处理 Minecraft 服务器上报的实时心跳事件。
service ServerStatusService {
  // ServerEventStream 服务器事件客户端流
  rpc ServerEventStream(stream ServerEventStreamRequest) returns (ServerEventStreamResponse);
}

// ServerEventType 服务器事件类型
enum ServerEventType {
  SERVER_EVENT_TYPE_UNSPECIFIED = 0;
  SERVER_EVENT_TYPE_HEARTBEAT = 1;
}

// ServerEventStreamRequest 服务器事件流请求
message ServerEventStreamRequest {
  // 事件类型标识
  ServerEventType event_type = 1;
  // 事件内容
  oneof event {
    // 心跳事件
    HeartbeatEvent heartbeat_event = 11;
  }
}

// ServerEventStreamResponse 服务器事件流响应
message ServerEventStreamResponse {
  xBase.BaseResponse base_response = 1;
}

// HeartbeatEvent 心跳事件
message HeartbeatEvent {
  // 服务器名称
  string server_name = 1;
  // 服务器 TPS
  double tps = 2;
  // 在线玩家数量
  int32 online_player = 3;
  // CPU 信息
  CpuInfo cpu_info = 5;
  // 内存信息
  MemoryInfo memory_info = 6;
  // 磁盘信息
  DiskInfo disk_info = 7;
  // JVM 信息
  JvmInfo jvm_info = 8;
  // 版本信息
  ServerVersionInfo version_info = 9;
  // 世界列表
  repeated WorldInfo worlds = 10;
}

// CpuInfo CPU 信息
message CpuInfo {
  // CPU 核心数
  int32 cores = 1;
  // CPU 使用率（百分比）
  double usage_percent = 2;
}

// MemoryInfo 内存信息
message MemoryInfo {
  // 总内存（字节）
  int64 total_bytes = 1;
  // 已用内存（字节）
  int64 used_bytes = 2;
  // 空闲内存（字节）
  int64 free_bytes = 3;
}

// DiskInfo 磁盘信息
message DiskInfo {
  // 总磁盘空间（字节）
  int64 total_bytes = 1;
  // 已用磁盘空间（字节）
  int64 used_bytes = 2;
}

// JvmInfo JVM 信息
message JvmInfo {
  // JVM 最大内存（字节）
  int64 max_memory_bytes = 1;
  // JVM 已用内存（字节）
  int64 used_memory_bytes = 2;
}

// ServerVersionInfo 服务器版本信息
message ServerVersionInfo {
  // 服务器版本
  string server_version = 1;
  // Minecraft 版本
  string mc_version = 2;
}

// WorldInfo 世界信息
message WorldInfo {
  // 世界名称
  string world_name = 1;
  // 玩家数量
  int32 player_count = 2;
  // 实体数量
  int32 entity_count = 3;
  // 已加载区块数
  int32 loaded_chunks = 4;
}
```

---

## 7. gRPC 连接示例代码

以下示例展示如何在 Bukkit 插件中采集负载数据并通过 gRPC 心跳流上报。

### 7.1 构建 HeartbeatEvent

```java
import com.sun.management.OperatingSystemMXBean;
import frontleaves.status.v1.Status.*;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.stub.StreamObserver;
import lang.Thread;
import org.bukkit.Bukkit;

import java.lang.management.ManagementFactory;

public class HeartbeatReporter {

    private final String serverName;
    private final ManagedChannel channel;
    private StreamObserver<ServerEventStreamRequest> requestObserver;

    public HeartbeatReporter(String target, String serverName, String secretKey) {
        this.serverName = serverName;
        this.channel = ManagedChannelBuilder.forTarget(target)
                .usePlaintext()
                .interceptor(new PluginSecretKeyInterceptor(secretKey))
                .build();

        ServerStatusServiceGrpc.ServerStatusServiceStub asyncStub =
                ServerStatusServiceGrpc.newStub(channel);

        this.requestObserver = asyncStub.serverEventStream(
                new StreamObserver<ServerEventStreamResponse>() {
                    @Override
                    public void onNext(ServerEventStreamResponse value) {
                        // 服务端响应，通常为空
                    }

                    @Override
                    public void onError(Throwable t) {
                        // 处理连接错误，触发重连逻辑
                    }

                    @Override
                    public void onCompleted() {
                        // 流关闭
                    }
                }
        );
    }

    public void sendHeartbeat() {
        HeartbeatEvent heartbeat = buildHeartbeatEvent();

        ServerEventStreamRequest request = ServerEventStreamRequest.newBuilder()
                .setEventType(ServerEventType.SERVER_EVENT_TYPE_HEARTBEAT)
                .setHeartbeatEvent(heartbeat)
                .build();

        requestObserver.onNext(request);
    }

    private HeartbeatEvent buildHeartbeatEvent() {
        // 采集 TPS（Bukkit 不直接提供 TPS，此处以 Spigot 的 MinecraftServer 为例）
        double tps = getTps();

        // 采集 CPU 信息
        OperatingSystemMXBean osBean = (OperatingSystemMXBean)
                ManagementFactory.getOperatingSystemMXBean();
        CpuInfo cpuInfo = CpuInfo.newBuilder()
                .setCores(osBean.getAvailableProcessors())
                .setUsagePercent(osBean.getProcessCpuLoad() * 100.0)
                .build();

        // 采集系统内存信息
        long totalMemory = osBean.getTotalPhysicalMemorySize();
        long freeMemory = osBean.getFreePhysicalMemorySize();
        MemoryInfo memoryInfo = MemoryInfo.newBuilder()
                .setTotalBytes(totalMemory)
                .setUsedBytes(totalMemory - freeMemory)
                .setFreeBytes(freeMemory)
                .build();

        // 采集 JVM 内存信息
        Runtime runtime = Runtime.getRuntime();
        JvmInfo jvmInfo = JvmInfo.newBuilder()
                .setMaxMemoryBytes(runtime.maxMemory())
                .setUsedMemoryBytes(runtime.totalMemory() - runtime.freeMemory())
                .build();

        return HeartbeatEvent.newBuilder()
                .setServerName(serverName)
                .setTps(tps)
                .setOnlinePlayer(Bukkit.getOnlinePlayers().size())
                .setCpuInfo(cpuInfo)
                .setMemoryInfo(memoryInfo)
                .setJvmInfo(jvmInfo)
                .build();
    }

    /**
     * 获取服务器 TPS。
     * Spigot/Paper 服务器可通过反射获取 MinecraftServer.recentTps。
     * 如果使用 Paper，可直接调用 Bukkit.getServer() 对应的 API。
     */
    private double getTps() {
        try {
            // Paper 服务器的简便方式
            // double[] tps = Bukkit.getServer().getTPS();
            // return tps[0];

            // Spigot 通用方式：反射获取 MinecraftServer.recentTps
            Object server = Class.forName(
                    "net.minecraft.server.MinecraftServer"
            ).getMethod("getServer").invoke(null);
            double[] recentTps = (double[]) server.getClass()
                    .getField("recentTps").get(server);
            return recentTps[0];
        } catch (Exception e) {
            return 20.0; // 无法获取时返回默认值
        }
    }

    public void shutdown() {
        requestObserver.onCompleted();
        channel.shutdown();
    }
}
```

### 7.2 定时上报

在插件主类中启动定时任务，每 15 秒上报一次心跳：

```java
import org.bukkit.plugin.java.JavaPlugin;
import org.bukkit.scheduler.BukkitRunnable;

public class ServerStatusPlugin extends JavaPlugin {

    private HeartbeatReporter reporter;

    @Override
    public void onEnable() {
        String target = getConfig().getString("grpc.target", "localhost:9090");
        String serverName = getConfig().getString("server.name", "survival");
        String secretKey = getConfig().getString("grpc.secret-key", "");

        reporter = new HeartbeatReporter(target, serverName, secretKey);

        // 每 300 tick（15 秒）上报一次心跳
        new BukkitRunnable() {
            @Override
            public void run() {
                reporter.sendHeartbeat();
            }
        }.runTaskTimerAsynchronously(this, 0L, 300L);
    }

    @Override
    public void onDisable() {
        if (reporter != null) {
            reporter.shutdown();
        }
    }
}
```

### 7.3 gRPC 认证拦截器

服务端要求所有 gRPC 请求携带 `plugin-secret-key` 元数据。以下是一个简单的 ClientInterceptor 实现：

```java
import io.grpc.CallOptions;
import io.grpc.Channel;
import io.grpc.ClientCall;
import io.grpc.ClientInterceptor;
import io.grpc.ForwardingClientCall;
import io.grpc.Metadata;
import io.grpc.MethodDescriptor;

public class PluginSecretKeyInterceptor implements ClientInterceptor {

    private static final Metadata.Key<String> SECRET_KEY_HEADER =
            Metadata.Key.of("plugin-secret-key", Metadata.ASCII_STRING_MARSHALLER);

    private final String secretKey;

    public PluginSecretKeyInterceptor(String secretKey) {
        this.secretKey = secretKey;
    }

    @Override
    public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
            MethodDescriptor<ReqT, RespT> method,
            CallOptions callOptions,
            Channel next) {
        return new ForwardingClientCall.SimpleForwardingClientCall<>(
                next.newCall(method, callOptions)) {
            @Override
            public void start(Listener<RespT> responseListener, Metadata headers) {
                headers.put(SECRET_KEY_HEADER, secretKey);
                super.start(responseListener, headers);
            }
        };
    }
}
```

### 7.4 关键依赖

在 `pom.xml` 或 `build.gradle` 中添加 gRPC 和 protobuf 相关依赖：

```xml
<!-- Maven 示例 -->
<dependency>
    <groupId>io.grpc</groupId>
    <artifactId>grpc-netty</artifactId>
    <version>1.62.2</version>
</dependency>
<dependency>
    <groupId>io.grpc</groupId>
    <artifactId>grpc-protobuf</artifactId>
    <version>1.62.2</version>
</dependency>
<dependency>
    <groupId>io.grpc</groupId>
    <artifactId>grpc-stub</artifactId>
    <version>1.62.2</version>
</dependency>
```

使用 `protobuf-maven-plugin` 或 Gradle 的 `protobuf` 插件从 `.proto` 文件生成 Java 代码，生成后的类位于 `frontleaves.status.v1` 包下。
