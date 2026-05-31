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

// dispatchTelemetry 根据遥测请求中的字段分发遥测数据
func (h *MatrixTelemetryHandler) dispatchTelemetry(ctx context.Context, req *matrixpb.MatrixTelemetryRequest) {
	// 处理生命周期事件（单次字段）
	if req.GetPlayerJoin() != nil {
		h.handlePlayerJoin(ctx, req.GetPlayerJoin(), req.GetServerName())
	}
	if req.GetPlayerQuit() != nil {
		h.handlePlayerQuit(ctx, req.GetPlayerQuit(), req.GetServerName())
	}
	// 处理批量事件（repeated 字段）
	h.dispatchBatchEvents(ctx, req)
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
		PlayerJoin: evt,
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
			PlayerQuit: evt,
		})
		session.MarkOffline()

		sessionKey := fmt.Sprintf("%s:%s", serverName, playerUUID.String())
		session.Stop()
		h.sessionManager.Remove(sessionKey)
	}
}

// dispatchBatchEvents 遍历所有 16 种 repeated 事件字段，提取 player_uuid 后逐条路由到对应 session
func (h *MatrixTelemetryHandler) dispatchBatchEvents(ctx context.Context, req *matrixpb.MatrixTelemetryRequest) {
	serverName := req.GetServerName()

	// sendEvent 辅助函数：将单条事件包装为 MatrixTelemetryRequest 后发送到 session
	sendEvent := func(playerUUIDStr string, setPayload func(*matrixpb.MatrixTelemetryRequest)) {
		if playerUUIDStr == "" {
			return
		}
		playerUUID, err := uuid.Parse(playerUUIDStr)
		if err != nil {
			h.log.Warn(ctx, "dispatchBatchEvents - UUID格式无效: "+playerUUIDStr)
			return
		}
		session := h.sessionManager.Get(serverName, playerUUID)
		if session == nil {
			return
		}
		eventReq := &matrixpb.MatrixTelemetryRequest{ServerName: serverName}
		setPayload(eventReq)
		session.Send(eventReq)
	}

	// TelemetryTick (field 13)
	for _, evt := range req.GetTelemetryTicks() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.TelemetryTicks = []*matrixpb.TelemetryTick{evt}
		})
	}
	// BlockBreak (field 14)
	for _, evt := range req.GetBlockBreaks() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.BlockBreaks = []*matrixpb.BlockBreakEvent{evt}
		})
	}
	// BlockPlace (field 15)
	for _, evt := range req.GetBlockPlaces() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.BlockPlaces = []*matrixpb.BlockPlaceEvent{evt}
		})
	}
	// EntityKill (field 16)
	for _, evt := range req.GetEntityKills() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.EntityKills = []*matrixpb.EntityKillEvent{evt}
		})
	}
	// EntityDamage (field 17)
	for _, evt := range req.GetEntityDamages() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.EntityDamages = []*matrixpb.EntityDamageEvent{evt}
		})
	}
	// PlayerDamage (field 18)
	for _, evt := range req.GetPlayerDamages() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.PlayerDamages = []*matrixpb.PlayerDamageEvent{evt}
		})
	}
	// PlayerDeath (field 19)
	for _, evt := range req.GetPlayerDeaths() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.PlayerDeaths = []*matrixpb.PlayerDeathEvent{evt}
		})
	}
	// ItemDrop (field 20)
	for _, evt := range req.GetItemDrops() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.ItemDrops = []*matrixpb.ItemDropEvent{evt}
		})
	}
	// ItemPickup (field 21)
	for _, evt := range req.GetItemPickups() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.ItemPickups = []*matrixpb.ItemPickupEvent{evt}
		})
	}
	// InventoryAction (field 22)
	for _, evt := range req.GetInventoryActions() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.InventoryActions = []*matrixpb.InventoryActionEvent{evt}
		})
	}
	// PlayerChat (field 23)
	for _, evt := range req.GetPlayerChats() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.PlayerChats = []*matrixpb.PlayerChatEvent{evt}
		})
	}
	// PlayerCommand (field 24)
	for _, evt := range req.GetPlayerCommands() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.PlayerCommands = []*matrixpb.PlayerCommandEvent{evt}
		})
	}
	// PlayerToggle (field 25)
	for _, evt := range req.GetPlayerToggles() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.PlayerToggles = []*matrixpb.PlayerToggleEvent{evt}
		})
	}
	// Teleport (field 26)
	for _, evt := range req.GetTeleports() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.Teleports = []*matrixpb.PlayerTeleportEvent{evt}
		})
	}
	// Respawn (field 27)
	for _, evt := range req.GetRespawns() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.Respawns = []*matrixpb.PlayerRespawnEvent{evt}
		})
	}
	// GameModeChange (field 28)
	for _, evt := range req.GetGameModeChanges() {
		sendEvent(evt.GetPlayerUuid(), func(r *matrixpb.MatrixTelemetryRequest) {
			r.GameModeChanges = []*matrixpb.GameModeChangeEvent{evt}
		})
	}
}
