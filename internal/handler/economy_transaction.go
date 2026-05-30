package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiEconomy "github.com/frontleaves-mc/frontleaves-plugin/api/economy"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EconomyTransactionHandler 经济交易流水 HTTP 适配层，负责参数校验与响应映射。
type EconomyTransactionHandler handler

// NewEconomyTransactionHandler 创建经济交易流水 Handler 实例。
func NewEconomyTransactionHandler(ctx context.Context) *EconomyTransactionHandler {
	return NewHandler[EconomyTransactionHandler](ctx, "EconomyTransactionHandler")
}

// ListMyTransactions 查询当前玩家的交易流水
//
// @Summary     [玩家] 查询我的交易流水
// @Description 根据登录用户的 Token 解析出游戏角色 UUID，分页查询该角色的交易流水记录
// @Tags        玩家-经济接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiEconomy.TransactionListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                     "请求参数错误"
// @Failure     401  {object}  xBase.BaseResponse                                     "未授权"
// @Failure     404  {object}  xBase.BaseResponse                                     "未找到关联的游戏角色"
// @Router      /user/economy/transactions [GET]
func (h *EconomyTransactionHandler) ListMyTransactions(ctx *gin.Context) {
	h.log.Info(ctx, "ListMyTransactions - 查询我的交易流水")

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

	page, pageSize := h.parsePagination(ctx)

	list, total, xErr := h.service.transactionLogLogic.GetPlayerTransactions(
		ctx.Request.Context(), playerUUID, page, pageSize,
	)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiEconomy.TransactionListResponse{
		List:     h.entitiesToDTOs(list),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListPlayerTransactions 管理端查询指定玩家的交易流水
//
// @Summary     [管理] 查询玩家交易流水
// @Description 管理员根据玩家 UUID 分页查询指定玩家的交易流水记录
// @Tags        管理-经济接口
// @Accept      json
// @Produce     json
// @Param       player_uuid  query  string  true   "玩家UUID"
// @Param       page         query  int     false  "页码"
// @Param       page_size    query  int     false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiEconomy.TransactionListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                     "请求参数错误"
// @Failure     401  {object}  xBase.BaseResponse                                     "未授权"
// @Failure     403  {object}  xBase.BaseResponse                                     "无权限"
// @Router      /admin/economy/transactions [GET]
func (h *EconomyTransactionHandler) ListPlayerTransactions(ctx *gin.Context) {
	h.log.Info(ctx, "ListPlayerTransactions - 管理端查询玩家交易流水")

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

	page, pageSize := h.parsePagination(ctx)

	list, total, xErr := h.service.transactionLogLogic.GetPlayerTransactions(
		ctx.Request.Context(), playerUUID, page, pageSize,
	)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiEconomy.TransactionListResponse{
		List:     h.entitiesToDTOs(list),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// resolvePrimaryPlayerUUID 通过用户 ID 查询关联的主游戏角色 UUID
func (h *EconomyTransactionHandler) resolvePrimaryPlayerUUID(ctx *gin.Context, userID xSnowflake.SnowflakeID) (uuid.UUID, *xError.Error) {
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

// parsePagination 解析并规范化分页参数，默认 page=1, pageSize=20, max 100
func (h *EconomyTransactionHandler) parsePagination(ctx *gin.Context) (int, int) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// entitiesToDTOs 将实体列表转换为 DTO 列表
func (h *EconomyTransactionHandler) entitiesToDTOs(logs []*entity.TransactionLog) []apiEconomy.TransactionDTO {
	dtos := make([]apiEconomy.TransactionDTO, 0, len(logs))
	for _, e := range logs {
		dtos = append(dtos, h.entityToDTO(e))
	}
	return dtos
}

// entityToDTO 将单条交易流水实体映射为 DTO
func (h *EconomyTransactionHandler) entityToDTO(e *entity.TransactionLog) apiEconomy.TransactionDTO {
	return apiEconomy.TransactionDTO{
		ID:            int64(e.ID),
		PlayerUUID:    e.PlayerUUID.String(),
		PlayerName:    e.PlayerName,
		Amount:        e.Amount,
		AmountDisplay: formatAmount(e.Amount),
		Type:          e.Type,
		TypeName:      mapTransactionType(e.Type),
		Counterparty:  e.CounterpartyName,
		Operator:      e.OperatorName,
		Comment:       e.Comment,
		CreatedAt:     e.CreatedAt.Format(time.RFC3339),
	}
}

// formatAmount 将分单位的金额格式化为 "X.XX" 显示字符串
func formatAmount(fen int64) string {
	return fmt.Sprintf("%.2f", float64(fen)/100.0)
}

// mapTransactionType 将交易类型编码映射为中文名称
func mapTransactionType(t int16) string {
	switch t {
	case entity.TransactionTypeTransfer:
		return "转账"
	case entity.TransactionTypeAdmin:
		return "管理员操作"
	default:
		return "未知"
	}
}
