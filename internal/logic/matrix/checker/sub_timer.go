package checker

import (
	"context"
	"fmt"
	"sync"
	"time"

	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

// TimerSub 基于事务时钟的 Timer 检测，通过对比客户端 tick 累计时间与服务器实际流逝时间，
// 识别加速器类作弊（Timer hack）。
//
// 原理：每个 tick 约 50ms，若客户端上报的 tick 频率远超实际时间流逝（考虑时钟漂移），
// 则判定为 Timer 作弊。
type TimerSub struct {
	BaseChecker

	mu             sync.Mutex
	timerBalance   float64 // 客户端 tick 累计时间（ms）
	sessionStartMs int64   // 会话开始时的服务器时间戳（ms）
	initialized    bool    // 是否已完成首次 tick 初始化

	clockDrift float64 // 时钟漂移容忍度（ms），从环境变量读取
}

// NewTimerSub 创建 TimerSub 实例，从环境变量读取配置。
func NewTimerSub(warner *components.AntiCheatWarning) *TimerSub {
	clockDrift := xEnv.GetEnvFloat(bConst.EnvMatrixAcTimerClockDrift, 150)
	vlThreshold := xEnv.GetEnvFloat(bConst.EnvMatrixAcTimerVlThreshold, 5.0)

	vl := component.NewVLTracker(0.5, vlThreshold)
	baseChecker := NewBaseChecker(warner, vl)

	return &TimerSub{
		BaseChecker: *baseChecker,
		clockDrift:  clockDrift,
	}
}

// Name 返回 sub 名称标识。
func (t *TimerSub) Name() string {
	return "timer"
}

// Process 处理单条遥测数据，根据 Payload 类型分发处理。
func (t *TimerSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, tick := range msg.GetTelemetryTicks() {
		t.checkTimer(ctx, tick)
	}
	if len(msg.GetTeleports()) > 0 || len(msg.GetRespawns()) > 0 || len(msg.GetGameModeChanges()) > 0 {
		// 传送/重生/游戏模式变更 → 重置时间追踪
		t.timerBalance = 0
		t.initialized = false
	}

	return nil
}

// Drain 排水完成时调用，无特殊刷盘逻辑。
func (t *TimerSub) Drain(_ context.Context) error {
	return nil
}

// checkTimer 执行事务时钟检测核心逻辑。
//
// 每个游戏 tick 约 50ms，将 tickBalance 累加 50ms，
// 与服务器实际流逝时间（含 clockDrift 容差）对比：
//   - 若 timerBalance > sessionElapsed + clockDrift → 疑似 Timer 加速
//   - 否则 → RewardViaVL 衰减违规值
//
// 同时设有上限（sessionElapsed + clockDrift + 500）防止漂移过大。
func (t *TimerSub) checkTimer(ctx context.Context, tick *matrixpb.TelemetryTick) {
	if tick == nil {
		return
	}

	// 首次 tick：初始化会话起始时间
	if !t.initialized {
		t.sessionStartMs = tick.GetTimestamp()
		t.initialized = true
	}

	// 每个 tick 约 50ms
	t.timerBalance += 50

	// 当前服务器时间
	nowMs := time.Now().UnixMilli()

	// 会话实际流逝时间
	sessionElapsed := float64(nowMs - t.sessionStartMs)

	if t.timerBalance > sessionElapsed+t.clockDrift {
		// Timer 加速检测：客户端 tick 累计时间超过实际时间 + 容差
		t.FlagViaVL(ctx, "TIMER", fmt.Sprintf(
			"客户端 tick 累计 %.0fms 超过实际流逝 %.0fms + 容差 %.0fms",
			t.timerBalance, sessionElapsed, t.clockDrift,
		), 1.0, map[string]any{
			"timer_balance":   t.timerBalance,
			"session_elapsed": sessionElapsed,
			"clock_drift":     t.clockDrift,
			"tick_timestamp":  tick.GetTimestamp(),
		})

		// 上限：防止 timerBalance 漂移过大
		upperBound := sessionElapsed + t.clockDrift + 500
		if t.timerBalance > upperBound {
			t.timerBalance = upperBound
		}
	} else {
		t.RewardViaVL()
	}
}
