package checker

import (
	"context"
	"fmt"
	"sync"

	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// NoFallSub 地面伪装检测：检测玩家声称 on_ground 但实际正在下落的 NoFall 作弊行为。
// 使用连续 tick 计数器 + VLTracker 实现渐进式违规累加与衰减。
type NoFallSub struct {
	BaseChecker

	mu            sync.Mutex
	prevPosY      float64
	hasPrev       bool
	noFallCounter int

	// Config
	fallSpeed        float64
	consecutiveTicks int
}

// NewNoFallSub 创建 NoFallSub 实例，从环境变量读取配置。
func NewNoFallSub(warner *components.AntiCheatWarning) *NoFallSub {
	fallSpeed := xEnv.GetEnvFloat(bConst.EnvMatrixAcNofallFallSpeed, 0.1)
	consecutiveTicks := xEnv.GetEnvInt(bConst.EnvMatrixAcNofallConsecutiveTicks, 3)
	vlThreshold := 5.0

	vl := component.NewVLTracker(0.5, vlThreshold)
	baseChecker := NewBaseChecker(warner, vl)

	return &NoFallSub{
		BaseChecker:      *baseChecker,
		fallSpeed:        fallSpeed,
		consecutiveTicks: consecutiveTicks,
	}
}

// Name 返回 sub 名称标识。
func (n *NoFallSub) Name() string {
	return "no_fall"
}

// Process 处理单条遥测数据，根据 Payload 类型分发处理。
func (n *NoFallSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	switch msg.Payload.(type) {
	case *matrixpb.MatrixTelemetryRequest_TelemetryTick:
		n.checkNoFall(ctx, msg.GetTelemetryTick())
	case *matrixpb.MatrixTelemetryRequest_Teleport,
		*matrixpb.MatrixTelemetryRequest_Respawn,
		*matrixpb.MatrixTelemetryRequest_GameModeChange:
		// 传送/重生/游戏模式变更 → 重置状态
		n.noFallCounter = 0
		n.hasPrev = false
	}

	return nil
}

// Drain 排水完成时调用，无特殊刷盘逻辑。
func (n *NoFallSub) Drain(_ context.Context) error {
	return nil
}

// checkNoFall 执行 NoFall 地面伪装检测核心逻辑。
//
// 检测原理：
//   - 豁免：攀爬（梯子/藤蔓）、水中、岩浆中 → 跳过检测
//   - 当玩家声称 is_on_ground=true 但 deltaY < -fallSpeed（实际正在下落），
//     连续触发 consecutiveTicks 次后判定为 NoFall 作弊
//   - 未触发时逐步恢复计数器（每 tick 最多减 1）
func (n *NoFallSub) checkNoFall(ctx context.Context, tick *matrixpb.TelemetryTick) {
	if tick == nil {
		return
	}

	// 豁免检查：攀爬、水中、岩浆中时不检测
	if tick.GetIsClimbing() || tick.GetIsInWater() || tick.GetIsInLava() {
		return
	}

	posY := tick.GetPosY()

	// 首次 tick 仅记录位置
	if !n.hasPrev {
		n.prevPosY = posY
		n.hasPrev = true
		return
	}

	deltaY := posY - n.prevPosY

	if tick.GetIsOnGround() {
		// 声称在地面但实际正在下落
		if deltaY < -n.fallSpeed {
			n.noFallCounter++
			if n.noFallCounter >= n.consecutiveTicks {
				n.FlagViaVL(ctx, "NOFALL", fmt.Sprintf(
					"地面伪装：声称 on_ground 但 deltaY=%.4f（下落速度超过阈值 %.4f），连续 %d tick",
					deltaY, n.fallSpeed, n.noFallCounter,
				), 1.0, map[string]any{
					"delta_y":           deltaY,
					"fall_speed":        n.fallSpeed,
					"consecutive_count": n.noFallCounter,
					"pos_y":             posY,
					"is_on_ground":      true,
				})
			}
		} else {
			// 正常着地 → 逐步恢复
			n.noFallCounter = max(0, n.noFallCounter-1)
		}
	} else {
		// 不在地面 → 逐步恢复
		n.noFallCounter = max(0, n.noFallCounter-1)
	}

	// 更新位置记录
	n.prevPosY = posY
}
