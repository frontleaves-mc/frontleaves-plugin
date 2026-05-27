package direct_message

// SendDirectMessageRequest 发送私信请求
type SendDirectMessageRequest struct {
	ReceiverID string `json:"receiver_id" binding:"required"`
	Message    string `json:"message" binding:"required,max=500"`
}

// ListDirectMessageRequest 查询私信列表请求
type ListDirectMessageRequest struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	TargetUser string `form:"target_user" binding:"omitempty"`
}

// MarkAsReadRequest 标记私信已读请求
type MarkAsReadRequest struct {
	SenderID string `json:"sender_id" binding:"required"`
}

// AdminListDirectMessageRequest 管理端查询私信列表请求
type AdminListDirectMessageRequest struct {
	Page         int    `form:"page" binding:"omitempty,min=1"`
	PageSize     int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SenderName   string `form:"sender_name" binding:"omitempty"`
	ReceiverName string `form:"receiver_name" binding:"omitempty"`
}