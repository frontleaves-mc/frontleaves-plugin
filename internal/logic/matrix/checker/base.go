package checker

import (
	"context"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/component"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix/components"
)

// BaseChecker 组合 VLTracker + AntiCheatWarning，供各 checker Sub 嵌入使用。
// 每个 checker Sub 拥有自己的 VLTracker 实例，但共享同一个 AntiCheatWarning（同玩家）。
type BaseChecker struct {
	warner *components.AntiCheatWarning
	vl     *component.VLTracker
}

// NewBaseChecker 创建 BaseChecker 实例。
func NewBaseChecker(warner *components.AntiCheatWarning, vl *component.VLTracker) *BaseChecker {
	return &BaseChecker{warner: warner, vl: vl}
}

// FlagViaVL 累加 VL；当达到阈值时触发警告并重置 VL。
func (b *BaseChecker) FlagViaVL(ctx context.Context, warningType, description string, amount float64, contextData map[string]any) {
	b.vl.Flag(amount)
	if b.vl.ShouldFlag() {
		b.warner.Trigger(ctx, warningType, description, contextData)
		b.vl.Reset()
	}
}

// RewardViaVL 奖励 VL（降低违规值）。
func (b *BaseChecker) RewardViaVL() {
	b.vl.Reward()
}

// HandleReset 基础空操作；个别 checker 可按需直接调用 vl.Reset()。
func (b *BaseChecker) HandleReset() {
	// no-op by default
}

// VL 返回底层的 VLTracker，用于测试目的。
func (b *BaseChecker) VL() *component.VLTracker {
	return b.vl
}
