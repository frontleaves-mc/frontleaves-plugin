package checker

import (
	"context"
	"testing"

	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/stretchr/testify/assert"
)

// newTestNoFallSub 创建一个用于测试的 NoFallSub，使用低 VL 阈值以快速触发。
func newTestNoFallSub() *NoFallSub {
	vl := component.NewVLTracker(0.5, 1.0) // decay=0.5, threshold=1.0 (low for testing)
	baseChecker := NewBaseChecker(nil, vl)

	return &NoFallSub{
		BaseChecker:      *baseChecker,
		fallSpeed:        0.1,
		consecutiveTicks: 3,
	}
}

// makeNoFallTickMsg 构造 TelemetryTick 类型的请求消息。
func makeNoFallTickMsg(posY float64, opts ...func(*matrixpb.TelemetryTick)) *matrixpb.MatrixTelemetryRequest {
	tick := &matrixpb.TelemetryTick{
		PlayerUuid: "test-uuid",
		PlayerName: "TestPlayer",
		PosY:       posY,
		Timestamp:  1000,
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

// TestNoFallSub_NormalFalling 验证正常空中下落不触发 flag。
func TestNoFallSub_NormalFalling(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置 Y=80
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 正常下落 Y=79，不在地面 → 正常
	_ = sub.Process(ctx, makeNoFallTickMsg(79.0, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = false
	}))

	// tick 3: 继续下落 Y=78，不在地面 → 正常
	_ = sub.Process(ctx, makeNoFallTickMsg(78.0, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = false
	}))

	assert.Equal(t, 0.0, sub.VL().Violations(), "normal falling should not flag")
}

// TestNoFallSub_NormalLanding 验证正常着地不触发 flag。
func TestNoFallSub_NormalLanding(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置 Y=65
	_ = sub.Process(ctx, makeNoFallTickMsg(65.0))

	// tick 2: 着地，deltaY 极小（-0.05），在 fallSpeed 阈值内 → 正常
	_ = sub.Process(ctx, makeNoFallTickMsg(64.95, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))

	assert.Equal(t, 0.0, sub.VL().Violations(), "normal landing should not flag")
	assert.Equal(t, 0, sub.noFallCounter, "normal landing should not increment counter")
}

// TestNoFallSub_GroundSpoof 验证声称 on_ground 但实际下落触发 flag。
func TestNoFallSub_GroundSpoof(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置 Y=80
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 声称 on_ground 但下落 0.5 blocks（超过 fallSpeed=0.1）
	_ = sub.Process(ctx, makeNoFallTickMsg(79.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	assert.Equal(t, 1, sub.noFallCounter, "first spoof tick should set counter to 1")

	// tick 3: 继续伪装
	_ = sub.Process(ctx, makeNoFallTickMsg(79.0, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	assert.Equal(t, 2, sub.noFallCounter, "second spoof tick should set counter to 2")

	// tick 4: 第三次 → 达到 consecutiveTicks=3 → flag
	_ = sub.Process(ctx, makeNoFallTickMsg(78.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	assert.Equal(t, 3, sub.noFallCounter, "third spoof tick should set counter to 3")
	assert.Greater(t, sub.VL().Violations(), 0.0, "ground spoof should flag after consecutive ticks")
}

// TestNoFallSub_WaterExempt 验证水中状态豁免检测。
func TestNoFallSub_WaterExempt(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 在水中，声称 on_ground 但正在下落 → 豁免
	_ = sub.Process(ctx, makeNoFallTickMsg(79.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
		tick.IsInWater = true
	}))

	assert.Equal(t, 0, sub.noFallCounter, "water should be exempt")
	assert.Equal(t, 0.0, sub.VL().Violations(), "water should not flag")
}

// TestNoFallSub_ClimbingExempt 验证攀爬状态豁免检测。
func TestNoFallSub_ClimbingExempt(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 攀爬中，声称 on_ground 但位置下落 → 豁免
	_ = sub.Process(ctx, makeNoFallTickMsg(79.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
		tick.IsClimbing = true
	}))

	assert.Equal(t, 0, sub.noFallCounter, "climbing should be exempt")
	assert.Equal(t, 0.0, sub.VL().Violations(), "climbing should not flag")
}

// TestNoFallSub_TeleportReset 验证传送事件重置计数器。
func TestNoFallSub_TeleportReset(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 伪装 on_ground，增加计数器
	_ = sub.Process(ctx, makeNoFallTickMsg(79.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	assert.Equal(t, 1, sub.noFallCounter, "should have counter=1 after spoof tick")

	// 发送传送事件 → 重置
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

	assert.Equal(t, 0, sub.noFallCounter, "teleport should reset counter")
	assert.False(t, sub.hasPrev, "teleport should reset hasPrev")
}

// TestNoFallSub_LavaExempt 验证岩浆状态豁免检测。
func TestNoFallSub_LavaExempt(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 在岩浆中 → 豁免
	_ = sub.Process(ctx, makeNoFallTickMsg(79.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
		tick.IsInLava = true
	}))

	assert.Equal(t, 0, sub.noFallCounter, "lava should be exempt")
	assert.Equal(t, 0.0, sub.VL().Violations(), "lava should not flag")
}

// TestNoFallSub_CounterRecovery 验证非伪装 tick 逐步恢复计数器。
func TestNoFallSub_CounterRecovery(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 伪装 on_ground
	_ = sub.Process(ctx, makeNoFallTickMsg(79.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	assert.Equal(t, 1, sub.noFallCounter)

	// tick 3: 正常空中 tick（不在地面，正常下落）→ 计数器恢复
	_ = sub.Process(ctx, makeNoFallTickMsg(79.0, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = false
	}))
	assert.Equal(t, 0, sub.noFallCounter, "non-ground tick should recover counter")
}

// TestNoFallSub_RespawnReset 验证重生事件重置计数器。
func TestNoFallSub_RespawnReset(t *testing.T) {
	sub := newTestNoFallSub()
	ctx := context.Background()

	// tick 1: 建立初始位置
	_ = sub.Process(ctx, makeNoFallTickMsg(80.0))

	// tick 2: 伪装
	_ = sub.Process(ctx, makeNoFallTickMsg(79.5, func(tick *matrixpb.TelemetryTick) {
		tick.IsOnGround = true
	}))
	assert.Equal(t, 1, sub.noFallCounter)

	// 发送重生事件 → 重置
	respawnMsg := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Payload: &matrixpb.MatrixTelemetryRequest_Respawn{
			Respawn: &matrixpb.PlayerRespawnEvent{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  2000,
			},
		},
	}
	_ = sub.Process(ctx, respawnMsg)

	assert.Equal(t, 0, sub.noFallCounter, "respawn should reset counter")
	assert.False(t, sub.hasPrev, "respawn should reset hasPrev")
}
