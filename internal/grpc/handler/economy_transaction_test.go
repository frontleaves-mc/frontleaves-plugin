package handler

import (
	"context"
	"io"
	"testing"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	economypb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/economy/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockClientStreamingServer 模拟 grpc.ClientStreamingServer 接口。
type mockClientStreamingServer struct {
	requests  []*economypb.RecordTransactionRequest
	recvIndex int
	response  *economypb.RecordTransactionResponse
	sendErr   error
	ctx       context.Context
	headers   metadata.MD
	trailers  metadata.MD
}

func (m *mockClientStreamingServer) Recv() (*economypb.RecordTransactionRequest, error) {
	if m.recvIndex >= len(m.requests) {
		return nil, io.EOF
	}
	req := m.requests[m.recvIndex]
	m.recvIndex++
	return req, nil
}

func (m *mockClientStreamingServer) SendAndClose(res *economypb.RecordTransactionResponse) error {
	m.response = res
	return m.sendErr
}

func (m *mockClientStreamingServer) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *mockClientStreamingServer) SetHeader(md metadata.MD) error  { m.headers = md; return nil }
func (m *mockClientStreamingServer) SendHeader(md metadata.MD) error { m.headers = md; return nil }
func (m *mockClientStreamingServer) SetTrailer(md metadata.MD)       { m.trailers = md }
func (m *mockClientStreamingServer) SendMsg(msg interface{}) error   { return nil }
func (m *mockClientStreamingServer) RecvMsg(msg interface{}) error   { return nil }

func newValidRequest(playerUUID, idempotencyKey string) *economypb.RecordTransactionRequest {
	return &economypb.RecordTransactionRequest{
		PlayerUuid:     playerUUID,
		IdempotencyKey: idempotencyKey,
		PlayerName:     "TestPlayer",
		Amount:         100,
		Type:           economypb.TransactionType_TRANSACTION_TYPE_TRANSFER,
		Timestamp:      time.Now().UnixMilli(),
	}
}

func newHandlerWithSQLite(t *testing.T) *EconomyTransactionHandler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(&entity.TransactionLog{})
	require.NoError(t, err)
	repo := repository.NewTransactionLogRepo(db)
	txnLogic := logic.NewTransactionLogLogic(repo)
	return &EconomyTransactionHandler{
		grpcHandler: grpcHandler{
			name: "TestEconomyTransaction",
			log:  xLog.WithName(xLog.NamedGRPC, "TestEconomyTransaction"),
		},
		economyService: &economyService{
			transactionLogLogic: txnLogic,
		},
		queryCh: make(chan *balanceQueryRequest, 100),
	}
}

func TestEconomyTransactionHandler_Stream_EmptyPlayerUUID_Skipped(t *testing.T) {
	h := newHandlerWithSQLite(t)

	validUUID := uuid.New().String()
	stream := &mockClientStreamingServer{
		requests: []*economypb.RecordTransactionRequest{
			{PlayerUuid: "", IdempotencyKey: "no-uuid-key", PlayerName: "Bad", Amount: 50, Type: economypb.TransactionType_TRANSACTION_TYPE_TRANSFER},
			newValidRequest(validUUID, "valid-key"),
		},
	}

	err := h.RecordTransactionStream(stream)
	require.NoError(t, err)
	require.NotNil(t, stream.response)
	assert.True(t, stream.response.Success)
}

func TestEconomyTransactionHandler_Stream_EmptyIdempotencyKey_Skipped(t *testing.T) {
	h := newHandlerWithSQLite(t)

	validUUID := uuid.New().String()
	stream := &mockClientStreamingServer{
		requests: []*economypb.RecordTransactionRequest{
			{PlayerUuid: validUUID, IdempotencyKey: "", PlayerName: "Bad", Amount: 50, Type: economypb.TransactionType_TRANSACTION_TYPE_TRANSFER},
			newValidRequest(validUUID, "valid-ik"),
		},
	}

	err := h.RecordTransactionStream(stream)
	require.NoError(t, err)
}

func TestEconomyTransactionHandler_Stream_InvalidPlayerUUID_Skipped(t *testing.T) {
	h := newHandlerWithSQLite(t)

	stream := &mockClientStreamingServer{
		requests: []*economypb.RecordTransactionRequest{
			{PlayerUuid: "not-a-uuid", IdempotencyKey: "bad-uuid-key", PlayerName: "BadUUID", Amount: 50, Type: economypb.TransactionType_TRANSACTION_TYPE_TRANSFER},
		},
	}

	err := h.RecordTransactionStream(stream)
	require.NoError(t, err)
	require.NotNil(t, stream.response)
	assert.True(t, stream.response.Success)
}

func TestEconomyTransactionHandler_Stream_EmptyStream(t *testing.T) {
	h := newHandlerWithSQLite(t)

	stream := &mockClientStreamingServer{requests: nil}

	err := h.RecordTransactionStream(stream)
	require.NoError(t, err)
	require.NotNil(t, stream.response)
	assert.True(t, stream.response.Success)
}

func TestEconomyTransactionHandler_Stream_AllSkipped(t *testing.T) {
	h := newHandlerWithSQLite(t)

	stream := &mockClientStreamingServer{
		requests: []*economypb.RecordTransactionRequest{
			{PlayerUuid: "", IdempotencyKey: "b1", PlayerName: "Bad1", Amount: 50},
			{PlayerUuid: "invalid", IdempotencyKey: "b2", PlayerName: "Bad2", Amount: 50},
		},
	}

	err := h.RecordTransactionStream(stream)
	require.NoError(t, err)
	require.NotNil(t, stream.response)
	assert.True(t, stream.response.Success)
}

func TestEconomyTransactionHandler_Stream_Idempotency(t *testing.T) {
	h := newHandlerWithSQLite(t)

	validUUID := uuid.New().String()
	stream1 := &mockClientStreamingServer{
		requests: []*economypb.RecordTransactionRequest{
			newValidRequest(validUUID, "idem-stream-key"),
		},
	}
	require.NoError(t, h.RecordTransactionStream(stream1))

	stream2 := &mockClientStreamingServer{
		requests: []*economypb.RecordTransactionRequest{
			newValidRequest(validUUID, "idem-stream-key"),
		},
	}
	err := h.RecordTransactionStream(stream2)
	require.NoError(t, err)
	assert.True(t, stream2.response.Success)
}

func TestEconomyTransactionHandler_Construction(t *testing.T) {
	h := newHandlerWithSQLite(t)
	require.NotNil(t, h, "EconomyTransactionHandler should not be nil")
	assert.NotNil(t, h.transactionLogLogic, "transactionLogLogic should not be nil")
}
