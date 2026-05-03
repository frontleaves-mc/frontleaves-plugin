package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
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
	statuspb.UnimplementedServerStatusServiceServer
}

func NewServerStatusHandler(ctx context.Context, server grpc.ServiceRegistrar) *ServerStatusHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "ServerStatusHandler")
	h := &ServerStatusHandler{grpcHandler: *base}

	statuspb.RegisterServerStatusServiceServer(server, h)
	xGrpcMiddle.UseUnary(statuspb.ServerStatusService_ServiceDesc, middleware.UnaryPluginVerify(ctx))
	xGrpcMiddle.UseStream(statuspb.ServerStatusService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

// ServerEventStream 实现服务器事件客户端流式 RPC
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

		h.dispatchEvent(ctx, req, &registeredServerName, stream)
	}
}

func (h *ServerStatusHandler) dispatchEvent(
	ctx context.Context,
	req *statuspb.ServerEventStreamRequest,
	registeredServerName *string,
	stream grpc.ClientStreamingServer[statuspb.ServerEventStreamRequest, statuspb.ServerEventStreamResponse],
) {
	switch evt := req.Event.(type) {
	case *statuspb.ServerEventStreamRequest_HeartbeatEvent:
		heartbeat := evt.HeartbeatEvent
		serverName := heartbeat.GetServerName()
		h.log.Info(ctx, "HeartbeatEvent - 服务器心跳: "+serverName)
		serverKey := string(bConst.CacheStatusServer.Get(serverName))
		h.rdb.HSet(ctx, serverKey, map[string]any{
			"online_players": strconv.FormatInt(int64(heartbeat.GetOnlinePlayers()), 10),
			"tps":           fmt.Sprintf("%.2f", heartbeat.GetTps()),
			"timestamp":      strconv.FormatInt(time.Now().UnixMilli(), 10),
		})
		h.rdb.Expire(ctx, serverKey, statusTTL)
		if *registeredServerName == "" {
			*registeredServerName = serverName
			h.setEventStream(serverName, &eventStream{
				stream:     stream,
				serverName: serverName,
				log:        h.log,
			})
		}

	case *statuspb.ServerEventStreamRequest_PlayerJoinEvent:
		join := evt.PlayerJoinEvent
		playerUUID := join.GetPlayerUuid()
		serverName := join.GetServerName()
		worldName := join.GetWorldName()
		playerName := join.GetPlayerName()
		h.log.Info(ctx, "PlayerJoinEvent - 玩家加入: "+playerName)
		playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
		h.rdb.HSet(ctx, playerKey, map[string]any{
			"server_name": serverName,
			"world_name":  worldName,
			"player_name": playerName,
			"online":      "true",
			"last_seen":   strconv.FormatInt(time.Now().UnixMilli(), 10),
		})
		h.rdb.Expire(ctx, playerKey, statusTTL)
		serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
		h.rdb.SAdd(ctx, serverPlayersKey, playerUUID)
		h.rdb.Expire(ctx, serverPlayersKey, statusTTL)
		parsedUUID, err := uuid.Parse(playerUUID)
		if err == nil {
			groupName := join.GetGroupName()
			if groupName == "" {
				groupName = "PLAYER"
			}
			if syncErr := h.service.gameProfileLogic.Upsert(ctx, 0, parsedUUID, playerName, groupName); syncErr != nil {
				h.log.Warn(ctx, "同步 GameProfile 失败: "+syncErr.Error())
			}
		}

	case *statuspb.ServerEventStreamRequest_PlayerQuitEvent:
		quit := evt.PlayerQuitEvent
		playerUUID := quit.GetPlayerUuid()
		serverName := quit.GetServerName()
		h.log.Info(ctx, "PlayerQuitEvent - 玩家离开: "+quit.GetPlayerName())
		playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
		h.rdb.HSet(ctx, playerKey, map[string]any{
			"online":    "false",
			"last_seen": strconv.FormatInt(time.Now().UnixMilli(), 10),
		})
		h.rdb.Expire(ctx, playerKey, statusTTL)
		serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
		h.rdb.SRem(ctx, serverPlayersKey, playerUUID)

	case *statuspb.ServerEventStreamRequest_PlayerSwitchWorldEvent:
		sw := evt.PlayerSwitchWorldEvent
		playerUUID := sw.GetPlayerUuid()
		h.log.Info(ctx, "PlayerSwitchWorldEvent - 玩家切换世界")
		playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
		h.rdb.HSet(ctx, playerKey, "world_name", sw.GetNewWorldName())
		h.rdb.Expire(ctx, playerKey, statusTTL)

	case *statuspb.ServerEventStreamRequest_PlayerChatEvent:
		chat := evt.PlayerChatEvent
		h.log.Info(ctx, "PlayerChatEvent - 玩家聊天: "+chat.GetPlayerName())
		parsedUUID, err := uuid.Parse(chat.GetPlayerUuid())
		if err != nil {
			h.log.Warn(ctx, "PlayerChatEvent - player_uuid 格式无效: "+err.Error())
			return
		}
		if xErr := h.service.playerChatLogic.RecordChat(ctx, parsedUUID, chat.GetPlayerName(),
			chat.GetServerName(), chat.GetWorldName(), chat.GetMessage()); xErr != nil {
			h.log.Warn(ctx, "记录聊天消息失败: "+xErr.Error())
		}

	case *statuspb.ServerEventStreamRequest_PlayerKickEvent:
		kick := evt.PlayerKickEvent
		h.log.Info(ctx, "PlayerKickEvent - 玩家被踢出: "+kick.GetPlayerName())
		parsedUUID, err := uuid.Parse(kick.GetPlayerUuid())
		if err != nil {
			h.log.Warn(ctx, "PlayerKickEvent - player_uuid 格式无效: "+err.Error())
			return
		}
		if xErr := h.service.playerEventLogic.RecordEvent(ctx, parsedUUID, kick.GetPlayerName(),
			kick.GetServerName(), kick.GetWorldName(), bConst.PlayerEventKick, kick.GetReason()); xErr != nil {
			h.log.Warn(ctx, "记录踢出事件失败: "+xErr.Error())
		}

	case *statuspb.ServerEventStreamRequest_PlayerDeathEvent:
		death := evt.PlayerDeathEvent
		h.log.Info(ctx, "PlayerDeathEvent - 玩家死亡: "+death.GetPlayerName())
		parsedUUID, err := uuid.Parse(death.GetPlayerUuid())
		if err != nil {
			h.log.Warn(ctx, "PlayerDeathEvent - player_uuid 格式无效: "+err.Error())
			return
		}
		if xErr := h.service.playerEventLogic.RecordEvent(ctx, parsedUUID, death.GetPlayerName(),
			death.GetServerName(), death.GetWorldName(), bConst.PlayerEventDeath, death.GetDeathMessage()); xErr != nil {
			h.log.Warn(ctx, "记录死亡事件失败: "+xErr.Error())
		}

	case *statuspb.ServerEventStreamRequest_PlayerGroupChangeEvent:
		gc := evt.PlayerGroupChangeEvent
		h.log.Info(ctx, "PlayerGroupChangeEvent - 权限组变更: "+gc.GetPlayerName()+" "+gc.GetOldGroupName()+" → "+gc.GetGroupName())
		parsedUUID, err := uuid.Parse(gc.GetPlayerUuid())
		if err != nil {
			h.log.Warn(ctx, "PlayerGroupChangeEvent - player_uuid 格式无效: "+err.Error())
			return
		}
		if updateErr := h.service.gameProfileLogic.UpdateGroupName(ctx, parsedUUID, gc.GetGroupName()); updateErr != nil {
			h.log.Warn(ctx, "更新 GameProfile 权限组失败: "+updateErr.Error())
		}
		if matchErr := h.service.titleLogic.MatchGroupTitle(ctx, parsedUUID, gc.GetGroupName()); matchErr != nil {
			h.log.Warn(ctx, "匹配权限组称号失败: "+matchErr.Error())
		}

	default:
		h.log.Warn(ctx, "收到未知事件类型")
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
}
