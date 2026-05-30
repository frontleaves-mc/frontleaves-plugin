package handler

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiEconomy "github.com/frontleaves-mc/frontleaves-plugin/api/economy"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	grpcHandler "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EconomyBalanceHandler 经济余额查询 HTTP 适配层，负责参数校验与响应映射。
type EconomyBalanceHandler handler

// NewEconomyBalanceHandler 创建经济余额查询 Handler 实例。
func NewEconomyBalanceHandler(ctx context.Context) *EconomyBalanceHandler {
	return NewHandler[EconomyBalanceHandler](ctx, "EconomyBalanceHandler")
}

// GetMyBalance 查询当前玩家的余额
//
//	@Summary		[玩家] 查询我的余额
//	@Description	根据登录用户的 Token 解析出游戏角色 UUID，查询该角色的经济余额
//	@Tags			玩家-经济接口
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	xBase.BaseResponse{data=apiEconomy.BalanceDTO}	"成功"
//	@Failure		400	{object}	xBase.BaseResponse								"请求参数错误"
//	@Failure		401	{object}	xBase.BaseResponse								"未授权"
//	@Failure		404	{object}	xBase.BaseResponse								"未找到关联的游戏角色"
//	@Router			/user/economy/balance [GET]
func (h *EconomyBalanceHandler) GetMyBalance(ctx *gin.Context) {
	h.log.Info(ctx, "GetMyBalance - 查询我的余额")

	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未获取到用户信息", false, nil))
		return
	}

	userID, err := xSnowflake.ParseSnowflakeID(userInfo.UserID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的用户 ID", true, err))
		return
	}

	playerUUID, xErr := h.resolvePrimaryPlayerUUID(ctx, userID)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	balance, xErr := grpcHandler.QueryBalance(ctx.Request.Context(), playerUUID.String())
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiEconomy.BalanceDTO{
		PlayerUUID:    playerUUID.String(),
		Balance:       balance,
		BalanceDisplay: formatAmount(balance),
		Currency:      "CNY",
	})
}

// GetPlayerBalance 管理端查询指定玩家的余额
//
//	@Summary		[管理] 查询玩家余额
//	@Description	管理员根据玩家 UUID 查询指定玩家的经济余额
//	@Tags			管理-经济接口
//	@Accept			json
//	@Produce		json
//	@Param			player_uuid	query		string	true	"玩家UUID"
//	@Success		200			{object}	xBase.BaseResponse{data=apiEconomy.BalanceDTO}	"成功"
//	@Failure		400			{object}	xBase.BaseResponse								"请求参数错误"
//	@Failure		401			{object}	xBase.BaseResponse								"未授权"
//	@Failure		403			{object}	xBase.BaseResponse								"无权限"
//	@Router			/admin/economy/balance [GET]
func (h *EconomyBalanceHandler) GetPlayerBalance(ctx *gin.Context) {
	h.log.Info(ctx, "GetPlayerBalance - 管理端查询玩家余额")

	playerUUIDStr := ctx.Query("player_uuid")
	if playerUUIDStr == "" {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "player_uuid 参数不能为空", true, nil))
		return
	}

	playerUUID, err := uuid.Parse(playerUUIDStr)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	balance, xErr := grpcHandler.QueryBalance(ctx.Request.Context(), playerUUID.String())
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiEconomy.BalanceDTO{
		PlayerUUID:    playerUUID.String(),
		Balance:       balance,
		BalanceDisplay: formatAmount(balance),
		Currency:      "CNY",
	})
}

// resolvePrimaryPlayerUUID 通过用户 ID 查询关联的主游戏角色 UUID
func (h *EconomyBalanceHandler) resolvePrimaryPlayerUUID(ctx *gin.Context, userID xSnowflake.SnowflakeID) (uuid.UUID, *xError.Error) {
	profiles, xErr := h.service.gameProfileLogic.ListByUserID(ctx.Request.Context(), userID)
	if xErr != nil {
		return uuid.Nil, xErr
	}
	if len(profiles) == 0 {
		return uuid.Nil, xError.NewError(nil, xError.NotFound, "未找到关联的游戏角色", false, nil)
	}

	parsed, err := uuid.Parse(profiles[0].UUID)
	if err != nil {
		return uuid.Nil, xError.NewError(nil, xError.ParameterError, "游戏角色 UUID 格式异常", true, err)
	}
	return parsed, nil
}
