package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestContext() context.Context {
	return context.Background()
}

func newTestGRPCHandler() *grpcHandler {
	return &grpcHandler{
		name: "TestHandler",
		log:  nil,
		rdb:  nil,
	}
}

func newTestEssentialsService() *essentialsService {
	return &essentialsService{
		gameProfileLogic:  nil,
		playerEventLogic:  nil,
		playerChatLogic:   nil,
		serverLogic:       nil,
		serverPlayerLogic: nil,
	}
}

func newTestTitleService() *titleService {
	return &titleService{
		titleLogic:       nil,
		gameProfileLogic: nil,
	}
}

func newTestStatusService() *statusService {
	return &statusService{
		serverLogic:       nil,
		serverPlayerLogic: nil,
	}
}

func TestGRPCHandlerConstruction(t *testing.T) {
	h := newTestGRPCHandler()

	require.NotNil(t, h, "grpcHandler should not be nil")
	assert.Equal(t, "TestHandler", h.name)
}

func TestNewTestContext(t *testing.T) {
	ctx := newTestContext()

	require.NotNil(t, ctx)
}

func TestEssentialsServiceFields(t *testing.T) {
	svc := newTestEssentialsService()

	require.NotNil(t, svc, "essentialsService should not be nil")
	_ = svc.gameProfileLogic
	_ = svc.playerEventLogic
	_ = svc.playerChatLogic
	_ = svc.serverLogic
	_ = svc.serverPlayerLogic
}

func TestStatusServiceFields(t *testing.T) {
	svc := newTestStatusService()

	require.NotNil(t, svc, "statusService should not be nil")
	_ = svc.serverLogic
	_ = svc.serverPlayerLogic
}

func TestTitleServiceFields(t *testing.T) {
	svc := newTestTitleService()

	require.NotNil(t, svc, "titleService should not be nil")
	_ = svc.titleLogic
	_ = svc.gameProfileLogic
}
