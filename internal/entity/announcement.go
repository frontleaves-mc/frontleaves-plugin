package entity

import (
	"time"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

type AnnouncementType int16

const (
	AnnouncementTypeSite   AnnouncementType = 1 // 站内公告
	AnnouncementTypeGlobal AnnouncementType = 2 // 全局公告（Minecraft 内显示）
)

func (t AnnouncementType) String() string {
	switch t {
	case AnnouncementTypeSite:
		return "站内公告"
	case AnnouncementTypeGlobal:
		return "全局公告"
	default:
		return "未知类型"
	}
}

type AnnouncementStatus int16

const (
	AnnouncementStatusDraft     AnnouncementStatus = 1 // 草稿
	AnnouncementStatusPublished AnnouncementStatus = 2 // 已发布
	AnnouncementStatusOffline   AnnouncementStatus = 3 // 已下线
)

func (s AnnouncementStatus) String() string {
	switch s {
	case AnnouncementStatusDraft:
		return "草稿"
	case AnnouncementStatusPublished:
		return "已发布"
	case AnnouncementStatusOffline:
		return "已下线"
	default:
		return "未知状态"
	}
}

type Announcement struct {
	xModels.BaseEntity
	Title       string             `gorm:"not null;type:varchar(128);comment:公告标题" json:"title"` // 公告标题
	Content     string             `gorm:"not null;type:text;comment:公告内容(Markdown)" json:"content"` // 公告内容(Markdown)
	Type        AnnouncementType   `gorm:"not null;type:smallint;index:idx_announcement_type;comment:公告类型" json:"type"` // 公告类型
	Status      AnnouncementStatus `gorm:"not null;type:smallint;index:idx_announcement_status;comment:公告状态" json:"status"` // 公告状态
	PublishedAt *time.Time         `gorm:"comment:发布时间" json:"published_at,omitempty"` // 发布时间
}

func (_ *Announcement) GetGene() xSnowflake.Gene {
	return bConst.GeneAnnouncement
}
