package handler

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiMessage "github.com/frontleaves-mc/frontleaves-plugin/api/message"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/sse"
	"github.com/gin-gonic/gin"
)

type ChatSendHandler handler

func NewChatSendHandler(ctx context.Context) *ChatSendHandler {
	return NewHandler[ChatSendHandler](ctx, "ChatSendHandler")
}

// SendChatMessage 发送聊天消息
//
// @Summary     [用户] 发送聊天消息
// @Description Web 端用户选择游戏角色发送聊天消息，转发到游戏内并广播给 SSE 客户端
// @Tags        用户-消息接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiMessage.SendChatMessageRequest  true  "发送消息请求"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /user/messages/chat [POST]
func (h *ChatSendHandler) SendChatMessage(ctx *gin.Context) {
	h.log.Info(ctx, "SendChatMessage - 发送聊天消息")

	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未获取到用户信息", false, nil))
		return
	}

	var req apiMessage.SendChatMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	senderID, err := xSnowflake.ParseSnowflakeID(userInfo.UserID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的用户 ID", true, err))
		return
	}

	// 校验用户对 GameProfile 的所有权并获取 MC 用户名
	profile, xErr := h.service.gameProfileLogic.ResolveProfileForUser(ctx.Request.Context(), senderID, req.ProfileUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	gameName := profile.Username
	playerUUID := profile.UUID

	// 记录到 DB
	_, recordErr := h.service.playerChatLogic.RecordWebChat(ctx.Request.Context(), senderID, gameName, playerUUID, req.Message)
	if recordErr != nil {
		_ = ctx.Error(xError.NewError(nil, xError.DatabaseError, "记录消息失败", false, recordErr))
		return
	}

	// 推送到 MC 插件
	if pushErr := logic.PushChatToGame(ctx.Request.Context(), gameName, req.Message); pushErr != nil {
		h.log.Warn(ctx, "SendChatMessage - 推送到MC插件失败: "+pushErr.Error())
	}

	// 广播到 SSE 客户端
	sse.BroadcastChatMessage(apiMessage.SSEChatMessage{
		PlayerName: gameName,
		PlayerUUID: playerUUID.String(),
		Message:    req.Message,
		Source:     2,
	})

	xResult.SuccessHasData(ctx, "发送成功", nil)
}
