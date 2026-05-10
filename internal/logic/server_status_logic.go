package logic

import (
	"context"
	"strconv"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiServerStatus "github.com/frontleaves-mc/frontleaves-plugin/api/server_status"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
)

const heartbeatTimeout = 2 * time.Minute

type serverStatusPlayerRepo struct {
	serverPlayer *repository.ServerPlayerRepo
}

type ServerStatusLogic struct {
	logic
	repo serverStatusPlayerRepo
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
		repo: serverStatusPlayerRepo{
			serverPlayer: repository.NewServerPlayerRepo(db),
		},
	}
}

func (l *ServerStatusLogic) GetAllServerStatus(ctx context.Context) ([]apiServerStatus.ServerStatusResponse, *xError.Error) {
	l.log.Info(ctx, "GetAllServerStatus - 查询所有服务器状态")

	var publicServers []entity.Server
	if err := l.db.WithContext(ctx).Where("is_public = ? AND is_enabled = ?", true, true).Order("sort_order ASC, created_at DESC").Find(&publicServers).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询公开服务器列表失败", false, err)
	}

	servers := make([]apiServerStatus.ServerStatusResponse, 0, len(publicServers))
	now := time.Now().UnixMilli()

	for _, srv := range publicServers {
		serverName := srv.Name
		resp := &apiServerStatus.ServerStatusResponse{
			ServerName:        serverName,
			ServerDisplayName: srv.DisplayName,
			Online:            false,
			Tps:               0,
			Players:           []apiServerStatus.PlayerStatusInfo{},
		}

		serverKey := string(bConst.CacheStatusServer.Get(serverName))
		serverData, err := l.rdb.HGetAll(ctx, serverKey).Result()
		if err != nil || len(serverData) == 0 {
			// Redis 缓存过期，尝试 DB 降级
			resp = l.dbFallbackServerStatus(ctx, srv.ID, resp)
			servers = append(servers, *resp)
			continue
		}

		resp = parseServerData(serverName, srv.DisplayName, serverData, now)

		playerUUIDs, pErr := l.rdb.SMembers(ctx, string(bConst.CacheStatusServerPlayers.Get(serverName))).Result()
		if pErr != nil {
			l.log.Warn(ctx, "查询服务器玩家列表失败，尝试 DB 降级: "+serverName)
			resp = l.dbFallbackServerStatus(ctx, srv.ID, resp)
			servers = append(servers, *resp)
			continue
		}

		resp.Players = l.getPlayerInfosGraceful(ctx, playerUUIDs)
		servers = append(servers, *resp)
	}

	return servers, nil
}

func (l *ServerStatusLogic) GetServerStatus(ctx context.Context, serverName string) (*apiServerStatus.ServerStatusResponse, *xError.Error) {
	l.log.Info(ctx, "GetServerStatus - 查询服务器状态: "+serverName)

	// 检查该 server_name 是否为公开且启用的服务器
	var server entity.Server
	if err := l.db.WithContext(ctx).Where("name = ? AND is_public = ? AND is_enabled = ?", serverName, true, true).First(&server).Error; err != nil {
		return nil, xError.NewError(ctx, xError.NotFound, "服务器不存在或未公开", false, err)
	}

	resp := &apiServerStatus.ServerStatusResponse{
		ServerName:        serverName,
		ServerDisplayName: server.DisplayName,
		Online:            false,
		OnlinePlayers:     0,
		Tps:               0,
		LastHeartbeat:     0,
		Players:           []apiServerStatus.PlayerStatusInfo{},
	}

	serverKey := string(bConst.CacheStatusServer.Get(serverName))
	serverData, err := l.rdb.HGetAll(ctx, serverKey).Result()
	if err != nil {
		l.log.Warn(ctx, "查询服务器状态失败，降级为离线: "+serverName)
		return resp, nil
	}

	if len(serverData) == 0 {
		// Redis 缓存过期，尝试 DB 降级
		resp = l.dbFallbackServerStatus(ctx, server.ID, resp)
		return resp, nil
	}

	now := time.Now().UnixMilli()
	resp = parseServerData(serverName, server.DisplayName, serverData, now)

	playerUUIDs, err := l.rdb.SMembers(ctx, string(bConst.CacheStatusServerPlayers.Get(serverName))).Result()
	if err != nil {
		l.log.Warn(ctx, "查询服务器玩家列表失败，尝试 DB 降级: "+serverName)
		resp = l.dbFallbackServerStatus(ctx, server.ID, resp)
		return resp, nil
	}

	resp.Players = l.getPlayerInfosGraceful(ctx, playerUUIDs)

	return resp, nil
}

func parseServerData(serverName string, displayName string, data map[string]string, now int64) *apiServerStatus.ServerStatusResponse {
	resp := &apiServerStatus.ServerStatusResponse{
		ServerName:        serverName,
		ServerDisplayName: displayName,
		Online:            false,
		Players:           []apiServerStatus.PlayerStatusInfo{},
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

func (l *ServerStatusLogic) getPlayerInfosGraceful(ctx context.Context, playerUUIDs []string) []apiServerStatus.PlayerStatusInfo {
	players := make([]apiServerStatus.PlayerStatusInfo, 0, len(playerUUIDs))
	for _, pUUID := range playerUUIDs {
		playerData, err := l.rdb.HGetAll(ctx, string(bConst.CacheStatusPlayer.Get(pUUID))).Result()
		if err != nil {
			l.log.Warn(ctx, "查询玩家状态失败，跳过: "+pUUID)
			continue
		}
		if len(playerData) == 0 {
			// 单个 player hash 过期，尝试 DB 降级查询
			parsedUUID, parseErr := uuid.Parse(pUUID)
			if parseErr != nil {
				l.log.Warn(ctx, "玩家 UUID 格式无效，跳过: "+pUUID)
				continue
			}
			dbPlayers, xErr := l.repo.serverPlayer.GetOnlineByPlayerUUIDs(ctx, []uuid.UUID{parsedUUID})
			if xErr != nil || len(dbPlayers) == 0 {
				continue
			}
			p := dbPlayers[0]
			players = append(players, apiServerStatus.PlayerStatusInfo{
				PlayerUUID: p.PlayerUUID.String(),
				PlayerName: p.PlayerName,
				WorldName:  p.WorldName,
			})
			continue
		}
		players = append(players, apiServerStatus.PlayerStatusInfo{
			PlayerUUID: pUUID,
			PlayerName: playerData["player_name"],
			WorldName:  playerData["world_name"],
		})
	}
	return players
}

func (l *ServerStatusLogic) dbFallbackServerStatus(ctx context.Context, serverID xSnowflake.SnowflakeID, resp *apiServerStatus.ServerStatusResponse) *apiServerStatus.ServerStatusResponse {
	dbPlayers, xErr := l.repo.serverPlayer.GetOnlineByServer(ctx, serverID)
	if xErr != nil {
		l.log.Warn(ctx, "DB 降级查询在线玩家失败: "+xErr.Error())
		return resp
	}
	if len(dbPlayers) == 0 {
		return resp
	}

	resp.Online = true
	resp.OnlinePlayers = int32(len(dbPlayers))

	players := make([]apiServerStatus.PlayerStatusInfo, 0, len(dbPlayers))
	for _, p := range dbPlayers {
		players = append(players, apiServerStatus.PlayerStatusInfo{
			PlayerUUID: p.PlayerUUID.String(),
			PlayerName: p.PlayerName,
			WorldName:  p.WorldName,
		})
	}
	resp.Players = players

	return resp
}
