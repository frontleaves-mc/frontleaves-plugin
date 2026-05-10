package logic

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
)

// PlayerInfo 心跳传递的玩家数据
type PlayerInfo struct {
	UUID  uuid.UUID
	Name  string
	World string
}

type serverPlayerRepo struct {
	serverPlayer *repository.ServerPlayerRepo
}

// ServerPlayerLogic 服务器在线玩家对账与状态管理
type ServerPlayerLogic struct {
	logic
	repo serverPlayerRepo
}

// NewServerPlayerLogic 从上下文中提取 db/rdb，构造 ServerPlayerLogic
func NewServerPlayerLogic(ctx context.Context) *ServerPlayerLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &ServerPlayerLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "ServerPlayerLogic"),
		},
		repo: serverPlayerRepo{
			serverPlayer: repository.NewServerPlayerRepo(db),
		},
	}
}

// ReconcilePlayers 心跳对账：将心跳上报的玩家列表与数据库同步
//
// 核心逻辑：
//   - 心跳中存在但数据库中不存在 → UpsertOnline（新上线）
//   - 数据库中存在但心跳中不存在 → MarkOffline（已下线）
//   - 两边都存在 → UpsertOnline（更新 name/world/last_seen）
//   - 心跳玩家列表为空时跳过对账，避免误标记全服下线
func (l *ServerPlayerLogic) ReconcilePlayers(
	ctx context.Context,
	serverID xSnowflake.SnowflakeID,
	heartbeatPlayers []PlayerInfo,
) error {
	l.log.Info(ctx, "ReconcilePlayers - 开始心跳对账")

	// 心跳玩家为空时跳过对账
	if len(heartbeatPlayers) == 0 {
		l.log.Info(ctx, "ReconcilePlayers - 心跳玩家列表为空，跳过对账")
		return nil
	}

	// 构建 heartbeat UUID 集合，O(1) 查找
	heartbeatUUIDs := make(map[uuid.UUID]PlayerInfo, len(heartbeatPlayers))
	for _, p := range heartbeatPlayers {
		heartbeatUUIDs[p.UUID] = p
	}

	// 获取数据库中当前在线的玩家 UUID
	dbUUIDs, xErr := l.repo.serverPlayer.GetOnlinePlayerUUIDsByServer(ctx, serverID)
	if xErr != nil {
		l.log.Warn(ctx, "ReconcilePlayers - 获取在线玩家UUID失败: "+xErr.Error())
		return xErr
	}

	// 数据库 UUID 集合，O(1) 查找
	dbUUIDSet := make(map[uuid.UUID]bool, len(dbUUIDs))
	for _, id := range dbUUIDs {
		dbUUIDSet[id] = true
	}

	// 心跳中存在 → UpsertOnline（新上线 + 已存在的都更新）
	for uuidVal, info := range heartbeatUUIDs {
		if xErr := l.repo.serverPlayer.UpsertOnline(ctx, serverID, uuidVal, info.Name, info.World); xErr != nil {
			l.log.Warn(ctx, "ReconcilePlayers - UpsertOnline 失败: "+xErr.Error())
			return xErr
		}
	}

	// 数据库中存在但心跳中不存在 → MarkOffline
	for _, dbID := range dbUUIDs {
		if _, exists := heartbeatUUIDs[dbID]; !exists {
			if xErr := l.repo.serverPlayer.MarkOffline(ctx, serverID, dbID); xErr != nil {
				l.log.Warn(ctx, "ReconcilePlayers - MarkOffline 失败: "+xErr.Error())
				return xErr
			}
		}
	}

	return nil
}

// PlayerJoined 玩家加入服务器
func (l *ServerPlayerLogic) PlayerJoined(
	ctx context.Context,
	serverID xSnowflake.SnowflakeID,
	playerUUID uuid.UUID,
	playerName string,
	worldName string,
) error {
	l.log.Info(ctx, "PlayerJoined - 玩家加入服务器")
	if xErr := l.repo.serverPlayer.UpsertOnline(ctx, serverID, playerUUID, playerName, worldName); xErr != nil {
		l.log.Warn(ctx, "PlayerJoined - UpsertOnline 失败: "+xErr.Error())
		return xErr
	}
	return nil
}

// PlayerLeft 玩家离开服务器
func (l *ServerPlayerLogic) PlayerLeft(
	ctx context.Context,
	serverID xSnowflake.SnowflakeID,
	playerUUID uuid.UUID,
) error {
	l.log.Info(ctx, "PlayerLeft - 玩家离开服务器")
	if xErr := l.repo.serverPlayer.MarkOffline(ctx, serverID, playerUUID); xErr != nil {
		l.log.Warn(ctx, "PlayerLeft - MarkOffline 失败: "+xErr.Error())
		return xErr
	}
	return nil
}

// ServerOffline 服务器下线，标记该服务器所有在线玩家为离线
func (l *ServerPlayerLogic) ServerOffline(
	ctx context.Context,
	serverID xSnowflake.SnowflakeID,
) error {
	l.log.Info(ctx, "ServerOffline - 服务器下线，标记全部玩家离线")
	if xErr := l.repo.serverPlayer.MarkAllOfflineByServer(ctx, serverID); xErr != nil {
		l.log.Warn(ctx, "ServerOffline - MarkAllOfflineByServer 失败: "+xErr.Error())
		return xErr
	}
	return nil
}

// GetOnlinePlayers 获取指定服务器当前在线的玩家列表
func (l *ServerPlayerLogic) GetOnlinePlayers(
	ctx context.Context,
	serverID xSnowflake.SnowflakeID,
) ([]entity.ServerPlayer, error) {
	l.log.Info(ctx, "GetOnlinePlayers - 获取在线玩家列表")
	players, xErr := l.repo.serverPlayer.GetOnlineByServer(ctx, serverID)
	if xErr != nil {
		l.log.Warn(ctx, "GetOnlinePlayers - GetOnlineByServer 失败: "+xErr.Error())
		return nil, xErr
	}
	return players, nil
}
