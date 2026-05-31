package checker

import (
	"context"
	"math"
	"testing"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/google/uuid"
)

// --- 测试辅助 ---

// newTestReachSub 创建用于测试的 ReachSub（不读取环境变量，手动注入配置）
func newTestReachSub(maxReach, expand, vlThreshold float64) *ReachSub {
	warner := components.NewAntiCheatWarning(
		uuid.New(), "TestPlayer", "TestServer", "test-session",
		nil, nil, nil, "", // warningRepo、monitorCache、rdb 为 nil，bufferKey 为空，测试不触发 DB/Redis
	)
	vl := component.NewVLTracker(0.25, vlThreshold)
	bc := NewBaseChecker(warner, vl)

	return &ReachSub{
		BaseChecker: *bc,
		maxReach:    maxReach,
		expand:      expand,
	}
}

// makeDamageEvent 构建测试用的 EntityDamageEvent
func makeDamageEvent(px, py, pz, ex, ey, ez float64, yaw, pitch float32, entityType string) *matrixpb.EntityDamageEvent {
	return &matrixpb.EntityDamageEvent{
		PlayerX:    px,
		PlayerY:    py,
		PlayerZ:    pz,
		EntityX:    ex,
		EntityY:    ey,
		EntityZ:    ez,
		PlayerYaw:  yaw,
		PlayerPitch: pitch,
		EntityType: entityType,
		Timestamp:  1000,
	}
}

// makeDamageMsg 将 EntityDamageEvent 包装为 MatrixTelemetryRequest
func makeDamageMsg(evt *matrixpb.EntityDamageEvent) *matrixpb.MatrixTelemetryRequest {
	return &matrixpb.MatrixTelemetryRequest{
		EntityDamages: []*matrixpb.EntityDamageEvent{evt},
	}
}

// --- Ray-AABB 纯数学测试 ---

// TestRayAABB_HitFrontFace 射线从 Z 负方向命中包围盒正面
func TestRayAABB_HitFrontFace(t *testing.T) {
	origin := vector3{X: 5, Y: 1, Z: 0}
	dir := vector3{X: 0, Y: 0, Z: 1} // 朝 +Z 方向
	box := aabb{
		Min: vector3{X: 4, Y: 0, Z: 3},
		Max: vector3{X: 6, Y: 2, Z: 5},
	}

	hit, dist := rayAABBIntersect(origin, dir, box)
	if !hit {
		t.Fatal("expected ray to hit AABB")
	}
	if math.Abs(dist-3.0) > 1e-9 {
		t.Fatalf("expected distance 3.0, got %.9f", dist)
	}
}

// TestRayAABB_Miss 射线完全未命中包围盒
func TestRayAABB_Miss(t *testing.T) {
	origin := vector3{X: 0, Y: 1, Z: 0}
	dir := vector3{X: 1, Y: 0, Z: 0} // 朝 +X 方向
	box := aabb{
		Min: vector3{X: 5, Y: 0, Z: 5}, // 完全在 Z=5 位置
		Max: vector3{X: 7, Y: 2, Z: 7},
	}

	hit, _ := rayAABBIntersect(origin, dir, box)
	if hit {
		t.Fatal("expected ray to miss AABB")
	}
}

// TestRayAABB_InsideBox 原点在包围盒内部
func TestRayAABB_InsideBox(t *testing.T) {
	origin := vector3{X: 5, Y: 1, Z: 4} // 在盒内
	dir := vector3{X: 0, Y: 0, Z: 1}
	box := aabb{
		Min: vector3{X: 4, Y: 0, Z: 3},
		Max: vector3{X: 6, Y: 2, Z: 5},
	}

	hit, dist := rayAABBIntersect(origin, dir, box)
	if !hit {
		t.Fatal("expected ray to hit (origin inside box)")
	}
	// 原点在盒内 → tmin < 0, 使用 tmax → dist = 到出口面距离
	if dist < 0 {
		t.Fatalf("expected non-negative distance for inside-box case, got %.9f", dist)
	}
}

// --- 集成测试 ---

// TestReachSub_NormalAttack 玩家面向实体，距离在最大范围内 → 不 flag
func TestReachSub_NormalAttack(t *testing.T) {
	r := newTestReachSub(3.0, 0.1, 3.0)

	// 玩家在 (0, 64, 0)，面向 +Z (yaw=0)，实体在 (0, 64, 2.5)
	evt := makeDamageEvent(0, 64, 0, 0, 64, 2.5, 0, 0, "PLAYER")
	msg := makeDamageMsg(evt)

	err := r.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	// 正常攻击，VL 应该减少（reward），初始为 0 所以 reward 后仍为 0
	vl := r.vl.Violations()
	if vl != 0 {
		t.Fatalf("expected VL 0 after normal attack, got %.4f", vl)
	}
}

