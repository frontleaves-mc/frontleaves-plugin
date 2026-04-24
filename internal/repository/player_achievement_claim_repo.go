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

type PlayerAchievementClaimRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewPlayerAchievementClaimRepo(db *gorm.DB, rdb *redis.Client) *PlayerAchievementClaimRepo {
	return &PlayerAchievementClaimRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "PlayerAchievementClaimRepo"),
	}
}

func (r *PlayerAchievementClaimRepo) Create(ctx context.Context, claim *entity.PlayerAchievementClaim) *xError.Error {
	r.log.Info(ctx, "Create - 创建申领记录")
	if err := r.db.WithContext(ctx).Create(claim).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建申领记录失败", false, err)
	}
	return nil
}

func (r *PlayerAchievementClaimRepo) GetByPlayerAndAchievement(ctx context.Context, playerUUID uuid.UUID, achievementID xSnowflake.SnowflakeID) (*entity.PlayerAchievementClaim, *xError.Error) {
	r.log.Info(ctx, "GetByPlayerAndAchievement - 查询申领记录")
	var claim entity.PlayerAchievementClaim
	if err := r.db.WithContext(ctx).Where("player_uuid = ? AND achievement_id = ?", playerUUID, achievementID).First(&claim).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询申领记录失败", false, err)
	}
	return &claim, nil
}

func (r *PlayerAchievementClaimRepo) UpdateTitleClaimed(ctx context.Context, id xSnowflake.SnowflakeID, claimed bool) *xError.Error {
	r.log.Info(ctx, "UpdateTitleClaimed - 更新称号申领状态")
	if err := r.db.WithContext(ctx).Model(&entity.PlayerAchievementClaim{}).Where("id = ?", id).Update("title_claimed", claimed).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新申领状态失败", false, err)
	}
	return nil
}
