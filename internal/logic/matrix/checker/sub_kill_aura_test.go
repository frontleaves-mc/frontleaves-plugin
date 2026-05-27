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

// --- KillAura 测试辅助 ---

// newTestKillAuraSub 创建用于测试的 KillAuraSub（不读取环境变量，手动注入配置）
func newTestKillAuraSub(windowMs int64, maxSwitch int, angleThreshold float64) *KillAuraSub {
	warner := components.NewAntiCheatWarning(
		uuid.New(), "TestPlayer", "TestServer", "test-session",
		nil, nil,
	)
	vl := component.NewVLTracker(0.3, 5.0)
	bc := NewBaseChecker(warner, vl)

	return &KillAuraSub{
		BaseChecker:    *bc,
		attackHistory:  component.NewSlidingWindow[AttackRecord](100, windowMs),
		windowMs:       windowMs,
		maxSwitch:      maxSwitch,
		angleThreshold: angleThreshold,
	}
}

// makeKillAuraDamageEvent 构建测试用 EntityDamageEvent
func makeKillAuraDamageEvent(
	entityID int32, entityType string,
	entityX, entityY, entityZ float64,
	playerX, playerY, playerZ float64,
	yaw, pitch float32,
	timestamp int64,
) *matrixpb.EntityDamageEvent {
	return &matrixpb.EntityDamageEvent{
		EntityId:    entityID,
		EntityType:  entityType,
		EntityX:     entityX,
		EntityY:     entityY,
		EntityZ:     entityZ,
		PlayerX:     playerX,
		PlayerY:     playerY,
		PlayerZ:     playerZ,
		PlayerYaw:   yaw,
		PlayerPitch: pitch,
		Timestamp:   timestamp,
	}
}

// --- 测试用例 ---

// TestKillAuraSub_NormalCombat 攻击同一目标多次，不应触发 flag
func TestKillAuraSub_NormalCombat(t *testing.T) {
	s := newTestKillAuraSub(3000, 3, 90.0)

	// 玩家在 (0, 64, 0)，面向 +Z (yaw=0)，攻击同一实体 (entityId=42)
	baseTs := time.Now().UnixMilli()
	for i := int64(0); i < 10; i++ {
		evt := makeKillAuraDamageEvent(42, "ZOMBIE", 0, 64, 2.0, 0, 64, 0, 0, 0, baseTs+i*100)
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

	// 同一目标，VL 应该为 0（方向正常 + 无切换）
	vl := s.VL().Violations()
	if vl > 0 {
		t.Fatalf("expected VL 0 after normal combat, got %.4f", vl)
	}
}

// TestKillAuraSub_RapidSwitching 短时间内快速切换多个不同目标 → flag
func TestKillAuraSub_RapidSwitching(t *testing.T) {
	s := newTestKillAuraSub(3000, 3, 90.0)

	// 在短时间内攻击 5 个不同目标（共 7 次攻击 > 5 次，unique=5 > maxSwitch=3）
	// switchScore = 5/7 ≈ 0.71 > 0.6
	entities := []struct {
		id   int32
		x, y, z float64
	}{
		{1, 2.0, 64.0, 0.0},
		{2, -2.0, 64.0, 0.0},
		{3, 0.0, 64.0, 2.0},
		{4, 0.0, 64.0, -2.0},
		{5, 1.0, 64.0, 1.0},
	}

	baseTs := time.Now().UnixMilli()
	// 前 5 次各打一个不同目标
	for i, e := range entities {
		evt := makeKillAuraDamageEvent(e.id, "ZOMBIE", e.x, e.y, e.z, 0, 64, 0, 0, 0, baseTs+int64(i)*50)
		msg := &matrixpb.MatrixTelemetryRequest{
			Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
				EntityDamage: evt,
			},
		}
		_ = s.Process(context.Background(), msg)
	}
	// 再打 2 次复用已有目标，保持 unique=5, total=7
	for i := 0; i < 2; i++ {
		evt := makeKillAuraDamageEvent(int32(i+1), "ZOMBIE", 2.0, 64.0, 0.0, 0, 64, 0, 0, 0, baseTs+500+int64(i)*50)
		msg := &matrixpb.MatrixTelemetryRequest{
			Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
				EntityDamage: evt,
			},
		}
		_ = s.Process(context.Background(), msg)
	}

	vl := s.VL().Violations()
	if vl <= 0 {
		t.Fatalf("expected VL > 0 after rapid target switching, got %.4f", vl)
	}
}

