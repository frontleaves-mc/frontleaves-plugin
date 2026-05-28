package bConst

// SchedulerConfigNamespace 公告调度配置命名空间
const SchedulerConfigNamespace = "scheduler"

// Scheduler 配置键常量
const (
	SchedulerConfigMode            = "mode"             // 调度模式: 1=FixedInterval, 2=Sequential
	SchedulerConfigIntervalSeconds = "interval_seconds" // 固定间隔秒数
	SchedulerConfigIsEnabled       = "is_enabled"       // 是否启用调度: true/false
)
