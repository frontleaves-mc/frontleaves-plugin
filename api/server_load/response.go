package apiServerLoad

// LoadSample 单个采样点
type LoadSample struct {
	TPS         float64 `json:"tps"`
	CPUUsagePct float64 `json:"cpu_usage_pct"`
	CPUCores    int     `json:"cpu_cores"`
	MemTotal    int64   `json:"mem_total_bytes"`
	MemUsed     int64   `json:"mem_used_bytes"`
	MemFree     int64   `json:"mem_free_bytes"`
	JVMMemMax   int64   `json:"jvm_max_bytes"`
	JVMMemUsed  int64   `json:"jvm_used_bytes"`
	CollectedAt int64   `json:"collected_at"`
}

// ServerRealtimeLoadResponse 单服务器实时负载
type ServerRealtimeLoadResponse struct {
	ServerID      int64       `json:"server_id"`
	ServerName    string      `json:"server_name"`
	DisplayName   string      `json:"display_name"`
	Online        bool        `json:"online"`
	TPS           float64     `json:"tps"`
	CPUInfo       *CPUInfo    `json:"cpu_info,omitempty"`
	MemoryInfo    *MemoryInfo `json:"memory_info,omitempty"`
	JVMInfo       *JVMInfo    `json:"jvm_info,omitempty"`
	LastHeartbeat int64       `json:"last_heartbeat"`
}

// CPUInfo CPU 信息
type CPUInfo struct {
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usage_percent"`
}

// MemoryInfo 内存信息
type MemoryInfo struct {
	TotalBytes int64 `json:"total_bytes"`
	UsedBytes  int64 `json:"used_bytes"`
	FreeBytes  int64 `json:"free_bytes"`
}

// JVMInfo JVM 内存信息
type JVMInfo struct {
	MaxMemoryBytes  int64 `json:"max_memory_bytes"`
	UsedMemoryBytes int64 `json:"used_memory_bytes"`
}

// ServerLoadHistoryResponse 历史趋势响应
type ServerLoadHistoryResponse struct {
	ServerID    int64               `json:"server_id"`
	ServerName  string              `json:"server_name"`
	DisplayName string              `json:"display_name"`
	Records     []LoadHistoryRecord  `json:"records"`
}

// LoadHistoryRecord 单分钟历史记录
type LoadHistoryRecord struct {
	MinuteTime  string       `json:"minute_time"`
	Samples     []LoadSample `json:"samples"`
	TpsAvg      float64      `json:"tps_avg"`
	CpuUsageAvg float64      `json:"cpu_usage_avg"`
	MemUsedAvg  int64        `json:"mem_used_avg"`
	JvmUsedAvg  int64        `json:"jvm_used_avg"`
}
