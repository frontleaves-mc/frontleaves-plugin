package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	xGrpcResult "github.com/bamboo-services/bamboo-base-go/plugins/grpc/result"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	statuspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/status/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	"google.golang.org/grpc"
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

	return h
}

func (h *ServerStatusHandler) PlayerJoin(
	ctx context.Context, req *statuspb.PlayerEventRequest,
) (*statuspb.PlayerEventResponse, error) {
	h.log.Info(ctx, "PlayerJoin - 玩家加入: "+req.GetPlayerName())

	playerUUID := req.GetPlayerUuid()
	serverName := req.GetServerName()
	worldName := req.GetWorldName()

	playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
	h.rdb.HSet(ctx, playerKey, map[string]any{
		"server_name": serverName,
		"world_name":  worldName,
		"player_name": req.GetPlayerName(),
		"online":      "true",
		"last_seen":   strconv.FormatInt(time.Now().UnixMilli(), 10),
	})
	h.rdb.Expire(ctx, playerKey, statusTTL)

	serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
	h.rdb.SAdd(ctx, serverPlayersKey, playerUUID)
	h.rdb.Expire(ctx, serverPlayersKey, statusTTL)

	parsedUUID, err := uuid.Parse(playerUUID)
	if err == nil {
		groupName := req.GetGroupName()
		if groupName == "" {
			groupName = "PLAYER"
		}
		if syncErr := h.service.gameProfileLogic.Upsert(ctx, 0, parsedUUID, req.GetPlayerName(), groupName); syncErr != nil {
			h.log.Warn(ctx, "同步 GameProfile 失败: "+syncErr.Error())
		}
	}

	resp := xGrpcResult.SuccessWith[*statuspb.PlayerEventResponse](ctx, "玩家加入事件已处理")
	return resp, nil
}

func (h *ServerStatusHandler) PlayerQuit(
	ctx context.Context, req *statuspb.PlayerEventRequest,
) (*statuspb.PlayerEventResponse, error) {
	h.log.Info(ctx, "PlayerQuit - 玩家离开: "+req.GetPlayerName())

	playerUUID := req.GetPlayerUuid()
	serverName := req.GetServerName()

	playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
	h.rdb.HSet(ctx, playerKey, map[string]any{
		"online":    "false",
		"last_seen": strconv.FormatInt(time.Now().UnixMilli(), 10),
	})
	h.rdb.Expire(ctx, playerKey, statusTTL)

	serverPlayersKey := string(bConst.CacheStatusServerPlayers.Get(serverName))
	h.rdb.SRem(ctx, serverPlayersKey, playerUUID)

	resp := xGrpcResult.SuccessWith[*statuspb.PlayerEventResponse](ctx, "玩家离开事件已处理")
	return resp, nil
}

func (h *ServerStatusHandler) PlayerSwitchWorld(
	ctx context.Context, req *statuspb.PlayerSwitchWorldRequest,
) (*statuspb.PlayerEventResponse, error) {
	h.log.Info(ctx, "PlayerSwitchWorld - 玩家切换世界")

	playerUUID := req.GetPlayerUuid()
	playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
	h.rdb.HSet(ctx, playerKey, "world_name", req.GetNewWorldName())
	h.rdb.Expire(ctx, playerKey, statusTTL)

	resp := xGrpcResult.SuccessWith[*statuspb.PlayerEventResponse](ctx, "切换世界事件已处理")
	return resp, nil
}

func (h *ServerStatusHandler) ServerHeartbeat(
	ctx context.Context, req *statuspb.ServerHeartbeatRequest,
) (*statuspb.ServerHeartbeatResponse, error) {
	h.log.Info(ctx, "ServerHeartbeat - 服务器心跳: "+req.GetServerName())

	serverName := req.GetServerName()
	serverKey := string(bConst.CacheStatusServer.Get(serverName))
	h.rdb.HSet(ctx, serverKey, map[string]any{
		"online_players": strconv.FormatInt(int64(req.GetOnlinePlayers()), 10),
		"tps":           fmt.Sprintf("%.2f", req.GetTps()),
		"timestamp":      strconv.FormatInt(time.Now().UnixMilli(), 10),
	})
	h.rdb.Expire(ctx, serverKey, statusTTL)

	resp := xGrpcResult.SuccessWith[*statuspb.ServerHeartbeatResponse](ctx, "心跳已处理")
	return resp, nil
}

