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
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

type TitleAdminHandler handler

func NewTitleAdminHandler(ctx context.Context) *TitleAdminHandler {
	return NewHandler[TitleAdminHandler](ctx, "TitleAdminHandler")
}

// @Summary     创建称号
// @Description 创建新的称号
// @Tags        管理员-称号接口
// @Accept      json
// @Produce     json
// @Param       request body apiTitle.CreateTitleRequest true "创建称号请求"
// @Success     200 {object} xBase.BaseResponse{data=apiTitle.TitleResponse}
// @Failure     400 {object} xBase.BaseResponse
// @Router      /api/v1/admin/titles [POST]
func (h *TitleAdminHandler) CreateTitle(ctx *gin.Context) {
	h.log.Info(ctx, "CreateTitle - 创建称号")

	var req apiTitle.CreateTitleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	title, xErr := h.service.titleLogic.CreateTitle(ctx, req.Name, req.Description, entity.TitleType(req.Type), req.PermissionGroup)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "创建成功", title)
}

// @Summary     更新称号
// @Description 更新指定称号信息
// @Tags        管理员-称号接口
// @Accept      json
// @Produce     json
// @Param       id path string true "称号ID"
// @Param       request body apiTitle.UpdateTitleRequest true "更新称号请求"
// @Success     200 {object} xBase.BaseResponse{data=apiTitle.TitleResponse}
// @Failure     400 {object} xBase.BaseResponse
// @Router      /api/v1/admin/titles/:id [PUT]
func (h *TitleAdminHandler) UpdateTitle(ctx *gin.Context) {
	h.log.Info(ctx, "UpdateTitle - 更新称号")

	titleID, xErr := h.parseTitleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiTitle.UpdateTitleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	title, xErr := h.service.titleLogic.UpdateTitle(ctx, titleID, req.Name, req.Description, entity.TitleType(req.Type), req.PermissionGroup, req.IsActive)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", title)
}

// @Summary     删除称号
// @Description 删除指定称号
// @Tags        管理员-称号接口
// @Accept      json
// @Produce     json
// @Param       id path string true "称号ID"
// @Success     200 {object} xBase.BaseResponse
// @Failure     400 {object} xBase.BaseResponse
// @Router      /api/v1/admin/titles/:id [DELETE]
func (h *TitleAdminHandler) DeleteTitle(ctx *gin.Context) {
	h.log.Info(ctx, "DeleteTitle - 删除称号")

	titleID, xErr := h.parseTitleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.titleLogic.DeleteTitle(ctx, titleID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "删除成功", nil)
}

// @Summary     称号详情
// @Description 查询指定称号详情
// @Tags        管理员-称号接口
// @Accept      json
// @Produce     json
// @Param       id path string true "称号ID"
// @Success     200 {object} xBase.BaseResponse{data=apiTitle.TitleResponse}
// @Failure     400 {object} xBase.BaseResponse
// @Router      /api/v1/admin/titles/:id [GET]
func (h *TitleAdminHandler) GetTitle(ctx *gin.Context) {
	h.log.Info(ctx, "GetTitle - 查询称号详情")

	titleID, xErr := h.parseTitleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	title, xErr := h.service.titleLogic.GetTitle(ctx, titleID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", title)
}

// @Summary     称号列表
// @Description 查询称号列表（分页）
// @Tags        管理员-称号接口
// @Accept      json
// @Produce     json
// @Param       page query int false "页码"
// @Param       page_size query int false "每页数量"
// @Param       type query int false "称号类型筛选"
// @Success     200 {object} xBase.BaseResponse
// @Failure     400 {object} xBase.BaseResponse
// @Router      /api/v1/admin/titles [GET]
func (h *TitleAdminHandler) ListTitles(ctx *gin.Context) {
	h.log.Info(ctx, "ListTitles - 查询称号列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var titleType *int16
	if t := ctx.Query("type"); t != "" {
		v, _ := strconv.Atoi(t)
		tv := int16(v)
		titleType = &tv
	}

	titles, total, xErr := h.service.titleLogic.ListTitles(ctx, page, pageSize, titleType)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", gin.H{
		"list":      titles,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// @Summary     分配称号
// @Description 将指定称号分配给玩家
// @Tags        管理员-称号接口
// @Accept      json
// @Produce     json
// @Param       id path string true "称号ID"
// @Param       request body apiTitle.AssignTitleRequest true "分配请求"
// @Success     200 {object} xBase.BaseResponse
// @Failure     400 {object} xBase.BaseResponse
// @Router      /api/v1/admin/titles/:id/assign [POST]
func (h *TitleAdminHandler) AssignTitle(ctx *gin.Context) {
	h.log.Info(ctx, "AssignTitle - 分配称号")

	titleID, xErr := h.parseTitleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiTitle.AssignTitleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerUUID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	if xErr := h.service.titleLogic.AssignTitleToPlayer(ctx, titleID, playerUUID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "分配成功", nil)
}

// @Summary     撤销称号
// @Description 撤销玩家的指定称号
// @Tags        管理员-称号接口
// @Accept      json
// @Produce     json
// @Param       id path string true "称号ID"
// @Param       request body apiTitle.AssignTitleRequest true "撤销请求"
// @Success     200 {object} xBase.BaseResponse
// @Failure     400 {object} xBase.BaseResponse
// @Router      /api/v1/admin/titles/:id/assign [DELETE]
func (h *TitleAdminHandler) RevokeTitle(ctx *gin.Context) {
	h.log.Info(ctx, "RevokeTitle - 撤销称号")

	titleID, xErr := h.parseTitleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiTitle.AssignTitleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerUUID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	if xErr := h.service.titleLogic.RevokeTitleFromPlayer(ctx, titleID, playerUUID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "撤销成功", nil)
}

func (h *TitleAdminHandler) parseTitleID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的称号 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}
