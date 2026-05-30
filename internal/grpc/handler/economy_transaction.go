package handler

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	economypb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/economy/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// balanceQueryRequest 表示一个待发送的余额查询请求。
//
// 由 QueryBalance 方法通过 queryCh 投递到 sendLoop 协程，
// resultCh 接收 MC 插件异步返回的 BalanceResult。
type balanceQueryRequest struct {
	playerUUID string
	resultCh   chan *economypb.BalanceResult
}

// EconomyTransactionHandler 处理经济交易流水的 Client-Streaming / Bidi-Streaming gRPC Handler。
//
// 接收 MC 插件通过客户端流式 RPC 上报的交易流水数据，
// 在流结束（EOF）时批量写入 PostgreSQL，由 logic 层执行逐条幂等校验。
//
// 同时维护 BalanceStream Bidi 流的多路复用，
// 支持 HTTP handler 通过 QueryBalance 发起余额查询。
type EconomyTransactionHandler struct {
	grpcHandler
	*economyService
	economypb.UnimplementedTransactionLogServiceServer

	// Bidi Stream 多路复用 — Go Server 为复用端（发送查询），MC 为回复端
	mu              sync.Mutex
	balanceStream   grpc.BidiStreamingServer[economypb.BalanceResult, economypb.BalanceQuery]
	pendingRequests sync.Map            // request_id(uint64) → chan *BalanceResult
	queryCh         chan *balanceQueryRequest
	nextRequestID   atomic.Uint64
}

