package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

// MatrixPlayerStatistic 玩家统计表（1:1 UUID）
type MatrixPlayerStatistic struct {
	xModels.BaseEntity

	PlayerUUID uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:uk_player_uuid;comment:玩家UUID" json:"player_uuid"` // 玩家UUID
	PlayerName string          `gorm:"type:varchar(64);not null;comment:玩家用户名" json:"player_name"`                   // 玩家用户名

	BlocksBreak  json.RawMessage `gorm:"type:jsonb;not null;default:'{}';comment:方块破坏统计 {material: count}" json:"blocks_break"`   // 方块破坏统计 {material: count}
	BlocksPlace  json.RawMessage `gorm:"type:jsonb;not null;default:'{}';comment:方块放置统计 {material: count}" json:"blocks_place"`   // 方块放置统计 {material: count}
	EntitiesKill json.RawMessage `gorm:"type:jsonb;not null;default:'{}';comment:实体击杀统计 {entity_type: count}" json:"entities_kill"` // 实体击杀统计 {entity_type: count}
	Deaths       json.RawMessage `gorm:"type:jsonb;not null;default:'{}';comment:死亡统计 {cause: count}" json:"deaths"`                 // 死亡统计 {cause: count}
	ItemsUsed    json.RawMessage `gorm:"type:jsonb;not null;default:'{}';comment:物品使用统计 {material: count}" json:"items_used"`       // 物品使用统计 {material: count}

	TotalBlocksBroken   int64 `gorm:"default:0;comment:总破坏方块数" json:"total_blocks_broken"`   // 总破坏方块数
	TotalBlocksPlaced   int64 `gorm:"default:0;comment:总放置方块数" json:"total_blocks_placed"`   // 总放置方块数
	TotalEntitiesKilled int64 `gorm:"default:0;comment:总击杀实体数" json:"total_entities_killed"` // 总击杀实体数
	TotalDeaths         int64 `gorm:"default:0;comment:总死亡次数" json:"total_deaths"`           // 总死亡次数
	TotalPlayTimeMs     int64 `gorm:"default:0;comment:总游戏时长(ms)" json:"total_play_time_ms"`  // 总游戏时长(ms)

	CurrentSessionStart time.Time `gorm:"type:timestamptz;comment:当前会话开始时间" json:"current_session_start"` // 当前会话开始时间
	TotalSessions       int64     `gorm:"default:0;comment:总会话数" json:"total_sessions"`                       // 总会话数
}

// GetGene 返回 xSnowflake.Gene，用于标识该实体在 ID 生成时使用的基因类型。
func (_ *MatrixPlayerStatistic) GetGene() xSnowflake.Gene {
	return bConst.GeneMatrixPlayerStatistic
}
