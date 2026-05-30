package repository

import (
	"context"
	"testing"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建内存 SQLite 数据库并自动迁移 TransactionLog 表。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&entity.TransactionLog{})
	require.NoError(t, err)
	return db
}

// newTestTransactionLog 创建一条测试用的交易流水记录。
func newTestTransactionLog(id int64, playerUUID uuid.UUID, idempotencyKey string, txType int16) *entity.TransactionLog {
	return &entity.TransactionLog{
		ID:             xSnowflake.SnowflakeID(id),
		PlayerUUID:     playerUUID,
		PlayerName:     "TestPlayer",
		Amount:         100,
		Type:           txType,
		IdempotencyKey: idempotencyKey,
		Comment:        "test transaction",
	}
}

func TestTransactionLogRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTransactionLogRepo(db)
	ctx := context.Background()

	playerUUID := uuid.New()
	txLog := newTestTransactionLog(1, playerUUID, "key-create-1", entity.TransactionTypeTransfer)

	xErr := repo.Create(ctx, txLog)
	require.Nil(t, xErr)

	// 验证已写入
	var count int64
	err := db.Model(&entity.TransactionLog{}).Where("idempotency_key = ?", "key-create-1").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestTransactionLogRepo_BatchCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTransactionLogRepo(db)
	ctx := context.Background()

	playerUUID := uuid.New()
	logs := []*entity.TransactionLog{
		newTestTransactionLog(1, playerUUID, "batch-1", entity.TransactionTypeTransfer),
		newTestTransactionLog(2, playerUUID, "batch-2", entity.TransactionTypeAdmin),
		newTestTransactionLog(3, playerUUID, "batch-3", entity.TransactionTypeTransfer),
	}

	xErr := repo.BatchCreate(ctx, logs)
	require.Nil(t, xErr)

	var count int64
	err := db.Model(&entity.TransactionLog{}).Where("player_uuid = ?", playerUUID).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestTransactionLogRepo_BatchCreate_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTransactionLogRepo(db)
	ctx := context.Background()

	xErr := repo.BatchCreate(ctx, nil)
	assert.Nil(t, xErr)

	xErr = repo.BatchCreate(ctx, []*entity.TransactionLog{})
	assert.Nil(t, xErr)
}

func TestTransactionLogRepo_FindByPlayerUUID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTransactionLogRepo(db)
	ctx := context.Background()

	playerUUID := uuid.New()
	otherUUID := uuid.New()

	// 插入 5 条玩家 A 的记录 + 1 条玩家 B 的记录
	for i := 1; i <= 5; i++ {
		xErr := repo.Create(ctx, newTestTransactionLog(int64(i), playerUUID, "find-player-"+string(rune('A'+i)), entity.TransactionTypeTransfer))
		require.Nil(t, xErr)
	}
	require.Nil(t, repo.Create(ctx, newTestTransactionLog(10, otherUUID, "find-other", entity.TransactionTypeTransfer)))

	t.Run("page 1 size 2", func(t *testing.T) {
		logs, total, xErr := repo.FindByPlayerUUID(ctx, playerUUID, 1, 2)
		require.Nil(t, xErr)
		assert.Equal(t, int64(5), total)
		require.Len(t, logs, 2)
		// 按 created_at DESC 排序，后插入的在前
		assert.Equal(t, xSnowflake.SnowflakeID(5), logs[0].ID)
	})

	t.Run("page 2 size 2", func(t *testing.T) {
		logs, total, xErr := repo.FindByPlayerUUID(ctx, playerUUID, 2, 2)
		require.Nil(t, xErr)
		assert.Equal(t, int64(5), total)
		require.Len(t, logs, 2)
	})

	t.Run("page 3 size 2 (last page)", func(t *testing.T) {
		logs, total, xErr := repo.FindByPlayerUUID(ctx, playerUUID, 3, 2)
		require.Nil(t, xErr)
		assert.Equal(t, int64(5), total)
		require.Len(t, logs, 1)
	})

	t.Run("page 0 clamped to 1", func(t *testing.T) {
		logs, total, xErr := repo.FindByPlayerUUID(ctx, playerUUID, 0, 2)
		require.Nil(t, xErr)
		assert.Equal(t, int64(5), total)
		require.Len(t, logs, 2)
	})

	t.Run("pageSize 0 clamped to 1", func(t *testing.T) {
		logs, total, xErr := repo.FindByPlayerUUID(ctx, playerUUID, 1, 0)
		require.Nil(t, xErr)
		assert.Equal(t, int64(5), total)
		require.Len(t, logs, 1)
	})

	t.Run("pageSize > 100 clamped to 100", func(t *testing.T) {
		logs, total, xErr := repo.FindByPlayerUUID(ctx, playerUUID, 1, 200)
		require.Nil(t, xErr)
		assert.Equal(t, int64(5), total)
		require.Len(t, logs, 5)
	})

	t.Run("player with no transactions", func(t *testing.T) {
		logs, total, xErr := repo.FindByPlayerUUID(ctx, uuid.New(), 1, 10)
		require.Nil(t, xErr)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, logs)
	})
}

