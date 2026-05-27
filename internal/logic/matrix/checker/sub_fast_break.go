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

// materialExpectedTime 方块预期破坏时间表（秒），用于 FastBreak 检测。
// 未知方块默认使用 1.0 秒。
var materialExpectedTime = map[string]float64{
	"STONE":                    1.25,
	"COBBLESTONE":              1.25,
	"DEEPSLATE":                1.5,
	"IRON_ORE":                 0.75,
	"DEEPSLATE_IRON_ORE":       0.85,
	"GOLD_ORE":                 0.75,
	"DEEPSLATE_GOLD_ORE":       0.85,
	"DIAMOND_ORE":              0.75,
	"DEEPSLATE_DIAMOND_ORE":    0.85,
	"EMERALD_ORE":              0.75,
	"DEEPSLATE_EMERALD_ORE":    0.85,
	"OBSIDIAN":                 9.4,
	"NETHERITE_BLOCK":          8.0,
	"ANCIENT_DEBRIS":           3.0,
	"ENDER_CHEST":              9.4,
}

// BreakRecord 单次方块破坏的记录。
type BreakRecord struct {
	Timestamp int64
	Material  string
	ToolUsed  string
}

// FastBreakSub 基于滑动窗口的快速破坏检测。
// 通过 RingBuffer 维护最近 N 次破坏记录，对比连续破坏的时间间隔与预期时间，
// 当 ratio = actualInterval / expectedTime 低于阈值时判定为作弊。
type FastBreakSub struct {
	BaseChecker

	mu           sync.Mutex
	breakRecords *component.RingBuffer[BreakRecord]

	windowSize     int
	ratioThreshold float64
}

// NewFastBreakSub 创建 FastBreakSub 实例，从环境变量读取配置。
func NewFastBreakSub(warner *components.AntiCheatWarning) *FastBreakSub {
	windowSize := int(xEnv.GetEnvFloat(bConst.EnvMatrixAcFastbreakWindowSize, 20))
	ratioThreshold := xEnv.GetEnvFloat(bConst.EnvMatrixAcFastbreakRatioThreshold, 0.5)

	vl := component.NewVLTracker(0.3, 5.0)
	baseChecker := NewBaseChecker(warner, vl)

	return &FastBreakSub{
		BaseChecker:    *baseChecker,
		breakRecords:   component.NewRingBuffer[BreakRecord](windowSize),
		windowSize:     windowSize,
		ratioThreshold: ratioThreshold,
	}
}

// Name 返回 sub 名称标识。
func (f *FastBreakSub) Name() string {
	return "fast_break"
}

// Process 处理单条遥测数据，根据 Payload 类型分发处理。
func (f *FastBreakSub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch msg.Payload.(type) {
	case *matrixpb.MatrixTelemetryRequest_BlockBreak:
		f.checkFastBreak(ctx, msg.GetBlockBreak())
	}

	return nil
}

// Drain 排水完成时调用，无特殊刷盘逻辑。
func (f *FastBreakSub) Drain(_ context.Context) error {
	return nil
}

// checkFastBreak 执行快速破坏检测核心逻辑。
//
// 检测流程：
//  1. 将当前破坏事件记录 Push 到 RingBuffer
//  2. 当记录数 >= 2 时，遍历连续的破坏对
//  3. 对每对计算 ratio = actualInterval / expectedTime
//  4. ratio < ratioThreshold → 太快了，FlagViaVL
//  5. 否则 → RewardViaVL
func (f *FastBreakSub) checkFastBreak(ctx context.Context, evt *matrixpb.BlockBreakEvent) {
	if evt == nil {
		return
	}

	record := BreakRecord{
		Timestamp: evt.GetTimestamp(),
		Material:  evt.GetMaterial(),
		ToolUsed:  evt.GetToolUsed(),
	}

	f.breakRecords.Push(record)

	if f.breakRecords.Size() < 2 {
		return
	}

	items := f.breakRecords.Items()
	flagged := false

	for i := 1; i < len(items); i++ {
		actualInterval := float64(items[i].Timestamp-items[i-1].Timestamp) / 1000.0
		expectedTime, ok := materialExpectedTime[items[i].Material]
		if !ok {
			expectedTime = 1.0
		}

		if expectedTime <= 0 {
			continue
		}

		ratio := actualInterval / expectedTime

		if ratio < f.ratioThreshold {
			f.FlagViaVL(ctx, "FAST_BREAK", fmt.Sprintf(
				"方块破坏速度异常：实际间隔 %.3fs / 预期 %.3fs = ratio %.3f < 阈值 %.2f",
				actualInterval, expectedTime, ratio, f.ratioThreshold,
			), 1.0, map[string]any{
				"actual_interval":   actualInterval,
				"expected_time":     expectedTime,
				"ratio":             ratio,
				"ratio_threshold":   f.ratioThreshold,
				"material":          items[i].Material,
				"tool_used":         items[i].ToolUsed,
				"window_size":       f.windowSize,
			})
			flagged = true
			break
		}
	}

	if !flagged {
		f.RewardViaVL()
	}
}
