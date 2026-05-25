package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MatrixStatisticRepo 玩家统计 Repository
type MatrixStatisticRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

// NewMatrixStatisticRepo 创建 MatrixStatisticRepo 实例
func NewMatrixStatisticRepo(db *gorm.DB) *MatrixStatisticRepo {
	return &MatrixStatisticRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "MatrixStatisticRepo"),
	}
}

// Upsert 按玩家 UUID 更新或创建统计记录（增量合并）
func (r *MatrixStatisticRepo) Upsert(ctx context.Context, stat *entity.MatrixPlayerStatistic) *xError.Error {
	r.log.Info(ctx, "Upsert - 更新或创建玩家统计")
	result := r.db.WithContext(ctx).
		Where("player_uuid = ?", stat.PlayerUUID).
		Assign(map[string]interface{}{
			"player_name":           stat.PlayerName,
			"blocks_break":          stat.BlocksBreak,
			"blocks_place":          stat.BlocksPlace,
			"entities_kill":         stat.EntitiesKill,
			"deaths":                stat.Deaths,
			"items_used":            stat.ItemsUsed,
			"total_blocks_broken":   stat.TotalBlocksBroken,
			"total_blocks_placed":   stat.TotalBlocksPlaced,
			"total_entities_killed": stat.TotalEntitiesKilled,
			"total_deaths":          stat.TotalDeaths,
			"total_play_time_ms":    stat.TotalPlayTimeMs,
			"current_session_start": stat.CurrentSessionStart,
			"total_sessions":        stat.TotalSessions,
		}).
		FirstOrCreate(stat)
	if result.Error != nil {
		return xError.NewError(ctx, xError.DatabaseError, "更新或创建玩家统计失败", false, result.Error)
	}
	return nil
}

// GetByUUID 按玩家 UUID 查询统计记录
func (r *MatrixStatisticRepo) GetByUUID(ctx context.Context, playerUUID uuid.UUID) (*entity.MatrixPlayerStatistic, *xError.Error) {
	r.log.Info(ctx, "GetByUUID - 查询玩家统计")
	var stat entity.MatrixPlayerStatistic
	if err := r.db.WithContext(ctx).
		Where("player_uuid = ?", playerUUID).
		First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询玩家统计失败", false, err)
	}
	return &stat, nil
}
