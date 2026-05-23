package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/sse"
	"github.com/gin-gonic/gin"
)

type ChatSSEHandler handler

func NewChatSSEHandler(ctx context.Context) *ChatSSEHandler {
	return NewHandler[ChatSSEHandler](ctx, "ChatSSEHandler")
}

// StreamChat SSE 实时聊天消息流
//
// @Summary     [用户] 实时聊天消息流
// @Description 通过 SSE 连接接收实时聊天消息，首次连接推送最近 50 条消息
// @Tags        用户-消息接口
// @Produce     text/event-stream
// @Success     200  {string}  string  "SSE 事件流"
// @Router      /user/messages/chat/stream [GET]
func (h *ChatSSEHandler) StreamChat(ctx *gin.Context) {
	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未获取到用户信息", false, nil))
		return
	}

	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	// 推送初始数据（最近 50 条消息）
	recentChats, xErr := h.service.playerChatLogic.ListRecentChats(ctx.Request.Context(), 50)
	if xErr != nil {
		h.log.Warn(ctx, "StreamChat - 获取最近消息失败: "+xErr.Error())
	}

	client := &sse.ChatSSEClient{
		Writer:  ctx.Writer,
		Flusher: ctx.Writer.(http.Flusher),
		UserID:  userInfo.UserID,
	}

	// 发送初始消息
	if len(recentChats) > 0 {
		if err := sse.SendInitMessages(client, recentChats); err != nil {
			h.log.Warn(ctx, "StreamChat - 发送初始消息失败: "+err.Error())
			return
		}
	}

	// 使用 userID + 时间戳作为 clientID，支持同一用户多 tab 连接
	clientID := fmt.Sprintf("%s-%d", userInfo.UserID, time.Now().UnixMilli())
	sse.RegisterSSEClient(clientID, client)
	defer sse.RemoveSSEClient(clientID)

	// 监听客户端断开连接
	ctx.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Request.Context().Done():
			return false
		}
	})
}
