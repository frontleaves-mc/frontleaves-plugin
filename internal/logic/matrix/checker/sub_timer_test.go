package checker

import (
	"context"
	"testing"
	"time"

	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/stretchr/testify/assert"
)

// newTestTimerSub 创建一个用于测试的 TimerSub，使用高 VL 阈值避免触发 warner（nil）。
func newTestTimerSub() *TimerSub {
	vl := component.NewVLTracker(0.5, 100.0) // decay=0.5, threshold=100.0 (high to avoid nil warner crash)
	baseChecker := NewBaseChecker(nil, vl)

	return &TimerSub{
		BaseChecker: *baseChecker,
		clockDrift:  150,
	}
}

// TestTimerSub_NormalSpeed 验证正常 tick 频率（每 50ms 一个 tick）不会触发 flag。
func TestTimerSub_NormalSpeed(t *testing.T) {
	sub := newTestTimerSub()
	ctx := context.Background()

	// 使用当前时间附近的 timestamp，模拟正常 tick 节奏
	now := time.Now().UnixMilli()

	// tick 1: 初始化会话（timestamp 设为当前时间 - 200ms）
	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now - 200,
			},
		},
	})

	// tick 2: 正常 tick（距离初始化约 200ms，两个 tick = 100ms，合理）
	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now - 100,
			},
		},
	})

	// timerBalance = 100, sessionElapsed ≈ 200（远大于 100+150=250 不成立）→ reward
	assert.Equal(t, 0.0, sub.VL().Violations(), "normal tick pace should not flag")
}

// TestTimerSub_Accelerated 验证过快的 tick 频率触发 flag。
func TestTimerSub_Accelerated(t *testing.T) {
	sub := newTestTimerSub()
	ctx := context.Background()

	// 使用当前时间，使得 sessionElapsed ≈ 0
	now := time.Now().UnixMilli()

	// tick 1: 初始化会话
	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now,
			},
		},
	})

	// 立即发送大量 tick，模拟 Timer 加速
	// timerBalance 将累加到 50*10=500ms，但实际流逝时间可能只有几 ms
	for i := 0; i < 9; i++ {
		_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
			ServerName: "test-server",
			TelemetryTicks: []*matrixpb.TelemetryTick{
				{
					PlayerUuid: "test-uuid",
					PlayerName: "TestPlayer",
					Timestamp:  now,
				},
			},
		})
	}

	// timerBalance = 500, sessionElapsed ≈ 0-几ms, 500 > 0+150=150 → flag
	assert.Greater(t, sub.VL().Violations(), 0.0, "accelerated ticks should increase VL")
}

// TestTimerSub_Reset 验证传送事件重置 Timer 状态。
func TestTimerSub_Reset(t *testing.T) {
	sub := newTestTimerSub()
	ctx := context.Background()

	now := time.Now().UnixMilli()

	// 先建立状态
	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now,
			},
		},
	})

	assert.True(t, sub.initialized, "should be initialized after first tick")
	assert.Greater(t, sub.timerBalance, 0.0, "timer balance should be > 0 after tick")

	// 发送传送事件 → 重置
	tpMsg := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Teleports: []*matrixpb.PlayerTeleportEvent{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now,
			},
		},
	}
	_ = sub.Process(ctx, tpMsg)

	assert.False(t, sub.initialized, "teleport should reset initialized")
	assert.Equal(t, 0.0, sub.timerBalance, "teleport should reset timerBalance")

	// 同样验证重生事件重置
	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now + 100,
			},
		},
	})
	assert.True(t, sub.initialized, "should be re-initialized after tick")

	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Respawns: []*matrixpb.PlayerRespawnEvent{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now + 200,
			},
		},
	})
	assert.False(t, sub.initialized, "respawn should reset initialized")

	// 验证游戏模式变更事件重置
	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		TelemetryTicks: []*matrixpb.TelemetryTick{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now + 300,
			},
		},
	})
	assert.True(t, sub.initialized, "should be re-initialized after tick")

	_ = sub.Process(ctx, &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		GameModeChanges: []*matrixpb.GameModeChangeEvent{
			{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Timestamp:  now + 400,
			},
		},
	})
	assert.False(t, sub.initialized, "game mode change should reset initialized")
}

// TestTimerSub_DrainAndName 验证基础方法正常工作。
func TestTimerSub_DrainAndName(t *testing.T) {
	sub := newTestTimerSub()

	// Name
	assert.Equal(t, "timer", sub.Name(), "Name should return 'timer'")

	// Drain 无错误
	ctx := context.Background()
	err := sub.Drain(ctx)
	assert.NoError(t, err, "Drain should not return error")
}

// TestTimerSub_NilTick 验证 nil tick 不触发 panic。
func TestTimerSub_NilTick(t *testing.T) {
	sub := newTestTimerSub()
	ctx := context.Background()

	// 发送 nil TelemetryTick（通过空 repeated 列表的情况）
	msg := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
	}

	// 不应 panic
	err := sub.Process(ctx, msg)
	assert.NoError(t, err, "nil tick should not cause error")
	assert.Equal(t, 0.0, sub.VL().Violations(), "nil tick should not affect VL")
}
