package repository

import (
	"context"
	"fmt"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
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

// Upsert 按玩家 UUID 更新或创建统计记录（jsonb 数值累加合并）
//
// 使用 PostgreSQL INSERT ... ON CONFLICT + jsonb_each/sum/object_agg 子查询，
// 对 blocks_break 等 5 个 jsonb 字段按 key 做**数值累加**（非简单覆盖）。
// 调用方在每次 flush 后清空 map，因此每次传入的是增量数据，需与已有数据按 key 求和。
func (r *MatrixStatisticRepo) Upsert(ctx context.Context, stat *entity.MatrixPlayerStatistic) *xError.Error {
	r.log.Info(ctx, "Upsert - 更新或创建玩家统计")

	// 手动生成 Snowflake ID（绕过 GORM BeforeCreate hook）
	stat.ID = xSnowflake.GenerateID(bConst.GeneMatrixPlayerStatistic)

	// 使用 GORM Statement 动态解析表名（含前缀）
	stmt := &gorm.Statement{DB: r.db}
	_ = stmt.Parse(&entity.MatrixPlayerStatistic{})
	tableName := stmt.Table

	upsertSQL := fmt.Sprintf(`
		INSERT INTO %s (
			id, player_uuid, player_name,
			blocks_break, blocks_place, entities_kill, deaths, items_used,
			total_blocks_broken, total_blocks_placed, total_entities_killed, total_deaths,
			total_play_time_ms, current_session_start, total_sessions,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (player_uuid) DO UPDATE SET
			player_name           = EXCLUDED.player_name,
			blocks_break = (
				SELECT COALESCE(jsonb_object_agg(key, to_jsonb(sum_val)), '{}'::jsonb)
				FROM (SELECT key, SUM(value::text::numeric) AS sum_val FROM (
					SELECT key, value FROM jsonb_each(COALESCE(%s.blocks_break, '{}'::jsonb))
					UNION ALL SELECT key, value FROM jsonb_each(EXCLUDED.blocks_break)
				) combined GROUP BY key) aggregated
			),
			blocks_place = (
				SELECT COALESCE(jsonb_object_agg(key, to_jsonb(sum_val)), '{}'::jsonb)
				FROM (SELECT key, SUM(value::text::numeric) AS sum_val FROM (
					SELECT key, value FROM jsonb_each(COALESCE(%s.blocks_place, '{}'::jsonb))
					UNION ALL SELECT key, value FROM jsonb_each(EXCLUDED.blocks_place)
				) combined GROUP BY key) aggregated
			),
			entities_kill = (
				SELECT COALESCE(jsonb_object_agg(key, to_jsonb(sum_val)), '{}'::jsonb)
				FROM (SELECT key, SUM(value::text::numeric) AS sum_val FROM (
					SELECT key, value FROM jsonb_each(COALESCE(%s.entities_kill, '{}'::jsonb))
					UNION ALL SELECT key, value FROM jsonb_each(EXCLUDED.entities_kill)
				) combined GROUP BY key) aggregated
			),
			deaths = (
				SELECT COALESCE(jsonb_object_agg(key, to_jsonb(sum_val)), '{}'::jsonb)
				FROM (SELECT key, SUM(value::text::numeric) AS sum_val FROM (
					SELECT key, value FROM jsonb_each(COALESCE(%s.deaths, '{}'::jsonb))
					UNION ALL SELECT key, value FROM jsonb_each(EXCLUDED.deaths)
				) combined GROUP BY key) aggregated
			),
			items_used = (
				SELECT COALESCE(jsonb_object_agg(key, to_jsonb(sum_val)), '{}'::jsonb)
				FROM (SELECT key, SUM(value::text::numeric) AS sum_val FROM (
					SELECT key, value FROM jsonb_each(COALESCE(%s.items_used, '{}'::jsonb))
					UNION ALL SELECT key, value FROM jsonb_each(EXCLUDED.items_used)
				) combined GROUP BY key) aggregated
			),
			total_blocks_broken   = EXCLUDED.total_blocks_broken,
			total_blocks_placed   = EXCLUDED.total_blocks_placed,
			total_entities_killed = EXCLUDED.total_entities_killed,
			total_deaths          = EXCLUDED.total_deaths,
			total_play_time_ms    = EXCLUDED.total_play_time_ms,
			current_session_start = EXCLUDED.current_session_start,
			total_sessions        = EXCLUDED.total_sessions,
			updated_at            = NOW()
	`, tableName, tableName, tableName, tableName, tableName, tableName)

	result := r.db.WithContext(ctx).Exec(upsertSQL,
		stat.ID, stat.PlayerUUID, stat.PlayerName,
		stat.BlocksBreak, stat.BlocksPlace, stat.EntitiesKill, stat.Deaths, stat.ItemsUsed,
		stat.TotalBlocksBroken, stat.TotalBlocksPlaced, stat.TotalEntitiesKilled, stat.TotalDeaths,
		stat.TotalPlayTimeMs, stat.CurrentSessionStart, stat.TotalSessions,
	)
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
