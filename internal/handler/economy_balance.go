package handler

import (
	"context"
	"math"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiEconomy "github.com/frontleaves-mc/frontleaves-plugin/api/economy"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	economypb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/economy/v1"
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

// AdjustPlayerBalance 管理员调整指定玩家的余额
//
//	@Summary		[管理] 调整玩家余额
//	@Description	管理员调整指定玩家的经济余额（增加/扣减/设置/重置）
//	@Tags			管理-经济接口
//	@Accept			json
//	@Produce		json
//	@Param			body	body	apiEconomy.AdjustBalanceRequest	true	"调整玩家余额请求"
//	@Success		200	{object}	xBase.BaseResponse{data=apiEconomy.AdjustBalanceResponse}	"成功"
//	@Failure		400	{object}	xBase.BaseResponse	"请求参数错误"
//	@Failure		401	{object}	xBase.BaseResponse	"未授权"
//	@Failure		403	{object}	xBase.BaseResponse	"无权限"
//	@Failure		503	{object}	xBase.BaseResponse	"服务不可用（MC 插件未连接或查询超时）"
//	@Router			/admin/economy/balance/adjust [POST]
func (h *EconomyBalanceHandler) AdjustPlayerBalance(ctx *gin.Context) {
	h.log.Info(ctx, "AdjustPlayerBalance - 管理员调整玩家余额")

	var req apiEconomy.AdjustBalanceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	if req.PlayerUUID == "" {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "player_uuid 不能为空", true, nil))
		return
	}
	playerUUID, err := uuid.Parse(req.PlayerUUID)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的玩家 UUID", true, err))
		return
	}

	validOperations := map[string]bool{
		"add":    true,
		"remove": true,
		"set":    true,
		"reset":  true,
	}
	if !validOperations[req.Operation] {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的操作类型，可选值：add, remove, set, reset", true, nil))
		return
	}

	var amountFen int64
	if req.Operation != "reset" {
		if req.Amount <= 0 {
			_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "操作金额必须大于 0", true, nil))
			return
		}
		amountFen = int64(math.Round(req.Amount * 100))
	}

	var protoOp economypb.BalanceOperation
	switch req.Operation {
	case "add":
		protoOp = economypb.BalanceOperation_BALANCE_OPERATION_ADD
	case "remove":
		protoOp = economypb.BalanceOperation_BALANCE_OPERATION_REMOVE
	case "set":
		protoOp = economypb.BalanceOperation_BALANCE_OPERATION_SET
	case "reset":
		protoOp = economypb.BalanceOperation_BALANCE_OPERATION_RESET
	}

	newBalance, xErr := grpcHandler.AdjustBalance(ctx.Request.Context(), playerUUID.String(), protoOp, amountFen)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "调整余额成功", &apiEconomy.AdjustBalanceResponse{
		PlayerUUID:        playerUUID.String(),
		NewBalance:        newBalance,
		NewBalanceDisplay: formatAmount(newBalance),
		Currency:          "CNY",
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
