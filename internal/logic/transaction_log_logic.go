package logic

import (
	"context"
	"fmt"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
)

// TransactionLogLogic 交易流水业务编排层，负责幂等校验与仓储调用编排。
type TransactionLogLogic struct {
	repo *repository.TransactionLogRepo
	log  *xLog.LogNamedLogger
}

// NewTransactionLogLogic 创建交易流水 Logic 实例。
func NewTransactionLogLogic(repo *repository.TransactionLogRepo) *TransactionLogLogic {
	return &TransactionLogLogic{
		repo: repo,
		log:  xLog.WithName(xLog.NamedLOGC, "TransactionLogLogic"),
	}
}

// RecordTransaction 记录单条交易流水（幂等）。
// 若 IdempotencyKey 已存在则视为幂等成功，直接返回 nil。
func (l *TransactionLogLogic) RecordTransaction(ctx context.Context, log *entity.TransactionLog) *xError.Error {
	l.log.Info(ctx, "RecordTransaction - 记录交易流水")

	exists, xErr := l.repo.ExistsByIdempotencyKey(ctx, log.IdempotencyKey)
	if xErr != nil {
		l.log.Warn(ctx, "RecordTransaction - 检查幂等键失败: "+xErr.Error())
		return xErr
	}
	if exists {
		l.log.Info(ctx, "RecordTransaction - 幂等键已存在，跳过: "+log.IdempotencyKey)
		return nil
	}

	return l.repo.Create(ctx, log)
}

// RecordBatchTransactions 批量记录交易流水（逐条幂等）。
// 对于每条记录，若 IdempotencyKey 已存在则跳过；其余通过 BatchCreate 批量插入。
func (l *TransactionLogLogic) RecordBatchTransactions(ctx context.Context, logs []*entity.TransactionLog) *xError.Error {
	l.log.Info(ctx, "RecordBatchTransactions - 批量记录交易流水")

	var batch []*entity.TransactionLog
	for _, item := range logs {
		exists, xErr := l.repo.ExistsByIdempotencyKey(ctx, item.IdempotencyKey)
		if xErr != nil {
			l.log.Warn(ctx, "RecordBatchTransactions - 检查幂等键失败: "+xErr.Error())
			return xErr
		}
		if exists {
			l.log.Info(ctx, "RecordBatchTransactions - 幂等键已存在，跳过: "+item.IdempotencyKey)
			continue
		}
		batch = append(batch, item)
	}

	if len(batch) == 0 {
		l.log.Info(ctx, "RecordBatchTransactions - 所有记录均幂等，无需插入")
		return nil
	}

	l.log.Info(ctx, fmt.Sprintf("RecordBatchTransactions - 批量插入 %d 条记录", len(batch)))
	return l.repo.BatchCreate(ctx, batch)
}

// GetPlayerTransactions 查询玩家交易流水（分页）。
func (l *TransactionLogLogic) GetPlayerTransactions(
	ctx context.Context, playerUUID uuid.UUID, page, pageSize int,
) ([]*entity.TransactionLog, int64, *xError.Error) {
	l.log.Info(ctx, "GetPlayerTransactions - 查询玩家交易流水")
	return l.repo.FindByPlayerUUID(ctx, playerUUID, page, pageSize)
}

// GetAdminAuditLogs 查询管理员操作日志（分页）。
func (l *TransactionLogLogic) GetAdminAuditLogs(
	ctx context.Context, page, pageSize int,
) ([]*entity.TransactionLog, int64, *xError.Error) {
	l.log.Info(ctx, "GetAdminAuditLogs - 查询管理员操作日志")
	return l.repo.FindAdminLogs(ctx, page, pageSize)
}
