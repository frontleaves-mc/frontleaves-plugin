package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MatrixWarningRepo 玩家警告 Repository
type MatrixWarningRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

// NewMatrixWarningRepo 创建 MatrixWarningRepo 实例
func NewMatrixWarningRepo(db *gorm.DB) *MatrixWarningRepo {
	return &MatrixWarningRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "MatrixWarningRepo"),
	}
}

// Create 创建玩家警告记录
func (r *MatrixWarningRepo) Create(ctx context.Context, warning *entity.MatrixPlayerWarning) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家警告记录")
	if err := r.db.WithContext(ctx).Create(warning).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家警告记录失败", false, err)
	}
	return nil
}

// ListByUUID 按玩家 UUID 分页查询警告记录
func (r *MatrixWarningRepo) ListByUUID(ctx context.Context, playerUUID uuid.UUID, limit, offset int) ([]*entity.MatrixPlayerWarning, *xError.Error) {
	r.log.Info(ctx, "ListByUUID - 查询玩家警告记录列表")
	var warnings []*entity.MatrixPlayerWarning
	if err := r.db.WithContext(ctx).
		Where("player_uuid = ?", playerUUID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&warnings).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询玩家警告记录失败", false, err)
	}
	return warnings, nil
}
