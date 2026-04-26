package repository

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type GameProfileAchievementClaimRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewGameProfileAchievementClaimRepo(db *gorm.DB) *GameProfileAchievementClaimRepo {
	return &GameProfileAchievementClaimRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileAchievementClaimRepo"),
	}
}

func (r *GameProfileAchievementClaimRepo) Create(ctx context.Context, claim *entity.GameProfileAchievementClaim) *xError.Error {
	r.log.Info(ctx, "Create - 创建申领记录")
	if err := r.db.WithContext(ctx).Create(claim).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建申领记录失败", false, err)
	}
	return nil
}

func (r *GameProfileAchievementClaimRepo) GetByGameProfileAndAchievement(ctx context.Context, profileUUID uuid.UUID, achievementID xSnowflake.SnowflakeID) (*entity.GameProfileAchievementClaim, *xError.Error) {
	r.log.Info(ctx, "GetByGameProfileAndAchievement - 查询申领记录")
	var claim entity.GameProfileAchievementClaim
	if err := r.db.WithContext(ctx).Where("game_profile_uuid = ? AND achievement_id = ?", profileUUID, achievementID).First(&claim).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询申领记录失败", false, err)
	}
	return &claim, nil
}

func (r *GameProfileAchievementClaimRepo) UpdateTitleClaimed(ctx context.Context, id xSnowflake.SnowflakeID, claimed bool) *xError.Error {
	r.log.Info(ctx, "UpdateTitleClaimed - 更新称号申领状态")
	if err := r.db.WithContext(ctx).Model(&entity.GameProfileAchievementClaim{}).Where("id = ?", id).Update("title_claimed", claimed).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新申领状态失败", false, err)
	}
	return nil
}
