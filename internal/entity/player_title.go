package entity

import (
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type TitleSource int16

const (
	TitleSourceAuto       TitleSource = 1 // 自动获得（通用称号）
	TitleSourceGroup      TitleSource = 2 // 权限组匹配
	TitleSourceAdmin      TitleSource = 3 // 管理员分配
	TitleSourceAchievement TitleSource = 4 // 成就奖励
)

func (s TitleSource) String() string {
	switch s {
	case TitleSourceAuto:
		return "自动获得"
	case TitleSourceGroup:
		return "权限组匹配"
	case TitleSourceAdmin:
		return "管理员分配"
	case TitleSourceAchievement:
		return "成就奖励"
	default:
		return "未知来源"
	}
}

type PlayerTitle struct {
	xModels.BaseEntity
	PlayerUUID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:uk_player_title;comment:玩家UUID" json:"player_uuid"`
	TitleID    xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_player_title;index:idx_pt_title;comment:称号ID" json:"title_id"`
	Source     TitleSource `gorm:"not null;type:smallint;comment:获得来源" json:"source"`
	IsEquipped bool        `gorm:"not null;default:false;index:idx_pt_equipped;comment:是否装备" json:"is_equipped"`
	GrantedAt  time.Time   `gorm:"not null;type:timestamptz;comment:授予时间" json:"granted_at"`

	Title *Title  `gorm:"foreignKey:TitleID;references:ID;constraint:OnDelete:CASCADE" json:"title,omitempty"`
	Player *Player `gorm:"foreignKey:PlayerUUID;references:UUID;constraint:OnDelete:CASCADE" json:"player,omitempty"`
}

func (_ *PlayerTitle) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerTitle
}
