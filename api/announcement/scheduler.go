package apiAnnouncement

type GetSchedulerConfigResponse struct {
	Mode            int16 `json:"mode"`
	IntervalSeconds int   `json:"interval_seconds"`
	IsEnabled       bool  `json:"is_enabled"`
}

type UpdateSchedulerConfigRequest struct {
	Mode            int16 `json:"mode" binding:"required,oneof=1 2"`
	IntervalSeconds int   `json:"interval_seconds" binding:"required,min=1"`
}
