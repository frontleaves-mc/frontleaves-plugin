package apiMessage

// ChatLogResponse 聊天记录响应
type ChatLogResponse struct {
	ID         string  `json:"id"`
	PlayerUUID string  `json:"player_uuid"`
	PlayerName string  `json:"player_name"`
	ServerName string  `json:"server_name"`
	WorldName  string  `json:"world_name"`
	Message    string  `json:"message"`
	Source     int     `json:"source"`
	SenderID   *string `json:"sender_id,omitempty"`
}

// CommandLogResponse 指令日志响应
type CommandLogResponse struct {
	ID         string `json:"id"`
	PlayerUUID string `json:"player_uuid"`
	PlayerName string `json:"player_name"`
	ServerName string `json:"server_name"`
	WorldName  string `json:"world_name"`
	Command    string `json:"command"`
}

// ChatHistoryListResponse 聊天记录分页列表响应
type ChatHistoryListResponse struct {
	List     []ChatLogResponse `json:"list"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// CommandHistoryListResponse 指令日志分页列表响应
type CommandHistoryListResponse struct {
	List     []CommandLogResponse `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// SSEChatMessage SSE 聊天消息推送
type SSEChatMessage struct {
	PlayerName string `json:"player_name"`
	ServerName string `json:"server_name"`
	Message    string `json:"message"`
	Source     int    `json:"source"`
}
