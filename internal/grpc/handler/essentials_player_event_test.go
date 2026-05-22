package handler

import (
	"testing"

	essentialspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/essentials/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEssentialsPlayerEventHandlerConstruction(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	require.NotNil(t, h, "EssentialsPlayerEventHandler should not be nil")
	assert.Equal(t, "TestHandler", h.name)
}

func TestEssentialsPlayerEventHandlerImplementsInterface(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	var _ essentialspb.PlayerEventServiceServer = h
}

func TestDispatchPlayerEventJoinWithNilEvent(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	req := &essentialspb.PlayerEventStreamRequest{
		EventType: essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_JOIN,
	}

	assert.NotPanics(t, func() {
		h.dispatchPlayerEvent(newTestContext(), req)
	}, "dispatchPlayerEvent with nil PlayerJoinEvent should not panic")
}

func TestDispatchPlayerEventQuitWithNilEvent(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	req := &essentialspb.PlayerEventStreamRequest{
		EventType: essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_QUIT,
	}

	assert.NotPanics(t, func() {
		h.dispatchPlayerEvent(newTestContext(), req)
	}, "dispatchPlayerEvent with nil PlayerQuitEvent should not panic")
}

func TestDispatchPlayerEventChatWithNilEvent(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	req := &essentialspb.PlayerEventStreamRequest{
		EventType: essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_CHAT,
	}

	assert.NotPanics(t, func() {
		h.dispatchPlayerEvent(newTestContext(), req)
	}, "dispatchPlayerEvent with nil PlayerChatEvent should not panic")
}

func TestDispatchPlayerEventKickWithNilEvent(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	req := &essentialspb.PlayerEventStreamRequest{
		EventType: essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_KICK,
	}

	assert.NotPanics(t, func() {
		h.dispatchPlayerEvent(newTestContext(), req)
	}, "dispatchPlayerEvent with nil PlayerKickEvent should not panic")
}

func TestDispatchPlayerEventDeathWithNilEvent(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	req := &essentialspb.PlayerEventStreamRequest{
		EventType: essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_DEATH,
	}

	assert.NotPanics(t, func() {
		h.dispatchPlayerEvent(newTestContext(), req)
	}, "dispatchPlayerEvent with nil PlayerDeathEvent should not panic")
}

func TestDispatchPlayerEventGroupChangeWithNilEvent(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	req := &essentialspb.PlayerEventStreamRequest{
		EventType: essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_PLAYER_GROUP_CHANGE,
	}

	assert.NotPanics(t, func() {
		h.dispatchPlayerEvent(newTestContext(), req)
	}, "dispatchPlayerEvent with nil PlayerGroupChangeEvent should not panic")
}

func TestDispatchPlayerEventUnspecifiedType(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	req := &essentialspb.PlayerEventStreamRequest{
		EventType: essentialspb.PlayerEventType_PLAYER_EVENT_TYPE_UNSPECIFIED,
	}

	assert.NotPanics(t, func() {
		h.dispatchPlayerEvent(newTestContext(), req)
	}, "dispatchPlayerEvent with UNSPECIFIED type should not panic")
}

func TestHandlePlayerJoinEmptyUUID(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	join := &essentialspb.PlayerJoinEvent{
		PlayerUuid: "",
		PlayerName: "TestPlayer",
	}

	assert.NotPanics(t, func() {
		h.handlePlayerJoin(newTestContext(), join)
	}, "handlePlayerJoin with empty UUID should not panic")
}

func TestHandlePlayerQuitEmptyUUID(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	quit := &essentialspb.PlayerQuitEvent{
		PlayerUuid: "",
		PlayerName: "TestPlayer",
	}

	assert.NotPanics(t, func() {
		h.handlePlayerQuit(newTestContext(), quit)
	}, "handlePlayerQuit with empty UUID should not panic")
}

func TestHandlePlayerChatEmptyUUID(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	chat := &essentialspb.PlayerChatEvent{
		PlayerUuid: "",
		PlayerName: "TestPlayer",
	}

	assert.NotPanics(t, func() {
		h.handlePlayerChat(newTestContext(), chat)
	}, "handlePlayerChat with empty UUID should not panic")
}

func TestHandlePlayerKickEmptyUUID(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	kick := &essentialspb.PlayerKickEvent{
		PlayerUuid: "",
		PlayerName: "TestPlayer",
	}

	assert.NotPanics(t, func() {
		h.handlePlayerKick(newTestContext(), kick)
	}, "handlePlayerKick with empty UUID should not panic")
}

func TestHandlePlayerDeathEmptyUUID(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	death := &essentialspb.PlayerDeathEvent{
		PlayerUuid: "",
		PlayerName: "TestPlayer",
	}

	assert.NotPanics(t, func() {
		h.handlePlayerDeath(newTestContext(), death)
	}, "handlePlayerDeath with empty UUID should not panic")
}

func TestHandlePlayerGroupChangeEmptyUUID(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	gc := &essentialspb.PlayerGroupChangeEvent{
		PlayerUuid:   "",
		PlayerName:   "TestPlayer",
		GroupName:    "ADMIN",
		OldGroupName: "PLAYER",
	}

	assert.NotPanics(t, func() {
		h.handlePlayerGroupChange(newTestContext(), gc)
	}, "handlePlayerGroupChange with empty UUID should not panic")
}

func TestHandlePlayerJoinInvalidUUID(t *testing.T) {
	h := &EssentialsPlayerEventHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	join := &essentialspb.PlayerJoinEvent{
		PlayerUuid: "not-a-valid-uuid",
		PlayerName: "TestPlayer",
		ServerName: "survival",
		WorldName:  "world",
		GroupName:  "PLAYER",
	}

	assert.NotPanics(t, func() {
		h.handlePlayerJoin(newTestContext(), join)
	}, "handlePlayerJoin with invalid UUID should not panic")
}
