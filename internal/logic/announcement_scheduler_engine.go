package logic

import (
	"context"
	"fmt"
	"sync"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

// PushFunc 是将公告推送到 MC 服务器的函数类型，由上层注入以解耦 gRPC 层
type PushFunc func(ctx context.Context, announcementID, title, content string, annType int32) error

// SchedulerEngine 公告调度引擎，负责按 FixedInterval 或 Sequential 模式循环推送公告
type SchedulerEngine struct {
	mu                sync.RWMutex
	scheduleLogic     *AnnouncementScheduleLogic
	announcementLogic *AnnouncementLogic
	pushFunc          PushFunc
	cancelFunc        context.CancelFunc
	currentScheduleID xSnowflake.SnowflakeID
	currentIndex      int
	ticker            *time.Ticker
	log               *xLog.LogNamedLogger
	running           bool
}

// NewSchedulerEngine 创建调度引擎实例
func NewSchedulerEngine(
	scheduleLogic *AnnouncementScheduleLogic,
	announcementLogic *AnnouncementLogic,
	pushFunc PushFunc,
) *SchedulerEngine {
	return &SchedulerEngine{
		scheduleLogic:     scheduleLogic,
		announcementLogic: announcementLogic,
		pushFunc:          pushFunc,
		log:               xLog.WithName(xLog.NamedLOGC, "SchedulerEngine"),
	}
}

// Start 启动调度引擎，根据 scheduleID 加载调度配置并开始循环推送
func (e *SchedulerEngine) Start(ctx context.Context, scheduleID xSnowflake.SnowflakeID) *xError.Error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 如果已在运行，先停止
	if e.running {
		e.stopLocked()
	}

	// 加载调度信息
	schedule, xErr := e.scheduleLogic.repo.schedule.GetByID(ctx, scheduleID)
	if xErr != nil {
		return xError.NewError(nil, xError.NotFound, "调度不存在或加载失败", true, xErr)
	}

	// 加载调度项
	items, xErr := e.scheduleLogic.repo.item.GetByScheduleID(ctx, scheduleID)
	if xErr != nil {
		return xError.NewError(nil, xError.DatabaseError, "加载调度项失败", false, xErr)
	}
	if len(items) == 0 {
		return xError.NewError(nil, xError.ParameterError, "调度项为空，无法启动", true, nil)
	}

	// 创建可取消的子 context
	childCtx, cancel := context.WithCancel(ctx)
	e.cancelFunc = cancel
	e.currentScheduleID = scheduleID
	e.currentIndex = 0
	e.running = true

	e.log.Info(ctx, fmt.Sprintf("启动调度引擎 - 调度: %s, 模式: %s, 项数: %d",
		schedule.Name, schedule.Mode.String(), len(items)))

	// 根据模式启动不同的 ticker
	switch schedule.Mode {
	case entity.ScheduleModeFixedInterval:
		e.startFixedInterval(childCtx, schedule, items)
	case entity.ScheduleModeSequential:
		e.startSequential(childCtx, schedule, items)
	default:
		cancel()
		e.running = false
		return xError.NewError(nil, xError.ParameterError, xError.ErrMessage(fmt.Sprintf("未知的调度模式: %d", schedule.Mode)), true, nil)
	}

	return nil
}

// Stop 停止调度引擎
func (e *SchedulerEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.stopLocked()
	e.log.Info(context.Background(), "调度引擎已停止")
	return nil
}

// Restart 重启调度引擎，从数据库重新加载活动调度
func (e *SchedulerEngine) Restart(ctx context.Context) *xError.Error {
	// 先停止
	_ = e.Stop()

	// 从数据库加载活动调度
	schedule, _, xErr := e.scheduleLogic.GetActiveScheduleWithItems(ctx)
	if xErr != nil {
		return xError.NewError(nil, xError.DatabaseError, "加载活动调度失败", false, xErr)
	}
	if schedule == nil {
		e.log.Info(ctx, "重启调度引擎 - 无活动调度，跳过启动")
		return nil
	}

	return e.Start(ctx, schedule.ID)
}

