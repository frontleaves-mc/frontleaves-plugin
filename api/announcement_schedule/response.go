package apiAnnouncementSchedule

import "time"

type ScheduleResponse struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Mode            int16                  `json:"mode"`
	IntervalSeconds int                    `json:"interval_seconds"`
	IsActive        bool                   `json:"is_active"`
	Status          int16                  `json:"status"`
	Items           []ScheduleItemResponse `json:"items"`
	CreatedAt       time.Time              `json:"created_at"`
}

type ScheduleItemResponse struct {
	AnnouncementID    string `json:"announcement_id"`
	AnnouncementTitle string `json:"announcement_title"`
	SortOrder         int    `json:"sort_order"`
	DelaySeconds      int    `json:"delay_seconds"`
}

// ScheduleListResponse 公告调度分页列表响应
type ScheduleListResponse struct {
	List     []ScheduleResponse `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}
