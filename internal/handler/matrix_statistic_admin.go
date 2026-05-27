package handler

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiMatrixStatistic "github.com/frontleaves-mc/frontleaves-plugin/api/matrix_statistic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatrixStatisticAdminHandler handler

func NewMatrixStatisticAdminHandler(ctx context.Context) *MatrixStatisticAdminHandler {
	return NewHandler[MatrixStatisticAdminHandler](ctx, "MatrixStatisticAdminHandler")
}

// GetByUUID 查询玩家统计数据
//
// @Summary     [管理] 查询玩家统计数据
// @Description 根据玩家 UUID 查询统计数据
// @Tags        管理-Matrix统计接口
// @Accept      json
// @Produce     json
// @Param       uuid  path  string  true  "玩家UUID"
// @Success     200  {object}  xBase.BaseResponse{data=apiMatrixStatistic.StatisticResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "统计数据不存在"
// @Router      /admin/matrix/statistics/:uuid [GET]
func (h *MatrixStatisticAdminHandler) GetByUUID(ctx *gin.Context) {
	h.log.Info(ctx, "GetByUUID - 查询玩家统计数据")

	uuidStr := ctx.Param("uuid")
	playerUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	stat, xErr := h.service.matrixStatisticQueryLogic.GetByUUID(ctx.Request.Context(), playerUUID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", apiMatrixStatistic.StatisticResponse{
		ID:                  stat.ID.String(),
		PlayerUUID:          stat.PlayerUUID.String(),
		PlayerName:          stat.PlayerName,
		BlocksBreak:         stat.BlocksBreak,
		BlocksPlace:         stat.BlocksPlace,
		EntitiesKill:        stat.EntitiesKill,
		Deaths:              stat.Deaths,
		ItemsUsed:           stat.ItemsUsed,
		TotalBlocksBroken:   stat.TotalBlocksBroken,
		TotalBlocksPlaced:   stat.TotalBlocksPlaced,
		TotalEntitiesKilled: stat.TotalEntitiesKilled,
		TotalDeaths:         stat.TotalDeaths,
		TotalPlayTimeMs:     stat.TotalPlayTimeMs,
		CurrentSessionStart: stat.CurrentSessionStart,
		TotalSessions:       stat.TotalSessions,
	})
}