package entity

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

// PluginCredential 插件授权凭证
//
// 用于 gRPC 插件认证，每个插件持有唯一的 name + secret_key 组合。
type PluginCredential struct {
	xModels.BaseEntity
	Name      string `gorm:"not null;type:varchar(64);uniqueIndex:uk_plugin_name;comment:插件名称"`
	SecretKey string `gorm:"not null;type:varchar(128);comment:插件密钥"`
	IsActive  bool   `gorm:"not null;default:true;comment:是否启用"`
}

func (_ *PluginCredential) GetGene() xSnowflake.Gene {
	return bConst.GenePluginCredential
}
