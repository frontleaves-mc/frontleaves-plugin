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

type GameProfileRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewGameProfileRepo(db *gorm.DB) *GameProfileRepo {
	return &GameProfileRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "GameProfileRepo"),
	}
}

func (r *GameProfileRepo) GetByUUID(ctx context.Context, profileUUID uuid.UUID) (*entity.GameProfile, *xError.Error) {
	r.log.Info(ctx, "GetByUUID - 查询玩家信息")
	var gp entity.GameProfile
	if err := r.db.WithContext(ctx).Where("uuid = ?", profileUUID).First(&gp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "玩家不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家失败", false, err)
	}
	return &gp, nil
}

func (r *GameProfileRepo) CreateOrUpdate(ctx context.Context, gp *entity.GameProfile) *xError.Error {
	r.log.Info(ctx, "CreateOrUpdate - 创建或更新玩家")
	assigns := map[string]interface{}{
		"username":    gp.Username,
		"group_name":  gp.GroupName,
		"reported_at": gp.ReportedAt,
	}
	if gp.UserID != 0 {
		assigns["user_id"] = gp.UserID
	}
	result := r.db.WithContext(ctx).Where("uuid = ?", gp.UUID).
		Assign(assigns).
		FirstOrCreate(gp)
	if result.Error != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建或更新玩家失败", false, result.Error)
	}
	return nil
}

func (r *GameProfileRepo) List(ctx context.Context, page, pageSize int) ([]entity.GameProfile, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询玩家列表")
	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.GameProfile{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家总数失败", false, err)
	}
	var gps []entity.GameProfile
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&gps).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家列表失败", false, err)
	}
	return gps, total, nil
}

func (r *GameProfileRepo) GetByUserID(ctx context.Context, userID xSnowflake.SnowflakeID) ([]entity.GameProfile, *xError.Error) {
	r.log.Info(ctx, "GetByUserID - 按用户ID查询玩家列表")
	var profiles []entity.GameProfile
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&profiles).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "按用户ID查询玩家列表失败", false, err)
	}
	return profiles, nil
}
