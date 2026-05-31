package matrix

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Sub ---

// mockSub is a mock implementation of MatrixSub for testing.
// All counters use sync/atomic for thread safety under -race.
type mockSub struct {
	name         string
	processCount int32
	drainCount   int32
	shouldError  bool
	delay        time.Duration
}

func (m *mockSub) Name() string { return m.name }

func (m *mockSub) Process(_ context.Context, _ *matrixpb.MatrixTelemetryRequest) error {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		default:
		}
	}
	if m.shouldError {
		return fmt.Errorf("mock error from %s", m.name)
	}
	atomic.AddInt32(&m.processCount, 1)
	return nil
}

func (m *mockSub) Drain(_ context.Context) error {
	atomic.AddInt32(&m.drainCount, 1)
	return nil
}

func (m *mockSub) getProcessCount() int32 { return atomic.LoadInt32(&m.processCount) }
func (m *mockSub) getDrainCount() int32   { return atomic.LoadInt32(&m.drainCount) }

// --- Helpers ---

// newMinimalSession creates a PlayerSession without Redis for unit testing.
// Only populates fields needed for Send/MarkOffline/syncBroadcast testing.
func newMinimalSession(t *testing.T) *PlayerSession {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	return &PlayerSession{
		ctx:        ctx,
		cancel:     cancel,
		playerUUID: uuid.New(),
		playerName: "TestPlayer",
		serverName: "test-server",
		sessionKey: "test-server:test-uuid",
		log:        xLog.WithName(xLog.NamedLOGC, "TestSession"),
		inputCh:    make(chan *matrixpb.MatrixTelemetryRequest, inputChSize),
		isOnline:   true,
		isDraining: false,
	}
}

// makeTestMsg creates a minimal MatrixTelemetryRequest with a TelemetryTick payload.
func makeTestMsg() *matrixpb.MatrixTelemetryRequest {
	return &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{
				PlayerUuid: uuid.New().String(),
				PlayerName: "TestPlayer",
				PosX:       100.0,
				PosY:       64.0,
				PosZ:       200.0,
				Timestamp:  time.Now().UnixMilli(),
			},
		},
	}
}

// --- Tests ---

// TestPlayerSessionLifecycle tests Send → MarkOffline → channel close behavior.
func TestPlayerSessionLifecycle(t *testing.T) {
	session := newMinimalSession(t)
	defer session.cancel()

	// Phase 1: Online — Send should accept messages
	msg := makeTestMsg()
	session.Send(msg)

	// Verify the message landed in inputCh
	select {
	case received := <-session.inputCh:
		assert.Equal(t, msg.ServerName, received.ServerName)
	case <-time.After(time.Second):
		t.Fatal("expected to receive message from inputCh within 1s")
	}

	// Phase 2: Mark offline — inputCh should be closed
	session.MarkOffline()

	// isOnline should be false
	session.mu.RLock()
	online := session.isOnline
	draining := session.isDraining
	session.mu.RUnlock()
	assert.False(t, online, "isOnline should be false after MarkOffline")
	assert.True(t, draining, "isDraining should be true after MarkOffline")

	// Phase 3: Send after offline — should return immediately (no panic, no block)
	done := make(chan struct{})
	go func() {
		session.Send(makeTestMsg())
		close(done)
	}()
	select {
	case <-done:
		// OK — Send returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Send should return immediately after MarkOffline")
	}

	// Phase 4: inputCh should be closed — reading should yield zero-value + false
	_, ok := <-session.inputCh
	assert.False(t, ok, "inputCh should be closed after MarkOffline")
}

// TestPlayerSessionGracefulDrain tests that Stop completes within drainTimeout
// even when no goroutines were started via Start().
func TestPlayerSessionGracefulDrain(t *testing.T) {
	session := newMinimalSession(t)

	// Stop without Start — wg is zero so Wait() returns immediately
	done := make(chan struct{})
	go func() {
		session.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(drainTimeout + 2*time.Second):
		t.Fatal("Stop() did not complete within drainTimeout + 2s")
	}
}

// TestPlayerSessionConcurrentSend hammers Send from many goroutines to detect data races.
func TestPlayerSessionConcurrentSend(t *testing.T) {
	session := newMinimalSession(t)
	defer session.cancel()

	const goroutines = 50
	const messagesPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				session.Send(makeTestMsg())
			}
		}()
	}

	wg.Wait()

	// Drain the channel to count received messages
	received := 0
	for {
		select {
		case <-session.inputCh:
			received++
		default:
			goto counted
		}
	}
