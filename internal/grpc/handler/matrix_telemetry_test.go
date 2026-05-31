package handler

import (
	"context"
	"sync"
	"testing"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMatrixHandler(t *testing.T) *MatrixTelemetryHandler {
	t.Helper()
	return &MatrixTelemetryHandler{
		grpcHandler: grpcHandler{
			name: "TestMatrixTelemetry",
			log:  xLog.WithName(xLog.NamedGRPC, "TestMatrixTelemetry"),
			rdb:  nil,
		},
		matrixService: &matrixService{
			sessionManager: &matrix.MatrixSessionManager{},
		},
	}
}

// TestMatrixTelemetryHandler_DispatchTelemetry_EmptyBatch verifies that an empty request
// does not crash the handler.
func TestMatrixTelemetryHandler_DispatchTelemetry_EmptyBatch(t *testing.T) {
	h := newTestMatrixHandler(t)
	ctx := context.Background()

	req := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TickNumber: 1,
		Timestamp:  time.Now().UnixMilli(),
	}

	assert.NotPanics(t, func() {
		h.dispatchTelemetry(ctx, req)
	})
}

// TestMatrixTelemetryHandler_DispatchBatchEvents_SingleEvent verifies single-event routing.
func TestMatrixTelemetryHandler_DispatchBatchEvents_SingleEvent(t *testing.T) {
	h := newTestMatrixHandler(t)
	ctx := context.Background()

	req := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		EntityDamages: []*matrixpb.EntityDamageEvent{
			{
				PlayerUuid: uuid.New().String(),
				PlayerName: "TestPlayer",
				Damage:     5.0,
				Timestamp:  time.Now().UnixMilli(),
			},
		},
	}

	assert.NotPanics(t, func() {
		h.dispatchBatchEvents(ctx, req)
	})
}

// TestMatrixTelemetryHandler_DispatchBatchEvents_MultiPlayerMixedEvents verifies
// multi-player mixed event types are handled without crash.
func TestMatrixTelemetryHandler_DispatchBatchEvents_MultiPlayerMixedEvents(t *testing.T) {
	h := newTestMatrixHandler(t)
	ctx := context.Background()

	player1 := uuid.New().String()
	player2 := uuid.New().String()

	req := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{PlayerUuid: player1, PlayerName: "Player1", PosX: 100, PosY: 64, PosZ: 200, Timestamp: 1000},
			{PlayerUuid: player2, PlayerName: "Player2", PosX: 200, PosY: 64, PosZ: 300, Timestamp: 1000},
		},
		BlockBreaks: []*matrixpb.BlockBreakEvent{
			{PlayerUuid: player1, PlayerName: "Player1", Material: "STONE", Timestamp: 1001},
		},
		EntityDamages: []*matrixpb.EntityDamageEvent{
			{PlayerUuid: player2, PlayerName: "Player2", Damage: 3.0, Timestamp: 1002},
		},
		PlayerChats: []*matrixpb.PlayerChatEvent{
			{PlayerUuid: player1, PlayerName: "Player1", Message: "hello", Timestamp: 1003},
			{PlayerUuid: player2, PlayerName: "Player2", Message: "world", Timestamp: 1004},
		},
		Teleports: []*matrixpb.PlayerTeleportEvent{
			{PlayerUuid: player1, PlayerName: "Player1", Cause: "PLUGIN", Timestamp: 1005},
		},
	}

	assert.NotPanics(t, func() {
		h.dispatchBatchEvents(ctx, req)
	})
}

// TestMatrixTelemetryHandler_DispatchBatchEvents_PlayerJoinWithBatch verifies
// PlayerJoin lifecycle event alongside batch events dispatch.
func TestMatrixTelemetryHandler_DispatchBatchEvents_PlayerJoinWithBatch(t *testing.T) {
	h := newTestMatrixHandler(t)
	ctx := context.Background()

	playerUUID := uuid.New()

	req := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		PlayerJoin: &matrixpb.PlayerJoinEvent{
			PlayerUuid: playerUUID.String(),
			PlayerName: "TestPlayer",
			WorldName:  "world",
			Timestamp:  time.Now().UnixMilli(),
		},
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{PlayerUuid: playerUUID.String(), PlayerName: "TestPlayer", PosX: 100, PosY: 64, PosZ: 200, Timestamp: 1000},
		},
	}

	assert.NotPanics(t, func() {
		h.dispatchBatchEvents(ctx, req)
	})
}

// TestMatrixTelemetryHandler_DispatchBatchEvents_EmptyUUIDSkipped verifies empty/invalid
// player_uuid events are silently skipped.
func TestMatrixTelemetryHandler_DispatchBatchEvents_EmptyUUIDSkipped(t *testing.T) {
	h := newTestMatrixHandler(t)
	ctx := context.Background()

	req := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		EntityDamages: []*matrixpb.EntityDamageEvent{
			{PlayerUuid: "", PlayerName: "EmptyUUID", Damage: 5.0, Timestamp: 1000},
			{PlayerUuid: "not-a-valid-uuid", PlayerName: "InvalidUUID", Damage: 3.0, Timestamp: 1001},
		},
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{PlayerUuid: "", PlayerName: "EmptyTick", Timestamp: 1002},
		},
	}

	assert.NotPanics(t, func() {
		h.dispatchBatchEvents(ctx, req)
	})
}

