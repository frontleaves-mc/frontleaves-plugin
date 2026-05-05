package entity

import (
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type AnnouncementScheduleItem struct {
	xModels.BaseEntity
	ScheduleID     xSnowflake.SnowflakeID `gorm:"not null;index:idx_announcement_schedule_item_schedule_id;comment:调度ID" json:"schedule_id"` // 调度ID
	AnnouncementID xSnowflake.SnowflakeID `gorm:"not null;index:idx_announcement_schedule_item_announcement_id;comment:公告ID" json:"announcement_id"` // 公告ID
	SortOrder      int                    `gorm:"not null;comment:排序权重" json:"sort_order"` // 排序权重
	DelaySeconds   int                    `gorm:"not null;comment:Sequential模式延迟秒数" json:"delay_seconds"` // Sequential模式延迟秒数
}

func (_ *AnnouncementScheduleItem) GetGene() xSnowflake.Gene {
	return bConst.GeneAnnouncementScheduleItem
}