// NewEconomyTransactionHandler 创建 EconomyTransactionHandler 实例并注册 gRPC 服务。
//
// 注册 TransactionLogServiceServer，附加 StreamPluginVerify 中间件进行插件身份认证。
func NewEconomyTransactionHandler(ctx context.Context, server grpc.ServiceRegistrar) *EconomyTransactionHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "EconomyTransaction")
	h := &EconomyTransactionHandler{
		grpcHandler:    *base,
		economyService: newEconomyService(ctx),
		queryCh:        make(chan *balanceQueryRequest, 100),
	}

	economypb.RegisterTransactionLogServiceServer(server, h)
	xGrpcMiddle.UseStream(economypb.TransactionLogService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

// RecordTransactionStream 实现交易流水客户端流式 RPC。
//
// 循环 Recv() 接收 RecordTransactionRequest 消息，逐条校验必填字段并映射为 entity.TransactionLog，
// 在流正常关闭（io.EOF）时调用 logic 层 RecordBatchTransactions 批量写入。
func (h *EconomyTransactionHandler) RecordTransactionStream(
	stream grpc.ClientStreamingServer[economypb.RecordTransactionRequest, economypb.RecordTransactionResponse],
) error {
	ctx := stream.Context()
	h.log.Info(ctx, "RecordTransactionStream - 新的客户端流连接")

	var entities []*entity.TransactionLog

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			if h.log != nil {
				h.log.Info(ctx, "RecordTransactionStream - 流正常关闭，即将批量写入")
			}
			break
		}
		if err != nil {
			if h.log != nil {
				h.log.Warn(ctx, "RecordTransactionStream - 流读取错误: "+err.Error())
			}
			return err
		}

		// 校验必填字段
		if req.GetPlayerUuid() == "" {
			if h.log != nil {
				h.log.Warn(ctx, "RecordTransactionStream - player_uuid 为空，跳过")
			}
			continue
		}
		if req.GetIdempotencyKey() == "" {
			if h.log != nil {
				h.log.Warn(ctx, "RecordTransactionStream - idempotency_key 为空，跳过")
			}
			continue
		}

		// 解析 PlayerUUID
		parsedUUID, err := uuid.Parse(req.GetPlayerUuid())
		if err != nil {
			if h.log != nil {
				h.log.Warn(ctx, "RecordTransactionStream - player_uuid 格式无效: "+err.Error())
			}
			continue
		}

		// Proto → Entity 字段映射
		txLog := &entity.TransactionLog{
			PlayerUUID:     parsedUUID,
			PlayerName:     req.GetPlayerName(),
			Amount:         req.GetAmount(),
			Type:           int16(req.GetType()),
			Comment:        req.GetComment(),
			IdempotencyKey: req.GetIdempotencyKey(),
		}

		// 可选字段：对方 UUID
		if counterpartyUUIDStr := req.GetCounterpartyUuid(); counterpartyUUIDStr != "" {
			if cpUUID, err := uuid.Parse(counterpartyUUIDStr); err == nil {
				txLog.CounterpartyUUID = &cpUUID
			} else if h.log != nil {
				h.log.Warn(ctx, "RecordTransactionStream - counterparty_uuid 格式无效: "+err.Error())
			}
		}
		if counterpartyName := req.GetCounterpartyName(); counterpartyName != "" {
			txLog.CounterpartyName = counterpartyName
		}

		// 可选字段：操作者 UUID
		if operatorUUIDStr := req.GetOperatorUuid(); operatorUUIDStr != "" {
			if opUUID, err := uuid.Parse(operatorUUIDStr); err == nil {
				txLog.OperatorUUID = &opUUID
			} else if h.log != nil {
				h.log.Warn(ctx, "RecordTransactionStream - operator_uuid 格式无效: "+err.Error())
			}
		}
		if operatorName := req.GetOperatorName(); operatorName != "" {
			txLog.OperatorName = operatorName
		}

		// 时间戳 → CreatedAt
		if ts := req.GetTimestamp(); ts > 0 {
			txLog.CreatedAt = time.UnixMilli(ts)
		}

		entities = append(entities, txLog)
	}

	// EOF 批量写入
	if len(entities) == 0 {
		if h.log != nil {
			h.log.Info(ctx, "RecordTransactionStream - 无待插入记录")
		}
		return stream.SendAndClose(&economypb.RecordTransactionResponse{
			Success: true,
			Message: "ok",
		})
	}

	if h.log != nil {
		h.log.Info(ctx, "RecordTransactionStream - 批量插入交易流水")
	}
	if xErr := h.transactionLogLogic.RecordBatchTransactions(ctx, entities); xErr != nil {
		if h.log != nil {
			h.log.Warn(ctx, "RecordTransactionStream - 批量插入失败: "+xErr.Error())
		}
		return status.Error(codes.Internal, "批量插入交易流水失败")
	}

	return stream.SendAndClose(&economypb.RecordTransactionResponse{
		Success: true,
		Message: "ok",
	})
}

// BalanceStream 实现余额查询双向流 RPC。
//
// MC 插件连接后建立 Bidi 流，Go Server 侧通过流发送 BalanceQuery 查询余额，
// MC 查 Vault 后返回 BalanceResult。内部使用 sendLoop/recvLoop 双协程模式
// 实现多路复用：HTTP handler 可通过 QueryBalance 并发发起查询。
func (h *EconomyTransactionHandler) BalanceStream(stream grpc.BidiStreamingServer[economypb.BalanceResult, economypb.BalanceQuery]) error {
	ctx := stream.Context()
	h.log.Info(ctx, "BalanceStream - MC 余额查询双向流已连接")

	h.mu.Lock()
	h.balanceStream = stream
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.balanceStream = nil
		h.mu.Unlock()
		h.failAllPending(ctx)
	}()

	go h.balanceSendLoop(ctx, stream)

	for {
		result, err := stream.Recv()
		if err == io.EOF {
			h.log.Info(ctx, "BalanceStream - 流正常关闭")
			return nil
		}
		if err != nil {
			h.log.Warn(ctx, "BalanceStream - 流读取错误: "+err.Error())
			return err
		}
		if ch, ok := h.pendingRequests.Load(result.GetRequestId()); ok {
			select {
			case ch.(chan *economypb.BalanceResult) <- result:
			default:
			}
			h.pendingRequests.Delete(result.GetRequestId())
		}
	}
}

