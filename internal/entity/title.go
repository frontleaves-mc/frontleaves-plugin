package entity

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type TitleType int16

const (
	TitleTypeGeneral   TitleType = 1 // 通用称号
	TitleTypeGroup     TitleType = 2 // 权限组称号
	TitleTypeExclusive TitleType = 3 // 玩家专属称号
)

func (t TitleType) String() string {
	switch t {
	case TitleTypeGeneral:
		return "通用称号"
	case TitleTypeGroup:
		return "权限组称号"
	case TitleTypeExclusive:
		return "玩家专属称号"
	default:
		return "未知类型"
	}
}

type Title struct {
	xModels.BaseEntity
	Name            string  `gorm:"not null;type:varchar(64);uniqueIndex:uk_title_name;comment:称号名称" json:"name"`
	Description     string  `gorm:"not null;type:varchar(255);comment:称号描述" json:"description"`
	Type            TitleType `gorm:"not null;type:smallint;index:idx_title_type;comment:称号类型" json:"type"`
	PermissionGroup *string `gorm:"type:varchar(64);index:idx_title_perm_group;comment:关联权限组" json:"permission_group,omitempty"`
	IsActive        bool    `gorm:"not null;default:true;comment:是否启用" json:"is_active"`
}

func (_ *Title) GetGene() xSnowflake.Gene {
	return bConst.GeneTitle
}
