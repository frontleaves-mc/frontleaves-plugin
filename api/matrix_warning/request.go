package apiMatrixWarning

import "time"

// ListWarningsRequest 列表查询请求
type ListWarningsRequest struct {
	Page          int         `form:"page" json:"page"`
	PageSize      int         `form:"page_size" json:"page_size"`
	PlayerUUID    string      `form:"player_uuid" json:"player_uuid"`
	WarningType   string      `form:"warning_type" json:"warning_type"`
	RiskScoreMin  *int32      `form:"risk_score_min" json:"risk_score_min"`
	RiskScoreMax  *int32      `form:"risk_score_max" json:"risk_score_max"`
	ServerName    string      `form:"server_name" json:"server_name"`
	StartTime     *time.Time  `form:"start_time" json:"start_time"`
	EndTime       *time.Time  `form:"end_time" json:"end_time"`
}