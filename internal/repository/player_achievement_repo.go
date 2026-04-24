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

type PlayerAchievementRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewPlayerAchievementRepo(db *gorm.DB, rdb *redis.Client) *PlayerAchievementRepo {
	return &PlayerAchievementRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "PlayerAchievementRepo"),
	}
}

func (r *PlayerAchievementRepo) Create(ctx context.Context, pa *entity.PlayerAchievement) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家成就")
	if err := r.db.WithContext(ctx).Create(pa).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家成就失败", false, err)
	}
	return nil
}

func (r *PlayerAchievementRepo) GetByPlayerAndAchievement(ctx context.Context, playerUUID uuid.UUID, achievementID xSnowflake.SnowflakeID) (*entity.PlayerAchievement, *xError.Error) {
	r.log.Info(ctx, "GetByPlayerAndAchievement - 查询玩家成就")
	var pa entity.PlayerAchievement
	if err := r.db.WithContext(ctx).Preload("Achievement").Where("player_uuid = ? AND achievement_id = ?", playerUUID, achievementID).First(&pa).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家成就失败", false, err)
	}
	return &pa, nil
}

func (r *PlayerAchievementRepo) ListByPlayer(ctx context.Context, playerUUID uuid.UUID) ([]entity.PlayerAchievement, *xError.Error) {
	r.log.Info(ctx, "ListByPlayer - 查询玩家所有成就")
	var pas []entity.PlayerAchievement
	if err := r.db.WithContext(ctx).Preload("Achievement").Where("player_uuid = ?", playerUUID).Order("created_at DESC").Find(&pas).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家成就列表失败", false, err)
	}
	return pas, nil
}

func (r *PlayerAchievementRepo) UpdateProgress(ctx context.Context, id xSnowflake.SnowflakeID, progress int64) *xError.Error {
	r.log.Info(ctx, "UpdateProgress - 更新成就进度")
	if err := r.db.WithContext(ctx).Model(&entity.PlayerAchievement{}).Where("id = ?", id).Update("progress", progress).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新成就进度失败", false, err)
	}
	return nil
}

func (r *PlayerAchievementRepo) UpdateStatus(ctx context.Context, id xSnowflake.SnowflakeID, status entity.AchievementStatus) *xError.Error {
	r.log.Info(ctx, "UpdateStatus - 更新成就状态")
	updates := map[string]interface{}{"status": status}
	if status == entity.AchievementStatusCompleted {
		updates["completed_at"] = gorm.Expr("NOW()")
	}
	if err := r.db.WithContext(ctx).Model(&entity.PlayerAchievement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新成就状态失败", false, err)
	}
	return nil
}
