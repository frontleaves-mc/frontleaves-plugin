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
	"github.com/redis/go-redis/v9"
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
	rdb          *redis.Client
	bufferKey    string
}

// NewAntiCheatWarning 创建 AntiCheatWarning 实例。
func NewAntiCheatWarning(
	playerUUID uuid.UUID,
	playerName, serverName, sessionKey string,
	warningRepo *repository.MatrixWarningRepo,
	monitorCache *cache.MatrixMonitorCache,
	rdb *redis.Client,
	bufferKey string,
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
		rdb:          rdb,
		bufferKey:    bufferKey,
	}
}

// Trigger 记录一次警告：累加风险分（上限 100）、采集窗口事件、写入 DB、更新 Redis monitor。
func (w *AntiCheatWarning) Trigger(ctx context.Context, warningType, description string, contextData map[string]any) {
	w.riskScore += riskScorePerHit
	if w.riskScore > maxRiskScore {
		w.riskScore = maxRiskScore
	}
	w.warningCount++

	contextJSON, _ := json.Marshal(contextData)
	recordJSON := w.collectWindowEvents(ctx, time.Now())

	warning := &entity.MatrixPlayerWarning{
		PlayerUUID:  w.playerUUID,
		PlayerName:  w.playerName,
		ServerName:  w.serverName,
		WarningType: warningType,
		Description: description,
		RiskScore:   w.riskScore,
		ContextData: contextJSON,
		Record:      recordJSON,
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

// collectWindowEvents 从 Redis List 采集触发窗口内的事件，并返回精简后的 JSON 记录。
// 窗口范围：触发前 15s 至触发后 5s，共 20s。
func (w *AntiCheatWarning) collectWindowEvents(ctx context.Context, triggerTime time.Time) json.RawMessage {
	windowStart := triggerTime.Add(-15 * time.Second).UnixMilli()
	windowEnd := triggerTime.Add(5 * time.Second).UnixMilli()

	var events []map[string]any

	if w.rdb != nil && w.bufferKey != "" {
		items, err := w.rdb.LRange(ctx, w.bufferKey, 0, -1).Result()
		if err == nil {
			for _, item := range items {
				var event map[string]any
				if err := json.Unmarshal([]byte(item), &event); err != nil {
					continue
				}
				tsRaw, ok := event["timestamp"]
				if !ok {
					continue
				}
				var ts int64
				switch v := tsRaw.(type) {
				case float64:
					ts = int64(v)
				case int64:
					ts = v
				case int:
					ts = int64(v)
				default:
					continue
				}
				if ts >= windowStart && ts <= windowEnd {
					events = append(events, w.slimEvent(event))
				}
			}
		}
	}

	result := map[string]any{
		"window_start": windowStart,
		"window_end":   windowEnd,
		"event_count":  len(events),
		"events":       events,
	}

	data, _ := json.Marshal(result)
	return data
}

// slimEvent 按事件类型精简字段，保留关键信息并对敏感内容进行脱敏。
func (w *AntiCheatWarning) slimEvent(event map[string]any) map[string]any {
	eventType, _ := event["event_type"].(string)

	// 通用字段
	slimmed := map[string]any{
		"timestamp":   event["timestamp"],
		"event_type":  event["event_type"],
		"server_name": event["server_name"],
	}

	switch eventType {
	case "MOVEMENT":
		slimmed["location"] = event["location"]
		slimmed["velocity"] = event["velocity"]
	case "ATTACK":
		slimmed["target"] = event["target"]
		slimmed["damage"] = event["damage"]
	case "BLOCK_BREAK":
		slimmed["block_type"] = event["block_type"]
		slimmed["location"] = event["location"]
	case "BLOCK_PLACE":
		slimmed["block_type"] = event["block_type"]
		slimmed["location"] = event["location"]
	case "CHAT":
		slimmed["message"] = "[REDACTED]"
	case "COMMAND":
		slimmed["command"] = "[REDACTED]"
	default:
		// 其他类型保留原始事件的所有字段
		return event
	}

	return slimmed
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