func (h *ServerStatusHandler) PlayerChat(
	ctx context.Context, req *statuspb.PlayerChatRequest,
) (*statuspb.PlayerEventResponse, error) {
	h.log.Info(ctx, "PlayerChat - 玩家聊天: "+req.GetPlayerName())

	parsedUUID, err := uuid.Parse(req.GetPlayerUuid())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "player_uuid 格式无效", true, err)
	}

	if xErr := h.service.playerChatLogic.RecordChat(ctx, parsedUUID, req.GetPlayerName(),
		req.GetServerName(), req.GetWorldName(), req.GetMessage()); xErr != nil {
		return nil, xErr
	}

	resp := xGrpcResult.SuccessWith[*statuspb.PlayerEventResponse](ctx, "聊天消息已记录")
	return resp, nil
}

func (h *ServerStatusHandler) PlayerKick(
	ctx context.Context, req *statuspb.PlayerKickRequest,
) (*statuspb.PlayerEventResponse, error) {
	h.log.Info(ctx, "PlayerKick - 玩家被踢出: "+req.GetPlayerName())

	parsedUUID, err := uuid.Parse(req.GetPlayerUuid())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "player_uuid 格式无效", true, err)
	}

	if xErr := h.service.playerEventLogic.RecordEvent(ctx, parsedUUID, req.GetPlayerName(),
		req.GetServerName(), req.GetWorldName(), bConst.PlayerEventKick, req.GetReason()); xErr != nil {
		return nil, xErr
	}

	resp := xGrpcResult.SuccessWith[*statuspb.PlayerEventResponse](ctx, "踢出事件已记录")
	return resp, nil
}

func (h *ServerStatusHandler) PlayerDeath(
	ctx context.Context, req *statuspb.PlayerDeathRequest,
) (*statuspb.PlayerEventResponse, error) {
	h.log.Info(ctx, "PlayerDeath - 玩家死亡: "+req.GetPlayerName())

	parsedUUID, err := uuid.Parse(req.GetPlayerUuid())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "player_uuid 格式无效", true, err)
	}

	if xErr := h.service.playerEventLogic.RecordEvent(ctx, parsedUUID, req.GetPlayerName(),
		req.GetServerName(), req.GetWorldName(), bConst.PlayerEventDeath, req.GetDeathMessage()); xErr != nil {
		return nil, xErr
	}

	resp := xGrpcResult.SuccessWith[*statuspb.PlayerEventResponse](ctx, "死亡事件已记录")
	return resp, nil
}

func (h *ServerStatusHandler) PlayerGroupChange(
	ctx context.Context, req *statuspb.PlayerGroupChangeRequest,
) (*statuspb.PlayerGroupChangeResponse, error) {
	h.log.Info(ctx, "PlayerGroupChange - 权限组变更: "+req.GetPlayerName()+" "+req.GetOldGroupName()+" → "+req.GetGroupName())

	playerUUID := req.GetPlayerUuid()
	groupName := req.GetGroupName()

	// 更新 GameProfile 权限组
	parsedUUID, err := uuid.Parse(playerUUID)
	if err == nil {
		if updateErr := h.service.gameProfileLogic.UpdateGroupName(ctx, parsedUUID, groupName); updateErr != nil {
			h.log.Warn(ctx, "更新 GameProfile 权限组失败: "+updateErr.Error())
		}

		// 触发权限组称号匹配
		if matchErr := h.service.titleLogic.MatchGroupTitle(ctx, parsedUUID, groupName); matchErr != nil {
			h.log.Warn(ctx, "匹配权限组称号失败: "+matchErr.Error())
		}
	}

	resp := xGrpcResult.SuccessWith[*statuspb.PlayerGroupChangeResponse](ctx, "权限组变更已处理")
	return resp, nil
}
