package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
)

type AchievementPlayerHandler handler

func NewAchievementPlayerHandler(ctx context.Context) *AchievementPlayerHandler {
	return NewHandler[AchievementPlayerHandler](ctx, "AchievementPlayerHandler")
}

func (h *AchievementPlayerHandler) GetPlayerAchievements(ctx *gin.Context) {
	h.log.Info(ctx, "GetPlayerAchievements - 查询玩家成就列表")

	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	achievements, xErr := h.service.achievementLogic.GetPlayerAchievements(ctx, playerUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", achievements)
}

func (h *AchievementPlayerHandler) ClaimReward(ctx *gin.Context) {
	h.log.Info(ctx, "ClaimReward - 领取成就奖励")

	playerUUID, err := uuid.Parse(ctx.Param("uuid"))
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	achIDStr := ctx.Param("achId")
	achID, err := strconv.ParseInt(achIDStr, 10, 64)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的成就 ID", true, err))
		return
	}

	claim, xErr := h.service.achievementLogic.ClaimReward(ctx, playerUUID, xSnowflake.SnowflakeID(achID))
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "领取成功", claim)
}

func (h *AchievementPlayerHandler) ListPublicAchievements(ctx *gin.Context) {
	h.log.Info(ctx, "ListPublicAchievements - 查询公开成就列表")

	achievements, xErr := h.service.achievementLogic.ListPublicAchievements(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", achievements)
}
