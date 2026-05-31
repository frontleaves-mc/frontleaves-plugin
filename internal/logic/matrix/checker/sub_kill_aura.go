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

// AttackRecord 记录一次攻击事件的基本信息。
type AttackRecord struct {
	Timestamp int64
	TargetID  string
	TargetPos [3]float64
}

// KillAuraSub 多目标快速切换攻击检测，实现 MatrixSub 接口。
//
// 检测逻辑：
//   - 目标切换频率：短时间内攻击不同目标数量超过阈值 → KillAura 嫌疑
//   - 攻击方向异常：玩家视线方向与目标方向夹角过大 → 视线外攻击
type KillAuraSub struct {
	BaseChecker
	mu            sync.Mutex
	attackHistory *component.SlidingWindow[AttackRecord]

	// 配置参数（从环境变量读取）
	windowMs       int64
	maxSwitch      int
	angleThreshold float64
}

// NewKillAuraSub 创建 KillAuraSub 实例，从环境变量读取配置。
func NewKillAuraSub(warner *components.AntiCheatWarning) *KillAuraSub {
	windowMs := xEnv.GetEnvInt64(bConst.EnvMatrixAcKillauraWindowMs, 3000)
	maxSwitch := xEnv.GetEnvInt(bConst.EnvMatrixAcKillauraMaxSwitch, 3)
	angleThreshold := xEnv.GetEnvFloat(bConst.EnvMatrixAcKillauraAngleThreshold, 90.0)

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

// Name 返回检查器名称。
func (s *KillAuraSub) Name() string {
	return "kill_aura"
}

// Process 处理单条遥测数据，根据事件类型分发处理。
func (s *KillAuraSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, evt := range msg.GetEntityDamages() {
		s.checkKillAura(ctx, evt)
	}

	return nil
}

// Drain 排水时无需操作。
func (s *KillAuraSub) Drain(_ context.Context) error {
	return nil
}

// checkKillAura 执行 KillAura 检测核心逻辑。
func (s *KillAuraSub) checkKillAura(ctx context.Context, evt *matrixpb.EntityDamageEvent) {
	if evt == nil {
		return
	}

	nowMs := evt.GetTimestamp()

	// 1. 提取目标 ID
	targetID := s.extractTargetID(evt)

	// 2. 记录攻击历史
	s.attackHistory.Add(nowMs, AttackRecord{
		Timestamp: nowMs,
		TargetID:  targetID,
		TargetPos: [3]float64{evt.GetEntityX(), evt.GetEntityY(), evt.GetEntityZ()},
	})

	// 3. 获取时间窗口内的攻击记录
	items := s.attackHistory.Items()

	// 4. 统计唯一目标数量
	uniqueTargets := countUniqueTargets(items)

	// 5. 目标切换频率检测
	if len(items) > 5 {
		switchScore := float64(uniqueTargets) / float64(len(items))
		if uniqueTargets > s.maxSwitch && switchScore > 0.6 {
			s.FlagViaVL(ctx, "KILL_AURA_SWITCH", fmt.Sprintf(
				"短时间内攻击 %d 个不同目标（共 %d 次攻击，切换分数 %.2f）",
				uniqueTargets, len(items), switchScore,
			), 1.0, map[string]any{
				"unique_targets": uniqueTargets,
				"total_attacks":  len(items),
				"switch_score":   switchScore,
				"max_switch":     s.maxSwitch,
				"window_ms":      s.windowMs,
			})
		}
	}

	// 6. 攻击方向检测
	s.checkDirection(ctx, evt)
}

// extractTargetID 从事件中提取目标标识。
//
// 优先使用 entity_id（非 0 时）；若 entity_id 为 0（proto3 默认值），
// 则回退到基于实体类型和坐标的复合 ID。
func (s *KillAuraSub) extractTargetID(evt *matrixpb.EntityDamageEvent) string {
	if evt.GetEntityId() != 0 {
		return fmt.Sprintf("%d", evt.GetEntityId())
	}
	// proto3 默认值 0 → 使用坐标哈希作为回退
	return fmt.Sprintf("%s:%.0f:%.0f:%.0f",
		evt.GetEntityType(),
		evt.GetEntityX(),
		evt.GetEntityY(),
		evt.GetEntityZ(),
	)
}

// checkDirection 检测攻击方向是否异常。
//
// 计算玩家视线方向（yaw/pitch）与玩家到目标方向之间的夹角，
// 若超过 angleThreshold 则标记为方向异常。
func (s *KillAuraSub) checkDirection(ctx context.Context, evt *matrixpb.EntityDamageEvent) {
	// 玩家眼睛位置（站立时 +1.62）
	eyePos := [3]float64{
		evt.GetPlayerX(),
		evt.GetPlayerY() + 1.62,
		evt.GetPlayerZ(),
	}

	// 目标中心位置
	targetPos := [3]float64{
		evt.GetEntityX(),
		evt.GetEntityY(),
		evt.GetEntityZ(),
	}

	// 玩家到目标的方向向量
	dx := targetPos[0] - eyePos[0]
	dy := targetPos[1] - eyePos[1]
	dz := targetPos[2] - eyePos[2]
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if dist < 1e-10 {
		return // 距离过近，无法判断方向
	}

	// 归一化目标方向
	ndx := dx / dist
	ndy := dy / dist
	ndz := dz / dist

	// 玩家视线方向（复用 sub_reach 的 yawPitchToDirection）
	lookDir := yawPitchToDirection(float64(evt.GetPlayerYaw()), float64(evt.GetPlayerPitch()))

	// 计算夹角（点积 → arccos）
	dot := lookDir.X*ndx + lookDir.Y*ndy + lookDir.Z*ndz
	// 钳制到 [-1, 1] 避免 arccof NaN
	if dot > 1.0 {
		dot = 1.0
	}
	if dot < -1.0 {
		dot = -1.0
	}
	angleDeg := math.Acos(dot) * 180.0 / math.Pi

	if angleDeg > s.angleThreshold {
		s.FlagViaVL(ctx, "KILL_AURA_DIRECTION", fmt.Sprintf(
			"攻击方向异常：玩家视线与目标方向夹角 %.1f° 超过阈值 %.1f°",
			angleDeg, s.angleThreshold,
		), 1.0, map[string]any{
			"angle":           angleDeg,
			"angle_threshold": s.angleThreshold,
			"player_yaw":      evt.GetPlayerYaw(),
			"player_pitch":    evt.GetPlayerPitch(),
			"player_x":        evt.GetPlayerX(),
			"player_y":        evt.GetPlayerY(),
			"player_z":        evt.GetPlayerZ(),
			"entity_x":        evt.GetEntityX(),
			"entity_y":        evt.GetEntityY(),
			"entity_z":        evt.GetEntityZ(),
		})
	}
}

// countUniqueTargets 统计攻击记录中不同目标 ID 的数量。
func countUniqueTargets(items []AttackRecord) int {
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		seen[item.TargetID] = true
	}
	return len(seen)
}