counted:
	t.Logf("received %d / %d messages (channel capacity = %d)", received, goroutines*messagesPerGoroutine, inputChSize)
	// At minimum, some messages should land (channel cap is 5000, total = 1000)
	assert.Greater(t, received, 0, "expected at least some messages to be received")
}

// TestSyncBroadcastWaitGroup verifies syncBroadcast dispatches to all subs
// and waits for completion before returning.
func TestSyncBroadcastWaitGroup(t *testing.T) {
	session := newMinimalSession(t)
	defer session.cancel()

	const subCount = 5
	subs := make([]MatrixSub, subCount)
	for i := 0; i < subCount; i++ {
		subs[i] = &mockSub{name: fmt.Sprintf("sub-%d", i)}
	}
	session.subs = subs

	const batchSize = 10
	batch := make([]*matrixpb.MatrixTelemetryRequest, batchSize)
	for i := range batch {
		batch[i] = makeTestMsg()
	}

	// syncBroadcast must complete and all subs must have processed all messages
	session.syncBroadcast(batch)

	for i, s := range subs {
		sub := s.(*mockSub)
		count := sub.getProcessCount()
		assert.Equal(t, int32(batchSize), count, "sub[%d] should have processed %d messages", i, batchSize)
	}
}

// TestSyncBroadcastErrorTolerance verifies that a sub returning errors
// does not prevent other subs from processing.
func TestSyncBroadcastErrorTolerance(t *testing.T) {
	session := newMinimalSession(t)
	defer session.cancel()

	errorSub := &mockSub{name: "error-sub", shouldError: true}
	okSub := &mockSub{name: "ok-sub"}
	session.subs = []MatrixSub{errorSub, okSub}

	batch := []*matrixpb.MatrixTelemetryRequest{makeTestMsg(), makeTestMsg()}
	session.syncBroadcast(batch)

	// okSub should still have processed all messages
	assert.Equal(t, int32(2), okSub.getProcessCount(), "ok-sub should process all messages despite error-sub failures")
	// errorSub should not have incremented processCount
	assert.Equal(t, int32(0), errorSub.getProcessCount(), "error-sub should not count errored messages")
}

// TestSyncBroadcastEmptyBatch verifies syncBroadcast handles empty batch gracefully.
func TestSyncBroadcastEmptyBatch(t *testing.T) {
	session := newMinimalSession(t)
	defer session.cancel()

	sub := &mockSub{name: "empty-sub"}
	session.subs = []MatrixSub{sub}

	// Should not panic with empty batch
	session.syncBroadcast(nil)
	assert.Equal(t, int32(0), sub.getProcessCount())
}

// TestSyncBroadcastNoSubs verifies syncBroadcast works with zero subs.
func TestSyncBroadcastNoSubs(t *testing.T) {
	session := newMinimalSession(t)
	defer session.cancel()

	session.subs = nil
	// Should not panic
	session.syncBroadcast([]*matrixpb.MatrixTelemetryRequest{makeTestMsg()})
}

// TestPlayerSessionDoubleMarkOffline verifies that calling MarkOffline twice
// panics (closing already-closed channel is a Go runtime panic).
// This test uses recover to catch the expected panic.
func TestPlayerSessionDoubleMarkOfflinePanics(t *testing.T) {
	session := newMinimalSession(t)
	defer session.cancel()

	session.MarkOffline()

	// Second call should panic due to close(closed channel)
	defer func() {
		r := recover()
		require.NotNil(t, r, "expected panic from double MarkOffline (close of closed channel)")
	}()

	session.MarkOffline()
}

// TestPlayerSessionSendAfterCancel verifies Send handles context cancellation gracefully.
func TestPlayerSessionSendAfterCancel(t *testing.T) {
	session := newMinimalSession(t)

	// Cancel the context (simulates session shutdown)
	session.cancel()

	// Send should still work on the inputCh as long as isOnline is true
	// (cancel doesn't directly affect Send, only MarkOffline does)
	msg := makeTestMsg()
	session.Send(msg)

	// Verify message landed
	select {
	case <-session.inputCh:
		// OK
	default:
		t.Fatal("expected message to be accepted even after context cancel")
	}
}
