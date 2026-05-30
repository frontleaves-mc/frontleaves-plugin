package matrix

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	inputChSize  = 5000
	popBatchSize = 100
	drainTimeout = 5 * time.Second
)

// manageInterval manageLoop 的轮询间隔，可通过 MATRIX_MANAGE_INTERVAL 环境变量配置
var manageInterval time.Duration

func init() {
	raw := xEnv.GetEnvString(bConst.EnvMatrixManageInterval, "5s")
	manageInterval = parseManageInterval(raw)
	log := xLog.WithName(xLog.NamedINIT, "PlayerSession")
	log.Info(context.Background(), fmt.Sprintf("manageInterval 配置为 %v", manageInterval))
}

// parseManageInterval 解析 manageInterval 配置值
// 当 raw 无法解析或解析结果 <= 0 时，返回默认值 5s
func parseManageInterval(raw string) time.Duration {
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		log := xLog.WithName(xLog.NamedINIT, "PlayerSession")
		log.Warn(context.Background(), fmt.Sprintf("manageInterval 配置无效 [%s]，使用默认值 5s", raw))
		return 5 * time.Second
	}
	return parsed
}

// PlayerSession 玩家会话，per server+player 的隔离处理单元
type PlayerSession struct {
	ctx          context.Context
	cancel       context.CancelFunc
	playerUUID   uuid.UUID
	playerName   string
	serverName   string
	sessionKey   string
	log          *xLog.LogNamedLogger
	rdb          *redis.Client
	wg           sync.WaitGroup
	mu           sync.RWMutex
	isOnline     bool
	isDraining   bool
	inputCh      chan *matrixpb.MatrixTelemetryRequest
	bufferKey    string
	monitorKey   string
	subs         []MatrixSub
	monitorCache *cache.MatrixMonitorCache
}

// NewPlayerSession 创建 PlayerSession 实例
func NewPlayerSession(
	ctx context.Context,
	serverName string,
	playerUUID uuid.UUID,
	playerName string,
	rdb *redis.Client,
	log *xLog.LogNamedLogger,
	subs []MatrixSub,
	monitorCache *cache.MatrixMonitorCache,
) *PlayerSession {
	sessionKey := fmt.Sprintf("%s:%s", serverName, playerUUID.String())
	childCtx, cancel := context.WithCancel(ctx)

	return &PlayerSession{
		ctx:          childCtx,
		cancel:       cancel,
		playerUUID:   playerUUID,
		playerName:   playerName,
		serverName:   serverName,
		sessionKey:   sessionKey,
		log:          log,
		rdb:          rdb,
		isOnline:     true,
		isDraining:   false,
		inputCh:      make(chan *matrixpb.MatrixTelemetryRequest, inputChSize),
		bufferKey:    string(bConst.CacheMatrixPlayerBuffer.Get(sessionKey)),
		monitorKey:   string(bConst.CacheMatrixPlayerMonitor.Get(sessionKey)),
		subs:         subs,
		monitorCache: monitorCache,
	}
}

// Start 启动 base 和 manage 协程
func (s *PlayerSession) Start() {
	s.wg.Add(2)
	go s.baseLoop()
	go s.manageLoop()
}

// Stop 停止 session（带 5s 超时）
func (s *PlayerSession) Stop() {
	s.cancel()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(drainTimeout):
		s.log.Warn(s.ctx, "PlayerSession.Stop - 排水超时，强制退出")
	}
}

// Send 非阻塞发送消息到 inputCh
func (s *PlayerSession) Send(msg *matrixpb.MatrixTelemetryRequest) {
	s.mu.RLock()
	online := s.isOnline
	s.mu.RUnlock()

	if !online {
		return
	}

	select {
	case s.inputCh <- msg:
	default:
		s.log.Warn(s.ctx, "PlayerSession.Send - inputCh 已满，丢弃消息")
	}
}

