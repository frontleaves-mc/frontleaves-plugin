package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/google/uuid"
)

const (
	reachThreshold  = 3.5
	speedThreshold  = 12.0 // blocks/second
	riskScorePerHit = 20
	maxRiskScore    = 100
)

// AntiCheatSub 反作弊监控 Sub，检测 Reach 和 Speed 异常
type AntiCheatSub struct {
	mu           sync.Mutex
	warningRepo  *repository.MatrixWarningRepo
	monitorCache *cache.MatrixMonitorCache
	playerUUID   uuid.UUID
	playerName   string
	serverName   string
	sessionKey   string
	riskScore    int32
	warningCount int32
	lastPosX     float64
	lastPosY     float64
	lastPosZ     float64
	lastTimestamp int64
	hasLastPos   bool
}

// NewAntiCheatSub 创建 AntiCheatSub 实例
func NewAntiCheatSub(
	playerUUID uuid.UUID,
	playerName, serverName, sessionKey string,
	warningRepo *repository.MatrixWarningRepo,
	monitorCache *cache.MatrixMonitorCache,
) *AntiCheatSub {
	return &AntiCheatSub{
		warningRepo:  warningRepo,
		monitorCache: monitorCache,
		playerUUID:   playerUUID,
		playerName:   playerName,
		serverName:   serverName,
		sessionKey:   sessionKey,
	}
}

// Name 返回 sub 名称
func (s *AntiCheatSub) Name() string {
	return "anti_cheat"
}

// Process 处理单条遥测数据
func (s *AntiCheatSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg.Payload.(type) {
	case *matrixpb.MatrixTelemetryRequest_EntityDamage:
		s.checkReach(ctx, msg.GetEntityDamage())

	case *matrixpb.MatrixTelemetryRequest_TelemetryTick:
		s.checkSpeed(ctx, msg.GetTelemetryTick())

	case *matrixpb.MatrixTelemetryRequest_Teleport:
		s.hasLastPos = false
		s.lastTimestamp = 0

	case *matrixpb.MatrixTelemetryRequest_GameModeChange:
		s.hasLastPos = false
		s.lastTimestamp = 0

	case *matrixpb.MatrixTelemetryRequest_Respawn:
		s.hasLastPos = false
		s.lastTimestamp = 0
	}

	return nil
}

// Drain 排水时写入最终风险分到 Redis monitor
func (s *AntiCheatSub) Drain(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateMonitor(ctx)
	return nil
}

// checkReach 检测攻击距离异常（Reach）
func (s *AntiCheatSub) checkReach(ctx context.Context, evt *matrixpb.EntityDamageEvent) {
	if evt == nil {
		return
	}

	dx := evt.GetPlayerX() - evt.GetEntityX()
	dy := evt.GetPlayerY() - evt.GetEntityY()
	dz := evt.GetPlayerZ() - evt.GetEntityZ()
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	if distance > reachThreshold {
		s.triggerWarning(ctx, "REACH", fmt.Sprintf(
			"攻击距离 %.2f blocks 超过阈值 %.1f",
			distance, reachThreshold,
		), map[string]any{
			"distance":     distance,
			"threshold":    reachThreshold,
			"entity_type":  evt.GetEntityType(),
			"damage_cause": evt.GetDamageCause(),
		})
	}
}

// checkSpeed 检测移动速度异常（Speed）
func (s *AntiCheatSub) checkSpeed(ctx context.Context, tick *matrixpb.TelemetryTick) {
	if tick == nil {
		return
	}

	if tick.GetIsFlying() {
		s.hasLastPos = false
		return
	}

	posX := tick.GetPosX()
	posY := tick.GetPosY()
	posZ := tick.GetPosZ()
	ts := tick.GetTimestamp()

	if s.hasLastPos && s.lastTimestamp > 0 {
		dt := float64(ts-s.lastTimestamp) / 1000.0
		if dt > 0 {
			dx := posX - s.lastPosX
			dy := posY - s.lastPosY
			dz := posZ - s.lastPosZ
			distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
			speed := distance / dt

			if speed > speedThreshold {
				s.triggerWarning(ctx, "SPEED", fmt.Sprintf(
					"移动速度 %.2f blocks/sec 超过阈值 %.1f",
					speed, speedThreshold,
				), map[string]any{
					"speed":     speed,
					"threshold": speedThreshold,
					"pos_x":     posX,
					"pos_y":     posY,
					"pos_z":     posZ,
				})
			}
		}
	}

	s.lastPosX = posX
	s.lastPosY = posY
	s.lastPosZ = posZ
	s.lastTimestamp = ts
	s.hasLastPos = true
}

// triggerWarning 触发警告：增加风险分，写 DB，更新 Redis monitor
func (s *AntiCheatSub) triggerWarning(ctx context.Context, warningType, description string, contextData map[string]any) {
	s.riskScore += riskScorePerHit
	if s.riskScore > maxRiskScore {
		s.riskScore = maxRiskScore
	}
	s.warningCount++

	contextJSON, _ := json.Marshal(contextData)

	warning := &entity.MatrixPlayerWarning{
		PlayerUUID:  s.playerUUID,
		PlayerName:  s.playerName,
		ServerName:  s.serverName,
		WarningType: warningType,
		Description: description,
		RiskScore:   s.riskScore,
		ContextData: contextJSON,
	}

	_ = s.warningRepo.Create(ctx, warning)

	s.updateMonitor(ctx)
}

// updateMonitor 更新 Redis 监控快照
func (s *AntiCheatSub) updateMonitor(ctx context.Context) {
	if s.monitorCache == nil {
		return
	}
	_ = s.monitorCache.Set(ctx, s.sessionKey, map[string]any{
		"risk_score":            fmt.Sprintf("%d", s.riskScore),
		"warning_count_session": fmt.Sprintf("%d", s.warningCount),
		"last_tick_processed":   fmt.Sprintf("%d", time.Now().UnixMilli()),
	})
}
