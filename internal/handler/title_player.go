package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiTitle "github.com/frontleaves-mc/frontleaves-plugin/api/title"
)

type TitlePlayerHandler handler

func NewTitlePlayerHandler(ctx context.Context) *TitlePlayerHandler {
	return NewHandler[TitlePlayerHandler](ctx, "TitlePlayerHandler")
}

// GetPlayerTitles 获取玩家拥有的所有称号
//
// @Summary     [玩家] 获取玩家称号列表
// @Description 根据玩家 UUID 获取该玩家拥有的所有称号
// @Tags        玩家-称号接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "玩家UUID"
// @Success     200  {object}  xBase.BaseResponse{data=[]apiTitle.PlayerTitleResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "玩家不存在"
// @Router      /players/:uuid/titles [GET]
func (h *TitlePlayerHandler) GetPlayerTitles(ctx *gin.Context) {
	h.log.Info(ctx, "GetPlayerTitles - 获取玩家称号列表")

	playerUUID, xErr := h.parsePlayerUUID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	titles, xErr := h.service.titleLogic.GetPlayerTitles(ctx, playerUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", titles)
}

// EquipTitle 装备指定称号
//
// @Summary     [玩家] 装备称号
// @Description 玩家装备指定称号
// @Tags        玩家-称号接口
// @Accept      json
// @Produce     json
// @Param       uuid      path  string                      true  "玩家UUID"
// @Param       request   body  apiTitle.EquipTitleRequest  true  "装备请求"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "称号不存在"
// @Router      /players/:uuid/titles/equip [PUT]
func (h *TitlePlayerHandler) EquipTitle(ctx *gin.Context) {
	h.log.Info(ctx, "EquipTitle - 装备称号")

	playerUUID, xErr := h.parsePlayerUUID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiTitle.EquipTitleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	titleID, err := strconv.ParseInt(req.TitleID, 10, 64)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的称号 ID", true, err))
		return
	}

	if xErr := h.service.titleLogic.EquipTitle(ctx, playerUUID, xSnowflake.SnowflakeID(titleID)); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "装备成功", nil)
}

// UnequipTitle 卸下当前装备的称号
//
// @Summary     [玩家] 卸下称号
// @Description 玩家卸下当前装备的称号
// @Tags        玩家-称号接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "玩家UUID"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /players/:uuid/titles/equip [DELETE]
func (h *TitlePlayerHandler) UnequipTitle(ctx *gin.Context) {
	h.log.Info(ctx, "UnequipTitle - 卸下称号")

	playerUUID, xErr := h.parsePlayerUUID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.titleLogic.UnequipTitle(ctx, playerUUID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "卸下成功", nil)
}

// GetEquippedTitle 获取当前装备的称号
//
// @Summary     [玩家] 获取当前装备称号
// @Description 根据玩家 UUID 获取该玩家当前装备的称号
// @Tags        玩家-称号接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "玩家UUID"
// @Success     200  {object}  xBase.BaseResponse{data=apiTitle.EquippedTitleResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "未装备称号"
// @Router      /players/:uuid/titles/equipped [GET]
func (h *TitlePlayerHandler) GetEquippedTitle(ctx *gin.Context) {
	h.log.Info(ctx, "GetEquippedTitle - 获取当前装备称号")

	playerUUID, xErr := h.parsePlayerUUID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	title, xErr := h.service.titleLogic.GetEquippedTitle(ctx, playerUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", title)
}

func (h *TitlePlayerHandler) parsePlayerUUID(ctx *gin.Context) (uuid.UUID, *xError.Error) {
	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		return uuid.Nil, xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err)
	}
	return playerUUID, nil
}