// MarkOffline 标记玩家离线，触发排水模式
func (s *PlayerSession) MarkOffline() {
	s.mu.Lock()
	s.isOnline = false
	s.isDraining = true
	s.mu.Unlock()
	close(s.inputCh)
}

// baseLoop 从 inputCh 读取数据 → Redis List 缓冲
func (s *PlayerSession) baseLoop() {
	defer s.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			s.log.Error(s.ctx, fmt.Sprintf("PlayerSession.baseLoop panic: %v", r))
		}
	}()

	for msg := range s.inputCh {
		data, err := protojson.Marshal(msg)
		if err != nil {
			s.log.Warn(s.ctx, "baseLoop - 序列化失败: "+err.Error())
			continue
		}

		pipe := s.rdb.Pipeline()
		pipe.RPush(s.ctx, s.bufferKey, data)
		pipe.LTrim(s.ctx, s.bufferKey, 0, inputChSize-1)
		if _, err := pipe.Exec(s.ctx); err != nil {
			s.log.Warn(s.ctx, "baseLoop - Redis 写入失败: "+err.Error())
		}
	}
}

// manageLoop 定时从 Redis List 批量消费数据并广播到所有 sub
func (s *PlayerSession) manageLoop() {
	defer s.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			s.log.Error(s.ctx, fmt.Sprintf("PlayerSession.manageLoop panic: %v", r))
		}
	}()

	ticker := time.NewTicker(manageInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.drainAll()
			return
		case <-ticker.C:
			batch := s.popBatch()
			if len(batch) > 0 {
				s.syncBroadcast(batch)
			}
		}
	}
}

// popBatch 从 Redis List 批量弹出消息
func (s *PlayerSession) popBatch() []*matrixpb.MatrixTelemetryRequest {
	results, err := s.rdb.LPopCount(s.ctx, s.bufferKey, popBatchSize).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		s.log.Warn(s.ctx, "popBatch - LPOP 失败: "+err.Error())
		return nil
	}

	batch := make([]*matrixpb.MatrixTelemetryRequest, 0, len(results))
	for _, data := range results {
		var msg matrixpb.MatrixTelemetryRequest
		if err := protojson.Unmarshal([]byte(data), &msg); err != nil {
			s.log.Warn(s.ctx, "popBatch - 反序列化失败: "+err.Error())
			continue
		}
		batch = append(batch, &msg)
	}
	return batch
}

// syncBroadcast 同步广播一批消息到所有 sub（并行处理，WaitGroup 等待）
func (s *PlayerSession) syncBroadcast(batch []*matrixpb.MatrixTelemetryRequest) {
	var wg sync.WaitGroup
	for _, sub := range s.subs {
		wg.Add(1)
		go func(sub MatrixSub) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					s.log.Error(s.ctx, fmt.Sprintf("Sub [%s] panic: %v", sub.Name(), r))
				}
			}()
			for _, msg := range batch {
				if err := sub.Process(s.ctx, msg); err != nil {
					s.log.Warn(s.ctx, fmt.Sprintf("Sub [%s] process error: %v", sub.Name(), err))
				}
			}
		}(sub)
	}
	wg.Wait()
}

// drainAll 排水：消费 Redis List 残留数据后调用 sub.Drain
func (s *PlayerSession) drainAll() {
	for {
		length, err := s.rdb.LLen(s.ctx, s.bufferKey).Result()
		if err != nil || length == 0 {
			break
		}
		batch := s.popBatch()
		if len(batch) > 0 {
			s.syncBroadcast(batch)
		}
	}

	// 最终 Drain
	for _, sub := range s.subs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.log.Error(s.ctx, fmt.Sprintf("Sub [%s] drain panic: %v", sub.Name(), r))
				}
			}()
			if err := sub.Drain(s.ctx); err != nil {
				s.log.Warn(s.ctx, fmt.Sprintf("Sub [%s] drain error: %v", sub.Name(), err))
			}
		}()
	}

	// 清理 Redis key
	s.rdb.Del(s.ctx, s.bufferKey)
	s.rdb.Del(s.ctx, s.monitorKey)
}
