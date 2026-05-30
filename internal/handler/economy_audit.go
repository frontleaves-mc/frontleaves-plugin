package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiEconomy "github.com/frontleaves-mc/frontleaves-plugin/api/economy"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EconomyAuditHandler 经济审计 HTTP 适配层，负责管理员操作日志的查询与筛选。
type EconomyAuditHandler handler

// NewEconomyAuditHandler 创建经济审计 Handler 实例。
func NewEconomyAuditHandler(ctx context.Context) *EconomyAuditHandler {
	return NewHandler[EconomyAuditHandler](ctx, "EconomyAuditHandler")
}

// ListAdminAuditLogs 查询管理员操作审计日志
//
// @Summary     [管理] 查询管理员操作审计日志
// @Description 管理员分页查询所有管理员操作记录，支持按操作员或目标玩家筛选
// @Tags        管理-经济接口
// @Accept      json
// @Produce     json
// @Param       operator_uuid  query  string  false  "操作员UUID"
// @Param       player_uuid    query  string  false  "目标玩家UUID"
// @Param       page           query  int     false  "页码"
// @Param       page_size      query  int     false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiEconomy.TransactionListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse                                     "请求参数错误"
// @Failure     401  {object}  xBase.BaseResponse                                     "未授权"
// @Failure     403  {object}  xBase.BaseResponse                                     "无权限"
// @Router      /admin/economy/audit-logs [GET]
func (h *EconomyAuditHandler) ListAdminAuditLogs(ctx *gin.Context) {
	h.log.Info(ctx, "ListAdminAuditLogs - 查询管理员操作审计日志")

	// 可选筛选参数
	operatorUUIDStr := ctx.Query("operator_uuid")
	playerUUIDStr := ctx.Query("player_uuid")

	var operatorUUID uuid.UUID
	var playerUUID uuid.UUID
	var filterByOperator bool
	var filterByPlayer bool

	if operatorUUIDStr != "" {
		parsed, err := uuid.Parse(operatorUUIDStr)
		if err != nil {
			_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的 operator_uuid", true, err))
			return
		}
		operatorUUID = parsed
		filterByOperator = true
	}

	if playerUUIDStr != "" {
		parsed, err := uuid.Parse(playerUUIDStr)
		if err != nil {
			_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的 player_uuid", true, err))
			return
		}
		playerUUID = parsed
		filterByPlayer = true
	}

	page, pageSize := h.parsePagination(ctx)

	list, total, xErr := h.service.transactionLogLogic.GetAdminAuditLogs(
		ctx.Request.Context(), page, pageSize,
	)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	// 内存筛选：按可选的操作员/玩家 UUID 过滤
	if filterByOperator || filterByPlayer {
		filtered := make([]*entity.TransactionLog, 0, len(list))
		for _, log := range list {
			if filterByOperator && (log.OperatorUUID == nil || *log.OperatorUUID != operatorUUID) {
				continue
			}
			if filterByPlayer && log.PlayerUUID != playerUUID {
				continue
			}
			filtered = append(filtered, log)
		}
		list = filtered
		total = int64(len(filtered))
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiEconomy.TransactionListResponse{
		List:     h.entitiesToDTOs(list),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// parsePagination 解析并规范化分页参数，默认 page=1, pageSize=20, max 100
func (h *EconomyAuditHandler) parsePagination(ctx *gin.Context) (int, int) {
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
func (h *EconomyAuditHandler) entitiesToDTOs(logs []*entity.TransactionLog) []apiEconomy.TransactionDTO {
	dtos := make([]apiEconomy.TransactionDTO, 0, len(logs))
	for _, e := range logs {
		dtos = append(dtos, h.entityToDTO(e))
	}
	return dtos
}

// entityToDTO 将单条交易流水实体映射为 DTO
func (h *EconomyAuditHandler) entityToDTO(e *entity.TransactionLog) apiEconomy.TransactionDTO {
	return apiEconomy.TransactionDTO{
		ID:            int64(e.ID),
		PlayerUUID:    e.PlayerUUID.String(),
		PlayerName:    e.PlayerName,
		Amount:        e.Amount,
		AmountDisplay: h.formatAmount(e.Amount),
		Type:          e.Type,
		TypeName:      h.mapTransactionType(e.Type),
		Counterparty:  e.CounterpartyName,
		Operator:      e.OperatorName,
		Comment:       e.Comment,
		CreatedAt:     e.CreatedAt.Format(time.RFC3339),
	}
}

// formatAmount 将分单位的金额格式化为 "X.XX" 显示字符串
func (h *EconomyAuditHandler) formatAmount(fen int64) string {
	return fmt.Sprintf("%.2f", float64(fen)/100.0)
}

// mapTransactionType 将交易类型编码映射为中文名称
func (h *EconomyAuditHandler) mapTransactionType(t int16) string {
	switch t {
	case entity.TransactionTypeTransfer:
		return "转账"
	case entity.TransactionTypeAdmin:
		return "管理员操作"
	default:
		return "未知"
	}
}
