package entity

import (
	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type PlayerAchievementClaim struct {
	xModels.BaseEntity
	PlayerUUID    uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_pac_player_ach;comment:玩家UUID" json:"player_uuid"`
	AchievementID xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_pac_player_ach;comment:成就ID" json:"achievement_id"`
	TitleClaimed  bool                   `gorm:"not null;default:false;comment:称号奖励是否已发放" json:"title_claimed"`

	Achievement *Achievement `gorm:"foreignKey:AchievementID;references:ID;constraint:OnDelete:CASCADE" json:"achievement,omitempty"`
	Player      *Player      `gorm:"foreignKey:PlayerUUID;references:UUID;constraint:OnDelete:CASCADE" json:"player,omitempty"`
}

func (_ *PlayerAchievementClaim) GetGene() xSnowflake.Gene {
	return bConst.GeneAchievementClaim
}
