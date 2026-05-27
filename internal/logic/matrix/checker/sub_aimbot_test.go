package checker

import (
	"context"
	"math"
	"testing"

	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/stretchr/testify/assert"
)

// newTestAimbotSub 创建一个用于测试的 AimbotSub，使用低 VL 阈值以快速触发。
func newTestAimbotSub() *AimbotSub {
	vl := component.NewVLTracker(0.3, 1.0) // decay=0.3, threshold=1.0 (low for testing)
	baseChecker := NewBaseChecker(nil, vl)

	return &AimbotSub{
		BaseChecker:      *baseChecker,
		yawDeltas:        component.NewRingBuffer[float64](30),
		pitchDeltas:      component.NewRingBuffer[float64](30),
		sampleWindow:     30,
		sensitivityDelta: 0.15,
	}
}

func makeAimbotYawPitchTick(yaw, pitch float32) *matrixpb.MatrixTelemetryRequest {
	return &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Payload: &matrixpb.MatrixTelemetryRequest_TelemetryTick{
			TelemetryTick: &matrixpb.TelemetryTick{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Yaw:        yaw,
				Pitch:      pitch,
				Timestamp:  1000,
			},
		},
	}
}

// makeAimbotDamageEvent creates an EntityDamageEvent message with yaw/pitch.
func makeAimbotDamageEvent(yaw, pitch float32) *matrixpb.MatrixTelemetryRequest {
	return &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
			EntityDamage: &matrixpb.EntityDamageEvent{
				PlayerUuid:  "test-uuid",
				PlayerName:  "TestPlayer",
				PlayerYaw:   yaw,
				PlayerPitch: pitch,
			},
		},
	}
}

// TestAimbotSub_Name 验证 Name() 返回 "aimbot"。
func TestAimbotSub_Name(t *testing.T) {
	sub := newTestAimbotSub()
	assert.Equal(t, "aimbot", sub.Name())
}

// TestAimbotSub_ConsistentSensitivity 验证灵敏度一致时不触发 flag。
// 使用固定的 pitch delta 模拟真实玩家操作，所有 delta 的 GCD 应保持一致。
func TestAimbotSub_ConsistentSensitivity(t *testing.T) {
	sub := newTestAimbotSub()
	ctx := context.Background()

	// 模拟一个灵敏度 2.0 的玩家：pitch delta ≈ 0.75
	// 连续输入相同的 pitch delta → GCD 稳定 → 不触发
	basePitch := float32(0.0)
	delta := float32(0.75)

	// 第一个 tick 建立初始值
	_ = sub.Process(ctx, makeAimbotYawPitchTick(0.0, basePitch))

	// 输入 20 个一致的变化
	for i := 0; i < 20; i++ {
		basePitch += delta
		_ = sub.Process(ctx, makeAimbotYawPitchTick(float32(i)*1.0, basePitch))
	}

	// 灵敏度一致 → 无 flag
	assert.Equal(t, 0.0, sub.VL().Violations(), "consistent sensitivity should not flag")
}

// TestAimbotSub_SensitivityJump 验证灵敏度突然跳变时触发 flag。
func TestAimbotSub_SensitivityJump(t *testing.T) {
	sub := newTestAimbotSub()
	ctx := context.Background()

	// Phase 1: 建立稳定灵敏度（delta=0.75）
	basePitch := float32(0.0)
	_ = sub.Process(ctx, makeAimbotYawPitchTick(0.0, basePitch))
	for i := 0; i < 20; i++ {
		basePitch += 0.75
		_ = sub.Process(ctx, makeAimbotYawPitchTick(float32(i)*1.0, basePitch))
	}

	// 手动设置一个已建立的灵敏度估计
	// 然后注入一组产生完全不同 GCD 的 pitch delta
	sub.mu.Lock()
	sub.sensitivityEstimate = 2.0 // 模拟已建立的灵敏度=2.0
	sub.hasEstimate = true
	sub.mu.Unlock()

	// 重新填充 buffer：清空后用 delta=0.3（产生完全不同的灵敏度）
	sub.mu.Lock()
	sub.pitchDeltas = component.NewRingBuffer[float64](30)
	sub.yawDeltas = component.NewRingBuffer[float64](30)
	sub.prevPitch = float64(basePitch)
	sub.prevYaw = float64(19.0)
	sub.hasPrev = true
	sub.mu.Unlock()

	// 输入 15 个 delta=0.3 的 tick → GCD=0.3 → sensitivity 远小于 2.0 → flag
	for i := 0; i < 16; i++ {
		basePitch += 0.3
		_ = sub.Process(ctx, makeAimbotYawPitchTick(float32(i+20)*1.0, basePitch))
	}

	assert.Greater(t, sub.VL().Violations(), 0.0, "sensitivity jump should flag")
}

