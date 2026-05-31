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

// AimbotSub 基于视角变化 GCD 的灵敏度估算检测。
// 通过计算 pitch delta 序列的最大公约数，反推鼠标灵敏度设置；
// 若灵敏度在短时间内发生跳变，则判定为疑似 Aimbot（外挂自动瞄准）。
type AimbotSub struct {
	BaseChecker

	mu          sync.Mutex
	yawDeltas   *component.RingBuffer[float64]
	pitchDeltas *component.RingBuffer[float64]
	prevYaw     float64
	prevPitch   float64
	hasPrev     bool

	sensitivityEstimate float64
	hasEstimate         bool

	// Config (from env vars)
	sampleWindow    int
	sensitivityDelta float64
}

// NewAimbotSub 创建 AimbotSub 实例，从环境变量读取配置。
func NewAimbotSub(warner *components.AntiCheatWarning) *AimbotSub {
	sampleWindow := xEnv.GetEnvInt(bConst.EnvMatrixAcAimbotSampleWindow, 30)
	sensitivityDelta := xEnv.GetEnvFloat(bConst.EnvMatrixAcAimbotSensitivityDelta, 0.15)

	vl := component.NewVLTracker(0.3, 5.0)
	baseChecker := NewBaseChecker(warner, vl)

	return &AimbotSub{
		BaseChecker:      *baseChecker,
		yawDeltas:        component.NewRingBuffer[float64](sampleWindow),
		pitchDeltas:      component.NewRingBuffer[float64](sampleWindow),
		sampleWindow:     sampleWindow,
		sensitivityDelta: sensitivityDelta,
	}
}

// Name 返回 sub 名称标识。
func (s *AimbotSub) Name() string {
	return "aimbot"
}

// Process 处理单条遥测数据，根据 Payload 类型分发处理。
func (s *AimbotSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tick := range msg.GetTelemetryTicks() {
		s.checkAimbot(ctx, float64(tick.GetYaw()), float64(tick.GetPitch()))
	}
	for _, evt := range msg.GetEntityDamages() {
		s.checkAimbot(ctx, float64(evt.GetPlayerYaw()), float64(evt.GetPlayerPitch()))
	}
	if len(msg.GetTeleports()) > 0 || len(msg.GetRespawns()) > 0 || len(msg.GetGameModeChanges()) > 0 {
		s.reset()
	}

	return nil
}

// Drain 排水完成时调用，无特殊刷盘逻辑。
func (s *AimbotSub) Drain(_ context.Context) error {
	return nil
}

// reset 重置追踪状态（传送/重生/模式变更时调用）。
func (s *AimbotSub) reset() {
	s.hasPrev = false
	s.sensitivityEstimate = 0
	s.hasEstimate = false
	s.yawDeltas = component.NewRingBuffer[float64](s.sampleWindow)
	s.pitchDeltas = component.NewRingBuffer[float64](s.sampleWindow)
}

// checkAimbot 执行 GCD 灵敏度估算核心逻辑。
//
// 原理：
//  1. 收集连续 tick 的 pitch 变化量（deltaPitch）到滑动窗口
//  2. 计算所有 pitch delta 的浮点 GCD
//  3. 由 GCD 反推鼠标灵敏度：sensitivity = (cbrt(gcd/0.15/8.0) - 0.2) / 0.6
//  4. 如果灵敏度突然跳变超过阈值 → 疑似外挂修改了瞄准行为
func (s *AimbotSub) checkAimbot(ctx context.Context, yaw, pitch float64) {
	// 首次 tick：仅存储初始值
	if !s.hasPrev {
		s.prevYaw = yaw
		s.prevPitch = pitch
		s.hasPrev = true
		return
	}

	deltaYaw := yaw - s.prevYaw
	deltaPitch := pitch - s.prevPitch

	// 跳过微小变化（玩家未移动视角）
	if math.Abs(deltaYaw) < 0.01 && math.Abs(deltaPitch) < 0.01 {
		s.prevYaw = yaw
		s.prevPitch = pitch
		return
	}

	s.yawDeltas.Push(deltaYaw)
	s.pitchDeltas.Push(deltaPitch)

	// 需要足够样本才能计算 GCD
	if s.pitchDeltas.Size() >= 15 {
		values := s.pitchDeltas.Items()
		gcd := floatGCDOfValues(values)

		// 由 GCD 反推灵敏度
		sensitivity := (math.Cbrt(gcd/0.15/8.0) - 0.2) / 0.6

		// 灵敏度跳变检测
		if s.hasEstimate && math.Abs(sensitivity-s.sensitivityEstimate) > s.sensitivityDelta {
			s.FlagViaVL(ctx, "AIMBOT", fmt.Sprintf(
				"灵敏度估算跳变: %.4f → %.4f (delta=%.4f > threshold=%.4f)",
				s.sensitivityEstimate, sensitivity,
				math.Abs(sensitivity-s.sensitivityEstimate), s.sensitivityDelta,
			), 1.0, map[string]any{
				"prev_sensitivity":  s.sensitivityEstimate,
				"current_sensitivity": sensitivity,
				"sensitivity_delta": s.sensitivityDelta,
				"gcd":               gcd,
				"sample_count":      len(values),
			})
		} else {
			s.RewardViaVL()
		}

		s.sensitivityEstimate = sensitivity
		s.hasEstimate = true
	} else {
		s.RewardViaVL()
	}

	// 更新上一帧视角
	s.prevYaw = yaw
	s.prevPitch = pitch
}

// floatGCD 计算两个浮点数的最大公约数（欧几里得算法的浮点版本）。
func floatGCD(a, b float64) float64 {
	for b > 1e-6 {
		a, b = b, math.Mod(a, b)
	}
	return a
}

// floatGCDOfValues 计算一组浮点数（取绝对值）的 GCD。
func floatGCDOfValues(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	gcd := math.Abs(values[0])
	for i := 1; i < len(values); i++ {
		gcd = floatGCD(gcd, math.Abs(values[i]))
	}
	return gcd
}
