package apiMatrixWarning

import (
	"encoding/json"
	"time"
)

// WarningListItemResponse 警告列表项响应
type WarningListItemResponse struct {
	ID          string    `json:"id"`
	PlayerUUID  string    `json:"player_uuid"`
	PlayerName  string    `json:"player_name"`
	ServerName  string    `json:"server_name"`
	WarningType string    `json:"warning_type"`
	Description string    `json:"description"`
	RiskScore   int32     `json:"risk_score"`
	CreatedAt   time.Time `json:"created_at"`
}

// WarningDetailResponse 警告详情响应
type WarningDetailResponse struct {
	ID          string          `json:"id"`
	PlayerUUID  string          `json:"player_uuid"`
	PlayerName  string          `json:"player_name"`
	ServerName  string          `json:"server_name"`
	WarningType string          `json:"warning_type"`
	Description string          `json:"description"`
	RiskScore   int32           `json:"risk_score"`
	ContextData json.RawMessage `json:"context_data"`
	CreatedAt   time.Time       `json:"created_at"`
}

// WarningListResponse 警告分页列表响应
type WarningListResponse struct {
	List     []WarningListItemResponse `json:"list"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"page_size"`
}