// TestAimbotSub_Reset 验证传送/重生/模式变更事件重置追踪状态。
func TestAimbotSub_Reset(t *testing.T) {
	sub := newTestAimbotSub()
	ctx := context.Background()

	// 建立足够状态使 hasEstimate=true（需要 pitchDeltas.Size() >= 15）
	basePitch := float32(0.0)
	_ = sub.Process(ctx, makeAimbotYawPitchTick(0.0, basePitch))

	for i := 0; i < 16; i++ {
		basePitch += 0.75
		_ = sub.Process(ctx, makeAimbotYawPitchTick(float32(i)*1.0, basePitch))
	}

	assert.True(t, sub.hasPrev, "should have prev after ticks")
	assert.True(t, sub.hasEstimate, "should have sensitivity estimate")
	assert.Equal(t, 16, sub.pitchDeltas.Size(), "should have 16 pitch deltas")

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

	// 验证重置
	assert.False(t, sub.hasPrev, "teleport should reset hasPrev")
	assert.False(t, sub.hasEstimate, "teleport should reset hasEstimate")
	assert.Equal(t, 0, sub.pitchDeltas.Size(), "teleport should reset pitchDeltas buffer")
	assert.Equal(t, 0, sub.yawDeltas.Size(), "teleport should reset yawDeltas buffer")
}

// TestAimbotSub_DamageEventSource 验证 EntityDamageEvent 也能触发检测。
func TestAimbotSub_DamageEventSource(t *testing.T) {
	sub := newTestAimbotSub()
	ctx := context.Background()

	basePitch := float32(0.0)
	delta := float32(0.75)

	// 使用 EntityDamageEvent 而非 TelemetryTick
	_ = sub.Process(ctx, makeAimbotDamageEvent(0.0, basePitch))

	for i := 0; i < 16; i++ {
		basePitch += delta
		_ = sub.Process(ctx, makeAimbotDamageEvent(float32(i)*1.0, basePitch))
	}

	// 一致的 delta → 无 flag
	assert.Equal(t, 0.0, sub.VL().Violations(), "consistent damage events should not flag")
}

// TestAimbotSub_FloatGCD 验证浮点 GCD 计算的正确性。
func TestAimbotSub_FloatGCD(t *testing.T) {
	tests := []struct {
		name     string
		a        float64
		b        float64
		expected float64
	}{
		{"identical values", 0.75, 0.75, 0.75},
		{"one is multiple", 1.5, 0.75, 0.75},
		{"coprime floats", 0.5, 0.3, 0.1},
		{"zero b", 5.0, 0.0, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := floatGCD(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 1e-4)
		})
	}
}

// TestAimbotSub_FloatGCDOfValues 验证多值 GCD 计算。
func TestAimbotSub_FloatGCDOfValues(t *testing.T) {
	// 所有值都是 0.75 的倍数 → GCD ≈ 0.75
	values := []float64{0.75, 1.5, 2.25, 3.0}
	result := floatGCDOfValues(values)
	assert.InDelta(t, 0.75, result, 1e-4)
}

// TestAimbotSub_RespawnReset 验证重生事件也触发重置。
func TestAimbotSub_RespawnReset(t *testing.T) {
	sub := newTestAimbotSub()
	ctx := context.Background()

	_ = sub.Process(ctx, makeAimbotYawPitchTick(10.0, -5.0))
	_ = sub.Process(ctx, makeAimbotYawPitchTick(15.0, -3.0))
	assert.True(t, sub.hasPrev)

	// 发送重生事件
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

	assert.False(t, sub.hasPrev, "respawn should reset hasPrev")
}

// TestAimbotSub_SmallDeltaSkipped 验证微小视角变化被跳过。
func TestAimbotSub_SmallDeltaSkipped(t *testing.T) {
	sub := newTestAimbotSub()
	ctx := context.Background()

	// 建立初始值
	_ = sub.Process(ctx, makeAimbotYawPitchTick(10.0, -5.0))

	// 发送微小变化（< 0.01）
	_ = sub.Process(ctx, makeAimbotYawPitchTick(10.005, -4.995))

	// 应该被跳过，不增加 buffer
	assert.Equal(t, 0, sub.pitchDeltas.Size(), "tiny delta should be skipped")
}

// TestAimbotSub_SensitivityFormula 验证灵敏度反推公式。
func TestAimbotSub_SensitivityFormula(t *testing.T) {
	// 已知 sensitivity=2.0 时，Minecraft 的 pitch delta GCD 应该接近某个值
	// 公式：sensitivity = (cbrt(gcd/0.15/8.0) - 0.2) / 0.6
	// 反推：gcd = ((sensitivity * 0.6 + 0.2) ^ 3) * 0.15 * 8.0
	sensitivity := 2.0
	expectedGCD := math.Pow(sensitivity*0.6+0.2, 3) * 0.15 * 8.0

	// 验证反推回来的灵敏度一致
	reverseSens := (math.Cbrt(expectedGCD/0.15/8.0) - 0.2) / 0.6
	assert.InDelta(t, sensitivity, reverseSens, 1e-6, "sensitivity formula should be reversible")
}
