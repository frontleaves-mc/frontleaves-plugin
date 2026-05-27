package repository

import (
	"context"
	"time"

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

// ListWithFilter 多条件筛选查询警告记录
func (r *MatrixWarningRepo) ListWithFilter(ctx context.Context, playerUUID, warningType, serverName string, riskScoreMin, riskScoreMax *int32, startTime, endTime *time.Time, page, pageSize int) ([]*entity.MatrixPlayerWarning, int64, *xError.Error) {
	r.log.Info(ctx, "ListWithFilter - 多条件筛选查询警告记录")

	query := r.db.WithContext(ctx).Model(&entity.MatrixPlayerWarning{})

	if playerUUID != "" {
		parsedUUID, err := uuid.Parse(playerUUID)
		if err == nil {
			query = query.Where("player_uuid = ?", parsedUUID)
		}
	}

	if warningType != "" {
		query = query.Where("warning_type = ?", warningType)
	}

	if riskScoreMin != nil && riskScoreMax != nil {
		min := *riskScoreMin
		max := *riskScoreMax
		if min > max {
			min, max = max, min
		}
		query = query.Where("risk_score BETWEEN ? AND ?", min, max)
	}

	if serverName != "" {
		query = query.Where("server_name = ?", serverName)
	}

	if startTime != nil && endTime != nil {
		start := *startTime
		end := *endTime
		if start.After(end) {
			start, end = end, start
		}
		query = query.Where("created_at BETWEEN ? AND ?", start, end)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(ctx, xError.DatabaseError, "查询警告记录失败", false, err)
	}

	var warnings []*entity.MatrixPlayerWarning
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&warnings).Error; err != nil {
		return nil, 0, xError.NewError(ctx, xError.DatabaseError, "查询警告记录失败", false, err)
	}

	return warnings, total, nil
}