// IsRunning 检查调度引擎是否正在运行
func (e *SchedulerEngine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// GetCurrentStatus 获取当前调度引擎的运行状态
func (e *SchedulerEngine) GetCurrentStatus() (scheduleID xSnowflake.SnowflakeID, index int, running bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentScheduleID, e.currentIndex, e.running
}

// RecoverFromDatabase 从数据库恢复调度状态（用于服务启动时自动恢复）
func (e *SchedulerEngine) RecoverFromDatabase(ctx context.Context) *xError.Error {
	e.log.Info(ctx, "RecoverFromDatabase - 从数据库恢复调度引擎")

	schedule, _, xErr := e.scheduleLogic.GetActiveScheduleWithItems(ctx)
	if xErr != nil {
		return xError.NewError(nil, xError.DatabaseError, "恢复调度引擎失败", false, xErr)
	}
	if schedule == nil {
		e.log.Info(ctx, "RecoverFromDatabase - 无活动调度，跳过恢复")
		return nil
	}

	e.log.Info(ctx, fmt.Sprintf("RecoverFromDatabase - 发现活动调度: %s (ID: %s)，开始恢复",
		schedule.Name, schedule.ID.String()))

	return e.Start(ctx, schedule.ID)
}

// --- 全局调度引擎单例 ---

// globalEngine 是全局调度引擎实例，供所有 logic 实例共享访问
var globalEngine *SchedulerEngine

// SetGlobalEngine 设置全局调度引擎实例
func SetGlobalEngine(engine *SchedulerEngine) {
	globalEngine = engine
}

// GetGlobalEngine 获取全局调度引擎实例（可能为 nil，调用方需自行判断）
func GetGlobalEngine() *SchedulerEngine {
	return globalEngine
}

// --- 私有方法 ---

// startFixedInterval 固定间隔模式：每个 tick 推送下一个公告
func (e *SchedulerEngine) startFixedInterval(ctx context.Context, schedule *entity.AnnouncementSchedule, items []entity.AnnouncementScheduleItem) {
	interval := time.Duration(schedule.IntervalSeconds) * time.Second
	e.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-e.ticker.C:
				e.tick(ctx, items)
			}
		}
	}()
}

// startSequential 顺序播放模式：按每个调度项的 DelaySeconds 推送当前公告，然后切换到下一个
func (e *SchedulerEngine) startSequential(ctx context.Context, schedule *entity.AnnouncementSchedule, items []entity.AnnouncementScheduleItem) {
	// 先立即推送第一个
	e.tick(ctx, items)

	// 设置下一个调度项的延迟
	e.resetSequentialTicker(ctx, items)
}

// resetSequentialTicker 重置 Sequential 模式的 ticker 为下一个调度项的延迟
func (e *SchedulerEngine) resetSequentialTicker(ctx context.Context, items []entity.AnnouncementScheduleItem) {
	if e.ticker != nil {
		e.ticker.Stop()
	}

	e.mu.RLock()
	nextIndex := (e.currentIndex + 1) % len(items)
	e.mu.RUnlock()

	delaySeconds := items[nextIndex].DelaySeconds
	if delaySeconds <= 0 {
		delaySeconds = 1 // 最小 1 秒
	}
	delay := time.Duration(delaySeconds) * time.Second

	e.ticker = time.NewTicker(delay)

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-e.ticker.C:
			e.tick(ctx, items)
			e.resetSequentialTicker(ctx, items)
		}
	}()
}

// tick 推送当前索引的公告，然后推进索引（循环）
func (e *SchedulerEngine) tick(ctx context.Context, items []entity.AnnouncementScheduleItem) {
	e.mu.RLock()
	if !e.running || len(items) == 0 {
		e.mu.RUnlock()
		return
	}
	idx := e.currentIndex
	item := items[idx]
	e.mu.RUnlock()

	// 加载公告详情
	announcement, xErr := e.announcementLogic.repo.announcement.GetByID(ctx, item.AnnouncementID)
	if xErr != nil {
		e.log.Error(ctx, fmt.Sprintf("加载公告失败 (ID: %s): %v", item.AnnouncementID.String(), xErr))
		// 即使加载失败也推进索引
		e.advanceIndex(len(items))
		return
	}

	// 推送公告（fire-and-forget：失败不停止引擎）
	if err := e.pushFunc(ctx, announcement.ID.String(), announcement.Title, announcement.Content, int32(announcement.Type)); err != nil {
		e.log.Error(ctx, fmt.Sprintf("推送公告失败 (ID: %s): %v", announcement.ID.String(), err))
	}

	// 推进索引（循环）
	e.advanceIndex(len(items))
}

// advanceIndex 推进当前索引（循环）
func (e *SchedulerEngine) advanceIndex(total int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.currentIndex = (e.currentIndex + 1) % total
}

// stopLocked 在已持有写锁的情况下停止引擎（内部使用）
func (e *SchedulerEngine) stopLocked() {
	if e.cancelFunc != nil {
		e.cancelFunc()
		e.cancelFunc = nil
	}
	if e.ticker != nil {
		e.ticker.Stop()
		e.ticker = nil
	}
	e.running = false
	e.currentScheduleID = 0
	e.currentIndex = 0
}
