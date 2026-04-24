package entity

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
)

type User struct {
	xModels.BaseEntity
	Username string  `gorm:"not null;type:varchar(64);uniqueIndex:uk_user_username;comment:用户名" json:"username"`
	Email    *string `gorm:"type:varchar(128);uniqueIndex:uk_user_email;comment:邮箱" json:"email,omitempty"`
}

func (_ *User) GetGene() xSnowflake.Gene {
	return xSnowflake.GeneUser
}
