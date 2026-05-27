package handler

import (
	"context"
	"strconv"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiDirectMessage "github.com/frontleaves-mc/frontleaves-plugin/api/direct_message"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
)

type DirectMessageHandler handler

func NewDirectMessageHandler(ctx context.Context) *DirectMessageHandler {
	return NewHandler[DirectMessageHandler](ctx, "DirectMessageHandler")
}

func getAuthUser(ctx *gin.Context) (*repository.AuthUserInfo, bool) {
	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未获取到用户信息", false, nil))
		return nil, false
	}
	return userInfo, true
}

// SendDirectMessage 发送私信
//
// @Summary     [用户] 发送私信
// @Description Web 端用户向指定用户发送私信
// @Tags        用户-私信接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiDirectMessage.SendDirectMessageRequest  true  "发送私信请求"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /user/messages/dm [POST]
// @Security    BearerAuth
func (h *DirectMessageHandler) SendDirectMessage(ctx *gin.Context) {
	h.log.Info(ctx, "SendDirectMessage - 发送私信")

	userInfo, ok := getAuthUser(ctx)
	if !ok {
		return
	}

	var req apiDirectMessage.SendDirectMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	xErr := h.service.directMessageLogic.RecordDirectMessageFromWeb(
		ctx.Request.Context(), userInfo.UserID, req.ReceiverID, req.Message,
	)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "发送成功", nil)
}

// ListMyDirectMessages 查询与指定用户的私信对话
//
// @Summary     [用户] 查询私信对话
// @Description 分页查询当前用户与指定用户之间的私信对话记录
// @Tags        用户-私信接口
// @Produce     json
// @Param       target_user  query  string  true   "目标用户 ID"
// @Param       page         query  int     false  "页码"
// @Param       page_size    query  int     false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiDirectMessage.DirectMessageListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /user/messages/dm [GET]
// @Security    BearerAuth
func (h *DirectMessageHandler) ListMyDirectMessages(ctx *gin.Context) {
	h.log.Info(ctx, "ListMyDirectMessages - 查询私信对话")

	userInfo, ok := getAuthUser(ctx)
	if !ok {
		return
	}

	var req apiDirectMessage.ListDirectMessageRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	if req.TargetUser == "" {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "目标用户 ID 不能为空", true, nil))
		return
	}

	page, pageSize := normalizePagination(req.Page, req.PageSize)

	messages, total, xErr := h.service.directMessageLogic.ListDirectMessages(
		ctx.Request.Context(), userInfo.UserID, req.TargetUser, page, pageSize,
	)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiDirectMessage.DirectMessageListResponse{
		List:     messages,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetUnreadCount 获取未读消息统计
//
// @Summary     [用户] 获取未读消息统计
// @Description 查询当前用户的未读私信数量，按发送者分组统计
// @Tags        用户-私信接口
// @Produce     json
// @Success     200  {object}  xBase.BaseResponse{data=apiDirectMessage.UnreadCountResponse}  "成功"
// @Router      /user/messages/dm/unread [GET]
// @Security    BearerAuth
func (h *DirectMessageHandler) GetUnreadCount(ctx *gin.Context) {
	h.log.Info(ctx, "GetUnreadCount - 查询未读消息统计")

	userInfo, ok := getAuthUser(ctx)
	if !ok {
		return
	}

	result, xErr := h.service.directMessageLogic.GetUnreadCount(ctx.Request.Context(), userInfo.UserID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", result)
}

// MarkAsRead 标记私信已读
//
// @Summary     [用户] 标记私信已读
// @Description 将指定发送者的未读私信标记为已读
// @Tags        用户-私信接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiDirectMessage.MarkAsReadRequest  true  "标记已读请求"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /user/messages/dm/read [PUT]
// @Security    BearerAuth
func (h *DirectMessageHandler) MarkAsRead(ctx *gin.Context) {
	h.log.Info(ctx, "MarkAsRead - 标记私信已读")

	userInfo, ok := getAuthUser(ctx)
	if !ok {
		return
	}

	var req apiDirectMessage.MarkAsReadRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	xErr := h.service.directMessageLogic.MarkAsRead(ctx.Request.Context(), userInfo.UserID, req.SenderID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "标记成功", nil)
}

// ListConversations 查询会话列表
//
// @Summary     [用户] 查询会话列表
// @Description 分页查询当前用户的所有私信会话，按最近消息时间排序
// @Tags        用户-私信接口
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiDirectMessage.ConversationListResponse}  "成功"
// @Router      /user/messages/dm/conversations [GET]
// @Security    BearerAuth
func (h *DirectMessageHandler) ListConversations(ctx *gin.Context) {
	h.log.Info(ctx, "ListConversations - 查询会话列表")

	userInfo, ok := getAuthUser(ctx)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	page, pageSize = normalizePagination(page, pageSize)

	conversations, total, xErr := h.service.directMessageLogic.ListConversations(
		ctx.Request.Context(), userInfo.UserID, page, pageSize,
	)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiDirectMessage.ConversationListResponse{
		List:     conversations,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListAllDirectMessages 管理端查询所有私信
//
// @Summary     [管理] 查询所有私信
// @Description 分页查询所有用户的私信记录，支持按发送者/接收者筛选
// @Tags        管理-私信接口
// @Produce     json
// @Param       page           query  int     false  "页码"
// @Param       page_size      query  int     false  "每页数量"
// @Param       sender_name    query  string  false  "发送者名称筛选"
// @Param       receiver_name  query  string  false  "接收者名称筛选"
// @Success     200  {object}  xBase.BaseResponse{data=apiDirectMessage.DirectMessageListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/messages/dm [GET]
// @Security    BearerAuth
func (h *DirectMessageHandler) ListAllDirectMessages(ctx *gin.Context) {
	h.log.Info(ctx, "ListAllDirectMessages - 管理端查询私信")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	page, pageSize = normalizePagination(page, pageSize)

	senderName := ctx.Query("sender_name")
	receiverName := ctx.Query("receiver_name")

	messages, total, xErr := h.service.directMessageLogic.ListAllForAdmin(
		ctx.Request.Context(), page, pageSize, senderName, receiverName,
	)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiDirectMessage.DirectMessageListResponse{
		List:     messages,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
