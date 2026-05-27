package checker

import (
	"context"
	"testing"

	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/stretchr/testify/assert"
)

// newTestSpeedSub 创建一个用于测试的 SpeedSub，使用低 VL 阈值以快速触发。
func newTestSpeedSub() *SpeedSub {
	vl := component.NewVLTracker(0.5, 1.0) // decay=0.5, threshold=1.0 (low for testing)
	baseChecker := NewBaseChecker(nil, vl)

	return &SpeedSub{
		BaseChecker: *baseChecker,
		baseSpeed: 4.317,
		tolerance: 0.15,
	}
}

func makeTickMsg(posX, posY, posZ float64, ts int64, opts ...func(*matrixpb.TelemetryTick)) *matrixpb.MatrixTelemetryRequest {
	tick := &matrixpb.TelemetryTick{
		PlayerUuid: "test-uuid",
		PlayerName: "TestPlayer",
		PosX:       posX,
		PosY:       posY,
		PosZ:       posZ,
		Timestamp:  ts,
	}
	for _, opt := range opts {
		opt(tick)
	}
	return &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Payload: &matrixpb.MatrixTelemetryRequest_TelemetryTick{
			TelemetryTick: tick,
		},
	}
}

// TestSpeedSub_NormalWalking 验证正常行走速度（4.0 blocks/s）不会触发 flag。
func TestSpeedSub_NormalWalking(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 建立初始位置 at t=1000ms
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: 正常行走距离 (4.0 blocks in 1000ms = 4.0 blocks/s)
	_ = sub.Process(ctx, makeTickMsg(104.0, 64.0, 200.0, 2000))

	// 阈值 = 4.317 * 1.0 * 1.15 ≈ 4.965, 4.0 < 4.965 → reward
	assert.Equal(t, 0.0, sub.VL().Violations(), "normal walk should not flag")
}

// TestSpeedSub_SprintExceeds 验证高速冲刺（8.0 blocks/s）最终触发 flag。
func TestSpeedSub_SprintExceeds(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置 at t=1000ms
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: 冲刺 8.0 blocks in 1000ms = 8.0 blocks/s (非 sprinting 状态)
	// threshold = 4.317 * 1.0 * 1.15 ≈ 4.965, 8.0 >> 4.965 → flag
	_ = sub.Process(ctx, makeTickMsg(108.0, 64.0, 200.0, 2000))

	assert.Greater(t, sub.VL().Violations(), 0.0, "excessive speed should increase VL")
}

// TestSpeedSub_FlyingSkip 验证飞行 tick 重置位置追踪且不检测。
func TestSpeedSub_FlyingSkip(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置 at t=1000ms
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: 飞行 tick → should reset
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.IsFlying = true
	}))

	assert.False(t, sub.hasLastPos, "flying tick should reset hasLastPos")
	assert.Equal(t, int64(0), sub.lastTimestamp, "flying tick should reset lastTimestamp")
}

// TestSpeedSub_TeleportReset 验证传送 payload 重置位置状态。
func TestSpeedSub_TeleportReset(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// 建立位置
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))
	_ = sub.Process(ctx, makeTickMsg(104.0, 64.0, 200.0, 2000))
	assert.True(t, sub.hasLastPos, "should have last pos after ticks")

	// 发送传送事件
	tpMsg := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Payload: &matrixpb.MatrixTelemetryRequest_Teleport{
			Teleport: &matrixpb.PlayerTeleportEvent{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  2000,
			},
		},
	}
	_ = sub.Process(ctx, tpMsg)

	assert.False(t, sub.hasLastPos, "teleport should reset hasLastPos")
	assert.Equal(t, int64(0), sub.lastTimestamp, "teleport should reset lastTimestamp")
}

