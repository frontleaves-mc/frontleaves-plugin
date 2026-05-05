package apiAnnouncementSchedule

import "time"

type ScheduleResponse struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Mode            int16                 `json:"mode"`
	IntervalSeconds int                   `json:"interval_seconds"`
	IsActive        bool                  `json:"is_active"`
	Status          int16                 `json:"status"`
	Items           []ScheduleItemResponse `json:"items"`
	CreatedAt       time.Time             `json:"created_at"`
}

type ScheduleItemResponse struct {
	AnnouncementID    string `json:"announcement_id"`
	AnnouncementTitle string `json:"announcement_title"`
	SortOrder         int    `json:"sort_order"`
	DelaySeconds      int    `json:"delay_seconds"`
}
