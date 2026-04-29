package apiPC

import "time"

type PluginCredentialResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SecretKey   string    `json:"secret_key"`
	CreatedAt   time.Time `json:"created_at"`
}

// PluginCredentialListResponse 插件凭证分页列表响应
type PluginCredentialListResponse struct {
	List     []PluginCredentialResponse `json:"list"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}
