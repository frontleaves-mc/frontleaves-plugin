package logic

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

// SchedulerConfigSnapshot 调度配置的内存强类型快照
type SchedulerConfigSnapshot struct {
	Mode            entity.ScheduleMode
	IntervalSeconds int
	IsEnabled       bool
}

// SchedulerConfigLoader 从 KV Config 表加载并缓存调度配置
// 提供线程安全的快照访问，避免引擎高频路径上的字符串解析
type SchedulerConfigLoader struct {
	repo *repository.ConfigRepository
	mu   sync.RWMutex
	cfg  *SchedulerConfigSnapshot
	log  *xLog.LogNamedLogger
}

// NewSchedulerConfigLoader 创建配置加载器
func NewSchedulerConfigLoader(repo *repository.ConfigRepository) *SchedulerConfigLoader {
	return &SchedulerConfigLoader{
		repo: repo,
		log:  xLog.WithName(xLog.NamedLOGC, "SchedulerConfigLoader"),
	}
}

// Load 从数据库重新加载配置并更新内存快照
func (l *SchedulerConfigLoader) Load(ctx context.Context) *xError.Error {
	l.log.Info(ctx, "Load - 加载调度配置")

	values, xErr := l.repo.BatchGet(ctx, bConst.SchedulerConfigNamespace)
	if xErr != nil {
		return xErr
	}

	// 解析配置值（带默认值保护）
	modeStr := values[bConst.SchedulerConfigMode]
	intervalStr := values[bConst.SchedulerConfigIntervalSeconds]
	enabledStr := values[bConst.SchedulerConfigIsEnabled]

	mode := entity.ScheduleModeFixedInterval
	if modeStr != "" {
		if parsed, err := strconv.ParseInt(modeStr, 10, 16); err == nil {
			mode = entity.ScheduleMode(parsed)
		}
	}

	interval := 60
	if intervalStr != "" {
		if parsed, err := strconv.Atoi(intervalStr); err == nil {
			interval = parsed
		}
	}

	isEnabled := false
	if enabledStr != "" {
		if parsed, err := strconv.ParseBool(enabledStr); err == nil {
			isEnabled = parsed
		}
	}

	newCfg := &SchedulerConfigSnapshot{
		Mode:            mode,
		IntervalSeconds: interval,
		IsEnabled:       isEnabled,
	}

	l.mu.Lock()
	l.cfg = newCfg
	l.mu.Unlock()

	l.log.Info(ctx, fmt.Sprintf("Load - 配置加载完成: mode=%v, interval=%d, enabled=%t", mode, interval, isEnabled))
	return nil
}

// Get 获取当前配置快照（线程安全）
// 如果从未加载过，返回零值快照（Mode=FixedInterval, Interval=60, IsEnabled=false）
func (l *SchedulerConfigLoader) Get() *SchedulerConfigSnapshot {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.cfg == nil {
		return &SchedulerConfigSnapshot{
			Mode:            entity.ScheduleModeFixedInterval,
			IntervalSeconds: 60,
			IsEnabled:       false,
		}
	}
	return l.cfg
}
