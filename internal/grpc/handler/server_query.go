package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	statuspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/status/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// queryStream 管理一个来自 Java 插件的双向流连接
type queryStream struct {
	stream  grpc.BidiStreamingServer[statuspb.ServerQueryRequest, statuspb.ServerQueryResponse]
	pending sync.Map // request_id -> chan *statuspb.ServerQueryRequest
	log     *xLog.LogNamedLogger
}

// queryStreamManager 管理活跃的 ServerQuery 双向流
// 仅保留最新的一条流连接（单 Java 插件场景）
var queryStreamManager struct {
	mu     sync.RWMutex
	stream *queryStream
}

// ServerQuery 实现 ServerQuery 双向流式 RPC handler
// Java 插件作为客户端连接到此流，Go 通过该流向 Java 发送查询并接收响应
func (h *ServerStatusHandler) ServerQuery(
	stream grpc.BidiStreamingServer[statuspb.ServerQueryRequest, statuspb.ServerQueryResponse],
) error {
	h.log.Info(stream.Context(), "ServerQuery - 新的双向流连接")

	qs := &queryStream{
		stream: stream,
		log:    h.log,
	}

	// 注册为活跃流
	h.setQueryStream(qs)
	defer h.clearQueryStream()

	// 循环读取 Java 端发来的响应
	for {
		resp, err := stream.Recv()
		if err != nil {
			if status.Code(err) == codes.Canceled || status.Code(err) == codes.OK {
				h.log.Info(stream.Context(), "ServerQuery - 流正常关闭")
				return nil
			}
			h.log.Warn(stream.Context(), "ServerQuery - 流读取错误: "+err.Error())
			return err
		}

		// 将响应分发给等待的调用者
		if ch, ok := qs.pending.LoadAndDelete(resp.GetRequestId()); ok {
			ch.(chan *statuspb.ServerQueryRequest) <- resp
		}
	}
}

// setQueryStream 注册活跃的查询流
func (h *ServerStatusHandler) setQueryStream(qs *queryStream) {
	queryStreamManager.mu.Lock()
	defer queryStreamManager.mu.Unlock()
	queryStreamManager.stream = qs
	h.log.Info(context.Background(), "ServerQuery - 活跃流已设置")
}

// getQueryStream 获取当前活跃的查询流
func (h *ServerStatusHandler) getQueryStream() *queryStream {
	queryStreamManager.mu.RLock()
	defer queryStreamManager.mu.RUnlock()
	return queryStreamManager.stream
}

// clearQueryStream 清除活跃的查询流
func (h *ServerStatusHandler) clearQueryStream() {
	queryStreamManager.mu.Lock()
	defer queryStreamManager.mu.Unlock()
	queryStreamManager.stream = nil
	h.log.Info(context.Background(), "ServerQuery - 活跃流已清除")
}

// generateRequestID 生成唯一的请求 ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// QueryPlayerStatus 通过 ServerQuery 查询玩家在线状态
func (h *ServerStatusHandler) QueryPlayerStatus(ctx context.Context, playerUUID string) (
	online bool, serverName, worldName, playerName string, lastSeen int64, err error,
) {
	qs := h.getQueryStream()
	if qs == nil {
		return false, "", "", "", 0, fmt.Errorf("无可用的 ServerQuery 连接")
	}

	reqID := generateRequestID()
	respCh := make(chan *statuspb.ServerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	// Go 通过 Send 向 Java 发送查询（使用 ServerQueryResponse 携带查询参数）
	if err := qs.stream.Send(&statuspb.ServerQueryResponse{
		RequestId: reqID,
		Event:     statuspb.QueryEvent_QUERY_EVENT_GET_PLAYER_STATUS,
	}); err != nil {
		return false, "", "", "", 0, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		// ServerQuery Go→Java 方向受 proto 限制: ServerQueryRequest 仅携带查询参数字段
		// (event, request_id, player_uuid, server_name, permission_node)，不携带结果字段
		_ = resp
		return false, "", "", "", 0, fmt.Errorf("ServerQuery Go→Java 方向受 proto 限制: ServerQueryRequest 不携带结果字段，需 proto 层面修改")
	case <-time.After(10 * time.Second):
		return false, "", "", "", 0, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return false, "", "", "", 0, ctx.Err()
	}
}

// QueryServerStatus 通过 ServerQuery 查询服务器状态
func (h *ServerStatusHandler) QueryServerStatus(ctx context.Context, serverName string) (
	players []*statuspb.PlayerStatus, onlinePlayers int32, tps float64, lastHeartbeat int64, err error,
) {
	qs := h.getQueryStream()
	if qs == nil {
		return nil, 0, 0, 0, fmt.Errorf("无可用的 ServerQuery 连接")
	}

	reqID := generateRequestID()
	respCh := make(chan *statuspb.ServerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	if err := qs.stream.Send(&statuspb.ServerQueryResponse{
		RequestId: reqID,
		Event:     statuspb.QueryEvent_QUERY_EVENT_GET_SERVER_STATUS,
	}); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		_ = resp
		return nil, 0, 0, 0, fmt.Errorf("ServerQuery Go→Java 方向受 proto 限制: ServerQueryRequest 不携带结果字段，需 proto 层面修改")
	case <-time.After(10 * time.Second):
		return nil, 0, 0, 0, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return nil, 0, 0, 0, ctx.Err()
	}
}

// QueryCheckPermission 通过 ServerQuery 检查玩家权限
func (h *ServerStatusHandler) QueryCheckPermission(ctx context.Context, playerUUID, permissionNode string) (bool, error) {
	qs := h.getQueryStream()
	if qs == nil {
		return false, fmt.Errorf("无可用的 ServerQuery 连接")
	}

	reqID := generateRequestID()
	respCh := make(chan *statuspb.ServerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	if err := qs.stream.Send(&statuspb.ServerQueryResponse{
		RequestId: reqID,
		Event:     statuspb.QueryEvent_QUERY_EVENT_CHECK_PERMISSION,
	}); err != nil {
		return false, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		_ = resp
		return false, fmt.Errorf("ServerQuery Go→Java 方向受 proto 限制: ServerQueryRequest 不携带结果字段，需 proto 层面修改")
	case <-time.After(10 * time.Second):
		return false, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// QueryPlayerGroups 通过 ServerQuery 获取玩家权限组
func (h *ServerStatusHandler) QueryPlayerGroups(ctx context.Context, playerUUID string) (
	primaryGroup string, groups []string, err error,
) {
	qs := h.getQueryStream()
	if qs == nil {
		return "", nil, fmt.Errorf("无可用的 ServerQuery 连接")
	}

	reqID := generateRequestID()
	respCh := make(chan *statuspb.ServerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	if err := qs.stream.Send(&statuspb.ServerQueryResponse{
		RequestId: reqID,
		Event:     statuspb.QueryEvent_QUERY_EVENT_GET_PLAYER_GROUPS,
	}); err != nil {
		return "", nil, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		_ = resp
		return "", nil, fmt.Errorf("ServerQuery Go→Java 方向受 proto 限制: ServerQueryRequest 不携带结果字段，需 proto 层面修改")
	case <-time.After(10 * time.Second):
		return "", nil, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return "", nil, ctx.Err()
	}
}
