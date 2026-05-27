package components

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/google/uuid"
)

const (
	riskScorePerHit = 20
	maxRiskScore    = 100
)

// AntiCheatWarning 反作弊警告组件，负责风险分累加、DB 警告写入和 Redis monitor 更新。
// 非 goroutine-safe，调用方需自行加锁。
type AntiCheatWarning struct {
	warningRepo  *repository.MatrixWarningRepo
	monitorCache *cache.MatrixMonitorCache
	playerUUID   uuid.UUID
	playerName   string
	serverName   string
	sessionKey   string
	riskScore    int32
	warningCount int32
}

// NewAntiCheatWarning 创建 AntiCheatWarning 实例。
func NewAntiCheatWarning(
	playerUUID uuid.UUID,
	playerName, serverName, sessionKey string,
	warningRepo *repository.MatrixWarningRepo,
	monitorCache *cache.MatrixMonitorCache,
) *AntiCheatWarning {
	return &AntiCheatWarning{
		warningRepo:  warningRepo,
		monitorCache: monitorCache,
		playerUUID:   playerUUID,
		playerName:   playerName,
		serverName:   serverName,
		sessionKey:   sessionKey,
		riskScore:    0,
		warningCount: 0,
	}
}

// Trigger 记录一次警告：累加风险分（上限 100）、写入 DB、更新 Redis monitor。
func (w *AntiCheatWarning) Trigger(ctx context.Context, warningType, description string, contextData map[string]any) {
	w.riskScore += riskScorePerHit
	if w.riskScore > maxRiskScore {
		w.riskScore = maxRiskScore
	}
	w.warningCount++

	contextJSON, _ := json.Marshal(contextData)

	warning := &entity.MatrixPlayerWarning{
		PlayerUUID:  w.playerUUID,
		PlayerName:  w.playerName,
		ServerName:  w.serverName,
		WarningType: warningType,
		Description: description,
		RiskScore:   w.riskScore,
		ContextData: contextJSON,
	}

	_ = w.warningRepo.Create(ctx, warning)

	w.updateMonitor(ctx)
}

// Drain 将最终状态刷写到 Redis monitor。
func (w *AntiCheatWarning) Drain(ctx context.Context) {
	w.updateMonitor(ctx)
}

// RiskScore 返回当前风险分。
func (w *AntiCheatWarning) RiskScore() int32 {
	return w.riskScore
}

// updateMonitor 更新 Redis 监控快照。
func (w *AntiCheatWarning) updateMonitor(ctx context.Context) {
	if w.monitorCache == nil {
		return
	}
	_ = w.monitorCache.Set(ctx, w.sessionKey, map[string]any{
		"risk_score":            fmt.Sprintf("%d", w.riskScore),
		"warning_count_session": fmt.Sprintf("%d", w.warningCount),
		"last_tick_processed":   fmt.Sprintf("%d", time.Now().UnixMilli()),
	})
}
