package entity

import (
	"github.com/google/uuid"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type PlayerChatLog struct {
	ID         xSnowflake.SnowflakeID `gorm:"primaryKey;comment:消息ID"`
	PlayerUUID uuid.UUID              `gorm:"type:uuid;not null;index;comment:玩家UUID"`
	PlayerName string                 `gorm:"type:varchar(64);not null;comment:玩家用户名"`
	ServerName string                 `gorm:"type:varchar(64);not null;comment:服务器名称"`
	WorldName  string                 `gorm:"type:varchar(64);not null;comment:世界名称"`
	Message    string                 `gorm:"type:text;not null;comment:聊天消息内容"`
}

func (_ *PlayerChatLog) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerChatLog
}
