package repository

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PlayerTitleRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewPlayerTitleRepo(db *gorm.DB, rdb *redis.Client) *PlayerTitleRepo {
	return &PlayerTitleRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "PlayerTitleRepo"),
	}
}

func (r *PlayerTitleRepo) Create(ctx context.Context, playerTitle *entity.PlayerTitle) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家称号关联")
	if err := r.db.WithContext(ctx).Create(playerTitle).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家称号关联失败", false, err)
	}
	return nil
}

func (r *PlayerTitleRepo) Delete(ctx context.Context, playerUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除玩家称号关联")
	if err := r.db.WithContext(ctx).Where("player_uuid = ? AND title_id = ?", playerUUID, titleID).Delete(&entity.PlayerTitle{}).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除玩家称号关联失败", false, err)
	}
	return nil
}

func (r *PlayerTitleRepo) GetByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) ([]entity.PlayerTitle, *xError.Error) {
	r.log.Info(ctx, "GetByPlayerUUID - 查询玩家拥有的称号")
	var playerTitles []entity.PlayerTitle
	if err := r.db.WithContext(ctx).Preload("Title").Where("player_uuid = ?", playerUUID).Find(&playerTitles).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家称号列表失败", false, err)
	}
	return playerTitles, nil
}

func (r *PlayerTitleRepo) GetEquippedByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (*entity.PlayerTitle, *xError.Error) {
	r.log.Info(ctx, "GetEquippedByPlayerUUID - 查询玩家装备的称号")
	var playerTitle entity.PlayerTitle
	if err := r.db.WithContext(ctx).Preload("Title").Where("player_uuid = ? AND is_equipped = ?", playerUUID, true).First(&playerTitle).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询装备称号失败", false, err)
	}
	return &playerTitle, nil
}

func (r *PlayerTitleRepo) HasTitle(ctx context.Context, playerUUID uuid.UUID, titleID xSnowflake.SnowflakeID) (bool, *xError.Error) {
	r.log.Info(ctx, "HasTitle - 检查玩家是否拥有称号")
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.PlayerTitle{}).Where("player_uuid = ? AND title_id = ?", playerUUID, titleID).Count(&count).Error; err != nil {
		return false, xError.NewError(nil, xError.DatabaseError, "检查玩家称号失败", false, err)
	}
	return count > 0, nil
}

func (r *PlayerTitleRepo) EquipTitle(ctx context.Context, db *gorm.DB, playerUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "EquipTitle - 装备称号")
	tx := db.WithContext(ctx)

	if err := tx.Model(&entity.PlayerTitle{}).Where("player_uuid = ?", playerUUID).Update("is_equipped", false).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "卸下旧称号失败", false, err)
	}

	if err := tx.Model(&entity.PlayerTitle{}).Where("player_uuid = ? AND title_id = ?", playerUUID, titleID).Update("is_equipped", true).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "装备称号失败", false, err)
	}
	return nil
}

func (r *PlayerTitleRepo) UnequipTitle(ctx context.Context, db *gorm.DB, playerUUID uuid.UUID) *xError.Error {
	r.log.Info(ctx, "UnequipTitle - 卸下称号")
	if err := db.WithContext(ctx).Model(&entity.PlayerTitle{}).Where("player_uuid = ? AND is_equipped = ?", playerUUID, true).Update("is_equipped", false).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "卸下称号失败", false, err)
	}
	return nil
}
