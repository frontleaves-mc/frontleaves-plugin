package matrix

import (
	"context"
	"sync"

	matrixpb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/matrix/v1"
)

// MatrixSub Matrix 数据处理插件接口
type MatrixSub interface {
	// Name 返回 sub 名称（用于日志标识）
	Name() string
	// Process 处理单条遥测数据
	Process(ctx context.Context, msg *matrixpb.MatrixTelemetryRequest) error
	// Drain 排水完成时调用，用于刷盘残留数据
	Drain(ctx context.Context) error
}

// MatrixSubRegistry MatrixSub 插件注册表
type MatrixSubRegistry struct {
	mu   sync.RWMutex
	subs []MatrixSub
}

// Register 注册一个新的 MatrixSub 插件
func (r *MatrixSubRegistry) Register(sub MatrixSub) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subs = append(r.subs, sub)
}

// GetAll 获取所有已注册的 MatrixSub 插件
func (r *MatrixSubRegistry) GetAll() []MatrixSub {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]MatrixSub, len(r.subs))
	copy(result, r.subs)
	return result
}
