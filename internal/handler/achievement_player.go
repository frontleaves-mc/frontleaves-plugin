package handler

import (
	"context"
	"strconv"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AchievementGameProfileHandler handler

func NewAchievementGameProfileHandler(ctx context.Context) *AchievementGameProfileHandler {
	return NewHandler[AchievementGameProfileHandler](ctx, "AchievementGameProfileHandler")
}

// GetPlayerAchievements 查询玩家成就列表
//
// @Summary     [玩家] 查询玩家成就列表
// @Description 根据玩家 UUID 查询该玩家已获得的所有成就
// @Tags        玩家-成就接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "玩家UUID"
// @Success     200  {object}  xBase.BaseResponse{data=[]apiAchievement.PlayerAchievementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                                "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse                                                "玩家不存在"
// @Router      /game-profiles/:uuid/achievements [GET]
func (h *AchievementGameProfileHandler) GetPlayerAchievements(ctx *gin.Context) {
	h.log.Info(ctx, "GetPlayerAchievements - 查询玩家成就列表")

	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	achievements, xErr := h.service.achievementLogic.GetPlayerAchievements(ctx.Request.Context(), playerUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", achievements)
}

// ClaimReward 领取成就奖励
//
// @Summary     [玩家] 领取成就奖励
// @Description 玩家领取指定成就的奖励
// @Tags        玩家-成就接口
// @Accept      json
// @Produce     json
// @Param       uuid   path  string  true  "玩家UUID"
// @Param       achId  path  string  true  "成就ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAchievement.AchievementClaimResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                             "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse                                             "成就不存在"
// @Router      /game-profiles/:uuid/achievements/:ach_id/claim [POST]
func (h *AchievementGameProfileHandler) ClaimReward(ctx *gin.Context) {
	h.log.Info(ctx, "ClaimReward - 领取成就奖励")

	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	achIDStr := ctx.Param("ach_id")
	achID, err := strconv.ParseInt(achIDStr, 10, 64)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的成就 ID", true, err))
		return
	}

	claim, xErr := h.service.achievementLogic.ClaimReward(ctx.Request.Context(), playerUUID, xSnowflake.SnowflakeID(achID))
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "领取成功", claim)
}

// ListPublicAchievements 查询公开成就列表
//
// @Summary     [公共] 查询公开成就列表
// @Description 查询所有公开可查看的成就列表
// @Tags        成就接口
// @Accept      json
// @Produce     json
// @Success     200  {object}  xBase.BaseResponse{data=[]apiAchievement.AchievementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                             "请求参数错误"
// @Router      /achievements [GET]
func (h *AchievementGameProfileHandler) ListPublicAchievements(ctx *gin.Context) {
	h.log.Info(ctx, "ListPublicAchievements - 查询公开成就列表")

	achievements, xErr := h.service.achievementLogic.ListPublicAchievements(ctx.Request.Context())
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", achievements)
}
