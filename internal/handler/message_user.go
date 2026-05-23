package handler

import (
	"context"
	"strconv"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	apiMessage "github.com/frontleaves-mc/frontleaves-plugin/api/message"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MessageUserHandler handler

func NewMessageUserHandler(ctx context.Context) *MessageUserHandler {
	return NewHandler[MessageUserHandler](ctx, "MessageUserHandler")
}

// ListMyChatHistory 查询当前用户的聊天记录
//
// @Summary     [用户] 查询我的聊天记录
// @Description 分页查询当前用户关联游戏角色的聊天记录
// @Tags        用户-消息接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiMessage.ChatHistoryListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /user/messages/chat [GET]
func (h *MessageUserHandler) ListMyChatHistory(ctx *gin.Context) {
	h.log.Info(ctx, "ListMyChatHistory - 查询我的聊天记录")

	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未获取到用户信息", false, nil))
		return
	}

	playerUUIDs, xErr := h.resolveUserPlayerUUIDs(ctx, userInfo.UserID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	chats, total, xErr := h.service.playerChatLogic.ListMyChatHistory(ctx.Request.Context(), page, pageSize, playerUUIDs)
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

// ListMyCommandHistory 查询当前用户的指令记录
//
// @Summary     [用户] 查询我的指令记录
// @Description 分页查询当前用户关联游戏角色的指令使用记录
// @Tags        用户-消息接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiMessage.CommandHistoryListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /user/messages/commands [GET]
func (h *MessageUserHandler) ListMyCommandHistory(ctx *gin.Context) {
	h.log.Info(ctx, "ListMyCommandHistory - 查询我的指令记录")

	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未获取到用户信息", false, nil))
		return
	}

	playerUUIDs, xErr := h.resolveUserPlayerUUIDs(ctx, userInfo.UserID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	commands, total, xErr := h.service.playerCommandLogic.ListMyCommandHistory(ctx.Request.Context(), page, pageSize, playerUUIDs)
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

// resolveUserPlayerUUIDs 通过用户 ID 查询关联的游戏角色 UUID 列表
func (h *MessageUserHandler) resolveUserPlayerUUIDs(ctx *gin.Context, userIDStr string) ([]uuid.UUID, *xError.Error) {
	userID, err := xSnowflake.ParseSnowflakeID(userIDStr)
	if err != nil {
		return nil, xError.NewError(nil, xError.ParameterError, "无效的用户 ID", true, err)
	}

	profiles, xErr := h.service.gameProfileLogic.ListByUserID(ctx.Request.Context(), userID)
	if xErr != nil {
		return nil, xErr
	}

	if len(profiles) == 0 {
		return []uuid.UUID{}, nil
	}

	uuids := make([]uuid.UUID, 0, len(profiles))
	for _, p := range profiles {
		parsed, err := uuid.Parse(p.UUID)
		if err != nil {
			continue
		}
		uuids = append(uuids, parsed)
	}
	return uuids, nil
}