// TestKillAuraSub_EntityIdFallback entity_id=0 时使用坐标哈希作为目标 ID
func TestKillAuraSub_EntityIdFallback(t *testing.T) {
	s := newTestKillAuraSub(3000, 3, 90.0)

	// 攻击两个不同的实体，entity_id 都为 0，但坐标不同
	// 应产生两个不同的 targetID
	nowTs := time.Now().UnixMilli()
	evt1 := makeKillAuraDamageEvent(0, "ZOMBIE", 5.0, 64.0, 0.0, 0, 64, 0, 0, 0, nowTs)
	evt2 := makeKillAuraDamageEvent(0, "ZOMBIE", -5.0, 64.0, 0.0, 0, 64, 0, 0, 0, nowTs+50)

	// 交替攻击两个目标各 4 次，共 8 次（> 5 次）
	for i := 0; i < 4; i++ {
		msg1 := &matrixpb.MatrixTelemetryRequest{
			Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
				EntityDamage: evt1,
			},
		}
		_ = s.Process(context.Background(), msg1)

		msg2 := &matrixpb.MatrixTelemetryRequest{
			Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
				EntityDamage: evt2,
			},
		}
		_ = s.Process(context.Background(), msg2)
	}

	// unique=2, total=8, switchScore=0.25, 不满足 > 0.6，但方向可能不触发
	// 此测试主要验证 entity_id=0 时不会 panic 且正确产生 targetID
	items := s.attackHistory.Items()
	seen := make(map[string]bool)
	for _, item := range items {
		seen[item.TargetID] = true
	}

	// 应有 2 个不同的 targetID（坐标不同）
	if len(seen) != 2 {
		t.Fatalf("expected 2 unique target IDs for entity_id=0 fallback, got %d", len(seen))
	}
}

// TestKillAuraSub_BehindTarget 攻击身后的目标 → 方向异常 flag
func TestKillAuraSub_BehindTarget(t *testing.T) {
	s := newTestKillAuraSub(3000, 3, 90.0)

	// 玩家在 (0, 64, 5)，面向 +Z (yaw=0, 即 South)
	// 实体在 (0, 64, 0) → 在玩家身后（-Z 方向）
	// 视线方向: (0, 0, +1)，目标方向: (0, 0, -1)
	// 夹角 = 180° > 90° → 方向异常
	evt := makeKillAuraDamageEvent(1, "ZOMBIE", 0, 64, 0, 0, 64, 5, 0, 0, time.Now().UnixMilli())
	msg := &matrixpb.MatrixTelemetryRequest{
		Payload: &matrixpb.MatrixTelemetryRequest_EntityDamage{
			EntityDamage: evt,
		},
	}
	_ = s.Process(context.Background(), msg)

	vl := s.VL().Violations()
	if vl <= 0 {
		t.Fatalf("expected VL > 0 for behind-target attack (direction anomaly), got %.4f", vl)
	}
}

// TestKillAuraSub_Name 返回 "kill_aura"
func TestKillAuraSub_Name(t *testing.T) {
	s := newTestKillAuraSub(3000, 3, 90.0)
	if s.Name() != "kill_aura" {
		t.Fatalf("expected name 'kill_aura', got '%s'", s.Name())
	}
}

// TestKillAuraSub_Drain 无操作，返回 nil
func TestKillAuraSub_Drain(t *testing.T) {
	s := newTestKillAuraSub(3000, 3, 90.0)
	err := s.Drain(context.Background())
	if err != nil {
		t.Fatalf("Drain returned error: %v", err)
	}
}

// TestKillAuraSub_NilEvent nil 事件不 panic
func TestKillAuraSub_NilEvent(t *testing.T) {
	s := newTestKillAuraSub(3000, 3, 90.0)

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
