package logic

import (
	"context"
	"testing"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTransactionLogLogic 创建测试用的 TransactionLogLogic 实例。
func setupTransactionLogLogic(t *testing.T) *TransactionLogLogic {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&entity.TransactionLog{})
	require.NoError(t, err)
	return NewTransactionLogLogic(repository.NewTransactionLogRepo(db))
}

// newTestTxnLog 创建一条测试交易流水。
func newTestTxnLog(id int64, playerUUID uuid.UUID, key string, txType int16) *entity.TransactionLog {
	return &entity.TransactionLog{
		ID:             xSnowflake.SnowflakeID(id),
		PlayerUUID:     playerUUID,
		PlayerName:     "TestPlayer",
		Amount:         100,
		Type:           txType,
		IdempotencyKey: key,
		Comment:        "test",
	}
}

func TestTransactionLogLogic_RecordTransaction_New(t *testing.T) {
	logicInstance := setupTransactionLogLogic(t)
	ctx := context.Background()
	playerUUID := uuid.New()

	txLog := newTestTxnLog(1, playerUUID, "idem-new", entity.TransactionTypeTransfer)

	xErr := logicInstance.RecordTransaction(ctx, txLog)
	require.Nil(t, xErr)

	// 验证已写入
	exists, xErr := logicInstance.repo.ExistsByIdempotencyKey(ctx, "idem-new")
	require.Nil(t, xErr)
	assert.True(t, exists)
}

func TestTransactionLogLogic_RecordTransaction_Idempotent(t *testing.T) {
	logicInstance := setupTransactionLogLogic(t)
	ctx := context.Background()
	playerUUID := uuid.New()

	txLog := newTestTxnLog(1, playerUUID, "idem-dup", entity.TransactionTypeTransfer)

	// 第一次插入正常
	require.Nil(t, logicInstance.RecordTransaction(ctx, txLog))

	// 第二次相同幂等键，应该幂等返回 nil
	xErr := logicInstance.RecordTransaction(ctx, txLog)
	assert.Nil(t, xErr)

	// 确认只有一条记录
	logs, total, xErr := logicInstance.GetPlayerTransactions(ctx, playerUUID, 1, 10)
	require.Nil(t, xErr)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
}

func TestTransactionLogLogic_RecordBatchTransactions(t *testing.T) {
	logicInstance := setupTransactionLogLogic(t)
	ctx := context.Background()
	playerUUID := uuid.New()

	logs := []*entity.TransactionLog{
		newTestTxnLog(1, playerUUID, "batch-new-1", entity.TransactionTypeTransfer),
		newTestTxnLog(2, playerUUID, "batch-new-2", entity.TransactionTypeAdmin),
		newTestTxnLog(3, playerUUID, "batch-new-3", entity.TransactionTypeTransfer),
	}

	xErr := logicInstance.RecordBatchTransactions(ctx, logs)
	require.Nil(t, xErr)

	_, total, xErr := logicInstance.GetPlayerTransactions(ctx, playerUUID, 1, 10)
	require.Nil(t, xErr)
	assert.Equal(t, int64(3), total)
}

func TestTransactionLogLogic_RecordBatchTransactions_MixedNewAndDuplicate(t *testing.T) {
	logicInstance := setupTransactionLogLogic(t)
	ctx := context.Background()
	playerUUID := uuid.New()

	// 先插入一条
	require.Nil(t, logicInstance.RecordTransaction(ctx, newTestTxnLog(1, playerUUID, "mixed-dup", entity.TransactionTypeTransfer)))

	// 批量：1 条重复 + 2 条新
	logs := []*entity.TransactionLog{
		newTestTxnLog(2, playerUUID, "mixed-dup", entity.TransactionTypeTransfer), // 幂等跳过
		newTestTxnLog(3, playerUUID, "mixed-new-1", entity.TransactionTypeAdmin),
		newTestTxnLog(4, playerUUID, "mixed-new-2", entity.TransactionTypeTransfer),
	}

	xErr := logicInstance.RecordBatchTransactions(ctx, logs)
	require.Nil(t, xErr)

	_, total, xErr := logicInstance.GetPlayerTransactions(ctx, playerUUID, 1, 10)
	require.Nil(t, xErr)
	assert.Equal(t, int64(3), total) // 1 条原始 + 2 条新
}

func TestTransactionLogLogic_RecordBatchTransactions_AllDuplicate(t *testing.T) {
	logicInstance := setupTransactionLogLogic(t)
	ctx := context.Background()
	playerUUID := uuid.New()

	// 插入两条
	require.Nil(t, logicInstance.RecordTransaction(ctx, newTestTxnLog(1, playerUUID, "all-dup-1", entity.TransactionTypeTransfer)))
	require.Nil(t, logicInstance.RecordTransaction(ctx, newTestTxnLog(2, playerUUID, "all-dup-2", entity.TransactionTypeAdmin)))

	// 批量全重复
	logs := []*entity.TransactionLog{
		newTestTxnLog(3, playerUUID, "all-dup-1", entity.TransactionTypeTransfer),
		newTestTxnLog(4, playerUUID, "all-dup-2", entity.TransactionTypeAdmin),
	}

	xErr := logicInstance.RecordBatchTransactions(ctx, logs)
	assert.Nil(t, xErr)

	_, total, xErr := logicInstance.GetPlayerTransactions(ctx, playerUUID, 1, 10)
	require.Nil(t, xErr)
	assert.Equal(t, int64(2), total) // 未增加
}

func TestTransactionLogLogic_GetPlayerTransactions(t *testing.T) {
	logicInstance := setupTransactionLogLogic(t)
	ctx := context.Background()
	playerUUID := uuid.New()

	for i := 1; i <= 3; i++ {
		key := "get-player-" + string(rune('A'+i))
		require.Nil(t, logicInstance.RecordTransaction(ctx, newTestTxnLog(int64(i), playerUUID, key, entity.TransactionTypeTransfer)))
	}

	logs, total, xErr := logicInstance.GetPlayerTransactions(ctx, playerUUID, 1, 10)
	require.Nil(t, xErr)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)
}

func TestTransactionLogLogic_GetAdminAuditLogs(t *testing.T) {
	logicInstance := setupTransactionLogLogic(t)
	ctx := context.Background()
	playerUUID := uuid.New()

	require.Nil(t, logicInstance.RecordTransaction(ctx, newTestTxnLog(1, playerUUID, "admin-1", entity.TransactionTypeAdmin)))
	require.Nil(t, logicInstance.RecordTransaction(ctx, newTestTxnLog(2, playerUUID, "admin-2", entity.TransactionTypeAdmin)))
	require.Nil(t, logicInstance.RecordTransaction(ctx, newTestTxnLog(3, playerUUID, "transfer-1", entity.TransactionTypeTransfer)))

	logs, total, xErr := logicInstance.GetAdminAuditLogs(ctx, 1, 10)
	require.Nil(t, xErr)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)
	for _, l := range logs {
		assert.Equal(t, entity.TransactionTypeAdmin, l.Type)
	}
}
