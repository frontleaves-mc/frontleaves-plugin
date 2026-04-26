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

type GameProfileTitleRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewGameProfileTitleRepo(db *gorm.DB, rdb *redis.Client) *GameProfileTitleRepo {
	return &GameProfileTitleRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileTitleRepo"),
	}
}

func (r *GameProfileTitleRepo) Create(ctx context.Context, gpt *entity.GameProfileTitle) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家称号关联")
	if err := r.db.WithContext(ctx).Create(gpt).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家称号关联失败", false, err)
	}
	return nil
}

func (r *GameProfileTitleRepo) Delete(ctx context.Context, profileUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除玩家称号关联")
	if err := r.db.WithContext(ctx).Where("game_profile_uuid = ? AND title_id = ?", profileUUID, titleID).Delete(&entity.GameProfileTitle{}).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除玩家称号关联失败", false, err)
	}
	return nil
}

func (r *GameProfileTitleRepo) GetByGameProfileUUID(ctx context.Context, profileUUID uuid.UUID) ([]entity.GameProfileTitle, *xError.Error) {
	r.log.Info(ctx, "GetByGameProfileUUID - 查询玩家拥有的称号")
	var gpts []entity.GameProfileTitle
	if err := r.db.WithContext(ctx).Preload("Title").Where("game_profile_uuid = ?", profileUUID).Find(&gpts).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家称号列表失败", false, err)
	}
	return gpts, nil
}

func (r *GameProfileTitleRepo) GetEquippedByGameProfileUUID(ctx context.Context, profileUUID uuid.UUID) (*entity.GameProfileTitle, *xError.Error) {
	r.log.Info(ctx, "GetEquippedByGameProfileUUID - 查询玩家装备的称号")
	var gpt entity.GameProfileTitle
	if err := r.db.WithContext(ctx).Preload("Title").Where("game_profile_uuid = ? AND is_equipped = ?", profileUUID, true).First(&gpt).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询装备称号失败", false, err)
	}
	return &gpt, nil
}

func (r *GameProfileTitleRepo) HasTitle(ctx context.Context, profileUUID uuid.UUID, titleID xSnowflake.SnowflakeID) (bool, *xError.Error) {
	r.log.Info(ctx, "HasTitle - 检查玩家是否拥有称号")
	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ? AND title_id = ?", profileUUID, titleID).Count(&count).Error; err != nil {
		return false, xError.NewError(nil, xError.DatabaseError, "检查玩家称号失败", false, err)
	}
	return count > 0, nil
}

func (r *GameProfileTitleRepo) EquipTitle(ctx context.Context, db *gorm.DB, profileUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "EquipTitle - 装备称号")
	tx := db.WithContext(ctx)
	if err := tx.Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ?", profileUUID).Update("is_equipped", false).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "卸下旧称号失败", false, err)
	}
	if err := tx.Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ? AND title_id = ?", profileUUID, titleID).Update("is_equipped", true).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "装备称号失败", false, err)
	}
	return nil
}

func (r *GameProfileTitleRepo) UnequipTitle(ctx context.Context, db *gorm.DB, profileUUID uuid.UUID) *xError.Error {
	r.log.Info(ctx, "UnequipTitle - 卸下称号")
	if err := db.WithContext(ctx).Model(&entity.GameProfileTitle{}).Where("game_profile_uuid = ? AND is_equipped = ?", profileUUID, true).Update("is_equipped", false).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "卸下称号失败", false, err)
	}
	return nil
}
