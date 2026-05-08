package entity

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

// Server 服务器
//
// 用于 Minecraft 服务器注册与管理，支持被动创建（gRPC 心跳自动注册）和管理员 CRUD。
type Server struct {
	xModels.BaseEntity
	Name        string `gorm:"not null;type:varchar(64);uniqueIndex:uk_server_name;comment:服务器标识名"`
	DisplayName string `gorm:"not null;type:varchar(64);comment:显示名称/别名"`
	Description string `gorm:"type:varchar(256);comment:服务器描述"`
	IsPublic    bool   `gorm:"not null;default:false;comment:是否对外公开"`
	IsEnabled   bool   `gorm:"not null;default:true;comment:是否启用"`
	Address     string `gorm:"type:varchar(128);comment:服务器地址"`
	SortOrder   int    `gorm:"not null;default:0;comment:排序权重"`
}

func (_ *Server) GetGene() xSnowflake.Gene {
	return bConst.GeneServer
}
