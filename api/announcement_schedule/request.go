package apiAnnouncementSchedule

type CreateScheduleRequest struct {
	Name            string              `json:"name" binding:"required"`
	Mode            int16               `json:"mode" binding:"required,oneof=1 2"`
	IntervalSeconds int                 `json:"interval_seconds"`
	Items           []ScheduleItemInput `json:"items" binding:"required,min=1"`
}

type ScheduleItemInput struct {
	AnnouncementID string `json:"announcement_id" binding:"required"`
	SortOrder      int    `json:"sort_order"`
	DelaySeconds   int    `json:"delay_seconds"`
}

type UpdateScheduleRequest struct {
	Name            string              `json:"name" binding:"required"`
	Mode            int16               `json:"mode" binding:"required,oneof=1 2"`
	IntervalSeconds int                 `json:"interval_seconds"`
	Items           []ScheduleItemInput `json:"items" binding:"required,min=1"`
}
