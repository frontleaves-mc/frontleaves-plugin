package checker

import (
	"context"
	"testing"

	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/stretchr/testify/assert"
)

// newTestFastBreakSub 创建一个用于测试的 FastBreakSub，使用低 VL 阈值以快速触发。
func newTestFastBreakSub() *FastBreakSub {
	vl := component.NewVLTracker(0.3, 1.0) // decay=0.3, threshold=1.0 (low for testing)
	baseChecker := NewBaseChecker(nil, vl)

	return &FastBreakSub{
		BaseChecker:    *baseChecker,
		breakRecords:   component.NewRingBuffer[BreakRecord](20),
		windowSize:     20,
		ratioThreshold: 0.5,
	}
}

// makeBreakMsg 构造 BlockBreakEvent 的 MatrixTelemetryRequest。
func makeBreakMsg(material, toolUsed string, ts int64) *matrixpb.MatrixTelemetryRequest {
	return &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Payload: &matrixpb.MatrixTelemetryRequest_BlockBreak{
			BlockBreak: &matrixpb.BlockBreakEvent{
				PlayerUuid: "test-uuid",
				PlayerName: "TestPlayer",
				Material:   material,
				ToolUsed:   toolUsed,
				Timestamp:  ts,
			},
		},
	}
}

// TestFastBreakSub_Name 验证 Name 返回 "fast_break"。
func TestFastBreakSub_Name(t *testing.T) {
	sub := newTestFastBreakSub()
	assert.Equal(t, "fast_break", sub.Name())
}

// TestFastBreakSub_NormalMining 验证正常挖矿速度不会触发 flag。
// STONE 预期 1.25s，以 1.3s 间隔挖矿 → ratio = 1.3/1.25 = 1.04 > 0.5 → 正常。
func TestFastBreakSub_NormalMining(t *testing.T) {
	sub := newTestFastBreakSub()
	ctx := context.Background()

	// 第 1 次破坏 STONE at t=1000ms
	_ = sub.Process(ctx, makeBreakMsg("STONE", "DIAMOND_PICKAXE", 1000))

	// 第 2 次破坏 STONE at t=2300ms → interval=1300ms=1.3s, ratio=1.3/1.25=1.04 > 0.5
	_ = sub.Process(ctx, makeBreakMsg("STONE", "DIAMOND_PICKAXE", 2300))

	// 第 3 次破坏 STONE at t=3600ms → interval=1300ms=1.3s, ratio=1.04 > 0.5
	_ = sub.Process(ctx, makeBreakMsg("STONE", "DIAMOND_PICKAXE", 3600))

	// 正常速度 → reward → VL 应为 0（初始 0，reward 后仍为 0）
	assert.Equal(t, 0.0, sub.VL().Violations(), "normal mining speed should not flag")
}

// TestFastBreakSub_TooFast 验证瞬间破坏触发 flag。
// STONE 预期 1.25s，以 0.1s 间隔破坏 → ratio = 0.1/1.25 = 0.08 < 0.5 → flag。
func TestFastBreakSub_TooFast(t *testing.T) {
	sub := newTestFastBreakSub()
	ctx := context.Background()

	// 第 1 次破坏 STONE at t=1000ms
	_ = sub.Process(ctx, makeBreakMsg("STONE", "DIAMOND_PICKAXE", 1000))

	// 第 2 次破坏 STONE at t=1100ms → interval=100ms=0.1s, ratio=0.1/1.25=0.08 < 0.5
	_ = sub.Process(ctx, makeBreakMsg("STONE", "DIAMOND_PICKAXE", 1100))

	assert.Greater(t, sub.VL().Violations(), 0.0, "instant break should flag")
}

