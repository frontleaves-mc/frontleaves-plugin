package matrix

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Mock ServerQuerier ---

type mockServerQuerier struct {
	servers []string
	players map[string][]PlayerStatusInfo
	errors  map[string]error
}

func (m *mockServerQuerier) GetAllConnectedServers() []string {
	return m.servers
}

func (m *mockServerQuerier) QueryServerStatus(_ context.Context, serverName string) ([]PlayerStatusInfo, error) {
	if err, ok := m.errors[serverName]; ok {
		return nil, err
	}
	return m.players[serverName], nil
}

// --- Tests ---

func TestRecoverSessionsNoServers(t *testing.T) {
	originalDelay := sessionRecoveryStartupDelay
	sessionRecoveryStartupDelay = 1 * time.Millisecond
	t.Cleanup(func() { sessionRecoveryStartupDelay = originalDelay })

	querier := &mockServerQuerier{servers: nil}
	RecoverSessions(context.Background(), querier)
}

func TestRecoverSessionsManagerNil(t *testing.T) {
	originalDelay := sessionRecoveryStartupDelay
	sessionRecoveryStartupDelay = 1 * time.Millisecond
	t.Cleanup(func() { sessionRecoveryStartupDelay = originalDelay })

	originalManager := GetGlobalMatrixSessionManager()
	SetGlobalMatrixSessionManager(nil)
	t.Cleanup(func() { SetGlobalMatrixSessionManager(originalManager) })

	querier := &mockServerQuerier{
		servers: []string{"lobby"},
		players: map[string][]PlayerStatusInfo{
			"lobby": {
				{PlayerUUID: "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1", PlayerName: "Alice", WorldName: "world"},
			},
		},
	}

	RecoverSessions(context.Background(), querier)
}

func TestRecoverSessionsContextCancelled(t *testing.T) {
	originalDelay := sessionRecoveryStartupDelay
	sessionRecoveryStartupDelay = 5 * time.Second
	t.Cleanup(func() { sessionRecoveryStartupDelay = originalDelay })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	querier := &mockServerQuerier{servers: []string{"lobby"}}
	RecoverSessions(ctx, querier)
}

func TestRecoverSessionsQueryError(t *testing.T) {
	originalDelay := sessionRecoveryStartupDelay
	sessionRecoveryStartupDelay = 1 * time.Millisecond
	t.Cleanup(func() { sessionRecoveryStartupDelay = originalDelay })

	originalManager := GetGlobalMatrixSessionManager()
	SetGlobalMatrixSessionManager(nil)
	t.Cleanup(func() { SetGlobalMatrixSessionManager(originalManager) })

	querier := &mockServerQuerier{
		servers: []string{"failing-server", "ok-server"},
		errors: map[string]error{
			"failing-server": errors.New("connection refused"),
		},
		players: map[string][]PlayerStatusInfo{
			"ok-server": {},
		},
	}

	RecoverSessions(context.Background(), querier)
}

func TestRecoverSessionsInvalidUUID(t *testing.T) {
	originalDelay := sessionRecoveryStartupDelay
	sessionRecoveryStartupDelay = 1 * time.Millisecond
	t.Cleanup(func() { sessionRecoveryStartupDelay = originalDelay })

	originalManager := GetGlobalMatrixSessionManager()
	SetGlobalMatrixSessionManager(nil)
	t.Cleanup(func() { SetGlobalMatrixSessionManager(originalManager) })

	querier := &mockServerQuerier{
		servers: []string{"lobby"},
		players: map[string][]PlayerStatusInfo{
			"lobby": {
				{PlayerUUID: "not-a-valid-uuid", PlayerName: "BadPlayer", WorldName: "world"},
				{PlayerUUID: "b2b2b2b2-b2b2-b2b2-b2b2-b2b2b2b2b2b2", PlayerName: "GoodPlayer", WorldName: "world"},
			},
		},
	}

	RecoverSessions(context.Background(), querier)
}

func TestRecoverSessionsCompletesWithinTimeout(t *testing.T) {
	originalDelay := sessionRecoveryStartupDelay
	sessionRecoveryStartupDelay = 10 * time.Millisecond
	t.Cleanup(func() { sessionRecoveryStartupDelay = originalDelay })

	originalManager := GetGlobalMatrixSessionManager()
	SetGlobalMatrixSessionManager(nil)
	t.Cleanup(func() { SetGlobalMatrixSessionManager(originalManager) })

	querier := &mockServerQuerier{
		servers: []string{"srv1", "srv2", "srv3"},
		players: map[string][]PlayerStatusInfo{
			"srv1": {
				{PlayerUUID: "11111111-1111-1111-1111-111111111111", PlayerName: "P1", WorldName: "w"},
			},
			"srv2": {},
			"srv3": {
				{PlayerUUID: "33333333-3333-3333-3333-333333333333", PlayerName: "P3", WorldName: "w"},
			},
		},
	}

	done := make(chan struct{})
	go func() {
		RecoverSessions(context.Background(), querier)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RecoverSessions did not complete within 5s")
	}
}

func TestRecoverSessionsEmptyPlayers(t *testing.T) {
	originalDelay := sessionRecoveryStartupDelay
	sessionRecoveryStartupDelay = 1 * time.Millisecond
	t.Cleanup(func() { sessionRecoveryStartupDelay = originalDelay })

	originalManager := GetGlobalMatrixSessionManager()
	SetGlobalMatrixSessionManager(nil)
	t.Cleanup(func() { SetGlobalMatrixSessionManager(originalManager) })

	querier := &mockServerQuerier{
		servers: []string{"empty-server"},
		players: map[string][]PlayerStatusInfo{
			"empty-server": {},
		},
	}

	done := make(chan struct{})
	go func() {
		RecoverSessions(context.Background(), querier)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("RecoverSessions hung with empty players")
	}
}

func TestMockServerQuerierGetAllConnectedServers(t *testing.T) {
	q := &mockServerQuerier{servers: []string{"a", "b", "c"}}
	assert.Equal(t, []string{"a", "b", "c"}, q.GetAllConnectedServers())
}

func TestMockServerQuerierQueryServerStatus(t *testing.T) {
	q := &mockServerQuerier{
		players: map[string][]PlayerStatusInfo{
			"lobby": {{PlayerUUID: "test", PlayerName: "P", WorldName: "w"}},
		},
		errors: map[string]error{
			"down": errors.New("unreachable"),
		},
	}

	players, err := q.QueryServerStatus(context.Background(), "lobby")
	assert.NoError(t, err)
	assert.Len(t, players, 1)

	_, err = q.QueryServerStatus(context.Background(), "down")
	assert.Error(t, err)
}