func (h *EconomyTransactionHandler) balanceSendLoop(ctx context.Context, stream grpc.BidiStreamingServer[economypb.BalanceResult, economypb.BalanceQuery]) {
	h.log.Info(ctx, "BalanceStream - sendLoop 已启动")
	for {
		select {
		case req, ok := <-h.queryCh:
			if !ok {
				return
			}
			rid := h.nextRequestID.Add(1)
			query := &economypb.BalanceQuery{
				RequestId:  rid,
				PlayerUuid: req.playerUUID,
			}
			h.pendingRequests.Store(rid, req.resultCh)
			if err := stream.Send(query); err != nil {
				h.log.Warn(ctx, "BalanceStream - 发送查询失败: "+err.Error())
				h.pendingRequests.Delete(rid)
				select {
				case req.resultCh <- &economypb.BalanceResult{
					Success:      false,
					ErrorMessage: err.Error(),
				}:
				default:
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *EconomyTransactionHandler) failAllPending(ctx context.Context) {
	var channels []chan *economypb.BalanceResult
	h.pendingRequests.Range(func(key, value any) bool {
		h.pendingRequests.Delete(key)
		channels = append(channels, value.(chan *economypb.BalanceResult))
		return true
	})
	for _, ch := range channels {
		select {
		case ch <- &economypb.BalanceResult{
			Success:      false,
			ErrorMessage: "余额查询流已断开",
		}:
		default:
		}
	}
	h.log.Info(ctx, "BalanceStream - 已清理所有待处理请求")
}

// QueryBalance 通过 Bidi 流向 MC 插件查询指定玩家余额（5s 超时）。
//
// 由 HTTP handler 或 logic 层调用。
func (h *EconomyTransactionHandler) QueryBalance(ctx context.Context, playerUUID string) (int64, *xError.Error) {
	h.mu.Lock()
	stream := h.balanceStream
	h.mu.Unlock()
	if stream == nil {
		return 0, xError.NewError(nil, xError.ServiceUnavailable, "余额查询流未建立，请确保 MC 插件已连接", true, nil)
	}

	resultCh := make(chan *economypb.BalanceResult, 1)
	h.queryCh <- &balanceQueryRequest{
		playerUUID: playerUUID,
		resultCh:   resultCh,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case result, ok := <-resultCh:
		if !ok || result == nil {
			return 0, xError.NewError(nil, xError.ServiceUnavailable, "余额查询流已断开", false, nil)
		}
		if !result.GetSuccess() {
			return 0, xError.NewError(nil, xError.ServiceUnavailable, xError.ErrMessage(result.GetErrorMessage()), true, nil)
		}
		return result.GetBalance(), nil
	case <-ctx.Done():
		return 0, xError.NewError(nil, xError.ServiceUnavailable, "查询余额超时", false, ctx.Err())
	}
}

// ---- 全局单例（供 HTTP handler / logic 层跨层调用） ----

var globalEconomyHandler *EconomyTransactionHandler

// SetGlobalEconomyHandler 注册全局经济系统 Handler。
//
// 仅在 grpc/register 层启动时调用一次。
func SetGlobalEconomyHandler(h *EconomyTransactionHandler) {
	globalEconomyHandler = h
}

// QueryBalance 全局余额查询入口。
//
// HTTP handler 通过此函数跨层访问 gRPC 层的 Bidi Stream 余额查询能力。
// 返回玩家余额（单位：分 fen）或错误。
func QueryBalance(ctx context.Context, playerUUID string) (int64, *xError.Error) {
	if globalEconomyHandler == nil {
		return 0, xError.NewError(nil, xError.ServiceUnavailable, "经济系统未初始化", false, nil)
	}
	return globalEconomyHandler.QueryBalance(ctx, playerUUID)
}
