package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiServer "github.com/frontleaves-mc/frontleaves-plugin/api/server"
)

type ServerAdminHandler handler

func NewServerAdminHandler(ctx context.Context) *ServerAdminHandler {
	return NewHandler[ServerAdminHandler](ctx, "ServerAdminHandler")
}

// CreateServer 创建服务器
//
// @Summary     [超管] 创建服务器
// @Description 创建新的服务器记录
// @Tags        超管-服务器管理接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiServer.CreateServerRequest  true  "创建服务器请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiServer.ServerResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/servers [POST]
func (h *ServerAdminHandler) CreateServer(ctx *gin.Context) {
	h.log.Info(ctx, "CreateServer - 创建服务器")

	var req apiServer.CreateServerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	resp, xErr := h.service.serverLogic.Create(ctx.Request.Context(), req.Name, req.DisplayName, req.Description, req.Address, req.SortOrder)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "创建成功", resp)
}

// UpdateServer 更新服务器
//
// @Summary     [超管] 更新服务器
// @Description 更新指定服务器的信息
// @Tags        超管-服务器管理接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                          true  "服务器ID"
// @Param       request  body  apiServer.UpdateServerRequest  true  "更新服务器请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiServer.ServerResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "服务器不存在"
// @Router      /admin/servers/{id} [PUT]
func (h *ServerAdminHandler) UpdateServer(ctx *gin.Context) {
	h.log.Info(ctx, "UpdateServer - 更新服务器")

	id, xErr := h.parseServerID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiServer.UpdateServerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	resp, xErr := h.service.serverLogic.Update(ctx.Request.Context(), id, req.DisplayName, req.Description, req.Address, req.SortOrder)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", resp)
}

// DeleteServer 删除服务器
//
// @Summary     [超管] 删除服务器
// @Description 硬删除指定的服务器记录
// @Tags        超管-服务器管理接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "服务器ID"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "服务器不存在"
// @Router      /admin/servers/{id} [DELETE]
func (h *ServerAdminHandler) DeleteServer(ctx *gin.Context) {
	h.log.Info(ctx, "DeleteServer - 删除服务器")

	id, xErr := h.parseServerID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.serverLogic.Delete(ctx.Request.Context(), id); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "删除成功", nil)
}

// GetServer 查询服务器详情
//
// @Summary     [超管] 查询服务器详情
// @Description 根据服务器 ID 查询详情
// @Tags        超管-服务器管理接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "服务器ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiServer.ServerResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "服务器不存在"
// @Router      /admin/servers/{id} [GET]
func (h *ServerAdminHandler) GetServer(ctx *gin.Context) {
	h.log.Info(ctx, "GetServer - 查询服务器详情")

	id, xErr := h.parseServerID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	resp, xErr := h.service.serverLogic.GetByID(ctx.Request.Context(), id)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", resp)
}

// ListServers 查询服务器列表
//
// @Summary     [超管] 查询服务器列表
// @Description 分页查询服务器列表
// @Tags        超管-服务器管理接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiServer.ServerListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/servers [GET]
func (h *ServerAdminHandler) ListServers(ctx *gin.Context) {
	h.log.Info(ctx, "ListServers - 查询服务器列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, xErr := h.service.serverLogic.List(ctx.Request.Context(), page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", resp)
}

// SetServerPublic 设置服务器公开状态
//
// @Summary     [超管] 设置服务器公开状态
// @Description 设置服务器是否对外公开可见
// @Tags        超管-服务器管理接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                            true  "服务器ID"
// @Param       request  body  apiServer.SetServerPublicRequest  true  "设置公开状态请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiServer.ServerResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "服务器不存在"
// @Router      /admin/servers/{id}/public [PUT]
func (h *ServerAdminHandler) SetServerPublic(ctx *gin.Context) {
	h.log.Info(ctx, "SetServerPublic - 设置服务器公开状态")

	id, xErr := h.parseServerID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiServer.SetServerPublicRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	resp, xErr := h.service.serverLogic.SetPublic(ctx.Request.Context(), id, req.IsPublic)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "设置成功", resp)
}

// SetServerEnabled 设置服务器启用状态
//
// @Summary     [超管] 设置服务器启用状态
// @Description 启用或禁用服务器
// @Tags        超管-服务器管理接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                             true  "服务器ID"
// @Param       request  body  apiServer.SetServerEnabledRequest  true  "设置启用状态请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiServer.ServerResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "服务器不存在"
// @Router      /admin/servers/{id}/enabled [PUT]
func (h *ServerAdminHandler) SetServerEnabled(ctx *gin.Context) {
	h.log.Info(ctx, "SetServerEnabled - 设置服务器启用状态")

	id, xErr := h.parseServerID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiServer.SetServerEnabledRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	resp, xErr := h.service.serverLogic.SetEnabled(ctx.Request.Context(), id, req.IsEnabled)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "设置成功", resp)
}

func (h *ServerAdminHandler) parseServerID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的服务器 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}
