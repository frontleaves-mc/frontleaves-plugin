package handler

import (
	"context"
	"fmt"
	"time"

	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/utility"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	essentialspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/essentials/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EssentialsPlayerQueryHandler struct {
	grpcHandler
	*essentialsService
	essentialspb.UnimplementedPlayerQueryServiceServer
}

func NewEssentialsPlayerQueryHandler(ctx context.Context, server grpc.ServiceRegistrar) *EssentialsPlayerQueryHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "EssentialsPlayerQueryHandler")
	h := &EssentialsPlayerQueryHandler{
		grpcHandler:       *base,
		essentialsService: newEssentialsService(ctx),
	}

	essentialspb.RegisterPlayerQueryServiceServer(server, h)
	xGrpcMiddle.UseStream(essentialspb.PlayerQueryService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

// PlayerQuery 实现双向流查询：接收 MC 插件返回的查询结果并路由到 pending channel
func (h *EssentialsPlayerQueryHandler) PlayerQuery(
	stream grpc.BidiStreamingServer[essentialspb.PlayerQueryRequest, essentialspb.PlayerQueryResponse],
) error {
	ctx := stream.Context()
	if h.log != nil { h.log.Info(ctx, "PlayerQuery - 新的双向流连接") }

	pluginName, err := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
	if err != nil {
		if h.log != nil { h.log.Warn(ctx, "PlayerQuery - 无法获取 plugin-name") }
		return status.Error(codes.Unauthenticated, "缺少 plugin-name")
	}
	if h.log != nil { h.log.Info(ctx, "PlayerQuery - 插件连接: "+pluginName) }

	qs := &essentialsQueryStream{
		stream:     stream,
		serverName: pluginName,
		log:        h.log,
	}

	registerEssentialsQueryStream(pluginName, qs)
	defer removeEssentialsQueryStream(pluginName)

	for {
		resp, err := stream.Recv()
		if err != nil {
			if status.Code(err) == codes.Canceled || status.Code(err) == codes.OK {
				if h.log != nil { h.log.Info(ctx, "PlayerQuery - 流正常关闭") }
				return nil
			}
			if h.log != nil { h.log.Warn(ctx, "PlayerQuery - 流读取错误: "+err.Error()) }
			return err
		}

		if ch, ok := qs.pending.LoadAndDelete(resp.GetRequestId()); ok {
			ch.(chan *essentialspb.PlayerQueryRequest) <- resp
		}
	}
}

// QueryPlayerStatus 查询玩家在线状态
func (h *EssentialsPlayerQueryHandler) QueryPlayerStatus(ctx context.Context, playerUUID string) (
	online bool, serverName, worldName, playerName string, lastSeen int64, err error,
) {
	qs := getAnyEssentialsQueryStream()
	if qs == nil {
		return false, "", "", "", 0, fmt.Errorf("无可用的 PlayerQuery 连接")
	}

	reqID := generateEssentialsRequestID()
	respCh := make(chan *essentialspb.PlayerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	if err := qs.stream.Send(&essentialspb.PlayerQueryResponse{
		RequestId: reqID,
		Query: &essentialspb.PlayerQueryResponse_PlayerStatusQuery{
			PlayerStatusQuery: &essentialspb.QueryPlayerStatusQuery{
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
		case *essentialspb.PlayerQueryRequest_PlayerStatusResult:
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

// QueryServerStatus 查询服务器状态
func (h *EssentialsPlayerQueryHandler) QueryServerStatus(ctx context.Context, serverName string) (
	players []*essentialspb.PlayerStatus, onlinePlayers int32, tps float64, lastHeartbeat int64, err error,
) {
	qs := getEssentialsQueryStream(serverName)
	if qs == nil {
		qs = getAnyEssentialsQueryStream()
	}
	if qs == nil {
		return nil, 0, 0, 0, fmt.Errorf("无可用的 PlayerQuery 连接")
	}

	reqID := generateEssentialsRequestID()
	respCh := make(chan *essentialspb.PlayerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	if err := qs.stream.Send(&essentialspb.PlayerQueryResponse{
		RequestId: reqID,
		Query: &essentialspb.PlayerQueryResponse_ServerStatusQuery{
			ServerStatusQuery: &essentialspb.QueryServerStatusQuery{
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
		case *essentialspb.PlayerQueryRequest_ServerStatusResult:
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

// QueryCheckPermission 检查玩家权限
func (h *EssentialsPlayerQueryHandler) QueryCheckPermission(ctx context.Context, playerUUID, permissionNode string) (bool, error) {
	qs := getAnyEssentialsQueryStream()
	if qs == nil {
		return false, fmt.Errorf("无可用的 PlayerQuery 连接")
	}

	reqID := generateEssentialsRequestID()
	respCh := make(chan *essentialspb.PlayerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	if err := qs.stream.Send(&essentialspb.PlayerQueryResponse{
		RequestId: reqID,
		Query: &essentialspb.PlayerQueryResponse_CheckPermissionQuery{
			CheckPermissionQuery: &essentialspb.QueryCheckPermissionQuery{
				PlayerUuid:     playerUUID,
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
		case *essentialspb.PlayerQueryRequest_CheckPermissionResult:
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

// QueryPlayerGroups 查询玩家权限组
func (h *EssentialsPlayerQueryHandler) QueryPlayerGroups(ctx context.Context, playerUUID string) (
	primaryGroup string, groups []string, err error,
) {
	qs := getAnyEssentialsQueryStream()
	if qs == nil {
		return "", nil, fmt.Errorf("无可用的 PlayerQuery 连接")
	}

	reqID := generateEssentialsRequestID()
	respCh := make(chan *essentialspb.PlayerQueryRequest, 1)
	qs.pending.Store(reqID, respCh)
	defer qs.pending.Delete(reqID)

	if err := qs.stream.Send(&essentialspb.PlayerQueryResponse{
		RequestId: reqID,
		Query: &essentialspb.PlayerQueryResponse_PlayerGroupsQuery{
			PlayerGroupsQuery: &essentialspb.QueryPlayerGroupsQuery{
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
		case *essentialspb.PlayerQueryRequest_PlayerGroupsResult:
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
