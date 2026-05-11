package apiTitle

import "time"

type TitleResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Color           string    `json:"color"`
	Type            int16     `json:"type"`
	PermissionGroup *string   `json:"permission_group,omitempty"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

type PlayerTitleResponse struct {
	TitleResponse
	Source     int16     `json:"source"`
	IsEquipped bool      `json:"is_equipped"`
	GrantedAt  time.Time `json:"granted_at"`
}

type EquippedTitleResponse struct {
	TitleID     string `json:"title_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Type        int16  `json:"type"`
}

// TitleListResponse 称号分页列表响应
type TitleListResponse struct {
	List     []TitleResponse `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}
