package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
)

type GameProfileHandler handler

func NewGameProfileHandler(ctx context.Context) *GameProfileHandler {
	return NewHandler[GameProfileHandler](ctx, "GameProfileHandler")
}

// GetGameProfile 查询游戏档案信息
//
// @Summary     [玩家] 查询游戏档案信息
// @Description 根据 UUID 查询游戏档案详细信息
// @Tags        游戏档案接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "档案UUID"
// @Success     200  {object}  xBase.BaseResponse{data=apiGameProfile.GameProfileResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                      "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse                                      "游戏档案不存在"
// @Router      /game-profiles/:uuid [GET]
func (h *GameProfileHandler) GetGameProfile(ctx *gin.Context) {
	h.log.Info(ctx, "GetGameProfile - 查询游戏档案信息")

	uuidVal, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的 UUID", true, err))
		return
	}

	gp, xErr := h.service.gameProfileLogic.GetPlayer(ctx.Request.Context(), uuidVal)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", gp)
}

// ListGameProfiles 查询游戏档案列表
//
// @Summary     [玩家] 查询游戏档案列表
// @Description 分页查询游戏档案列表
// @Tags        游戏档案接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /game-profiles [GET]
func (h *GameProfileHandler) ListGameProfiles(ctx *gin.Context) {
	h.log.Info(ctx, "ListGameProfiles - 查询游戏档案列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	gps, total, xErr := h.service.gameProfileLogic.ListPlayers(ctx.Request.Context(), page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", gin.H{
		"list":      gps,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}