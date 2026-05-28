package entity

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type ScheduleMode int16

const (
	ScheduleModeFixedInterval ScheduleMode = 1 // 固定间隔
	ScheduleModeSequential    ScheduleMode = 2 // 顺序播放
)

func (m ScheduleMode) String() string {
	switch m {
	case ScheduleModeFixedInterval:
		return "固定间隔"
	case ScheduleModeSequential:
		return "顺序播放"
	default:
		return "未知模式"
	}
}

// Config 通用键值对配置实体，支持按命名空间隔离不同模块的配置
type Config struct {
	xModels.BaseEntity
	Namespace string `gorm:"not null;type:varchar(64);index:idx_config_namespace_key,unique,priority:1;comment:配置命名空间" json:"namespace"` // 配置命名空间
	Key       string `gorm:"not null;type:varchar(128);index:idx_config_namespace_key,unique,priority:2;comment:配置键" json:"key"`       // 配置键
	Value     string `gorm:"not null;type:text;comment:配置值" json:"value"`                                                           // 配置值
}

// GetGene 返回 xSnowflake.Gene，用于标识该实体在 ID 生成时使用的基因类型。
func (_ *Config) GetGene() xSnowflake.Gene {
	return bConst.GeneConfig
}
