package handler

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiAchievement "github.com/frontleaves-mc/frontleaves-plugin/api/achievement"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

type AchievementAdminHandler handler

func NewAchievementAdminHandler(ctx context.Context) *AchievementAdminHandler {
	return NewHandler[AchievementAdminHandler](ctx, "AchievementAdminHandler")
}

func (h *AchievementAdminHandler) CreateAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "CreateAchievement - 创建成就")

	var req apiAchievement.CreateAchievementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	ach, xErr := h.service.achievementLogic.CreateAchievement(ctx, req.Name, req.Description, entity.AchievementType(req.Type), req.ConditionKey, req.ConditionParams, req.RewardConfig, req.SortOrder)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "创建成功", ach)
}

func (h *AchievementAdminHandler) UpdateAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "UpdateAchievement - 更新成就")

	achID, xErr := h.parseAchievementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiAchievement.UpdateAchievementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	ach, xErr := h.service.achievementLogic.UpdateAchievement(ctx, achID, req.Name, req.Description, entity.AchievementType(req.Type), req.ConditionKey, req.ConditionParams, req.RewardConfig, req.SortOrder, req.IsActive)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", ach)
}

func (h *AchievementAdminHandler) DeleteAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "DeleteAchievement - 删除成就")

	achID, xErr := h.parseAchievementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.achievementLogic.DeleteAchievement(ctx, achID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "删除成功", nil)
}

func (h *AchievementAdminHandler) GetAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "GetAchievement - 查询成就详情")

	achID, xErr := h.parseAchievementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	ach, xErr := h.service.achievementLogic.GetAchievement(ctx, achID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", ach)
}

func (h *AchievementAdminHandler) ListAchievements(ctx *gin.Context) {
	h.log.Info(ctx, "ListAchievements - 查询成就列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var achType *int16
	if t := ctx.Query("type"); t != "" {
		v, _ := strconv.Atoi(t)
		tv := int16(v)
		achType = &tv
	}

	achievements, total, xErr := h.service.achievementLogic.ListAchievements(ctx, page, pageSize, achType)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", gin.H{
		"list":      achievements,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AchievementAdminHandler) GrantAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "GrantAchievement - 手动授予成就")

	achID, xErr := h.parseAchievementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiAchievement.GrantAchievementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerUUID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	if xErr := h.service.achievementLogic.GrantAchievement(ctx, achID, playerUUID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "授予成功", nil)
}

func (h *AchievementAdminHandler) parseAchievementID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的成就 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}

// unused but kept for potential future use
var _ = json.RawMessage{}
