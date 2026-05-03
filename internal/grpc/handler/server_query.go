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

type queryStream struct {
	stream  grpc.BidiStreamingServer[statuspb.ServerQueryRequest, statuspb.ServerQueryResponse]
	pending sync.Map
	log     *xLog.LogNamedLogger
}

var queryStreamManager struct {
	mu     sync.RWMutex
	stream *queryStream
}

func (h *ServerStatusHandler) ServerQuery(
	stream grpc.BidiStreamingServer[statuspb.ServerQueryRequest, statuspb.ServerQueryResponse],
) error {
	h.log.Info(stream.Context(), "ServerQuery - 新的双向流连接")

	qs := &queryStream{
		stream: stream,
		log:    h.log,
	}

	h.setQueryStream(qs)
	defer h.clearQueryStream()

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

		if ch, ok := qs.pending.LoadAndDelete(resp.GetRequestId()); ok {
			ch.(chan *statuspb.ServerQueryRequest) <- resp
		}
	}
}

func (h *ServerStatusHandler) setQueryStream(qs *queryStream) {
	queryStreamManager.mu.Lock()
	defer queryStreamManager.mu.Unlock()
	queryStreamManager.stream = qs
	h.log.Info(context.Background(), "ServerQuery - 活跃流已设置")
}

func (h *ServerStatusHandler) getQueryStream() *queryStream {
	queryStreamManager.mu.RLock()
	defer queryStreamManager.mu.RUnlock()
	return queryStreamManager.stream
}

func (h *ServerStatusHandler) clearQueryStream() {
	queryStreamManager.mu.Lock()
	defer queryStreamManager.mu.Unlock()
	queryStreamManager.stream = nil
	h.log.Info(context.Background(), "ServerQuery - 活跃流已清除")
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

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

	if err := qs.stream.Send(&statuspb.ServerQueryResponse{
		RequestId: reqID,
		Query: &statuspb.ServerQueryResponse_PlayerStatusQuery{
			PlayerStatusQuery: &statuspb.QueryPlayerStatusQuery{
				PlayerUuid: playerUUID,
			},
		},
	}); err != nil {
		return false, "", "", "", 0, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp.Result == nil {
			return false, "", "", "", 0, fmt.Errorf("收到空响应")
		}
		switch result := resp.Result.(type) {
		case *statuspb.ServerQueryRequest_PlayerStatusResult:
			r := result.PlayerStatusResult
			return r.GetOnline(), r.GetServerName(), r.GetWorldName(), r.GetPlayerName(), r.GetLastSeen(), nil
		default:
			return false, "", "", "", 0, fmt.Errorf("收到非预期的响应类型")
		}
	case <-time.After(10 * time.Second):
		return false, "", "", "", 0, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return false, "", "", "", 0, ctx.Err()
	}
}

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
		Query: &statuspb.ServerQueryResponse_ServerStatusQuery{
			ServerStatusQuery: &statuspb.QueryServerStatusQuery{
				ServerName: serverName,
			},
		},
	}); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp.Result == nil {
			return nil, 0, 0, 0, fmt.Errorf("收到空响应")
		}
		switch result := resp.Result.(type) {
		case *statuspb.ServerQueryRequest_ServerStatusResult:
			r := result.ServerStatusResult
			return r.GetPlayers(), r.GetOnlinePlayers(), r.GetTps(), r.GetLastHeartbeat(), nil
		default:
			return nil, 0, 0, 0, fmt.Errorf("收到非预期的响应类型")
		}
	case <-time.After(10 * time.Second):
		return nil, 0, 0, 0, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return nil, 0, 0, 0, ctx.Err()
	}
}

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
		Query: &statuspb.ServerQueryResponse_CheckPermissionQuery{
			CheckPermissionQuery: &statuspb.QueryCheckPermissionQuery{
				PlayerUuid:    playerUUID,
				PermissionNode: permissionNode,
			},
		},
	}); err != nil {
		return false, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp.Result == nil {
			return false, fmt.Errorf("收到空响应")
		}
		switch result := resp.Result.(type) {
		case *statuspb.ServerQueryRequest_CheckPermissionResult:
			return result.CheckPermissionResult.GetHasPermission(), nil
		default:
			return false, fmt.Errorf("收到非预期的响应类型")
		}
	case <-time.After(10 * time.Second):
		return false, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

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
		Query: &statuspb.ServerQueryResponse_PlayerGroupsQuery{
			PlayerGroupsQuery: &statuspb.QueryPlayerGroupsQuery{
				PlayerUuid: playerUUID,
			},
		},
	}); err != nil {
		return "", nil, fmt.Errorf("发送查询失败: %w", err)
	}

	select {
	case resp := <-respCh:
		if resp.Result == nil {
			return "", nil, fmt.Errorf("收到空响应")
		}
		switch result := resp.Result.(type) {
		case *statuspb.ServerQueryRequest_PlayerGroupsResult:
			r := result.PlayerGroupsResult
			return r.GetPrimaryGroup(), r.GetGroups(), nil
		default:
			return "", nil, fmt.Errorf("收到非预期的响应类型")
		}
	case <-time.After(10 * time.Second):
		return "", nil, fmt.Errorf("查询超时")
	case <-ctx.Done():
		return "", nil, ctx.Err()
	}
}
