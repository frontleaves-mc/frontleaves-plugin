package handler

import (
	"context"
	"sync"
	"testing"
	"time"

	essentialspb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/essentials/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestEssentialsQueryStreamManager_RegisterAndRemove(t *testing.T) {
	qs := &essentialsQueryStream{
		serverName: "test-server",
		pending:    sync.Map{},
	}

	registerEssentialsQueryStream("test-server", qs)

	got := getEssentialsQueryStream("test-server")
	require.NotNil(t, got, "stream should be registered")
	assert.Equal(t, "test-server", got.serverName)

	removeEssentialsQueryStream("test-server")

	got = getEssentialsQueryStream("test-server")
	assert.Nil(t, got, "stream should be removed")
}

func TestEssentialsQueryStreamManager_GetAny(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	got := getAnyEssentialsQueryStream()
	assert.Nil(t, got, "no streams should return nil")

	qs := &essentialsQueryStream{
		serverName: "srv-a",
		pending:    sync.Map{},
	}
	registerEssentialsQueryStream("srv-a", qs)
	defer removeEssentialsQueryStream("srv-a")

	got = getAnyEssentialsQueryStream()
	require.NotNil(t, got, "should find one stream")
	assert.Equal(t, "srv-a", got.serverName)
}

func TestEssentialsQueryStreamManager_Overwrite(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	qs1 := &essentialsQueryStream{serverName: "srv", pending: sync.Map{}}
	qs2 := &essentialsQueryStream{serverName: "srv", pending: sync.Map{}}

	registerEssentialsQueryStream("srv", qs1)
	registerEssentialsQueryStream("srv", qs2)

	got := getEssentialsQueryStream("srv")
	require.NotNil(t, got)
	assert.Same(t, qs2, got, "later registration should overwrite")

	removeEssentialsQueryStream("srv")
}

func TestGenerateEssentialsRequestID(t *testing.T) {
	id := generateEssentialsRequestID()
	assert.NotEmpty(t, id)
}

func TestQueryPlayerStatus_NoStream(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	h := &EssentialsPlayerQueryHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	online, sName, wName, pName, lastSeen, err := h.QueryPlayerStatus(context.Background(), "test-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无可用的 PlayerQuery 连接")
	assert.False(t, online)
	assert.Empty(t, sName)
	assert.Empty(t, wName)
	assert.Empty(t, pName)
	assert.Zero(t, lastSeen)
}

func TestQueryCheckPermission_NoStream(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	h := &EssentialsPlayerQueryHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	hasPerm, err := h.QueryCheckPermission(context.Background(), "uuid", "node")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无可用的 PlayerQuery 连接")
	assert.False(t, hasPerm)
}

func TestQueryPlayerGroups_NoStream(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	h := &EssentialsPlayerQueryHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	pg, groups, err := h.QueryPlayerGroups(context.Background(), "uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无可用的 PlayerQuery 连接")
	assert.Empty(t, pg)
	assert.Nil(t, groups)
}

func TestQueryServerStatus_NoStream(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	h := &EssentialsPlayerQueryHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	players, online, tps, hb, err := h.QueryServerStatus(context.Background(), "srv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无可用的 PlayerQuery 连接")
	assert.Nil(t, players)
	assert.Zero(t, online)
	assert.Zero(t, tps)
	assert.Zero(t, hb)
}

func TestQueryPlayerStatus_Timeout(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	mockStream := &mockBidiStream{
		sendCh: make(chan *essentialspb.PlayerQueryResponse, 10),
	}
	qs := &essentialsQueryStream{
		stream:     mockStream,
		serverName: "test-timeout",
		pending:    sync.Map{},
	}
	registerEssentialsQueryStream("test-timeout", qs)
	defer removeEssentialsQueryStream("test-timeout")

	h := &EssentialsPlayerQueryHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, _, _, _, err := h.QueryPlayerStatus(ctx, "test-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestQueryPlayerStatus_Success(t *testing.T) {
	essentialsQueryStreamManager.mu.Lock()
	essentialsQueryStreamManager.streams = make(map[string]*essentialsQueryStream)
	essentialsQueryStreamManager.mu.Unlock()

	mockStream := &mockBidiStream{
		sendCh: make(chan *essentialspb.PlayerQueryResponse, 10),
	}
	qs := &essentialsQueryStream{
		stream:     mockStream,
		serverName: "test-success",
		pending:    sync.Map{},
	}
	registerEssentialsQueryStream("test-success", qs)
	defer removeEssentialsQueryStream("test-success")

	go func() {
		resp := <-mockStream.sendCh
		reqID := resp.GetRequestId()

		if ch, ok := qs.pending.Load(reqID); ok {
			ch.(chan *essentialspb.PlayerQueryRequest) <- &essentialspb.PlayerQueryRequest{
				RequestId: reqID,
				Result: &essentialspb.PlayerQueryRequest_PlayerStatusResult{
					PlayerStatusResult: &essentialspb.QueryPlayerStatusResult{
						Online:     true,
						ServerName: "lobby",
						WorldName:  "world",
						PlayerName: "TestPlayer",
						LastSeen:   1234567890,
					},
				},
			}
		}
	}()

	h := &EssentialsPlayerQueryHandler{
		grpcHandler:       *newTestGRPCHandler(),
		essentialsService: newTestEssentialsService(),
	}

	online, sName, wName, pName, lastSeen, err := h.QueryPlayerStatus(context.Background(), "test-uuid")
	require.NoError(t, err)
	assert.True(t, online)
	assert.Equal(t, "lobby", sName)
	assert.Equal(t, "world", wName)
	assert.Equal(t, "TestPlayer", pName)
	assert.Equal(t, int64(1234567890), lastSeen)
}

// mockBidiStream 模拟 grpc.BidiStreamingServer
type mockBidiStream struct {
	sendCh chan *essentialspb.PlayerQueryResponse
}

func (m *mockBidiStream) Send(msg *essentialspb.PlayerQueryResponse) error {
	m.sendCh <- msg
	return nil
}

func (m *mockBidiStream) Recv() (*essentialspb.PlayerQueryRequest, error) {
	// no-op for test
	return nil, nil
}

func (m *mockBidiStream) Context() context.Context {
	return context.Background()
}

func (m *mockBidiStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockBidiStream) SendHeader(metadata.MD) error { return nil }
func (m *mockBidiStream) SetTrailer(metadata.MD)       {}
func (m *mockBidiStream) SendMsg(msg any) error        { return nil }
func (m *mockBidiStream) RecvMsg(msg any) error        { return nil }
