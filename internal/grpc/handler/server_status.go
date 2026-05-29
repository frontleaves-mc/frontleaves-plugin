package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	statuspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/status/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const statusTTL = 5 * time.Minute

type ServerStatusHandler struct {
	grpcHandler
	*statusService
	statuspb.UnimplementedServerStatusServiceServer
}

func NewServerStatusHandler(ctx context.Context, server grpc.ServiceRegistrar) *ServerStatusHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "ServerStatusHandler")
	h := &ServerStatusHandler{
		grpcHandler:   *base,
		statusService: newStatusService(ctx),
	}

	statuspb.RegisterServerStatusServiceServer(server, h)
	xGrpcMiddle.UseUnary(statuspb.ServerStatusService_ServiceDesc, middleware.UnaryPluginVerify(ctx))
	xGrpcMiddle.UseStream(statuspb.ServerStatusService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

func (h *ServerStatusHandler) ServerEventStream(
	stream grpc.ClientStreamingServer[statuspb.ServerEventStreamRequest, statuspb.ServerEventStreamResponse],
) error {
	ctx := stream.Context()
	h.log.Info(ctx, "ServerEventStream - 新的客户端流连接")

	var registeredServerName string

	for {
		req, err := stream.Recv()
		if err != nil {
			if status.Code(err) == codes.Canceled || status.Code(err) == codes.OK {
				if registeredServerName != "" {
					h.cleanupServerStatus(ctx, registeredServerName)
					h.removeEventStream(registeredServerName)
				}
				h.log.Info(ctx, "ServerEventStream - 流正常关闭")
				return nil
			}
			h.log.Warn(ctx, "ServerEventStream - 流读取错误: "+err.Error())
			if registeredServerName != "" {
				h.cleanupServerStatus(ctx, registeredServerName)
				h.removeEventStream(registeredServerName)
			}
			return err
		}

		h.handleHeartbeat(ctx, req, &registeredServerName, stream)
	}
}

func (h *ServerStatusHandler) handleHeartbeat(
	ctx context.Context,
	req *statuspb.ServerEventStreamRequest,
	registeredServerName *string,
	stream grpc.ClientStreamingServer[statuspb.ServerEventStreamRequest, statuspb.ServerEventStreamResponse],
) {
	evt, ok := req.Event.(*statuspb.ServerEventStreamRequest_HeartbeatEvent)
	if !ok {
		h.log.Warn(ctx, "收到非心跳事件类型，跳过")
		return
	}

	heartbeat := evt.HeartbeatEvent
	serverName := heartbeat.GetServerName()
	if serverName == "" {
		h.log.Warn(ctx, "HeartbeatEvent - 收到空服务器名，跳过")
		return
	}
	h.log.Info(ctx, "HeartbeatEvent - 服务器心跳: "+serverName)

	server, xErr := h.serverLogic.GetOrCreateByName(ctx, serverName)
	if xErr != nil {
		h.log.Warn(ctx, "HeartbeatEvent - 被动创建服务器失败: "+xErr.Error())
	}

	serverKey := string(bConst.CacheStatusServer.Get(serverName))
	if server == nil || !server.IsEnabled {
		if *registeredServerName == "" {
			*registeredServerName = serverName
			h.setEventStream(serverName, &eventStream{
				stream:     stream,
				serverName: serverName,
				log:        h.log,
			})
		}
		return
	}

	serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
	onlineCount := h.rdb.SCard(ctx, serverPlayersKey).Val()

	fields := map[string]any{
		"online_players": strconv.FormatInt(onlineCount, 10),
		"tps":            fmt.Sprintf("%.2f", heartbeat.GetTps()),
		"timestamp":      strconv.FormatInt(time.Now().UnixMilli(), 10),
	}

	if cpu := heartbeat.GetCpuInfo(); cpu != nil {
		cpuJSON, _ := json.Marshal(map[string]any{
			"cores":         cpu.GetCores(),
			"usage_percent": cpu.GetUsagePercent(),
		})
		fields["cpu_info"] = string(cpuJSON)
	}

	if mem := heartbeat.GetMemoryInfo(); mem != nil {
		memJSON, _ := json.Marshal(map[string]any{
			"total_bytes": mem.GetTotalBytes(),
			"used_bytes":  mem.GetUsedBytes(),
			"free_bytes":  mem.GetFreeBytes(),
		})
		fields["memory_info"] = string(memJSON)
	}

	if disk := heartbeat.GetDiskInfo(); disk != nil {
		diskJSON, _ := json.Marshal(map[string]any{
			"total_bytes": disk.GetTotalBytes(),
			"used_bytes":  disk.GetUsedBytes(),
		})
		fields["disk_info"] = string(diskJSON)
	}

	if jvm := heartbeat.GetJvmInfo(); jvm != nil {
		jvmJSON, _ := json.Marshal(map[string]any{
			"max_memory_bytes":  jvm.GetMaxMemoryBytes(),
			"used_memory_bytes": jvm.GetUsedMemoryBytes(),
		})
		fields["jvm_info"] = string(jvmJSON)
	}

	if ver := heartbeat.GetVersionInfo(); ver != nil {
		verJSON, _ := json.Marshal(map[string]any{
			"server_version": ver.GetServerVersion(),
			"mc_version":     ver.GetMcVersion(),
		})
		fields["version_info"] = string(verJSON)
	}

	if worlds := heartbeat.GetWorlds(); len(worlds) > 0 {
		worldList := make([]map[string]any, 0, len(worlds))
		for _, w := range worlds {
			worldList = append(worldList, map[string]any{
				"world_name":    w.GetWorldName(),
				"player_count":  w.GetPlayerCount(),
				"entity_count":  w.GetEntityCount(),
				"loaded_chunks": w.GetLoadedChunks(),
			})
		}
		worldsJSON, _ := json.Marshal(worldList)
		fields["worlds"] = string(worldsJSON)
	}

	h.rdb.HSet(ctx, serverKey, fields)
	h.rdb.Expire(ctx, serverKey, statusTTL)

	h.rdb.HIncrByFloat(ctx, serverKey, "tps_sum", float64(heartbeat.GetTps()))
	h.rdb.HIncrBy(ctx, serverKey, "tps_count", 1)

	if cpu := heartbeat.GetCpuInfo(); cpu != nil {
		h.rdb.HIncrByFloat(ctx, serverKey, "cpu_usage_sum", float64(cpu.GetUsagePercent()))
		h.rdb.HIncrBy(ctx, serverKey, "cpu_usage_count", 1)
	}

	if mem := heartbeat.GetMemoryInfo(); mem != nil {
		h.rdb.HIncrBy(ctx, serverKey, "mem_used_sum", int64(mem.GetUsedBytes()))
		h.rdb.HIncrBy(ctx, serverKey, "mem_used_count", 1)
	}

	if jvm := heartbeat.GetJvmInfo(); jvm != nil {
		h.rdb.HIncrBy(ctx, serverKey, "jvm_used_sum", int64(jvm.GetUsedMemoryBytes()))
		h.rdb.HIncrBy(ctx, serverKey, "jvm_used_count", 1)
	}

	if *registeredServerName == "" {
		*registeredServerName = serverName
		h.setEventStream(serverName, &eventStream{
			stream:     stream,
			serverName: serverName,
			log:        h.log,
		})
	}
}

func (h *ServerStatusHandler) cleanupServerStatus(ctx context.Context, serverName string) {
	h.log.Info(ctx, "清理服务器状态: "+serverName)

	serverKey := string(bConst.CacheStatusServer.Get(serverName))
	h.rdb.Del(ctx, serverKey)

	serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
	playerUUIDs := h.rdb.SMembers(ctx, serverPlayersKey).Val()
	for _, playerUUID := range playerUUIDs {
		playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
		h.rdb.HSet(ctx, playerKey, "online", "false", "last_seen", strconv.FormatInt(time.Now().UnixMilli(), 10))
		h.rdb.Expire(ctx, playerKey, statusTTL)
	}
	h.rdb.Del(ctx, serverPlayersKey)

	if server, xErr := h.serverLogic.GetOrCreateByName(ctx, serverName); xErr == nil && server != nil {
		if err := h.serverPlayerLogic.ServerOffline(ctx, server.ID); err != nil {
			h.log.Warn(ctx, "cleanupServerStatus - DB 标记服务器玩家离线失败: "+err.Error())
		}
	}
}
