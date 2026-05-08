package apiServer

import "time"

// ServerResponse 服务器响应
type ServerResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	IsEnabled   bool      `json:"is_enabled"`
	Address     string    `json:"address"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ServerListResponse 服务器分页列表响应
type ServerListResponse struct {
	List     []ServerResponse `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// OnlineGameProfileResponse 在线游戏账号响应
type OnlineGameProfileResponse struct {
	UUID       string `json:"uuid"`
	Username   string `json:"username"`
	ServerName string `json:"server_name"`
	WorldName  string `json:"world_name"`
}

// PlayerOnlineResponse 玩家在线状态响应
type PlayerOnlineResponse struct {
	Online     bool   `json:"online"`
	PlayerUUID string `json:"player_uuid"`
	PlayerName string `json:"player_name"`
	ServerName string `json:"server_name"`
	WorldName  string `json:"world_name"`
	LastSeen   int64  `json:"last_seen"`
}
