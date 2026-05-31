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

// XRaySub 基于稀有矿物采集比例异常的 XRay 检测。
// 在滑动窗口内统计各类矿物采集数量，当稀有矿物（钻石、绿宝石、远古残骸）
// 的采集比例超过阈值时累加 VL。
type XRaySub struct {
	BaseChecker

	mu           sync.Mutex
	breakHistory map[string]int // material → count
	totalBreaks  int

	// Config (from env vars)
	windowSize   int
	diamondRatio float64
	emeraldRatio float64
	ancientRatio float64
}

// 稀有矿物 material 名称集合
var (
	diamondMaterials = map[string]bool{
		"DIAMOND_ORE":          true,
		"DEEPSLATE_DIAMOND_ORE": true,
	}
	emeraldMaterials = map[string]bool{
		"EMERALD_ORE":          true,
		"DEEPSLATE_EMERALD_ORE": true,
	}
	ancientMaterials = map[string]bool{
		"ANCIENT_DEBRIS": true,
	}
)

// NewXRaySub 创建 XRaySub 实例，从环境变量读取配置。
func NewXRaySub(warner *components.AntiCheatWarning) *XRaySub {
	windowSize := xEnv.GetEnvInt(bConst.EnvMatrixAcXrayWindowSize, 100)
	diamondRatio := xEnv.GetEnvFloat(bConst.EnvMatrixAcXrayDiamondRatio, 0.05)
	emeraldRatio := xEnv.GetEnvFloat(bConst.EnvMatrixAcXrayEmeraldRatio, 0.03)
	ancientRatio := xEnv.GetEnvFloat(bConst.EnvMatrixAcXrayAncientRatio, 0.01)
	// VL 配置：decay=0.3, setback=5.0
	vl := component.NewVLTracker(0.3, 5.0)
	baseChecker := NewBaseChecker(warner, vl)

	return &XRaySub{
		BaseChecker:  *baseChecker,
		breakHistory: make(map[string]int),
		windowSize:   windowSize,
		diamondRatio: diamondRatio,
		emeraldRatio: emeraldRatio,
		ancientRatio: ancientRatio,
	}
}

// Name 返回 sub 名称标识。
func (x *XRaySub) Name() string {
	return "xray"
}

// Process 处理单条遥测数据，仅对 BlockBreakEvent 执行检测。
func (x *XRaySub) Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	for _, evt := range msg.GetBlockBreaks() {
		x.checkXRay(ctx, evt)
	}

	return nil
}

// Drain 排水完成时调用，无特殊刷盘逻辑。
func (x *XRaySub) Drain(_ context.Context) error {
	return nil
}

// checkXRay 执行 XRay 检测核心逻辑。
//
// 在滑动窗口内统计方块破坏记录，当窗口满时计算稀有矿物比例。
// 若任何稀有矿物比例超过阈值，则按超出量 × 10 累加 suspiciousScore，
// 当 suspiciousScore > 0 时通过 VLTracker 上报。
func (x *XRaySub) checkXRay(ctx context.Context, evt *matrixpb.BlockBreakEvent) {
	if evt == nil {
		return
	}

	material := evt.GetMaterial()
	x.breakHistory[material]++
	x.totalBreaks++

	if x.totalBreaks < x.windowSize {
		return
	}

	// 窗口已满，计算稀有矿物比例
	suspiciousScore := 0.0

	diamondCount := x.countMaterials(diamondMaterials)
	emeraldCount := x.countMaterials(emeraldMaterials)
	ancientCount := x.countMaterials(ancientMaterials)

	diamondActual := float64(diamondCount) / float64(x.windowSize)
	emeraldActual := float64(emeraldCount) / float64(x.windowSize)
	ancientActual := float64(ancientCount) / float64(x.windowSize)

	if excess := diamondActual - x.diamondRatio; excess > 0 {
		suspiciousScore += excess * 10
	}
	if excess := emeraldActual - x.emeraldRatio; excess > 0 {
		suspiciousScore += excess * 10
	}
	if excess := ancientActual - x.ancientRatio; excess > 0 {
		suspiciousScore += excess * 10
	}

	if suspiciousScore > 0 {
		desc := fmt.Sprintf(
			"稀有矿物采集比例异常 - 钻石: %.2f%%, 绿宝石: %.2f%%, 远古残骸: %.2f%%",
			diamondActual*100, emeraldActual*100, ancientActual*100,
		)
		data := map[string]any{
			"window_size":    x.windowSize,
			"total_breaks":   x.totalBreaks,
			"diamond_ratio":  diamondActual,
			"emerald_ratio":  emeraldActual,
			"ancient_ratio":  ancientActual,
			"suspicious_score": suspiciousScore,
		}
		x.FlagViaVL(ctx, "XRAY", desc, suspiciousScore, data)
	}

	// 重置窗口
	x.breakHistory = make(map[string]int)
	x.totalBreaks = 0
}

// countMaterials 统计指定矿物集合在当前窗口中的总数量。
func (x *XRaySub) countMaterials(materials map[string]bool) int {
	count := 0
	for mat, ok := range materials {
		if ok {
			count += x.breakHistory[mat]
		}
	}
	return count
}
