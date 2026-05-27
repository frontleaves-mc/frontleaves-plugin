package checker

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"

	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// entityDimensions 已知实体类型的碰撞箱尺寸: {halfWidth, height}
var entityDimensions = map[string][2]float64{
	"ZOMBIE":     {0.3, 1.95},
	"SKELETON":   {0.3, 1.99},
	"CREEPER":    {0.3, 1.7},
	"SPIDER":     {0.7, 0.55},
	"ENDERMAN":   {0.3, 2.9},
	"PLAYER":     {0.3, 1.8},
	"VILLAGER":   {0.3, 1.95},
	"IRON_GOLEM": {0.7, 2.7},
}

// defaultEntityDimension 默认实体尺寸 (halfWidth=0.3, height=1.8)
var defaultEntityDimension = [2]float64{0.3, 1.8}

// vector3 三维向量
type vector3 struct {
	X, Y, Z float64
}

// aabb 轴对齐包围盒
type aabb struct {
	Min, Max vector3
}

// ReachSub 基于射线-AABB碰撞检测的攻击距离检查器，实现 MatrixSub 接口。
type ReachSub struct {
	BaseChecker
	mu         sync.Mutex

	// 配置参数
	maxReach float64
	expand   float64
}

// NewReachSub 创建 ReachSub 实例，从环境变量读取配置。
func NewReachSub(warner *components.AntiCheatWarning) *ReachSub {
	maxReach := xEnv.GetEnvFloat(bConst.EnvMatrixAcReachMaxReach, 3.0)
	expand := xEnv.GetEnvFloat(bConst.EnvMatrixAcReachExpand, 0.1)
	vlThreshold := xEnv.GetEnvFloat(bConst.EnvMatrixAcReachVlThreshold, 3.0)

	vl := component.NewVLTracker(0.25, vlThreshold)
	bc := NewBaseChecker(warner, vl)

	return &ReachSub{
		BaseChecker: *bc,
		maxReach: maxReach,
		expand:   expand,
	}
}

// Name 返回检查器名称。
func (r *ReachSub) Name() string {
	return "reach"
}

// Process 处理单条遥测数据，根据事件类型分发处理。
func (r *ReachSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch msg.Payload.(type) {
	case *matrixpb.MatrixTelemetryRequest_EntityDamage:
		r.checkReach(ctx, msg.GetEntityDamage())
	default:
		// 其他事件不处理，无持久状态
	}

	return nil
}

// Drain 排水时无需操作。
func (r *ReachSub) Drain(_ context.Context) error {
	return nil
}

// checkReach 使用射线-AABB碰撞检测判定攻击距离。
func (r *ReachSub) checkReach(ctx context.Context, evt *matrixpb.EntityDamageEvent) {
	if evt == nil {
		return
	}

	// 1. 构建玩家眼睛位置 (站立时眼睛高度 +1.62)
	origin := vector3{
		X: evt.GetPlayerX(),
		Y: evt.GetPlayerY() + 1.62,
		Z: evt.GetPlayerZ(),
	}

	// 2. 从 yaw/pitch 计算视线方向
	dir := yawPitchToDirection(float64(evt.GetPlayerYaw()), float64(evt.GetPlayerPitch()))

	// 3. 构建目标实体 AABB
	box := buildEntityAABB(evt.GetEntityType(), evt.GetEntityX(), evt.GetEntityY(), evt.GetEntityZ(), r.expand)

	// 4. 射线-AABB 碰撞检测
	hit, distance := rayAABBIntersect(origin, dir, box)

	if !hit {
		// 射线未命中目标碰撞箱 — 视线未对准实体但仍造成伤害，判定为 hitbox 异常
		r.FlagViaVL(ctx, "REACH_HITBOX", fmt.Sprintf(
			"射线未命中实体碰撞箱，可能存在 hitbox 穿透 (entityType=%s)",
			evt.GetEntityType(),
		), 1.0, map[string]any{
			"entity_type":  evt.GetEntityType(),
			"player_x":     evt.GetPlayerX(),
			"player_y":     evt.GetPlayerY(),
			"player_z":     evt.GetPlayerZ(),
			"entity_x":     evt.GetEntityX(),
			"entity_y":     evt.GetEntityY(),
			"entity_z":     evt.GetEntityZ(),
			"player_yaw":   evt.GetPlayerYaw(),
			"player_pitch": evt.GetPlayerPitch(),
		})
		return
	}

	if distance > r.maxReach {
		// 攻击距离超过最大值
		r.FlagViaVL(ctx, "REACH", fmt.Sprintf(
			"攻击距离 %.2f blocks 超过阈值 %.1f",
			distance, r.maxReach,
		), 1.0, map[string]any{
			"distance":    distance,
			"threshold":   r.maxReach,
			"entity_type": evt.GetEntityType(),
			"damage_cause": evt.GetDamageCause(),
		})
		return
	}

	// 正常攻击 — 奖励降低 VL
	r.RewardViaVL()
}

// yawPitchToDirection 将 Minecraft 的 yaw/pitch 转换为视线方向向量。
//
// MC 坐标系约定:
//   - Yaw: 0°=South(+Z), 90°=West(-X), 180°=North(-Z), 270°=East(+X)
//   - Pitch: -90°=Up, 0°=Horizontal, 90°=Down
func yawPitchToDirection(yaw, pitch float64) vector3 {
	yawRad := yaw * math.Pi / 180.0
	pitchRad := pitch * math.Pi / 180.0

	return vector3{
		X: -math.Sin(yawRad) * math.Cos(pitchRad),
		Y: -math.Sin(pitchRad),
		Z: math.Cos(yawRad) * math.Cos(pitchRad),
	}
}

// buildEntityAABB 根据实体类型构建 AABB 并扩展。
func buildEntityAABB(entityType string, x, y, z, expand float64) aabb {
	dim, ok := entityDimensions[entityType]
	if !ok {
		dim = defaultEntityDimension
	}
	halfW := dim[0]
	height := dim[1]

	return aabb{
		Min: vector3{X: x - halfW - expand, Y: y - expand, Z: z - halfW - expand},
		Max: vector3{X: x + halfW + expand, Y: y + height + expand, Z: z + halfW + expand},
	}
}

// rayAABBIntersect 使用 slab method 检测射线与 AABB 的碰撞。
// 返回是否命中及碰撞点距离。
func rayAABBIntersect(origin, dir vector3, box aabb) (bool, float64) {
	tmin := -math.MaxFloat64
	tmax := math.MaxFloat64

	for _, axis := range []int{0, 1, 2} {
		o := getAxis(origin, axis)
		d := getAxis(dir, axis)
		bmin := getAxis(box.Min, axis)
		bmax := getAxis(box.Max, axis)

		if math.Abs(d) < 1e-10 {
			// 射线与该轴平行 — 检查原点是否在 slab 内
			if o < bmin || o > bmax {
				return false, 0
			}
			continue
		}

		t1 := (bmin - o) / d
		t2 := (bmax - o) / d
		if t1 > t2 {
			t1, t2 = t2, t1
		}

		if t1 > tmin {
			tmin = t1
		}
		if t2 < tmax {
			tmax = t2
		}

		if tmin > tmax || tmax < 0 {
			return false, 0
		}
	}

	dist := tmin
	if dist < 0 {
		dist = tmax
	}
	return true, dist
}

// getAxis 按索引获取向量分量: 0=X, 1=Y, 2=Z。
func getAxis(v vector3, axis int) float64 {
	switch axis {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		return v.Z
	}
}
