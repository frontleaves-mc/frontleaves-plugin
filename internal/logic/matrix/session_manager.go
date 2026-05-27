package matrix

import (
	"context"
	"fmt"
	"sync"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/checker"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// MatrixSessionManager 玩家会话管理器（全局单例）
type MatrixSessionManager struct {
	mu           sync.RWMutex
	sessions     map[string]*PlayerSession
	rdb          *redis.Client
	log          *xLog.LogNamedLogger
	db           *gorm.DB
	monitorCache *cache.MatrixMonitorCache
	statRepo     *repository.MatrixStatisticRepo
	warningRepo  *repository.MatrixWarningRepo
}

// NewMatrixSessionManager 创建 MatrixSessionManager 实例
func NewMatrixSessionManager(ctx context.Context, db *gorm.DB, rdb *redis.Client, monitorCache *cache.MatrixMonitorCache, statRepo *repository.MatrixStatisticRepo, warningRepo *repository.MatrixWarningRepo) *MatrixSessionManager {
	return &MatrixSessionManager{
		sessions:     make(map[string]*PlayerSession),
		rdb:          rdb,
		log:          xLog.WithName(xLog.NamedLOGC, "MatrixSessionManager"),
		db:           db,
		monitorCache: monitorCache,
		statRepo:     statRepo,
		warningRepo:  warningRepo,
	}
}

// GetOrCreate 获取或创建玩家会话（写锁）
func (m *MatrixSessionManager) GetOrCreate(ctx context.Context, serverName string, playerUUID uuid.UUID, playerName string) *PlayerSession {
	sessionKey := fmt.Sprintf("%s:%s", serverName, playerUUID.String())

	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[sessionKey]; ok {
		return session
	}

	// 为每个玩家创建独立的 sub 实例（subs 持有 per-player 状态）
	subs := []MatrixSub{
		NewStatisticsSub(playerUUID, playerName, serverName, m.statRepo),
	}

	// Matrix 反作弊检测：共享 AntiCheatWarning 实例，各 checker Sub 独立持有 VLTracker
	if xEnv.GetEnvBool(bConst.EnvMatrixAcEnabled, true) {
		warner := components.NewAntiCheatWarning(playerUUID, playerName, serverName, sessionKey, m.warningRepo, m.monitorCache)
		antiCheatSubs := []MatrixSub{
			checker.NewSpeedSub(warner),
			checker.NewReachSub(warner),
			checker.NewTimerSub(warner),
			checker.NewFlySub(warner),
			checker.NewXRaySub(warner),
			checker.NewKillAuraSub(warner),
			checker.NewAimbotSub(warner),
			checker.NewAutoClickerSub(warner),
			checker.NewFastBreakSub(warner),
			checker.NewNoFallSub(warner),
		}
		subs = append(subs, antiCheatSubs...)
	}

	session := NewPlayerSession(ctx, serverName, playerUUID, playerName, m.rdb, m.log, subs, m.monitorCache)
	session.Start()
	m.sessions[sessionKey] = session

	m.log.Info(ctx, "GetOrCreate - 创建新会话: "+sessionKey)
	return session
}

// Remove 移除玩家会话
func (m *MatrixSessionManager) Remove(sessionKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionKey)
}

// Get 获取玩家会话（读锁）
func (m *MatrixSessionManager) Get(serverName string, playerUUID uuid.UUID) *PlayerSession {
	sessionKey := fmt.Sprintf("%s:%s", serverName, playerUUID.String())

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.sessions[sessionKey]
}

// ShutdownAll 关闭所有会话
func (m *MatrixSessionManager) ShutdownAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, session := range m.sessions {
		session.Stop()
		delete(m.sessions, key)
	}
}

// --- 全局单例 ---

var (
	globalMSMMu sync.RWMutex
	globalMSM   *MatrixSessionManager
)

// SetGlobalMatrixSessionManager 设置全局单例
func SetGlobalMatrixSessionManager(m *MatrixSessionManager) {
	globalMSMMu.Lock()
	defer globalMSMMu.Unlock()
	globalMSM = m
}

// GetGlobalMatrixSessionManager 获取全局单例
func GetGlobalMatrixSessionManager() *MatrixSessionManager {
	globalMSMMu.RLock()
	defer globalMSMMu.RUnlock()
	return globalMSM
}
