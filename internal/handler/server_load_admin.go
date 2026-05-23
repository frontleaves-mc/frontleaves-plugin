package handler

import (
	"context"
	"strconv"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	"github.com/gin-gonic/gin"
)

type ServerLoadAdminHandler handler

func NewServerLoadAdminHandler(ctx context.Context) *ServerLoadAdminHandler {
	return NewHandler[ServerLoadAdminHandler](ctx, "ServerLoadAdminHandler")
}

// GetAllRealtimeLoad 批量查询所有服务器实时负载
//
// @Summary     [超管] 批量查询服务器实时负载
// @Description 查询所有启用服务器的实时负载数据，包括 TPS、CPU、内存、JVM 等指标
// @Tags        超管-服务器负载接口
// @Accept      json
// @Produce     json
// @Success     200  {object}  xBase.BaseResponse{data=[]apiServerLoad.ServerRealtimeLoadResponse}  "成功"
// @Failure     401  {object}  xBase.BaseResponse  "未授权"
// @Failure     403  {object}  xBase.BaseResponse  "无权限"
// @Router      /admin/servers/load/realtime [GET]
func (h *ServerLoadAdminHandler) GetAllRealtimeLoad(ctx *gin.Context) {
	h.log.Info(ctx, "GetAllRealtimeLoad - 批量查询服务器实时负载")

	resp, xErr := h.service.serverLoadLogic.GetRealtimeAll(ctx.Request.Context())
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", resp)
}

// GetRealtimeLoad 查询单台服务器实时负载
//
// @Summary     [超管] 查询单台服务器实时负载
// @Description 根据服务器 ID 查询单台服务器的实时负载数据
// @Tags        超管-服务器负载接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "服务器ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiServerLoad.ServerRealtimeLoadResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "服务器不存在"
// @Router      /admin/servers/load/{id}/realtime [GET]
func (h *ServerLoadAdminHandler) GetRealtimeLoad(ctx *gin.Context) {
	h.log.Info(ctx, "GetRealtimeLoad - 查询单台服务器实时负载")

	id, xErr := h.parseServerLoadID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	resp, xErr := h.service.serverLoadLogic.GetRealtimeByServerID(ctx.Request.Context(), id)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", resp)
}

// GetLoadHistory 查询服务器历史负载趋势
//
// @Summary     [超管] 查询服务器历史负载趋势
// @Description 根据服务器 ID 和时间范围查询历史负载数据趋势，支持分页
// @Tags        超管-服务器负载接口
// @Accept      json
// @Produce     json
// @Param       id        path   string  true   "服务器ID"
// @Param       start     query  string  true   "开始时间 (RFC3339)"
// @Param       end       query  string  true   "结束时间 (RFC3339)"
// @Param       page      query  int     false  "页码"
// @Param       page_size query  int     false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiServerLoad.ServerLoadHistoryResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "服务器不存在"
// @Router      /admin/servers/load/{id}/history [GET]
func (h *ServerLoadAdminHandler) GetLoadHistory(ctx *gin.Context) {
	h.log.Info(ctx, "GetLoadHistory - 查询服务器历史负载趋势")

	id, xErr := h.parseServerLoadID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	startStr := ctx.Query("start")
	endStr := ctx.Query("end")
	if startStr == "" || endStr == "" {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "开始时间和结束时间不能为空", true, nil))
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "开始时间格式错误，请使用 RFC3339 格式", true, err))
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "结束时间格式错误，请使用 RFC3339 格式", true, err))
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "100"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}

	resp, xErr := h.service.serverLoadLogic.GetHistoryByServerID(ctx.Request.Context(), id, start, end, page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", resp)
}

func (h *ServerLoadAdminHandler) parseServerLoadID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的服务器 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}
