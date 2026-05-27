package handler

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiMatrixStatistic "github.com/frontleaves-mc/frontleaves-plugin/api/matrix_statistic"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatrixStatisticPlayerHandler handler

func NewMatrixStatisticPlayerHandler(ctx context.Context) *MatrixStatisticPlayerHandler {
	return NewHandler[MatrixStatisticPlayerHandler](ctx, "MatrixStatisticPlayerHandler")
}

// GetMyStatistic 查询自己的统计数据
//
// @Summary     [玩家] 查询自己的统计数据
// @Description 玩家查询自己的游戏统计数据
// @Tags        玩家-Matrix统计接口
// @Accept      json
// @Produce     json
// @Success     200  {object}  xBase.BaseResponse{data=apiMatrixStatistic.StatisticResponse}  "成功"
// @Failure     401  {object}  xBase.BaseResponse  "未登录"
// @Failure     404  {object}  xBase.BaseResponse  "统计数据不存在"
// @Router      /matrix/statistics/me [GET]
func (h *MatrixStatisticPlayerHandler) GetMyStatistic(ctx *gin.Context) {
	h.log.Info(ctx, "GetMyStatistic - 查询自己的统计数据")

	// 从认证上下文获取用户信息
	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未获取到用户信息", true, nil))
		return
	}

	// 解析 UserID
	userID, err := xSnowflake.ParseSnowflakeID(userInfo.UserID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的用户ID", true, err))
		return
	}

	// 通过 GameProfile 获取关联的玩家 UUID
	profiles, xErr := h.service.gameProfileLogic.ListByUserID(ctx.Request.Context(), userID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}
	if len(profiles) == 0 {
		_ = ctx.Error(xError.NewError(nil, xError.NotFound, "未找到关联的游戏角色", false, nil))
		return
	}

	// 取第一个 GameProfile 的 UUID
	profileUUID, err := uuid.Parse(profiles[0].UUID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "解析玩家UUID失败", true, err))
		return
	}

	// 查询统计数据
	stat, xErr := h.service.matrixStatisticQueryLogic.GetByUUID(ctx.Request.Context(), profileUUID)
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