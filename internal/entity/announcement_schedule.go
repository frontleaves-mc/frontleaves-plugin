package entity

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type ScheduleMode int16

const (
	ScheduleModeFixedInterval ScheduleMode = 1 // 固定间隔
	ScheduleModeSequential   ScheduleMode = 2 // 顺序播放
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

type AnnouncementSchedule struct {
	xModels.BaseEntity
	Name             string             `gorm:"not null;type:varchar(128);comment:调度名称" json:"name"` // 调度名称
	Mode             ScheduleMode       `gorm:"not null;type:smallint;comment:调度模式" json:"mode"` // 调度模式
	IntervalSeconds  int                `gorm:"not null;comment:固定间隔秒数" json:"interval_seconds"` // 固定间隔秒数
	IsActive         bool               `gorm:"not null;default:false;comment:是否为活动调度" json:"is_active"` // 是否为活动调度
	Status           AnnouncementStatus `gorm:"not null;type:smallint;comment:调度状态" json:"status"` // 调度状态
}

func (_ *AnnouncementSchedule) GetGene() xSnowflake.Gene {
	return bConst.GeneAnnouncementSchedule
}
