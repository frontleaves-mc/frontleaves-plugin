package repository

import (
	"context"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServerPlayerRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewServerPlayerRepo(db *gorm.DB) *ServerPlayerRepo {
	return &ServerPlayerRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "ServerPlayerRepo"),
	}
}

func (r *ServerPlayerRepo) UpsertOnline(ctx context.Context, serverID xSnowflake.SnowflakeID, playerUUID uuid.UUID, playerName, worldName string) *xError.Error {
	r.log.Info(ctx, "UpsertOnline - 更新或创建玩家在线状态")
	sp := &entity.ServerPlayer{
		ServerID:   serverID,
		PlayerUUID: playerUUID,
		PlayerName: playerName,
		WorldName:  worldName,
		Online:     true,
		LastSeen:   time.Now(),
	}
	result := r.db.WithContext(ctx).
		Where("server_id = ? AND player_uuid = ?", serverID, playerUUID).
		Assign(map[string]interface{}{
			"player_name": playerName,
			"world_name":  worldName,
			"online":      true,
			"last_seen":   time.Now(),
		}).
		FirstOrCreate(sp)
	if result.Error != nil {
		return xError.NewError(ctx, xError.DatabaseError, "更新或创建玩家在线状态失败", false, result.Error)
	}
	return nil
}

func (r *ServerPlayerRepo) MarkOffline(ctx context.Context, serverID xSnowflake.SnowflakeID, playerUUID uuid.UUID) *xError.Error {
	r.log.Info(ctx, "MarkOffline - 标记玩家离线")
	result := r.db.WithContext(ctx).
		Model(&entity.ServerPlayer{}).
		Where("server_id = ? AND player_uuid = ?", serverID, playerUUID).
		Updates(map[string]interface{}{
			"online":    false,
			"last_seen": time.Now(),
		})
	if result.Error != nil {
		return xError.NewError(ctx, xError.DatabaseError, "标记玩家离线失败", false, result.Error)
	}
	return nil
}

func (r *ServerPlayerRepo) MarkAllOfflineByServer(ctx context.Context, serverID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "MarkAllOfflineByServer - 标记服务器所有玩家离线")
	result := r.db.WithContext(ctx).
		Model(&entity.ServerPlayer{}).
		Where("server_id = ? AND online = ?", serverID, true).
		Updates(map[string]interface{}{
			"online":    false,
			"last_seen": time.Now(),
		})
	if result.Error != nil {
		return xError.NewError(ctx, xError.DatabaseError, "标记服务器所有玩家离线失败", false, result.Error)
	}
	return nil
}

func (r *ServerPlayerRepo) GetOnlineByServer(ctx context.Context, serverID xSnowflake.SnowflakeID) ([]entity.ServerPlayer, *xError.Error) {
	r.log.Info(ctx, "GetOnlineByServer - 查询服务器在线玩家")
	var players []entity.ServerPlayer
	if err := r.db.WithContext(ctx).Where("server_id = ? AND online = ?", serverID, true).Find(&players).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询服务器在线玩家失败", false, err)
	}
	return players, nil
}

func (r *ServerPlayerRepo) GetOnlineByServerAndUUIDs(ctx context.Context, serverID xSnowflake.SnowflakeID, uuids []uuid.UUID) ([]entity.ServerPlayer, *xError.Error) {
	r.log.Info(ctx, "GetOnlineByServerAndUUIDs - 查询指定UUID在线玩家")
	var players []entity.ServerPlayer
	if err := r.db.WithContext(ctx).
		Where("server_id = ? AND online = ? AND player_uuid IN ?", serverID, true, uuids).
		Find(&players).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询指定UUID在线玩家失败", false, err)
	}
	return players, nil
}

func (r *ServerPlayerRepo) GetOnlinePlayerUUIDsByServer(ctx context.Context, serverID xSnowflake.SnowflakeID) ([]uuid.UUID, *xError.Error) {
	r.log.Info(ctx, "GetOnlinePlayerUUIDsByServer - 查询服务器在线玩家UUID列表")
	var uuids []uuid.UUID
	if err := r.db.WithContext(ctx).
		Model(&entity.ServerPlayer{}).
		Where("server_id = ? AND online = ?", serverID, true).
		Pluck("player_uuid", &uuids).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询服务器在线玩家UUID列表失败", false, err)
	}
	return uuids, nil
}

func (r *ServerPlayerRepo) GetOnlineByPlayerUUIDs(ctx context.Context, uuids []uuid.UUID) ([]entity.ServerPlayer, *xError.Error) {
	r.log.Info(ctx, "GetOnlineByPlayerUUIDs - 按UUID列表查询在线玩家")
	var players []entity.ServerPlayer
	if err := r.db.WithContext(ctx).Where("online = ? AND player_uuid IN ?", true, uuids).Find(&players).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "按UUID列表查询在线玩家失败", false, err)
	}
	return players, nil
}

func (r *ServerPlayerRepo) GetOnlineByPlayerName(ctx context.Context, playerName string) ([]entity.ServerPlayer, *xError.Error) {
	r.log.Info(ctx, "GetOnlineByPlayerName - 按玩家名称查询在线玩家")
	var players []entity.ServerPlayer
	if err := r.db.WithContext(ctx).Where("player_name = ? AND online = ?", playerName, true).Find(&players).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "按玩家名称查询在线玩家失败", false, err)
	}
	return players, nil
}
