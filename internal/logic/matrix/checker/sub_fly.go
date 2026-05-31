package checker

import (
	"context"
	"fmt"
	"math"
	"sync"

	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// FlySub 基于状态机的飞行/悬停检测，使用 VLTracker 实现渐进式违规累加与衰减。
//
// 检测逻辑：
//   - 豁免：飞行状态 / 水中 / 攀爬 → 重置状态，跳过检测
//   - 地面：更新 lastGroundY，清零 consecutiveAirTicks，RewardViaVL
//   - 空中上升：连续滞空 >2 tick 且 ΔY 显著违背重力衰减模型 → FlagViaVL
//   - 悬停：连续滞空超过 hoverThreshold 且 |ΔY| < 0.01 → FlagViaVL
//
// 重力衰减模型：expectedΔY ≈ lastDeltaY × 0.98 − 0.08（模拟 MC 重力）。
// 若实际 ΔY 显著超过预期值，判定为异常上升。
type FlySub struct {
	BaseChecker

	mu sync.Mutex

	// 状态追踪
	lastGroundY         float64
	consecutiveAirTicks int
	lastDeltaY          float64
	prevPosY            float64
	hasLastGround       bool
	hasPrevPosY         bool

	// 配置（来自环境变量）
	vlThreshold    float64
	hoverThreshold int
}

// NewFlySub 创建 FlySub 实例，从环境变量读取配置。
func NewFlySub(warner *components.AntiCheatWarning) *FlySub {
	vlThreshold := xEnv.GetEnvFloat(bConst.EnvMatrixAcFlyVlThreshold, 5.0)
	hoverThreshold := xEnv.GetEnvInt(bConst.EnvMatrixAcFlyHoverThreshold, 40)

	vl := component.NewVLTracker(0.5, vlThreshold)
	baseChecker := NewBaseChecker(warner, vl)

	return &FlySub{
		BaseChecker:    *baseChecker,
		vlThreshold:    vlThreshold,
		hoverThreshold: hoverThreshold,
	}
}

// Name 返回 sub 名称标识。
func (f *FlySub) Name() string {
	return "fly"
}

// Process 处理单条遥测数据，根据 Payload 类型分发处理。
func (f *FlySub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, tick := range msg.GetTelemetryTicks() {
		f.checkFly(ctx, tick)
	}
	if len(msg.GetTeleports()) > 0 || len(msg.GetRespawns()) > 0 || len(msg.GetGameModeChanges()) > 0 {
		f.resetState()
	}

	return nil
}

// Drain 排水完成时调用，无特殊刷盘逻辑。
func (f *FlySub) Drain(_ context.Context) error {
	return nil
}

// resetState 重置所有飞行检测追踪状态。
func (f *FlySub) resetState() {
	f.consecutiveAirTicks = 0
	f.hasLastGround = false
	f.hasPrevPosY = false
	f.lastDeltaY = 0
}

// checkFly 执行飞行/悬停检测核心逻辑。
func (f *FlySub) checkFly(ctx context.Context, tick *matrixpb.TelemetryTick) {
	if tick == nil {
		return
	}

	// 1. 豁免检查：飞行 / 水中 / 攀爬 → 重置状态，跳过检测
	if tick.GetIsFlying() || tick.GetIsInWater() || tick.GetIsClimbing() {
		f.resetState()
		return
	}

	posY := tick.GetPosY()

	// 2. 地面检查：更新 lastGroundY，清零空中计数，奖励 VL
	if tick.GetIsOnGround() {
		f.lastGroundY = posY
		f.consecutiveAirTicks = 0
		f.hasLastGround = true
		f.lastDeltaY = 0
		f.hasPrevPosY = false
		f.RewardViaVL()
		return
	}

	// 计算当前 ΔY
	var deltaY float64
	if f.hasPrevPosY {
		deltaY = posY - f.prevPosY
	}

	// 3. 空中上升检查：连续滞空 >2 tick 且 ΔY 违背重力衰减
	if f.consecutiveAirTicks > 2 && f.hasPrevPosY {
		// 重力衰减模型：expectedΔY ≈ lastDeltaY * 0.98 - 0.08
		expectedDeltaY := f.lastDeltaY*0.98 - 0.08
		// 上升判定：ΔY > 0.1 且显著超过重力衰减预期
		if deltaY > 0.1 && deltaY > expectedDeltaY+0.15 {
			f.FlagViaVL(ctx, "FLY", fmt.Sprintf(
				"异常上升：ΔY=%.4f，预期ΔY=%.4f，连续滞空=%d tick",
				deltaY, expectedDeltaY, f.consecutiveAirTicks,
			), 1.0, map[string]any{
				"delta_y":             deltaY,
				"expected_delta_y":    expectedDeltaY,
				"consecutive_air_ticks": f.consecutiveAirTicks,
				"pos_y":               posY,
				"last_ground_y":       f.lastGroundY,
			})
		}
	}

	// 4. 悬停检查：连续滞空超过阈值且 |ΔY| 极小
	if f.consecutiveAirTicks > f.hoverThreshold && f.hasPrevPosY {
		if math.Abs(deltaY) < 0.01 {
			f.FlagViaVL(ctx, "FLY", fmt.Sprintf(
				"悬停检测：|ΔY|=%.4f < 0.01，连续滞空=%d tick",
				math.Abs(deltaY), f.consecutiveAirTicks,
			), 1.0, map[string]any{
				"delta_y":             deltaY,
				"consecutive_air_ticks": f.consecutiveAirTicks,
				"pos_y":               posY,
				"last_ground_y":       f.lastGroundY,
			})
		}
	}

	// 更新状态
	f.lastDeltaY = deltaY
	f.prevPosY = posY
	f.hasPrevPosY = true
	f.consecutiveAirTicks++
}
