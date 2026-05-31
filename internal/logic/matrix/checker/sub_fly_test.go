package checker

import (
	"context"
	"testing"

	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/stretchr/testify/assert"
)

// newTestFlySub 创建一个用于测试的 FlySub，使用低 VL 阈值以快速触发。
func newTestFlySub() *FlySub {
	vl := component.NewVLTracker(0.5, 1.0) // decay=0.5, threshold=1.0 (low for testing)
	baseChecker := NewBaseChecker(nil, vl)

	return &FlySub{
		BaseChecker:    *baseChecker,
		vlThreshold:    1.0,
		hoverThreshold: 5, // 低阈值方便测试悬停
	}
}

// TestFlySub_NormalGround 验证站在地面上不会触发 flag。
func TestFlySub_NormalGround(t *testing.T) {
	sub := newTestFlySub()
	ctx := context.Background()

	// tick 1: 站在地面上
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))

	// tick 2: 仍然站在地面
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 2000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))

	assert.Equal(t, 0.0, sub.VL().Violations(), "standing on ground should not flag")
	assert.True(t, sub.hasLastGround, "should have last ground after ground ticks")
	assert.Equal(t, 0, sub.consecutiveAirTicks, "consecutiveAirTicks should be 0 on ground")
}

// TestFlySub_JumpAndFall 验证正常跳跃/下落重力曲线不会触发 flag。
func TestFlySub_JumpAndFall(t *testing.T) {
	sub := newTestFlySub()
	ctx := context.Background()

	// tick 1: 地面上 Y=64.0
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))

	// tick 2: 跳跃上升 Y=64.42（正常 MC 跳跃初始速度 ~0.42 blocks/tick）
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.42, 200.0, 2000))

	// tick 3: 继续上升但减速 Y=64.72（ΔY=0.30，符合重力衰减）
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.72, 200.0, 3000))

	// tick 4: 到达最高点附近 Y=64.90（ΔY=0.18）
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.90, 200.0, 4000))

	// tick 5: 开始下落 Y=64.94（ΔY=0.04）
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.94, 200.0, 5000))

	// tick 6: 下落加速 Y=64.82（ΔY=-0.12）
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.82, 200.0, 6000))

	// tick 7: 落地
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 7000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))

	// 正常跳跃/下落不应触发飞行检测
	assert.Equal(t, 0.0, sub.VL().Violations(), "normal jump/fall should not flag")
}

// TestFlySub_FlyingExempt 验证飞行状态跳过检测并重置状态。
func TestFlySub_FlyingExempt(t *testing.T) {
	sub := newTestFlySub()
	ctx := context.Background()

	// 先建立一些空中状态
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	_ = sub.Process(ctx, makeTickMsg(100.0, 65.0, 200.0, 2000))
	assert.Equal(t, 1, sub.consecutiveAirTicks, "should have 1 air tick")

	// 飞行状态 → 重置
	_ = sub.Process(ctx, makeTickMsg(100.0, 66.0, 200.0, 3000, func(tick *matrixpb.TelemetryTick) {
		tick.IsFlying = true
	}))

	assert.Equal(t, 0, sub.consecutiveAirTicks, "flying tick should reset consecutiveAirTicks")
	assert.False(t, sub.hasLastGround, "flying tick should reset hasLastGround")
	assert.False(t, sub.hasPrevPosY, "flying tick should reset hasPrevPosY")
}

// TestFlySub_WaterExempt 验证水中状态跳过检测并重置状态。
func TestFlySub_WaterExempt(t *testing.T) {
	sub := newTestFlySub()
	ctx := context.Background()

	// 建立空中状态
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	_ = sub.Process(ctx, makeTickMsg(100.0, 65.0, 200.0, 2000))

	// 水中状态 → 重置
	_ = sub.Process(ctx, makeTickMsg(100.0, 65.5, 200.0, 3000, func(tick *matrixpb.TelemetryTick) {
		tick.IsInWater = true
	}))

	assert.Equal(t, 0, sub.consecutiveAirTicks, "water tick should reset consecutiveAirTicks")
	assert.False(t, sub.hasLastGround, "water tick should reset hasLastGround")
}

// TestFlySub_ClimbingExempt 验证攀爬状态跳过检测并重置状态。
func TestFlySub_ClimbingExempt(t *testing.T) {
	sub := newTestFlySub()
	ctx := context.Background()

	// 建立空中状态
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	_ = sub.Process(ctx, makeTickMsg(100.0, 65.0, 200.0, 2000))

	// 攀爬状态（梯子/藤蔓）→ 重置
	_ = sub.Process(ctx, makeTickMsg(100.0, 65.5, 200.0, 3000, func(tick *matrixpb.TelemetryTick) {
		tick.IsClimbing = true
	}))

	assert.Equal(t, 0, sub.consecutiveAirTicks, "climbing tick should reset consecutiveAirTicks")
	assert.False(t, sub.hasLastGround, "climbing tick should reset hasLastGround")
}

// TestFlySub_HoverDetected 验证长时间悬停触发 flag。
func TestFlySub_HoverDetected(t *testing.T) {
	sub := newTestFlySub()
	ctx := context.Background()

	// tick 1: 地面
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))

	// tick 2~8: 在空中几乎不动（hoverThreshold=5，需要 >5 个 air tick）
	// 每个 tick ΔY ≈ 0，模拟悬停
	for i := 1; i <= 7; i++ {
		ts := int64(1000 + i*50)
		// 极微小变动，|ΔY| < 0.01
		_ = sub.Process(ctx, makeTickMsg(100.0, 68.0+float64(i)*0.001, 200.0, ts))
	}

	// 悬停检测应该触发 flag
	assert.Greater(t, sub.VL().Violations(), 0.0, "hovering in air should flag")
}

// TestFlySub_TeleportReset 验证传送事件重置全部状态。
func TestFlySub_TeleportReset(t *testing.T) {
	sub := newTestFlySub()
	ctx := context.Background()

	// 建立空中状态
	_ = sub.Process(ctx, makeTickMsg(100.0, 64.0, 200.0, 1000, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	_ = sub.Process(ctx, makeTickMsg(100.0, 65.0, 200.0, 2000))
	assert.Equal(t, 1, sub.consecutiveAirTicks, "should have 1 air tick")
	assert.True(t, sub.hasLastGround, "should have last ground")

	// 发送传送事件
	tpMsg := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Teleports: []*matrixpb.PlayerTeleportEvent{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  2000,
			},
		},
	}
	_ = sub.Process(ctx, tpMsg)

	assert.Equal(t, 0, sub.consecutiveAirTicks, "teleport should reset consecutiveAirTicks")
	assert.False(t, sub.hasLastGround, "teleport should reset hasLastGround")
	assert.False(t, sub.hasPrevPosY, "teleport should reset hasPrevPosY")
}
