package entity

import (
	"encoding/json"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type AchievementType int16

const (
	AchievementTypeStat    AchievementType = 1 // 统计类
	AchievementTypeEvent   AchievementType = 2 // 事件类
	AchievementTypeSpecial AchievementType = 3 // 特殊条件
	AchievementTypeManual  AchievementType = 4 // 管理员手动
)

func (t AchievementType) String() string {
	switch t {
	case AchievementTypeStat:
		return "统计类"
	case AchievementTypeEvent:
		return "事件类"
	case AchievementTypeSpecial:
		return "特殊条件"
	case AchievementTypeManual:
		return "管理员手动"
	default:
		return "未知类型"
	}
}

type Achievement struct {
	xModels.BaseEntity
	Name             string                  `gorm:"not null;type:varchar(64);comment:成就名称" json:"name"`
	Description      string                  `gorm:"not null;type:varchar(255);comment:成就描述" json:"description"`
	Type             AchievementType         `gorm:"not null;type:smallint;index:idx_ach_type;comment:成就类型" json:"type"`
	ConditionKey     string                  `gorm:"not null;type:varchar(64);uniqueIndex:uk_ach_condition_key;comment:条件标识" json:"condition_key"`
	ConditionParams  json.RawMessage         `gorm:"type:jsonb;comment:条件参数" json:"condition_params,omitempty"`
	RewardConfig     json.RawMessage         `gorm:"type:jsonb;comment:奖励配置" json:"reward_config,omitempty"`
	IsActive         bool                    `gorm:"not null;default:true;comment:是否启用" json:"is_active"`
	SortOrder        int                     `gorm:"not null;default:0;comment:展示排序" json:"sort_order"`
}

func (_ *Achievement) GetGene() xSnowflake.Gene {
	return bConst.GeneAchievement
}
