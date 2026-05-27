package checker

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// SpeedSub 基于动态摩擦力的速度阈值检测，替代旧的固定 12.0 blocks/s 阈值。
// 使用 VLTracker 实现渐进式违规累加与衰减。
type SpeedSub struct {
	BaseChecker

	mu           sync.Mutex
	prevPosX     float64
	prevPosY     float64
	prevPosZ     float64
	lastTimestamp int64
	hasLastPos   bool

	// Config (from env vars)
	baseSpeed float64
	tolerance float64
}

// NewSpeedSub 创建 SpeedSub 实例，从环境变量读取配置。
func NewSpeedSub(warner *components.AntiCheatWarning) *SpeedSub {
	baseSpeed := xEnv.GetEnvFloat(bConst.EnvMatrixAcSpeedBaseSpeed, 4.317)
	tolerance := xEnv.GetEnvFloat(bConst.EnvMatrixAcSpeedTolerance, 0.15)
	vlThreshold := xEnv.GetEnvFloat(bConst.EnvMatrixAcSpeedVlThreshold, 5.0)

	vl := component.NewVLTracker(0.5, vlThreshold)
	baseChecker := NewBaseChecker(warner, vl)

	return &SpeedSub{
		BaseChecker: *baseChecker,
		baseSpeed:   baseSpeed,
		tolerance:   tolerance,
	}
}

// Name 返回 sub 名称标识。
func (s *SpeedSub) Name() string {
	return "speed"
}

// Process 处理单条遥测数据，根据 Payload 类型分发处理。
func (s *SpeedSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg.Payload.(type) {
	case *matrixpb.MatrixTelemetryRequest_TelemetryTick:
		s.checkSpeed(ctx, msg.GetTelemetryTick())
	case *matrixpb.MatrixTelemetryRequest_Teleport,
		*matrixpb.MatrixTelemetryRequest_Respawn,
		*matrixpb.MatrixTelemetryRequest_GameModeChange:
		// 传送/重生/游戏模式变更 → 重置位置追踪
		s.hasLastPos = false
		s.lastTimestamp = 0
	}

	return nil
}

// Drain 排水完成时调用，无特殊刷盘逻辑。
func (s *SpeedSub) Drain(_ context.Context) error {
	return nil
}

// checkSpeed 执行动态摩擦力速度检测核心逻辑。
//
// 动态阈值公式：
//
//	threshold = baseSpeed × movementMultiplier × (1 + tolerance) + speedPotionBonus
//
// movementMultiplier:
//   - 潜行: 0.3
//   - 冲刺: 1.3
//   - 行走: 1.0
//
// speedPotionBonus:
//
//	如果 active_effects 中包含 SPEED 效果，
//	bonus = baseSpeed × 0.2 × (amplifier + 1)
func (s *SpeedSub) checkSpeed(ctx context.Context, tick *matrixpb.TelemetryTick) {
	if tick == nil {
		return
	}

	// 豁免检查：飞行或水中时跳过检测
	if tick.GetIsFlying() || tick.GetIsInWater() {
		s.hasLastPos = false
		s.lastTimestamp = 0
		return
	}

	posX := tick.GetPosX()
	posY := tick.GetPosY()
	posZ := tick.GetPosZ()
	ts := tick.GetTimestamp()

	if s.hasLastPos && s.lastTimestamp > 0 {
		dt := float64(ts-s.lastTimestamp) / 1000.0
		if dt > 0 {
			// 仅计算水平速度（忽略 Y 轴跳跃/下落）
			dx := posX - s.prevPosX
			dz := posZ - s.prevPosZ
			horizontalSpeed := math.Sqrt(dx*dx+dz*dz) / dt

			// 动态移动倍率
			multiplier := 1.0 // 行走
			if tick.GetIsSprinting() {
				multiplier = 1.3
			}
			if tick.GetIsSneaking() {
				multiplier = 0.3
			}

			// 基础阈值
			threshold := s.baseSpeed * multiplier * (1.0 + s.tolerance)

			// 速度药水加成（保守解析）
			threshold += s.parseSpeedPotionBonus(tick.GetActiveEffects())

			if horizontalSpeed > threshold {
				s.FlagViaVL(ctx, "SPEED", fmt.Sprintf(
					"水平移动速度 %.2f blocks/s 超过动态阈值 %.2f",
					horizontalSpeed, threshold,
				), 1.0, map[string]any{
					"speed":          horizontalSpeed,
					"threshold":      threshold,
					"base_speed":     s.baseSpeed,
					"movement_mult":  multiplier,
					"tolerance":      s.tolerance,
					"pos_x":          posX,
					"pos_y":          posY,
					"pos_z":          posZ,
					"is_sprinting":   tick.GetIsSprinting(),
					"is_sneaking":    tick.GetIsSneaking(),
					"active_effects": tick.GetActiveEffects(),
				})
			} else {
				s.RewardViaVL()
			}
		}
	}

	// 更新位置记录
	s.prevPosX = posX
	s.prevPosY = posY
	s.prevPosZ = posZ
	s.lastTimestamp = ts
	s.hasLastPos = true
}

// parseSpeedPotionBonus 从 active_effects 中解析 SPEED 效果的 amplifier。
// 格式预期: "SPEED:2" 或 "SPEED" (amplifier=0)
// 解析失败时保守返回 0（不给予加成）。
func (s *SpeedSub) parseSpeedPotionBonus(activeEffects []string) float64 {
	for _, effect := range activeEffects {
		// 查找 SPEED 前缀的效果
		if !strings.HasPrefix(effect, "SPEED") {
			continue
		}

		// 尝试解析 amplifier
		parts := strings.SplitN(effect, ":", 2)
		if len(parts) < 2 {
			// "SPEED" 无冒号 → amplifier = 0
			return s.baseSpeed * 0.2 * (0 + 1)
		}

		amplifier, err := strconv.Atoi(parts[1])
		if err != nil {
			// 解析失败 → 保守不加成
			return 0
		}

		return s.baseSpeed * 0.2 * float64(amplifier+1)
	}

	return 0
}
