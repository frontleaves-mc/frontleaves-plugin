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

// CreateAchievement 创建新成就
//
// @Summary     [管理] 创建成就
// @Description 创建新的成就定义
// @Tags        管理-成就接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiAchievement.CreateAchievementRequest  true  "创建成就请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiAchievement.AchievementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                          "请求参数错误"
// @Router      /admin/achievements [POST]
func (h *AchievementAdminHandler) CreateAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "CreateAchievement - 创建成就")

	var req apiAchievement.CreateAchievementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	ach, xErr := h.service.achievementLogic.CreateAchievement(ctx.Request.Context(), req.Name, req.Description, entity.AchievementType(req.Type), req.ConditionKey, req.ConditionParams, req.RewardConfig, req.SortOrder)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "创建成功", ach)
}

// UpdateAchievement 更新成就信息
//
// @Summary     [管理] 更新成就
// @Description 更新指定成就的信息
// @Tags        管理-成就接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                                   true  "成就ID"
// @Param       request  body  apiAchievement.UpdateAchievementRequest  true  "更新成就请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiAchievement.AchievementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                          "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse                                          "成就不存在"
// @Router      /admin/achievements/:id [PUT]
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

	ach, xErr := h.service.achievementLogic.UpdateAchievement(ctx.Request.Context(), achID, req.Name, req.Description, entity.AchievementType(req.Type), req.ConditionKey, req.ConditionParams, req.RewardConfig, req.SortOrder, req.IsActive)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", ach)
}

// DeleteAchievement 删除成就
//
// @Summary     [管理] 删除成就
// @Description 删除指定成就
// @Tags        管理-成就接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "成就ID"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "成就不存在"
// @Router      /admin/achievements/:id [DELETE]
func (h *AchievementAdminHandler) DeleteAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "DeleteAchievement - 删除成就")

	achID, xErr := h.parseAchievementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.achievementLogic.DeleteAchievement(ctx.Request.Context(), achID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "删除成功", nil)
}

// GetAchievement 查询成就详情
//
// @Summary     [管理] 查询成就详情
// @Description 根据成就 ID 查询成就详情
// @Tags        管理-成就接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "成就ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAchievement.AchievementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                          "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse                                          "成就不存在"
// @Router      /admin/achievements/:id [GET]
func (h *AchievementAdminHandler) GetAchievement(ctx *gin.Context) {
	h.log.Info(ctx, "GetAchievement - 查询成就详情")

	achID, xErr := h.parseAchievementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	ach, xErr := h.service.achievementLogic.GetAchievement(ctx.Request.Context(), achID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", ach)
}

// ListAchievements 查询成就列表
//
// @Summary     [管理] 查询成就列表
// @Description 分页查询成就列表，可按类型筛选
// @Tags        管理-成就接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Param       type       query  int  false  "成就类型筛选"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/achievements [GET]
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

	achievements, total, xErr := h.service.achievementLogic.ListAchievements(ctx.Request.Context(), page, pageSize, achType)
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

// GrantAchievement 手动授予成就
//
// @Summary     [管理] 手动授予成就
// @Description 管理员手动将指定成就授予玩家
// @Tags        管理-成就接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                                  true  "成就ID"
// @Param       request  body  apiAchievement.GrantAchievementRequest  true  "授予成就请求"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "成就不存在"
// @Router      /admin/achievements/:id/grant [POST]
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

	if xErr := h.service.achievementLogic.GrantAchievement(ctx.Request.Context(), achID, playerUUID); xErr != nil {
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
