package entity

import (
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

// PlayerDirectMessage 玩家私信实体，用于记录玩家之间的私信消息。
type PlayerDirectMessage struct {
	ID           xSnowflake.SnowflakeID `gorm:"primaryKey;comment:私信ID"`                           // 私信ID
	SenderID     uuid.UUID              `gorm:"type:uuid;not null;index:idx_dm_sender;comment:发送者UUID"`     // 发送者UUID
	ReceiverID   uuid.UUID              `gorm:"type:uuid;not null;index:idx_dm_receiver;comment:接收者UUID"`   // 接收者UUID
	SenderName   string                 `gorm:"type:varchar(64);not null;comment:发送者用户名"`           // 发送者用户名
	ReceiverName string                 `gorm:"type:varchar(64);not null;comment:接收者用户名"`           // 接收者用户名
	Message      string                 `gorm:"type:text;not null;comment:消息内容"`                    // 消息内容
	Source       uint8                  `gorm:"type:smallint;not null;default:1;comment:消息来源(1=Game,2=Web)"` // 消息来源(1=Game,2=Web)
	IsRead       bool                   `gorm:"type:boolean;not null;default:false;comment:是否已读"`     // 是否已读
	ReadAt       *time.Time             `gorm:"type:timestamp;default:null;comment:已读时间"`             // 已读时间
}

// TableName 返回数据库表名。
func (PlayerDirectMessage) TableName() string {
	return "fp_player_direct_messages"
}

// GetGene 返回 xSnowflake.Gene，用于标识该实体在 ID 生成时使用的基因类型。
func (_ *PlayerDirectMessage) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerDirectMessage
}