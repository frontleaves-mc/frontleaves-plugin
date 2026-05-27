package checker

import (
	"context"
	"testing"
	"time"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/google/uuid"
)

// --- AutoClicker 测试辅助 ---

// newTestAutoClickerSub 创建用于测试的 AutoClickerSub（不读取环境变量，手动注入配置）
func newTestAutoClickerSub(maxCps float64, windowMs int64, minStddev float64) *AutoClickerSub {
	warner := components.NewAntiCheatWarning(
		uuid.New(), "TestPlayer", "TestServer", "test-session",
		nil, nil,
	)
	vl := component.NewVLTracker(0.3, 5.0)
	bc := NewBaseChecker(warner, vl)

	return &AutoClickerSub{
		BaseChecker:     *bc,
		clickTimestamps: component.NewSlidingWindow[int64](100, windowMs),
		maxCps:          maxCps,
		windowMs:        windowMs,
		minStddev:       minStddev,
	}
}

// makeAutoClickerDamageEvent 构建测试用 EntityDamageEvent（仅需要 timestamp）
func makeAutoClickerDamageEvent(timestamp int64) *matrixpb.EntityDamageEvent {
	return &matrixpb.EntityDamageEvent{
		EntityId:   1,
		EntityType: "ZOMBIE",
		EntityX:    2.0,
		EntityY:    64.0,
		EntityZ:    0.0,
		PlayerX:    0,
		PlayerY:    64.0,
		PlayerZ:    0,
		PlayerYaw:  0,
		PlayerPitch: 0,
		Timestamp:  timestamp,
	}
}

// --- 测试用例 ---

// TestAutoClickerSub_NormalClicking 正常点击频率，不应触发 flag
func TestAutoClickerSub_NormalClicking(t *testing.T) {
	// maxCps=16, windowMs=1000 → 每 1000ms 内最多 16 次
	// 正常点击：1000ms 内 8 次 → CPS=8 < 16 → 安全
	s := newTestAutoClickerSub(16.0, 1000, 15.0)

	baseTs := time.Now().UnixMilli()
	for i := int64(0); i < 8; i++ {
		evt := makeAutoClickerDamageEvent(baseTs + i*125) // 每 125ms 一次 = 8 CPS
		msg := &matrixpb.MatrixTelemetryRequest{
			Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
				EntityDamage: evt,
			},
		}
		err := s.Process(context.Background(), msg)
		if err != nil {
			t.Fatalf("Process returned error: %v", err)
		}
	}

	vl := s.VL().Violations()
	if vl > 0 {
		t.Fatalf("expected VL 0 after normal clicking, got %.4f", vl)
	}
}

// TestAutoClickerSub_HighCPS 超高 CPS 点击 → 触发 AUTOCLICKER_CPS flag
func TestAutoClickerSub_HighCPS(t *testing.T) {
	// maxCps=16, windowMs=1000 → 1000ms 内 20 次点击 → CPS=20 > 16
	s := newTestAutoClickerSub(16.0, 1000, 15.0)

	baseTs := time.Now().UnixMilli()
	for i := int64(0); i < 20; i++ {
		evt := makeAutoClickerDamageEvent(baseTs + i*50) // 每 50ms 一次 = 20 CPS
		msg := &matrixpb.MatrixTelemetryRequest{
			Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
				EntityDamage: evt,
			},
		}
		_ = s.Process(context.Background(), msg)
	}

	vl := s.VL().Violations()
	if vl <= 0 {
		t.Fatalf("expected VL > 0 after high CPS clicking, got %.4f", vl)
	}
}

// TestAutoClickerSub_UniformDistribution 完美均匀分布的点击间隔 → 触发 AUTOCLICKER_UNIFORM flag
func TestAutoClickerSub_UniformDistribution(t *testing.T) {
	// maxCps=50（允许高CPS），windowMs=1000，minStddev=15.0
	// 25 次点击，间隔恒定 40ms → stddev=0 < 15.0
	s := newTestAutoClickerSub(50.0, 1000, 15.0)

	baseTs := time.Now().UnixMilli()
	for i := int64(0); i < 25; i++ {
		// 严格均匀间隔 40ms → stddev = 0
		evt := makeAutoClickerDamageEvent(baseTs + i*40)
		msg := &matrixpb.MatrixTelemetryRequest{
			Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
				EntityDamage: evt,
			},
		}
		_ = s.Process(context.Background(), msg)
	}

	vl := s.VL().Violations()
	if vl <= 0 {
		t.Fatalf("expected VL > 0 after uniform click distribution, got %.4f", vl)
	}
}

// TestAutoClickerSub_Name 返回 "auto_clicker"
func TestAutoClickerSub_Name(t *testing.T) {
	s := newTestAutoClickerSub(16.0, 1000, 15.0)
	if s.Name() != "auto_clicker" {
		t.Fatalf("expected name 'auto_clicker', got '%s'", s.Name())
	}
}

// TestAutoClickerSub_Drain 无操作，返回 nil
func TestAutoClickerSub_Drain(t *testing.T) {
	s := newTestAutoClickerSub(16.0, 1000, 15.0)
	err := s.Drain(context.Background())
	if err != nil {
		t.Fatalf("Drain returned error: %v", err)
	}
}

// TestAutoClickerSub_NilEvent nil 事件不 panic
func TestAutoClickerSub_NilEvent(t *testing.T) {
	s := newTestAutoClickerSub(16.0, 1000, 15.0)

	msg := &matrixpb.MatrixTelemetryRequest{
		Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
			EntityDamage: nil,
		},
	}
	err := s.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
}
