package entity

import (
	"github.com/google/uuid"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type PlayerCommandLog struct {
	ID         xSnowflake.SnowflakeID `gorm:"primaryKey;comment:指令日志ID"`
	PlayerUUID uuid.UUID              `gorm:"type:uuid;not null;index:idx_player_command_log_uuid;comment:玩家UUID"`
	PlayerName string                 `gorm:"type:varchar(64);not null;comment:玩家用户名"`
	ServerName string                 `gorm:"type:varchar(64);not null;comment:服务器名称"`
	WorldName  string                 `gorm:"type:varchar(64);not null;comment:世界名称"`
	Command    string                 `gorm:"type:text;not null;comment:指令内容"`
}

func (_ *PlayerCommandLog) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerCommand
}
