package apiMessage

// SendChatMessageRequest 发送聊天消息请求
type SendChatMessageRequest struct {
	Message string `json:"message" binding:"required,max=500"`
}
