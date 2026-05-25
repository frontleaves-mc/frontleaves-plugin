package handler

import (
	"context"
	"fmt"

	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MatrixTelemetryHandler Matrix 遥测 gRPC Handler
type MatrixTelemetryHandler struct {
	grpcHandler
	*matrixService
	matrixpb.UnimplementedMatrixTelemetryServiceServer
}

// NewMatrixTelemetryHandler 创建 MatrixTelemetryHandler 实例并注册 gRPC 服务
func NewMatrixTelemetryHandler(ctx context.Context, server grpc.ServiceRegistrar) *MatrixTelemetryHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "MatrixTelemetryHandler")
	db := xCtxUtil.MustGetDB(ctx)
	h := &MatrixTelemetryHandler{
		grpcHandler:   *base,
		matrixService: newMatrixService(ctx, db, base.rdb),
	}

	matrixpb.RegisterMatrixTelemetryServiceServer(server, h)
	xGrpcMiddle.UseStream(matrixpb.MatrixTelemetryService_ServiceDesc, middleware.StreamPluginVerify(ctx))

	return h
}

// TelemetryStream 实现 Matrix 遥测数据客户端流式 RPC
func (h *MatrixTelemetryHandler) TelemetryStream(
	stream grpc.ClientStreamingServer[matrixpb.MatrixTelemetryRequest, matrixpb.MatrixTelemetryResponse],
) error {
	ctx := stream.Context()
	h.log.Info(ctx, "TelemetryStream - 新的客户端流连接")

	var registeredServerName string

	for {
		req, err := stream.Recv()
		if err != nil {
			if status.Code(err) == codes.Canceled || status.Code(err) == codes.OK {
				if registeredServerName != "" {
					h.log.Info(ctx, "TelemetryStream - 流正常关闭, server="+registeredServerName)
				}
				return nil
			}
			h.log.Warn(ctx, "TelemetryStream - 流读取错误: "+err.Error())
			return err
		}

		serverName := req.GetServerName()
		if serverName != "" && registeredServerName == "" {
			registeredServerName = serverName
		}

		h.dispatchTelemetry(ctx, req)
	}
}

// dispatchTelemetry 根据 oneof payload 类型分发遥测数据
func (h *MatrixTelemetryHandler) dispatchTelemetry(ctx context.Context, req *matrixpb.MatrixTelemetryRequest) {
	switch payload := req.Payload.(type) {
	case *matrixpb.MatrixTelemetryRequest_PlayerJoin:
		h.handlePlayerJoin(ctx, payload.PlayerJoin, req.GetServerName())
	case *matrixpb.MatrixTelemetryRequest_PlayerQuit:
		h.handlePlayerQuit(ctx, payload.PlayerQuit, req.GetServerName())
	default:
		h.handleGenericEvent(req)
	}
}

// handlePlayerJoin 处理玩家加入事件
func (h *MatrixTelemetryHandler) handlePlayerJoin(ctx context.Context, evt *matrixpb.PlayerJoinEvent, serverName string) {
	if evt == nil {
		return
	}
	playerUUID, err := uuid.Parse(evt.GetPlayerUuid())
	if err != nil {
		h.log.Warn(ctx, "PlayerJoinEvent - UUID 格式无效: "+evt.GetPlayerUuid())
		return
	}

	session := h.sessionManager.GetOrCreate(ctx, serverName, playerUUID, evt.GetPlayerName())
	session.Send(&matrixpb.MatrixTelemetryRequest{
		ServerName: serverName,
		Payload:    &matrixpb.MatrixTelemetryRequest_PlayerJoin{PlayerJoin: evt},
	})
}

// handlePlayerQuit 处理玩家退出事件
func (h *MatrixTelemetryHandler) handlePlayerQuit(ctx context.Context, evt *matrixpb.PlayerQuitEvent, serverName string) {
	if evt == nil {
		return
	}
	playerUUID, err := uuid.Parse(evt.GetPlayerUuid())
	if err != nil {
		h.log.Warn(ctx, "PlayerQuitEvent - UUID 格式无效: "+evt.GetPlayerUuid())
		return
	}

	session := h.sessionManager.Get(serverName, playerUUID)
	if session != nil {
		session.Send(&matrixpb.MatrixTelemetryRequest{
			ServerName: serverName,
			Payload:    &matrixpb.MatrixTelemetryRequest_PlayerQuit{PlayerQuit: evt},
		})
		session.MarkOffline()

		sessionKey := fmt.Sprintf("%s:%s", serverName, playerUUID.String())
		session.Stop()
		h.sessionManager.Remove(sessionKey)
	}
}

// handleGenericEvent 处理所有其他遥测事件
func (h *MatrixTelemetryHandler) handleGenericEvent(req *matrixpb.MatrixTelemetryRequest) {
	var playerUUIDStr string

	switch payload := req.Payload.(type) {
	case *matrixpb.MatrixTelemetryRequest_TelemetryTick:
		playerUUIDStr = payload.TelemetryTick.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_BlockBreak:
		playerUUIDStr = payload.BlockBreak.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_BlockPlace:
		playerUUIDStr = payload.BlockPlace.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_EntityKill:
		playerUUIDStr = payload.EntityKill.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_EntityDamage:
		playerUUIDStr = payload.EntityDamage.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_PlayerDamage:
		playerUUIDStr = payload.PlayerDamage.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_PlayerDeath:
		playerUUIDStr = payload.PlayerDeath.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_ItemDrop:
		playerUUIDStr = payload.ItemDrop.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_ItemPickup:
		playerUUIDStr = payload.ItemPickup.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_InventoryAction:
		playerUUIDStr = payload.InventoryAction.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_PlayerChat:
		playerUUIDStr = payload.PlayerChat.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_PlayerCommand:
		playerUUIDStr = payload.PlayerCommand.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_PlayerToggle:
		playerUUIDStr = payload.PlayerToggle.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_Teleport:
		playerUUIDStr = payload.Teleport.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_Respawn:
		playerUUIDStr = payload.Respawn.GetPlayerUuid()
	case *matrixpb.MatrixTelemetryRequest_GameModeChange:
		playerUUIDStr = payload.GameModeChange.GetPlayerUuid()
	default:
		return
	}

	if playerUUIDStr == "" {
		return
	}

	playerUUID, err := uuid.Parse(playerUUIDStr)
	if err != nil {
		return
	}

	session := h.sessionManager.Get(req.GetServerName(), playerUUID)
	if session != nil {
		session.Send(req)
	}
}
