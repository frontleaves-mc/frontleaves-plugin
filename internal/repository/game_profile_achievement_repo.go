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

type GameProfileAchievementRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewGameProfileAchievementRepo(db *gorm.DB, rdb *redis.Client) *GameProfileAchievementRepo {
	return &GameProfileAchievementRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileAchievementRepo"),
	}
}

func (r *GameProfileAchievementRepo) Create(ctx context.Context, gpa *entity.GameProfileAchievement) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家成就")
	if err := r.db.WithContext(ctx).Create(gpa).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家成就失败", false, err)
	}
	return nil
}

func (r *GameProfileAchievementRepo) GetByGameProfileAndAchievement(ctx context.Context, profileUUID uuid.UUID, achievementID xSnowflake.SnowflakeID) (*entity.GameProfileAchievement, *xError.Error) {
	r.log.Info(ctx, "GetByGameProfileAndAchievement - 查询玩家成就")
	var gpa entity.GameProfileAchievement
	if err := r.db.WithContext(ctx).Preload("Achievement").Where("game_profile_uuid = ? AND achievement_id = ?", profileUUID, achievementID).First(&gpa).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家成就失败", false, err)
	}
	return &gpa, nil
}

func (r *GameProfileAchievementRepo) ListByGameProfile(ctx context.Context, profileUUID uuid.UUID) ([]entity.GameProfileAchievement, *xError.Error) {
	r.log.Info(ctx, "ListByGameProfile - 查询玩家所有成就")
	var gpas []entity.GameProfileAchievement
	if err := r.db.WithContext(ctx).Preload("Achievement").Where("game_profile_uuid = ?", profileUUID).Order("created_at DESC").Find(&gpas).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家成就列表失败", false, err)
	}
	return gpas, nil
}

func (r *GameProfileAchievementRepo) UpdateProgress(ctx context.Context, id xSnowflake.SnowflakeID, progress int64) *xError.Error {
	r.log.Info(ctx, "UpdateProgress - 更新成就进度")
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileAchievement{}).Where("id = ?", id).Update("progress", progress).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新成就进度失败", false, err)
	}
	return nil
}

func (r *GameProfileAchievementRepo) UpdateStatus(ctx context.Context, id xSnowflake.SnowflakeID, status entity.AchievementStatus) *xError.Error {
	r.log.Info(ctx, "UpdateStatus - 更新成就状态")
	updates := map[string]interface{}{"status": status}
	if status == entity.AchievementStatusCompleted {
		updates["completed_at"] = gorm.Expr("NOW()")
	}
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileAchievement{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新成就状态失败", false, err)
	}
	return nil
}