// TestReachSub_ExceedsDistance 玩家面向实体但距离超过最大范围 → flag
func TestReachSub_ExceedsDistance(t *testing.T) {
	r := newTestReachSub(3.0, 0.1, 3.0)

	// 玩家在 (0, 64, 0)，面向 +Z (yaw=0)，实体在 (0, 64, 5.0) → 距离约5
	evt := makeDamageEvent(0, 64, 0, 0, 64, 5.0, 0, 0, "PLAYER")
	msg := makeDamageMsg(evt)

	err := r.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	vl := r.vl.Violations()
	if vl <= 0 {
		t.Fatalf("expected VL > 0 after exceed-distance attack, got %.4f", vl)
	}
}

// TestReachSub_BehindTarget 玩家背对实体 → 射线不命中 → hitbox 异常 flag
func TestReachSub_BehindTarget(t *testing.T) {
	r := newTestReachSub(3.0, 0.1, 3.0)

	// 玩家在 (0, 64, 5)，面向 +Z (yaw=0)，实体在身后 (0, 64, 0)
	// 射线朝 +Z，实体在 -Z → 完全背对
	evt := makeDamageEvent(0, 64, 5, 0, 64, 0, 0, 0, "PLAYER")
	msg := makeDamageMsg(evt)

	err := r.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}

	vl := r.vl.Violations()
	if vl <= 0 {
		t.Fatalf("expected VL > 0 for behind-target attack (hitbox anomaly), got %.4f", vl)
	}
}

// TestReachSub_DifferentEntityTypes 测试不同实体类型的碰撞箱尺寸
func TestReachSub_DifferentEntityTypes(t *testing.T) {
	// SPIDER: 宽体矮 (halfW=0.7, height=0.55)
	spiderBox := buildEntityAABB("SPIDER", 0, 0, 0, 0)
	if spiderBox.Max.X-spiderBox.Min.X != 1.4 {
		t.Fatalf("SPIDER width expected 1.4, got %.2f", spiderBox.Max.X-spiderBox.Min.X)
	}
	if spiderBox.Max.Y-spiderBox.Min.Y != 0.55 {
		t.Fatalf("SPIDER height expected 0.55, got %.2f", spiderBox.Max.Y-spiderBox.Min.Y)
	}

	// ENDERMAN: 窄体高 (halfW=0.3, height=2.9)
	enderBox := buildEntityAABB("ENDERMAN", 0, 0, 0, 0)
	if enderBox.Max.Y-enderBox.Min.Y != 2.9 {
		t.Fatalf("ENDERMAN height expected 2.9, got %.2f", enderBox.Max.Y-enderBox.Min.Y)
	}

	// UNKNOWN: 默认尺寸 (halfW=0.3, height=1.8)
	unknownBox := buildEntityAABB("UNKNOWN_MOB", 0, 0, 0, 0)
	if unknownBox.Max.Y-unknownBox.Min.Y != 1.8 {
		t.Fatalf("UNKNOWN height expected 1.8, got %.2f", unknownBox.Max.Y-unknownBox.Min.Y)
	}

	// 带 expand
	expandedBox := buildEntityAABB("SPIDER", 0, 0, 0, 0.1)
	width := expandedBox.Max.X - expandedBox.Min.X
	if math.Abs(width-1.6) > 1e-9 { // 1.4 + 0.2 (每侧 expand 0.1)
		t.Fatalf("expanded SPIDER width expected 1.6, got %.2f", width)
	}
}

// TestReachSub_RewardDecay 连续正常攻击应使 VL 逐渐降低
func TestReachSub_RewardDecay(t *testing.T) {
	r := newTestReachSub(3.0, 0.1, 3.0)

	// 先人为注入一些 VL
	r.vl.Flag(1.0)

	// 发送一次正常攻击 → reward (decay=0.25)
	evt := makeDamageEvent(0, 64, 0, 0, 64, 2.0, 0, 0, "PLAYER")
	msg := makeDamageMsg(evt)
	_ = r.Process(context.Background(), msg)

	vlAfterOne := r.vl.Violations()
	if vlAfterOne >= 1.0 {
		t.Fatalf("expected VL < 1.0 after reward, got %.4f", vlAfterOne)
	}
	if vlAfterOne != 0.75 {
		t.Fatalf("expected VL 0.75 after one reward, got %.4f", vlAfterOne)
	}
}

// TestReachSub_NilEvent nil 事件不 panic
func TestReachSub_NilEvent(t *testing.T) {
	r := newTestReachSub(3.0, 0.1, 3.0)

	// Process nil payload 不应 panic
	msg := &matrixpb.MatrixTelemetryRequest{}
	err := r.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process returned error: %v", err)
	}
}

// TestReachSub_Drain 无操作，返回 nil
func TestReachSub_Drain(t *testing.T) {
	r := newTestReachSub(3.0, 0.1, 3.0)
	err := r.Drain(context.Background())
	if err != nil {
		t.Fatalf("Drain returned error: %v", err)
	}
}

// TestReachSub_Name 返回 "reach"
func TestReachSub_Name(t *testing.T) {
	r := newTestReachSub(3.0, 0.1, 3.0)
	if r.Name() != "reach" {
		t.Fatalf("expected name 'reach', got '%s'", r.Name())
	}
}
