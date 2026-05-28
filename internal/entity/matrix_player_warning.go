package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

// MatrixPlayerWarning 玩家警告日志
type MatrixPlayerWarning struct {
	ID           xSnowflake.SnowflakeID `gorm:"primaryKey;comment:警告ID" json:"id"`                   // 警告ID
	PlayerUUID   uuid.UUID              `gorm:"type:uuid;not null;index;comment:玩家UUID" json:"player_uuid"`   // 玩家UUID
	PlayerName   string                 `gorm:"type:varchar(64);not null;comment:玩家用户名" json:"player_name"` // 玩家用户名
	ServerName   string                 `gorm:"type:varchar(64);not null;comment:服务器名称" json:"server_name"` // 服务器名称
	WarningType  string                 `gorm:"type:varchar(32);not null;index;comment:警告类型" json:"warning_type"` // 警告类型
	Description  string                 `gorm:"type:text;not null;comment:警告描述" json:"description"`             // 警告描述
	RiskScore    int32                  `gorm:"default:0;comment:风险分数(0-100)" json:"risk_score"`                // 风险分数(0-100)
	ContextData  json.RawMessage        `gorm:"type:jsonb;comment:上下文数据快照" json:"context_data"`               // 上下文数据快照
	Record       json.RawMessage        `gorm:"type:jsonb;comment:行为记录快照" json:"record"`                     // 行为记录快照
	CreatedAt    time.Time              `gorm:"not null;type:timestamptz;autoCreateTime:milli;comment:创建时间" json:"created_at"` // 创建时间
}

// GetGene 返回 xSnowflake.Gene，用于标识该实体在 ID 生成时使用的基因类型。
func (_ *MatrixPlayerWarning) GetGene() xSnowflake.Gene {
	return bConst.GeneMatrixPlayerWarning
}
