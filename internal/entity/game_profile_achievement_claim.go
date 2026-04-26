package entity

import (
	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type GameProfileAchievementClaim struct {
	xModels.BaseEntity
	GameProfileUUID uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_gpac_gameprofile_ach;comment:玩家UUID" json:"game_profile_uuid"`
	AchievementID   xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_gpac_gameprofile_ach;comment:成就ID" json:"achievement_id"`
	TitleClaimed    bool                   `gorm:"not null;default:false;comment:称号奖励是否已发放" json:"title_claimed"`

	Achievement *Achievement  `gorm:"foreignKey:AchievementID;references:ID;constraint:OnDelete:CASCADE" json:"achievement,omitempty"`
	GameProfile *GameProfile `gorm:"foreignKey:GameProfileUUID;references:UUID;constraint:OnDelete:CASCADE" json:"game_profile,omitempty"`
}

func (_ *GameProfileAchievementClaim) GetGene() xSnowflake.Gene {
	return bConst.GeneAchievementClaim
}
