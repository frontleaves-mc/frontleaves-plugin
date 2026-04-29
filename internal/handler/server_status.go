package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
)

type ServerStatusHandler handler

func NewServerStatusHandler(ctx context.Context) *ServerStatusHandler {
	return NewHandler[ServerStatusHandler](ctx, "ServerStatusHandler")
}

// ListServerStatus 获取所有服务器状态
//
// @Summary     [用户] 获取所有服务器状态
// @Description 获取所有 Minecraft 服务器的在线状态、TPS 和玩家列表
// @Tags        服务器状态接口
// @Accept      json
// @Produce     json
// @Success     200  {object}  xBase.BaseResponse{data=[]apiServerStatus.ServerStatusResponse}  "成功"
// @Failure     500  {object}  xBase.BaseResponse  "服务器内部错误"
// @Router      /servers/status [get]
func (h *ServerStatusHandler) ListServerStatus(ctx *gin.Context) {
	servers, xErr := h.service.serverStatusLogic.GetAllServerStatus(ctx.Request.Context())
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", servers)
}

// RefreshServerStatus 刷新指定服务器状态
//
// @Summary     [用户] 刷新服务器状态
// @Description 重新获取指定服务器的最新状态信息
// @Tags        服务器状态接口
// @Accept      json
// @Produce     json
// @Param       name  path  string  true  "服务器名称"
// @Success     200  {object}  xBase.BaseResponse{data=apiServerStatus.ServerStatusResponse}  "成功"
// @Failure     500  {object}  xBase.BaseResponse  "服务器内部错误"
// @Router      /servers/{name}/refresh [post]
func (h *ServerStatusHandler) RefreshServerStatus(ctx *gin.Context) {
	name := ctx.Param("name")

	serverStatus, xErr := h.service.serverStatusLogic.GetServerStatus(ctx.Request.Context(), name)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "刷新成功", serverStatus)
}
