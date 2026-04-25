package handler

import (
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	"github.com/gin-gonic/gin"
)

// Ping 健康检查
//
// @Summary     健康检查
// @Description 检查服务及基础设施运行状态，返回应用信息、数据库和 Redis 连通性
// @Tags        健康检查接口
// @Accept      json
// @Produce     json
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /health/ping [GET]
func (h *HealthHandler) Ping(ctx *gin.Context) {
	h.log.Info(ctx, "Ping - 健康检查")

	status, xErr := h.service.healthLogic.Ping(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "pong", status)
}
