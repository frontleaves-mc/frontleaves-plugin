package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TransactionLogRepo 交易流水数据访问层
type TransactionLogRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

// NewTransactionLogRepo 创建交易流水仓库实例
func NewTransactionLogRepo(db *gorm.DB) *TransactionLogRepo {
	return &TransactionLogRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "TransactionLogRepo"),
	}
}

// Create 创建单条交易流水记录
func (r *TransactionLogRepo) Create(ctx context.Context, log *entity.TransactionLog) *xError.Error {
	r.log.Info(ctx, "Create - 创建交易流水")
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建交易流水失败", false, err)
	}
	return nil
}

// BatchCreate 批量创建交易流水记录
func (r *TransactionLogRepo) BatchCreate(ctx context.Context, logs []*entity.TransactionLog) *xError.Error {
	r.log.Info(ctx, "BatchCreate - 批量创建交易流水")
	if len(logs) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(logs).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "批量创建交易流水失败", false, err)
	}
	return nil
}

// FindByPlayerUUID 按玩家UUID分页查询交易流水，按创建时间倒序
// page 页码（从1开始），无效时 clamp 为1
// pageSize 每页条数，clamp 到 [1, 100]
func (r *TransactionLogRepo) FindByPlayerUUID(
	ctx context.Context, playerUUID uuid.UUID, page, pageSize int,
) ([]*entity.TransactionLog, int64, *xError.Error) {
	r.log.Info(ctx, "FindByPlayerUUID - 按玩家UUID分页查询交易流水")

	// 边界保护
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	} else if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).Model(&entity.TransactionLog{}).
		Where("player_uuid = ?", playerUUID)

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家交易流水总数失败", false, err)
	}

	var logs []*entity.TransactionLog
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).
		Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家交易流水失败", false, err)
	}

	return logs, total, nil
}

// FindAdminLogs 查询管理员操作日志，按创建时间倒序
// page 页码（从1开始），无效时 clamp 为1
// pageSize 每页条数，clamp 到 [1, 100]
func (r *TransactionLogRepo) FindAdminLogs(
	ctx context.Context, page, pageSize int,
) ([]*entity.TransactionLog, int64, *xError.Error) {
	r.log.Info(ctx, "FindAdminLogs - 查询管理员操作日志")

	// 边界保护
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	} else if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).Model(&entity.TransactionLog{}).
		Where("type = ?", entity.TransactionTypeAdmin)

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询管理员操作日志总数失败", false, err)
	}

	var logs []*entity.TransactionLog
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).
		Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询管理员操作日志失败", false, err)
	}

	return logs, total, nil
}

// ExistsByIdempotencyKey 检查指定幂等键是否已存在
func (r *TransactionLogRepo) ExistsByIdempotencyKey(ctx context.Context, key string) (bool, *xError.Error) {
	r.log.Info(ctx, "ExistsByIdempotencyKey - 检查幂等键是否存在")

	var count int64
	if err := r.db.WithContext(ctx).Model(&entity.TransactionLog{}).
		Where("idempotency_key = ?", key).Count(&count).Error; err != nil {
		return false, xError.NewError(nil, xError.DatabaseError, "检查幂等键失败", false, err)
	}

	return count > 0, nil
}
