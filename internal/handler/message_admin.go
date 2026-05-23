package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiMessage "github.com/frontleaves-mc/frontleaves-plugin/api/message"
)

type MessageAdminHandler handler

func NewMessageAdminHandler(ctx context.Context) *MessageAdminHandler {
	return NewHandler[MessageAdminHandler](ctx, "MessageAdminHandler")
}

// ListAllChatHistory 查询所有用户的聊天记录
//
// @Summary     [管理] 查询所有聊天记录
// @Description 分页查询所有用户的聊天记录，支持按玩家UUID、服务器、来源筛选
// @Tags        管理-消息接口
// @Accept      json
// @Produce     json
// @Param       page        query  int     false  "页码"
// @Param       page_size   query  int     false  "每页数量"
// @Param       player_uuid query  string  false  "玩家UUID筛选"
// @Param       server_name query  string  false  "服务器名称筛选"
// @Param       source      query  int     false  "消息来源筛选(1=Game,2=Web)"
// @Success     200  {object}  xBase.BaseResponse{data=apiMessage.ChatHistoryListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/messages/chat [GET]
func (h *MessageAdminHandler) ListAllChatHistory(ctx *gin.Context) {
	h.log.Info(ctx, "ListAllChatHistory - 查询所有聊天记录")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var playerUUID *uuid.UUID
	if puid := ctx.Query("player_uuid"); puid != "" {
		parsed, err := uuid.Parse(puid)
		if err != nil {
			_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
			return
		}
		playerUUID = &parsed
	}

	var serverName *string
	if sn := ctx.Query("server_name"); sn != "" {
		serverName = &sn
	}

	var source *uint8
	if s := ctx.Query("source"); s != "" {
		v, _ := strconv.Atoi(s)
		sv := uint8(v)
		source = &sv
	}

	chats, total, xErr := h.service.playerChatLogic.ListChatHistory(ctx.Request.Context(), page, pageSize, playerUUID, serverName, source)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiMessage.ChatHistoryListResponse{
		List:     chats,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListAllCommandHistory 查询所有玩家的指令使用日志
//
// @Summary     [管理] 查询所有指令记录
// @Description 分页查询所有玩家的指令使用日志，支持按玩家UUID、服务器筛选
// @Tags        管理-消息接口
// @Accept      json
// @Produce     json
// @Param       page        query  int     false  "页码"
// @Param       page_size   query  int     false  "每页数量"
// @Param       player_uuid query  string  false  "玩家UUID筛选"
// @Param       server_name query  string  false  "服务器名称筛选"
// @Success     200  {object}  xBase.BaseResponse{data=apiMessage.CommandHistoryListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/messages/commands [GET]
func (h *MessageAdminHandler) ListAllCommandHistory(ctx *gin.Context) {
	h.log.Info(ctx, "ListAllCommandHistory - 查询所有指令记录")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var playerUUID *uuid.UUID
	if puid := ctx.Query("player_uuid"); puid != "" {
		parsed, err := uuid.Parse(puid)
		if err != nil {
			_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
			return
		}
		playerUUID = &parsed
	}

	var serverName *string
	if sn := ctx.Query("server_name"); sn != "" {
		serverName = &sn
	}

	commands, total, xErr := h.service.playerCommandLogic.ListCommandHistory(ctx.Request.Context(), page, pageSize, playerUUID, serverName)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiMessage.CommandHistoryListResponse{
		List:     commands,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