func TestTransactionLogRepo_FindAdminLogs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTransactionLogRepo(db)
	ctx := context.Background()

	playerUUID := uuid.New()

	// 插入 3 条管理员日志 + 2 条转账日志
	require.Nil(t, repo.Create(ctx, newTestTransactionLog(1, playerUUID, "admin-1", entity.TransactionTypeAdmin)))
	require.Nil(t, repo.Create(ctx, newTestTransactionLog(2, playerUUID, "admin-2", entity.TransactionTypeAdmin)))
	require.Nil(t, repo.Create(ctx, newTestTransactionLog(3, playerUUID, "transfer-1", entity.TransactionTypeTransfer)))
	require.Nil(t, repo.Create(ctx, newTestTransactionLog(4, playerUUID, "admin-3", entity.TransactionTypeAdmin)))
	require.Nil(t, repo.Create(ctx, newTestTransactionLog(5, playerUUID, "transfer-2", entity.TransactionTypeTransfer)))

	t.Run("page 1 size 2", func(t *testing.T) {
		logs, total, xErr := repo.FindAdminLogs(ctx, 1, 2)
		require.Nil(t, xErr)
		assert.Equal(t, int64(3), total)
		require.Len(t, logs, 2)
		for _, l := range logs {
			assert.Equal(t, entity.TransactionTypeAdmin, l.Type)
		}
	})

	t.Run("page 2 size 2", func(t *testing.T) {
		logs, total, xErr := repo.FindAdminLogs(ctx, 2, 2)
		require.Nil(t, xErr)
		assert.Equal(t, int64(3), total)
		require.Len(t, logs, 1)
	})

	t.Run("no admin logs", func(t *testing.T) {
		// 清空表
		db.Where("1=1").Delete(&entity.TransactionLog{})
		logs, total, xErr := repo.FindAdminLogs(ctx, 1, 10)
		require.Nil(t, xErr)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, logs)
	})
}

func TestTransactionLogRepo_ExistsByIdempotencyKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTransactionLogRepo(db)
	ctx := context.Background()

	playerUUID := uuid.New()
	require.Nil(t, repo.Create(ctx, newTestTransactionLog(1, playerUUID, "exists-key", entity.TransactionTypeTransfer)))

	t.Run("exists", func(t *testing.T) {
		exists, xErr := repo.ExistsByIdempotencyKey(ctx, "exists-key")
		require.Nil(t, xErr)
		assert.True(t, exists)
	})

	t.Run("not exists", func(t *testing.T) {
		exists, xErr := repo.ExistsByIdempotencyKey(ctx, "not-exists-key")
		require.Nil(t, xErr)
		assert.False(t, exists)
	})
}
