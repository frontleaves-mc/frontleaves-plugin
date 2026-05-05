package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

type AnnouncementUserHandler handler

func NewAnnouncementUserHandler(ctx context.Context) *AnnouncementUserHandler {
	return NewHandler[AnnouncementUserHandler](ctx, "AnnouncementUserHandler")
}

// ListPublishedAnnouncements 查询已发布公告列表
//
// @Summary     查询已发布公告列表
// @Description 分页查询已发布公告列表，支持按类型筛选（公开接口，无需认证）
// @Tags        公开-公告接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Param       type       query  int  false  "公告类型筛选(1=站内,2=全局)"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /announcements [GET]
func (h *AnnouncementUserHandler) ListPublishedAnnouncements(ctx *gin.Context) {
	h.log.Info(ctx, "ListPublishedAnnouncements - 查询已发布公告列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var typeFilter *int16
	if t := ctx.Query("type"); t != "" {
		v, _ := strconv.Atoi(t)
		tv := int16(v)
		typeFilter = &tv
	}

	statusFilter := int16(entity.AnnouncementStatusPublished)

	announcements, total, xErr := h.service.announcementLogic.ListAnnouncements(ctx.Request.Context(), page, pageSize, typeFilter, &statusFilter)
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

// GetPublishedAnnouncement 查询已发布公告详情
//
// @Summary     查询已发布公告详情
// @Description 根据公告 ID 查询已发布公告详情，返回完整 Markdown 内容（公开接口）
// @Tags        公开-公告接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "公告ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiAnnouncement.AnnouncementResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "公告不存在"
// @Router      /announcements/:id [GET]
func (h *AnnouncementUserHandler) GetPublishedAnnouncement(ctx *gin.Context) {
	h.log.Info(ctx, "GetPublishedAnnouncement - 查询已发布公告详情")

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

	if announcement.Status != int16(entity.AnnouncementStatusPublished) {
		_ = ctx.Error(xError.NewError(nil, xError.NotFound, "公告不存在", true, nil))
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", announcement)
}

func (h *AnnouncementUserHandler) parseAnnouncementID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的公告 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}