// TestSpeedSub_RewardDecay 验证正常行为降低 VL。
func TestSpeedSub_RewardDecay(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// 先手动 flag 一次建立 VL
	sub.VL().Flag(0.8)
	initialVL := sub.VL().Violations()
	assert.Equal(t, 0.8, initialVL)

	// tick 1: 初始位置 at t=1000ms
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: 正常速度 → reward → VL decay
	_ = sub.Process(ctx, makeTickMsg(104.0, 64.0, 200.0, 2000))

	// VL should have decayed by 0.5 (decay rate) from the reward
	assert.Less(t, sub.VL().Violations(), initialVL, "normal speed should decay VL")
}

// TestSpeedSub_SpeedPotionBonus 验证 SPEED 效果增加阈值。
func TestSpeedSub_SpeedPotionBonus(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: SPEED II 效果下的速度 (amplifier=1, bonus = 4.317 * 0.2 * 2 = 1.7268)
	// threshold = 4.317 * 1.0 * 1.15 + 1.7268 ≈ 6.692
	// 以 5.5 blocks/s 行进 → 5.5 < 6.692 → 不触发
	_ = sub.Process(ctx, makeTickMsg(105.5, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.ActiveEffects = []string{"SPEED:1"}
	}))

	// 5.5 < threshold with SPEED bonus → reward, not flag
	assert.Equal(t, 0.0, sub.VL().Violations(), "speed with SPEED effect should not flag")
}

// TestSpeedSub_WaterExempt 验证水中状态豁免检测。
func TestSpeedSub_WaterExempt(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: 在水中 → 豁免
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.IsInWater = true
	}))

	assert.False(t, sub.hasLastPos, "in-water tick should reset hasLastPos")
}

// TestSpeedSub_SneakingLowerThreshold 验证潜行使用更低阈值。
func TestSpeedSub_SneakingLowerThreshold(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: 潜行但速度过快
	// 潜行 threshold = 4.317 * 0.3 * 1.15 ≈ 1.489
	// 移动 2.0 blocks in 1000ms = 2.0 blocks/s > 1.489 → flag
	_ = sub.Process(ctx, makeTickMsg(102.0, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.IsSneaking = true
	}))

	assert.Greater(t, sub.VL().Violations(), 0.0, "fast sneak should flag")
}

// TestSpeedSub_SprintMultiplier 验证冲刺使用更高阈值且不误报。
func TestSpeedSub_SprintMultiplier(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: 冲刺速度 5.5 blocks/s
	// sprint threshold = 4.317 * 1.3 * 1.15 ≈ 6.454
	// 5.5 < 6.454 → 不触发
	_ = sub.Process(ctx, makeTickMsg(105.5, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.IsSprinting = true
	}))

	assert.Equal(t, 0.0, sub.VL().Violations(), "sprint at normal speed should not flag")
}

// TestSpeedSub_SpeedPotionNoAmplifier 验证 SPEED 效果无 amplifier 时默认为 0。
func TestSpeedSub_SpeedPotionNoAmplifier(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: "SPEED" 无 amplifier → bonus = 4.317 * 0.2 * 1 = 0.8634
	// threshold = 4.317 * 1.0 * 1.15 + 0.8634 ≈ 5.828
	// 5.0 < 5.828 → 不触发
	_ = sub.Process(ctx, makeTickMsg(105.0, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.ActiveEffects = []string{"SPEED"}
	}))

	assert.Equal(t, 0.0, sub.VL().Violations(), "SPEED without amplifier should give small bonus")
}

// TestSpeedSub_SpeedPotionInvalidAmplifier 验证无效 amplifier 保守不加成。
func TestSpeedSub_SpeedPotionInvalidAmplifier(t *testing.T) {
	sub := newTestSpeedSub()
	ctx := context.Background()

	// tick 1: 初始位置
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000))

	// tick 2: "SPEED:abc" 无效 amplifier → bonus = 0
	// threshold = 4.317 * 1.0 * 1.15 ≈ 4.965
	// 4.5 < 4.965 → 不触发
	_ = sub.Process(ctx, makeTickMsg(104.5, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.ActiveEffects = []string{"SPEED:abc"}
	}))

	assert.Equal(t, 0.0, sub.VL().Violations(), "invalid amplifier should not give bonus")
}
