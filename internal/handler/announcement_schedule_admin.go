package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiAnnouncementSchedule "github.com/frontleaves-mc/frontleaves-plugin/api/announcement_schedule"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

type AnnouncementScheduleAdminHandler handler

func NewAnnouncementScheduleAdminHandler(ctx context.Context) *AnnouncementScheduleAdminHandler {
	return NewHandler[AnnouncementScheduleAdminHandler](ctx, "AnnouncementScheduleAdminHandler")
}

// CreateSchedule 创建公告调度
//
// @Summary     [管理] 创建公告调度
// @Description 创建新的公告调度，指定调度名称、模式、间隔秒数和调度项
// @Tags        管理-公告调度接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiAnnouncementSchedule.CreateScheduleRequest  true  "创建调度请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncementSchedule.ScheduleResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/announcements/schedules [POST]
func (h *AnnouncementScheduleAdminHandler) CreateSchedule(ctx *gin.Context) {
	h.log.Info(ctx, "CreateSchedule - 创建公告调度")

	var req apiAnnouncementSchedule.CreateScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	schedule, xErr := h.service.announcementScheduleLogic.CreateSchedule(ctx.Request.Context(), req.Name, entity.ScheduleMode(req.Mode), req.IntervalSeconds, req.Items)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "创建成功", schedule)
}

// UpdateSchedule 更新公告调度
//
// @Summary     [管理] 更新公告调度
// @Description 更新指定公告调度的名称、模式、间隔秒数和调度项
// @Tags        管理-公告调度接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                                        true  "调度ID"
// @Param       request  body  apiAnnouncementSchedule.UpdateScheduleRequest  true  "更新调度请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncementSchedule.ScheduleResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "调度不存在"
// @Router      /admin/announcements/schedules/:id [PUT]
func (h *AnnouncementScheduleAdminHandler) UpdateSchedule(ctx *gin.Context) {
	h.log.Info(ctx, "UpdateSchedule - 更新公告调度")

	scheduleID, xErr := h.parseScheduleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiAnnouncementSchedule.UpdateScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	schedule, xErr := h.service.announcementScheduleLogic.UpdateSchedule(ctx.Request.Context(), scheduleID, req.Name, entity.ScheduleMode(req.Mode), req.IntervalSeconds, req.Items)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", schedule)
}

// DeleteSchedule 删除公告调度
//
// @Summary     [管理] 删除公告调度
// @Description 删除指定公告调度，活动调度需先停用才能删除
// @Tags        管理-公告调度接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "调度ID"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "调度不存在"
// @Router      /admin/announcements/schedules/:id [DELETE]
func (h *AnnouncementScheduleAdminHandler) DeleteSchedule(ctx *gin.Context) {
	h.log.Info(ctx, "DeleteSchedule - 删除公告调度")

	scheduleID, xErr := h.parseScheduleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.announcementScheduleLogic.DeleteSchedule(ctx.Request.Context(), scheduleID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "删除成功", nil)
}

// GetSchedule 查询公告调度详情
//
// @Summary     [管理] 查询公告调度详情
// @Description 根据调度 ID 查询公告调度详情，包含调度项列表
// @Tags        管理-公告调度接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "调度ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncementSchedule.ScheduleResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "调度不存在"
// @Router      /admin/announcements/schedules/:id [GET]
func (h *AnnouncementScheduleAdminHandler) GetSchedule(ctx *gin.Context) {
	h.log.Info(ctx, "GetSchedule - 查询公告调度详情")

	scheduleID, xErr := h.parseScheduleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	schedule, xErr := h.service.announcementScheduleLogic.GetSchedule(ctx.Request.Context(), scheduleID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", schedule)
}

// ListSchedules 查询公告调度列表
//
// @Summary     [管理] 查询公告调度列表
// @Description 分页查询公告调度列表
// @Tags        管理-公告调度接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/announcements/schedules [GET]
func (h *AnnouncementScheduleAdminHandler) ListSchedules(ctx *gin.Context) {
	h.log.Info(ctx, "ListSchedules - 查询公告调度列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	schedules, total, xErr := h.service.announcementScheduleLogic.ListSchedules(ctx.Request.Context(), page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", gin.H{
		"list":      schedules,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ActivateSchedule 激活公告调度
//
// @Summary     [管理] 激活公告调度
// @Description 将调度设为活动状态并启动推送引擎
// @Tags        管理-公告调度接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "调度ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncementSchedule.ScheduleResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "调度不存在"
// @Router      /admin/announcements/schedules/:id/activate [POST]
func (h *AnnouncementScheduleAdminHandler) ActivateSchedule(ctx *gin.Context) {
	h.log.Info(ctx, "ActivateSchedule - 激活公告调度")

	scheduleID, xErr := h.parseScheduleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	schedule, xErr := h.service.announcementScheduleLogic.ActivateSchedule(ctx.Request.Context(), scheduleID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "激活成功", schedule)
}

// DeactivateSchedule 停用公告调度
//
// @Summary     [管理] 停用公告调度
// @Description 停用调度并停止推送引擎
// @Tags        管理-公告调度接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "调度ID"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "调度不存在"
// @Router      /admin/announcements/schedules/:id/deactivate [POST]
func (h *AnnouncementScheduleAdminHandler) DeactivateSchedule(ctx *gin.Context) {
	h.log.Info(ctx, "DeactivateSchedule - 停用公告调度")

	scheduleID, xErr := h.parseScheduleID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.announcementScheduleLogic.DeactivateSchedule(ctx.Request.Context(), scheduleID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "停用成功", nil)
}

func (h *AnnouncementScheduleAdminHandler) parseScheduleID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的调度 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}
