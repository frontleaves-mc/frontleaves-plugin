package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiAnnouncement "github.com/frontleaves-mc/frontleaves-plugin/api/announcement"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

type AnnouncementAdminHandler handler

func NewAnnouncementAdminHandler(ctx context.Context) *AnnouncementAdminHandler {
	return NewHandler[AnnouncementAdminHandler](ctx, "AnnouncementAdminHandler")
}

// CreateAnnouncement 创建新公告
//
// @Summary     [管理] 创建公告
// @Description 创建新的公告
// @Tags        管理-公告接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiAnnouncement.CreateAnnouncementRequest  true  "创建公告请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncement.AnnouncementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/announcements [POST]
func (h *AnnouncementAdminHandler) CreateAnnouncement(ctx *gin.Context) {
	h.log.Info(ctx, "CreateAnnouncement - 创建公告")

	var req apiAnnouncement.CreateAnnouncementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	announcement, xErr := h.service.announcementLogic.CreateAnnouncement(ctx.Request.Context(), req.Title, req.Content, entity.AnnouncementType(req.Type))
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "创建成功", announcement)
}

// UpdateAnnouncement 更新指定公告信息
//
// @Summary     [管理] 更新公告
// @Description 更新指定公告的信息
// @Tags        管理-公告接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                                  true  "公告ID"
// @Param       request  body  apiAnnouncement.UpdateAnnouncementRequest  true  "更新公告请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncement.AnnouncementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "公告不存在"
// @Router      /admin/announcements/:id [PUT]
func (h *AnnouncementAdminHandler) UpdateAnnouncement(ctx *gin.Context) {
	h.log.Info(ctx, "UpdateAnnouncement - 更新公告")

	announcementID, xErr := h.parseAnnouncementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiAnnouncement.UpdateAnnouncementRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	announcement, xErr := h.service.announcementLogic.UpdateAnnouncement(ctx.Request.Context(), announcementID, req.Title, req.Content, entity.AnnouncementType(req.Type))
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", announcement)
}

// DeleteAnnouncement 删除指定公告
//
// @Summary     [管理] 删除公告
// @Description 删除指定公告
// @Tags        管理-公告接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "公告ID"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "公告不存在"
// @Router      /admin/announcements/:id [DELETE]
func (h *AnnouncementAdminHandler) DeleteAnnouncement(ctx *gin.Context) {
	h.log.Info(ctx, "DeleteAnnouncement - 删除公告")

	announcementID, xErr := h.parseAnnouncementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.announcementLogic.DeleteAnnouncement(ctx.Request.Context(), announcementID); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "删除成功", nil)
}

// GetAnnouncement 查询公告详情
//
// @Summary     [管理] 查询公告详情
// @Description 根据公告 ID 查询公告详情
// @Tags        管理-公告接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "公告ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncement.AnnouncementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "公告不存在"
// @Router      /admin/announcements/:id [GET]
func (h *AnnouncementAdminHandler) GetAnnouncement(ctx *gin.Context) {
	h.log.Info(ctx, "GetAnnouncement - 查询公告详情")

	announcementID, xErr := h.parseAnnouncementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	announcement, xErr := h.service.announcementLogic.GetAnnouncement(ctx.Request.Context(), announcementID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", announcement)
}

// ListAnnouncements 查询公告列表
//
// @Summary     [管理] 查询公告列表
// @Description 分页查询公告列表，可按类型和状态筛选
// @Tags        管理-公告接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Param       type       query  int  false  "公告类型筛选"
// @Param       status     query  int  false  "公告状态筛选"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/announcements [GET]
func (h *AnnouncementAdminHandler) ListAnnouncements(ctx *gin.Context) {
	h.log.Info(ctx, "ListAnnouncements - 查询公告列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var annType *int16
	if t := ctx.Query("type"); t != "" {
		v, _ := strconv.Atoi(t)
		tv := int16(v)
		annType = &tv
	}

	var annStatus *int16
	if s := ctx.Query("status"); s != "" {
		v, _ := strconv.Atoi(s)
		sv := int16(v)
		annStatus = &sv
	}

	announcements, total, xErr := h.service.announcementLogic.ListAnnouncements(ctx.Request.Context(), page, pageSize, annType, annStatus)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", gin.H{
		"list":      announcements,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// PublishAnnouncement 发布公告
//
// @Summary     [管理] 发布公告
// @Description 将指定公告发布，仅草稿状态可发布
// @Tags        管理-公告接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "公告ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncement.AnnouncementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "公告不存在"
// @Router      /admin/announcements/:id/publish [POST]
func (h *AnnouncementAdminHandler) PublishAnnouncement(ctx *gin.Context) {
	h.log.Info(ctx, "PublishAnnouncement - 发布公告")

	announcementID, xErr := h.parseAnnouncementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	announcement, xErr := h.service.announcementLogic.PublishAnnouncement(ctx.Request.Context(), announcementID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "发布成功", announcement)
}

// OfflineAnnouncement 下线公告
//
// @Summary     [管理] 下线公告
// @Description 将指定公告下线，仅已发布状态可下线
// @Tags        管理-公告接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "公告ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncement.AnnouncementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "公告不存在"
// @Router      /admin/announcements/:id/offline [POST]
func (h *AnnouncementAdminHandler) OfflineAnnouncement(ctx *gin.Context) {
	h.log.Info(ctx, "OfflineAnnouncement - 下线公告")

	announcementID, xErr := h.parseAnnouncementID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	announcement, xErr := h.service.announcementLogic.OfflineAnnouncement(ctx.Request.Context(), announcementID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "下线成功", announcement)
}

func (h *AnnouncementAdminHandler) parseAnnouncementID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的公告 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}