// TestFastBreakSub_ObsidianSlow 验证黑曜石正常耗时不会触发 flag。
// OBSIDIAN 预期 9.4s，以 10.0s 间隔破坏 → ratio = 10.0/9.4 = 1.064 > 0.5 → 正常。
func TestFastBreakSub_ObsidianSlow(t *testing.T) {
	sub := newTestFastBreakSub()
	ctx := context.Background()

	// 第 1 次破坏 OBSIDIAN at t=0ms
	_ = sub.Process(ctx, makeBreakMsg("OBSIDIAN", "DIAMOND_PICKAXE", 0))

	// 第 2 次破坏 OBSIDIAN at t=10000ms → interval=10000ms=10.0s, ratio=10.0/9.4=1.064 > 0.5
	_ = sub.Process(ctx, makeBreakMsg("OBSIDIAN", "DIAMOND_PICKAXE", 10000))

	// 第 3 次破坏 OBSIDIAN at t=20000ms → interval=10000ms=10.0s, ratio=1.064 > 0.5
	_ = sub.Process(ctx, makeBreakMsg("OBSIDIAN", "DIAMOND_PICKAXE", 20000))

	assert.Equal(t, 0.0, sub.VL().Violations(), "obsidian at normal speed should not flag")
}

// TestFastBreakSub_UnknownMaterial 验证未知方块使用默认 1.0s 预期时间。
// 未知方块 DIRT 默认 1.0s，以 0.8s 间隔 → ratio = 0.8/1.0 = 0.8 > 0.5 → 正常。
func TestFastBreakSub_UnknownMaterial(t *testing.T) {
	sub := newTestFastBreakSub()
	ctx := context.Background()

	// DIRT 不在 materialExpectedTime 中，默认 1.0s
	_ = sub.Process(ctx, makeBreakMsg("DIRT", "DIAMOND_SHOVEL", 1000))
	_ = sub.Process(ctx, makeBreakMsg("DIRT", "DIAMOND_SHOVEL", 1800)) // interval=0.8s, ratio=0.8/1.0=0.8

	assert.Equal(t, 0.0, sub.VL().Violations(), "unknown material at 0.8s interval should not flag (ratio=0.8>0.5)")
}

// TestFastBreakSub_ObsidianTooFast 验证黑曜石瞬间破坏触发 flag。
// OBSIDIAN 预期 9.4s，以 0.5s 间隔破坏 → ratio = 0.5/9.4 = 0.053 < 0.5 → flag。
func TestFastBreakSub_ObsidianTooFast(t *testing.T) {
	sub := newTestFastBreakSub()
	ctx := context.Background()

	_ = sub.Process(ctx, makeBreakMsg("OBSIDIAN", "DIAMOND_PICKAXE", 1000))
	_ = sub.Process(ctx, makeBreakMsg("OBSIDIAN", "DIAMOND_PICKAXE", 1500)) // interval=0.5s, ratio=0.053

	assert.Greater(t, sub.VL().Violations(), 0.0, "obsidian instant break should flag")
}

// TestFastBreakSub_SingleBreakNoFlag 验证单次破坏不触发检测（记录数 < 2）。
func TestFastBreakSub_SingleBreakNoFlag(t *testing.T) {
	sub := newTestFastBreakSub()
	ctx := context.Background()

	_ = sub.Process(ctx, makeBreakMsg("STONE", "DIAMOND_PICKAXE", 1000))

	assert.Equal(t, 0.0, sub.VL().Violations(), "single break should not flag")
}

// TestFastBreakSub_NilPayload 验证 nil BlockBreakEvent 不崩溃。
func TestFastBreakSub_NilPayload(t *testing.T) {
	sub := newTestFastBreakSub()
	ctx := context.Background()

	msg := &matrixpb.MatrixTelemetryRequest{
		ServerName: "test-server",
		Payload: &matrixpb.MatrixTelemetryRequest_BlockBreak{
			BlockBreak: nil,
		},
	}

	// 不应 panic
	_ = sub.Process(ctx, msg)
	assert.Equal(t, 0.0, sub.VL().Violations(), "nil block break should not flag")
}
