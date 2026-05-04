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
	TitleSourceAuto        TitleSource = 1
	TitleSourceGroup       TitleSource = 2
	TitleSourceAdmin       TitleSource = 3
	TitleSourceAchievement TitleSource = 4
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

type GameProfileTitle struct {
	xModels.BaseEntity
	GameProfileUUID uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_gpt_gameprofile_title;comment:玩家UUID" json:"game_profile_uuid"`
	TitleID         xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_gpt_gameprofile_title;index:idx_gpt_title;comment:称号ID" json:"title_id"`
	Source          TitleSource            `gorm:"not null;type:smallint;comment:获得来源" json:"source"`
	GrantedAt       time.Time              `gorm:"not null;comment:授予时间" json:"granted_at"`
	IsEquipped      bool                   `gorm:"not null;default:false;index:idx_gpt_equipped;comment:是否装备" json:"is_equipped"`

	Title       *Title       `gorm:"foreignKey:TitleID;references:ID;constraint:OnDelete:CASCADE" json:"title,omitempty"`
	GameProfile *GameProfile `gorm:"foreignKey:GameProfileUUID;references:UUID;constraint:OnDelete:CASCADE" json:"game_profile,omitempty"`
}

func (_ *GameProfileTitle) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerTitle
}
