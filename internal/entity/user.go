package entity

import (
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type User struct {
	xModels.BaseEntity
	Username string `gorm:"not null;type:varchar(64);comment:用户名" json:"username"`

	GameProfiles []GameProfile `gorm:"foreignKey:UserID" json:"-"`
}

func (_ *User) GetGene() xSnowflake.Gene {
	return bConst.GeneUser
}
