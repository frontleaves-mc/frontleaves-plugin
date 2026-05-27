package direct_message

import "time"

// DirectMessageResponse 私信详情响应
type DirectMessageResponse struct {
	ID           int64      `json:"id"`
	SenderID     string     `json:"sender_id"`
	SenderName   string     `json:"sender_name"`
	ReceiverID   string     `json:"receiver_id"`
	ReceiverName string     `json:"receiver_name"`
	Message      string     `json:"message"`
	Source       uint8      `json:"source"`       // 1=Game, 2=Web
	IsRead       bool       `json:"is_read"`
	ReadAt       *time.Time `json:"read_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// UnreadCountResponse 未读消息统计响应
type UnreadCountResponse struct {
	Total  int                `json:"total"`
	Detail []UnreadCountByUser `json:"detail"`
}

// UnreadCountByUser 按用户统计未读消息
type UnreadCountByUser struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Count    int64  `json:"count"`
}

// ConversationResponse 会话列表响应
type ConversationResponse struct {
	UserID        string     `json:"user_id"`
	UserName      string     `json:"user_name"`
	LastMessage   string     `json:"last_message"`
	LastMessageAt time.Time  `json:"last_message_at"`
	UnreadCount   int64      `json:"unread_count"`
}

// DirectMessageListResponse 私信分页列表响应
type DirectMessageListResponse struct {
	List     []DirectMessageResponse `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

// ConversationListResponse 会话分页列表响应
type ConversationListResponse struct {
	List     []ConversationResponse `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

// SSEDirectMessage SSE 私信推送
type SSEDirectMessage struct {
	SenderID     string `json:"sender_id"`
	SenderName   string `json:"sender_name"`
	ReceiverID   string `json:"receiver_id"`
	ReceiverName string `json:"receiver_name"`
	Message      string `json:"message"`
	Source       uint8  `json:"source"`
	IsRead       bool   `json:"is_read"`
}