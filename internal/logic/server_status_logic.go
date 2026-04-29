package logic

import (
	"context"
	"strconv"
	"strings"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiServerStatus "github.com/frontleaves-mc/frontleaves-plugin/api/server_status"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

const heartbeatTimeout = 2 * time.Minute

type ServerStatusLogic struct {
	logic
}

func NewServerStatusLogic(ctx context.Context) *ServerStatusLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &ServerStatusLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "ServerStatusLogic"),
		},
	}
}

func (l *ServerStatusLogic) GetAllServerStatus(ctx context.Context) ([]apiServerStatus.ServerStatusResponse, *xError.Error) {
	l.log.Info(ctx, "GetAllServerStatus - 查询所有服务器状态")

	wildcardKey := string(bConst.CacheStatusServer.Get("*"))
	keys, err := l.rdb.Keys(ctx, wildcardKey).Result()
	if err != nil {
		return nil, xError.NewError(ctx, xError.CacheError, "查询服务器列表失败", false, err)
	}

	serverKeyPrefix := string(bConst.CacheStatusServer.Get(""))
	servers := make([]apiServerStatus.ServerStatusResponse, 0, len(keys))
	now := time.Now().UnixMilli()

	for _, key := range keys {
		if strings.HasSuffix(key, ":players") {
			continue
		}

		serverName := strings.TrimPrefix(key, serverKeyPrefix)
		serverData, err := l.rdb.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, xError.NewError(ctx, xError.CacheError, xError.ErrMessage("查询服务器状态失败: "+serverName), false, err)
		}
		if len(serverData) == 0 {
			continue
		}

		resp := parseServerData(serverName, serverData, now)

		playerUUIDs, pErr := l.rdb.SMembers(ctx, string(bConst.CacheStatusServerPlayers.Get(serverName))).Result()
		if pErr != nil {
			return nil, xError.NewError(ctx, xError.CacheError, xError.ErrMessage("查询服务器玩家列表失败: "+serverName), false, pErr)
		}

		var xErr *xError.Error
		resp.Players, xErr = l.getPlayerInfos(ctx, playerUUIDs)
		if xErr != nil {
			return nil, xErr
		}

		servers = append(servers, *resp)
	}

	return servers, nil
}

func (l *ServerStatusLogic) GetServerStatus(ctx context.Context, serverName string) (*apiServerStatus.ServerStatusResponse, *xError.Error) {
	l.log.Info(ctx, "GetServerStatus - 查询服务器状态: "+serverName)

	serverKey := string(bConst.CacheStatusServer.Get(serverName))
	serverData, err := l.rdb.HGetAll(ctx, serverKey).Result()
	if err != nil {
		return nil, xError.NewError(ctx, xError.CacheError, "查询服务器状态失败", false, err)
	}

	now := time.Now().UnixMilli()

	resp := &apiServerStatus.ServerStatusResponse{
		ServerName: serverName,
		Players:    []apiServerStatus.PlayerStatusInfo{},
	}

	if len(serverData) == 0 {
		return resp, nil
	}

	resp = parseServerData(serverName, serverData, now)

	playerUUIDs, err := l.rdb.SMembers(ctx, string(bConst.CacheStatusServerPlayers.Get(serverName))).Result()
	if err != nil {
		return nil, xError.NewError(ctx, xError.CacheError, "查询服务器玩家列表失败", false, err)
	}

	players, xErr := l.getPlayerInfos(ctx, playerUUIDs)
	if xErr != nil {
		return nil, xErr
	}

	resp.Players = players

	return resp, nil
}

func parseServerData(serverName string, data map[string]string, now int64) *apiServerStatus.ServerStatusResponse {
	resp := &apiServerStatus.ServerStatusResponse{
		ServerName: serverName,
		Online:     false,
		Players:    []apiServerStatus.PlayerStatusInfo{},
	}

	if op, parseErr := strconv.ParseInt(data["online_players"], 10, 32); parseErr == nil {
		resp.OnlinePlayers = int32(op)
	}
	if tps, parseErr := strconv.ParseFloat(data["tps"], 64); parseErr == nil {
		resp.Tps = tps
	}
	if ts, parseErr := strconv.ParseInt(data["timestamp"], 10, 64); parseErr == nil {
		resp.LastHeartbeat = ts
		resp.Online = now-ts < heartbeatTimeout.Milliseconds()
	}

	return resp
}

func (l *ServerStatusLogic) getPlayerInfos(ctx context.Context, playerUUIDs []string) ([]apiServerStatus.PlayerStatusInfo, *xError.Error) {
	players := make([]apiServerStatus.PlayerStatusInfo, 0, len(playerUUIDs))
	for _, pUUID := range playerUUIDs {
		playerData, err := l.rdb.HGetAll(ctx, string(bConst.CacheStatusPlayer.Get(pUUID))).Result()
		if err != nil {
			return nil, xError.NewError(ctx, xError.CacheError, xError.ErrMessage("查询玩家状态失败: "+pUUID), false, err)
		}
		if len(playerData) == 0 {
			continue
		}
		players = append(players, apiServerStatus.PlayerStatusInfo{
			PlayerUUID: pUUID,
			PlayerName: playerData["player_name"],
			WorldName:  playerData["world_name"],
		})
	}
	return players, nil
}
