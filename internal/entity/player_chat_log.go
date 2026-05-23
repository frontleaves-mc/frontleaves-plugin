package entity

import (
	"github.com/google/uuid"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type PlayerChatLog struct {
	ID         xSnowflake.SnowflakeID `gorm:"primaryKey;comment:消息ID"`
	PlayerUUID *uuid.UUID             `gorm:"type:uuid;index:idx_player_chat_log_uuid;comment:玩家UUID"`
	PlayerName string                 `gorm:"type:varchar(64);not null;comment:玩家用户名"`
	ServerName string                 `gorm:"type:varchar(64);not null;comment:服务器名称"`
	WorldName  string                 `gorm:"type:varchar(64);not null;comment:世界名称"`
	Message    string                 `gorm:"type:text;not null;comment:聊天消息内容"`
	Source     uint8                  `gorm:"type:smallint;not null;default:1;comment:消息来源(1=Game,2=Web)"`
	SenderID   xSnowflake.SnowflakeID `gorm:"comment:Web发送者用户ID"`
	UserID     *xSnowflake.SnowflakeID `gorm:"index:idx_player_chat_log_user_id;comment:关联用户ID"`
}

func (_ *PlayerChatLog) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerChatLog
}