// TestMatrixTelemetryHandler_AllEventTypes_NoCrash verifies all 16 event types
// in a single request without crash.
func TestMatrixTelemetryHandler_AllEventTypes_NoCrash(t *testing.T) {
	h := newTestMatrixHandler(t)
	ctx := context.Background()

	playerUUID := uuid.New().String()
	ts := time.Now().UnixMilli()

	req := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{PlayerUuid: playerUUID, PosX: 100, PosY: 64, PosZ: 200, Timestamp: ts},
		},
		BlockBreaks: []*matrixpb.BlockBreakEvent{
			{PlayerUuid: playerUUID, Material: "STONE", Timestamp: ts},
		},
		BlockPlaces: []*matrixpb.BlockPlaceEvent{
			{PlayerUuid: playerUUID, Material: "DIRT", Timestamp: ts},
		},
		EntityKills: []*matrixpb.EntityKillEvent{
			{PlayerUuid: playerUUID, EntityType: "ZOMBIE", Timestamp: ts},
		},
		EntityDamages: []*matrixpb.EntityDamageEvent{
			{PlayerUuid: playerUUID, Damage: 5.0, Timestamp: ts},
		},
		PlayerDamages: []*matrixpb.PlayerDamageEvent{
			{PlayerUuid: playerUUID, Damage: 3.0, Timestamp: ts},
		},
		PlayerDeaths: []*matrixpb.PlayerDeathEvent{
			{PlayerUuid: playerUUID, DeathCause: "FALL", Timestamp: ts},
		},
		ItemDrops: []*matrixpb.ItemDropEvent{
			{PlayerUuid: playerUUID, Material: "DIAMOND", Amount: 1, Timestamp: ts},
		},
		ItemPickups: []*matrixpb.ItemPickupEvent{
			{PlayerUuid: playerUUID, Material: "IRON_INGOT", Amount: 5, Timestamp: ts},
		},
		InventoryActions: []*matrixpb.InventoryActionEvent{
			{PlayerUuid: playerUUID, ActionType: "CLICK", Timestamp: ts},
		},
		PlayerChats: []*matrixpb.PlayerChatEvent{
			{PlayerUuid: playerUUID, Message: "hello", Timestamp: ts},
		},
		PlayerCommands: []*matrixpb.PlayerCommandEvent{
			{PlayerUuid: playerUUID, Command: "/spawn", Timestamp: ts},
		},
		PlayerToggles: []*matrixpb.PlayerToggleEvent{
			{PlayerUuid: playerUUID, ToggleType: "SNEAK", IsEnabled: true, Timestamp: ts},
		},
		Teleports: []*matrixpb.PlayerTeleportEvent{
			{PlayerUuid: playerUUID, Cause: "PLUGIN", Timestamp: ts},
		},
		Respawns: []*matrixpb.PlayerRespawnEvent{
			{PlayerUuid: playerUUID, WorldName: "world", Timestamp: ts},
		},
		GameModeChanges: []*matrixpb.GameModeChangeEvent{
			{PlayerUuid: playerUUID, OldGameMode: "SURVIVAL", NewGameMode: "CREATIVE", Timestamp: ts},
		},
	}

	assert.NotPanics(t, func() {
		h.dispatchBatchEvents(ctx, req)
	})
}

// TestMatrixTelemetryHandler_RequestConstruction verifies request field getters.
func TestMatrixTelemetryHandler_RequestConstruction(t *testing.T) {
	playerUUID := uuid.New().String()
	ts := time.Now().UnixMilli()

	req := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{PlayerUuid: playerUUID, PosX: 100, PosY: 64, PosZ: 200, Timestamp: ts},
		},
		EntityDamages: []*matrixpb.EntityDamageEvent{
			{PlayerUuid: playerUUID, Damage: 5.0, Timestamp: ts},
		},
		BlockBreaks: []*matrixpb.BlockBreakEvent{
			{PlayerUuid: playerUUID, Material: "STONE", Timestamp: ts},
		},
	}

	require.Len(t, req.GetTelemetryTicks(), 1)
	require.Len(t, req.GetEntityDamages(), 1)
	require.Len(t, req.GetBlockBreaks(), 1)
	assert.Equal(t, playerUUID, req.GetTelemetryTicks()[0].GetPlayerUuid())
	assert.Equal(t, playerUUID, req.GetEntityDamages()[0].GetPlayerUuid())
	assert.Equal(t, playerUUID, req.GetBlockBreaks()[0].GetPlayerUuid())
	assert.Nil(t, req.GetPlayerJoin())
	assert.Nil(t, req.GetPlayerQuit())
}

// TestMatrixTelemetryHandler_DispatchBatchEvents_Concurrent verifies thread safety.
func TestMatrixTelemetryHandler_DispatchBatchEvents_Concurrent(t *testing.T) {
	h := newTestMatrixHandler(t)
	ctx := context.Background()

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			req := &matrixpb.MatrixTelemetryRequest{
				ServerName: "test-server",
				TelemetryTicks: []*matrixpb.TelemetryTick{
					{PlayerUuid: uuid.New().String(), PosX: float64(id), PosY: 64, PosZ: 200, Timestamp: time.Now().UnixMilli()},
				},
			}
			h.dispatchBatchEvents(ctx, req)
		}(i)
	}

	wg.Wait()
}
