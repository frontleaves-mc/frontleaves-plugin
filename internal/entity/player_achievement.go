package entity

import (
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type AchievementStatus int16

const (
	AchievementStatusInProgress AchievementStatus = 0 // 进行中
	AchievementStatusCompleted  AchievementStatus = 1 // 已完成
	AchievementStatusClaimed    AchievementStatus = 2 // 已领奖
)

func (s AchievementStatus) String() string {
	switch s {
	case AchievementStatusInProgress:
		return "进行中"
	case AchievementStatusCompleted:
		return "已完成"
	case AchievementStatusClaimed:
		return "已领奖"
	default:
		return "未知状态"
	}
}

type PlayerAchievement struct {
	xModels.BaseEntity
	PlayerUUID    uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_pa_player_ach;comment:玩家UUID" json:"player_uuid"`
	AchievementID xSnowflake.SnowflakeID `gorm:"not null;uniqueIndex:uk_pa_player_ach;index:idx_pa_ach;comment:成就ID" json:"achievement_id"`
	Status        AchievementStatus      `gorm:"not null;type:smallint;default:0;comment:状态" json:"status"`
	Progress      int64                  `gorm:"not null;default:0;comment:当前进度" json:"progress"`
	CompletedAt   *time.Time             `gorm:"type:timestamptz;comment:完成时间" json:"completed_at,omitempty"`

	Achievement *Achievement `gorm:"foreignKey:AchievementID;references:ID;constraint:OnDelete:CASCADE" json:"achievement,omitempty"`
	Player      *Player      `gorm:"foreignKey:PlayerUUID;references:UUID;constraint:OnDelete:CASCADE" json:"player,omitempty"`
}

func (_ *PlayerAchievement) GetGene() xSnowflake.Gene {
	return bConst.GenePlayerAchievement
}
