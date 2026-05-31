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

// AutoClickerSub 基于 CPS（每秒点击次数）与点击间隔标准差的自动连点器检测。
//
// 检测逻辑：
//   - CPS 检测：时间窗口内点击频率超过 maxCps → 自动连点器嫌疑
//   - 均匀分布检测：点击间隔标准差低于 minStddev → 机械式均匀点击特征
type AutoClickerSub struct {
	BaseChecker
	mu              sync.Mutex
	clickTimestamps *component.SlidingWindow[int64]

	maxCps    float64
	windowMs  int64
	minStddev float64
}

// NewAutoClickerSub 创建 AutoClickerSub 实例，从环境变量读取配置。
func NewAutoClickerSub(warner *components.AntiCheatWarning) *AutoClickerSub {
	maxCps := xEnv.GetEnvFloat(bConst.EnvMatrixAcAutoclickerMaxCps, 16)
	windowMs := xEnv.GetEnvInt64(bConst.EnvMatrixAcAutoclickerWindowMs, 1000)
	minStddev := xEnv.GetEnvFloat(bConst.EnvMatrixAcAutoclickerMinStddev, 15.0)

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

// Name 返回检查器名称。
func (s *AutoClickerSub) Name() string {
	return "auto_clicker"
}

// Process 处理单条遥测数据，根据事件类型分发处理。
func (s *AutoClickerSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, evt := range msg.GetEntityDamages() {
		s.checkAutoClicker(ctx, evt)
	}

	return nil
}

// Drain 排水时无需操作。
func (s *AutoClickerSub) Drain(_ context.Context) error {
	return nil
}

// checkAutoClicker 执行自动连点器检测核心逻辑。
func (s *AutoClickerSub) checkAutoClicker(ctx context.Context, evt *matrixpb.EntityDamageEvent) {
	if evt == nil {
		return
	}

	timestamp := evt.GetTimestamp()

	// 1. 记录点击时间戳
	s.clickTimestamps.Add(timestamp, timestamp)

	// 2. 获取窗口内的点击记录
	items := s.clickTimestamps.Items()

	// 3. CPS 检测：至少 10 次点击才有统计意义
	if len(items) >= 10 {
		cps := float64(len(items)) / (float64(s.windowMs) / 1000.0)

		if cps > s.maxCps {
			s.FlagViaVL(ctx, "AUTOCLICKER_CPS", fmt.Sprintf(
				"CPS %.2f 超过最大阈值 %.2f（窗口 %d ms 内 %d 次点击）",
				cps, s.maxCps, s.windowMs, len(items),
			), 1.0, map[string]any{
				"cps":       cps,
				"max_cps":   s.maxCps,
				"window_ms": s.windowMs,
				"clicks":    len(items),
			})
			return
		}
	}

	// 4. 均匀分布检测：至少 20 次点击才计算标准差
	if len(items) > 20 {
		intervals := make([]float64, len(items)-1)
		for i := 1; i < len(items); i++ {
			intervals[i-1] = float64(items[i] - items[i-1])
		}

		stddev := standardDeviation(intervals)
		if stddev < s.minStddev {
			s.FlagViaVL(ctx, "AUTOCLICKER_UNIFORM", fmt.Sprintf(
				"点击间隔标准差 %.2f 低于阈值 %.2f（疑似机械式均匀点击）",
				stddev, s.minStddev,
			), 1.0, map[string]any{
				"stddev":     stddev,
				"min_stddev": s.minStddev,
				"clicks":     len(items),
			})
			return
		}
	}

	// 5. 正常行为 → 奖励 VL 衰减
	s.RewardViaVL()
}

// average 计算浮点数切片的算术平均值。
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// standardDeviation 计算浮点数切片的总体标准差。
func standardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	avg := average(values)
	sum := 0.0
	for _, v := range values {
		diff := v - avg
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(values)))
}
