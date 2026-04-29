package entity

import (
	"github.com/google/uuid"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type PlayerEvent struct {
	ID         xSnowflake.SnowflakeID `gorm:"primaryKey;comment:事件ID"`
	PlayerUUID uuid.UUID              `gorm:"type:uuid;not null;index;comment:玩家UUID"`
	PlayerName string                 `gorm:"type:varchar(64);not null;comment:玩家用户名"`
	ServerName string                 `gorm:"type:varchar(64);not null;comment:服务器名称"`
	WorldName  string                 `gorm:"type:varchar(64);not null;comment:世界名称"`
	EventType  bConst.PlayerEventType `gorm:"type:smallint;not null;index;comment:事件类型"`
	Content    string                 `gorm:"type:text;not null;comment:事件内容"`
}

func (_ *PlayerEvent) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerEvent
}
