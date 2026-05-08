package apiServer

// CreateServerRequest 创建服务器请求
type CreateServerRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description,omitempty"`
	Address     string `json:"address,omitempty"`
	SortOrder   int    `json:"sort_order,omitempty"`
}

// UpdateServerRequest 更新服务器请求
type UpdateServerRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description,omitempty"`
	Address     string `json:"address,omitempty"`
	SortOrder   int    `json:"sort_order,omitempty"`
}

// SetServerPublicRequest 设置服务器公开状态请求
type SetServerPublicRequest struct {
	IsPublic bool `json:"is_public"`
}

// SetServerEnabledRequest 设置服务器启用状态请求
type SetServerEnabledRequest struct {
	IsEnabled bool `json:"is_enabled"`
}
