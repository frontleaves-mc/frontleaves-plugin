package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiPlayer "github.com/frontleaves-mc/frontleaves-plugin/api/player"
)

type PlayerHandler handler

func NewPlayerHandler(ctx context.Context) *PlayerHandler {
	return NewHandler[PlayerHandler](ctx, "PlayerHandler")
}

// GetPlayer 查询玩家信息
//
// @Summary     [玩家] 查询玩家信息
// @Description 根据玩家 UUID 查询玩家详细信息
// @Tags        玩家接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "玩家UUID"
// @Success     200  {object}  xBase.BaseResponse{data=apiPlayer.PlayerResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse                                "玩家不存在"
// @Router      /players/:uuid [GET]
func (h *PlayerHandler) GetPlayer(ctx *gin.Context) {
	h.log.Info(ctx, "GetPlayer - 查询玩家信息")

	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	player, xErr := h.service.playerLogic.GetPlayer(ctx, playerUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", player)
}

// ListPlayers 查询玩家列表
//
// @Summary     [玩家] 查询玩家列表
// @Description 分页查询玩家列表
// @Tags        玩家接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /players [GET]
func (h *PlayerHandler) ListPlayers(ctx *gin.Context) {
	h.log.Info(ctx, "ListPlayers - 查询玩家列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	players, total, xErr := h.service.playerLogic.ListPlayers(ctx, page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", gin.H{
		"list":      players,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdatePlayerGroup 更新玩家权限组
//
// @Summary     [超管] 更新玩家权限组
// @Description 更新指定玩家的权限组
// @Tags        玩家接口
// @Accept      json
// @Produce     json
// @Param       uuid      path  string                              true  "玩家UUID"
// @Param       request   body  apiPlayer.UpdatePlayerGroupRequest  true  "更新权限组请求"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /internal/players/:uuid/group [PUT]
func (h *PlayerHandler) UpdatePlayerGroup(ctx *gin.Context) {
	h.log.Info(ctx, "UpdatePlayerGroup - 更新玩家权限组")

	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	var req apiPlayer.UpdatePlayerGroupRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	if xErr := h.service.playerLogic.UpdatePlayerGroup(ctx, playerUUID, "", req.GroupName); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", nil)
}
