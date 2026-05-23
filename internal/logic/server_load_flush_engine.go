package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiServerLoad "github.com/frontleaves-mc/frontleaves-plugin/api/server_load"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ServerLoadFlushEngine 服务器负载刷盘引擎，每分钟从 Redis 读取服务器负载数据并持久化到 PostgreSQL
type ServerLoadFlushEngine struct {
	db         *gorm.DB
	rdb        *redis.Client
	repo       *repository.ServerLoadLogRepo
	serverRepo *repository.ServerRepo
	cancel     context.CancelFunc
	log        *xLog.LogNamedLogger
	mu         sync.RWMutex
	running    bool
}

// NewServerLoadFlushEngine 创建负载刷盘引擎实例
func NewServerLoadFlushEngine(ctx context.Context) *ServerLoadFlushEngine {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &ServerLoadFlushEngine{
		db:         db,
		rdb:        rdb,
		repo:       repository.NewServerLoadLogRepo(db),
		serverRepo: repository.NewServerRepo(db),
		log:        xLog.WithName(xLog.NamedLOGC, "ServerLoadFlushEngine"),
	}
}

// Start 启动后台刷盘协程
func (e *ServerLoadFlushEngine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return
	}

	childCtx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.running = true

	e.log.Info(childCtx, "ServerLoadFlushEngine 已启动")

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				e.flush(childCtx)
			}
		}
	}()
}

// Stop 停止刷盘协程
func (e *ServerLoadFlushEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.running = false
	e.log.Info(context.Background(), "ServerLoadFlushEngine 已停止")
}

// flush 执行一次刷盘逻辑：读取 Redis 数据 → 构建采样 → 持久化到 DB → 清理过期数据
func (e *ServerLoadFlushEngine) flush(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			e.log.Error(ctx, fmt.Sprintf("ServerLoadFlushEngine panic recovered: %v", r))
		}
	}()

	// 查询所有启用的服务器
	var servers []entity.Server
	if err := e.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&servers).Error; err != nil {
		e.log.Error(ctx, "flush - 查询服务器列表失败: "+err.Error())
		return
	}

	for _, srv := range servers {
		e.flushServer(ctx, srv)
	}

	// 清理 30 天前的数据
	if xErr := e.repo.DeleteBeforeTime(ctx, time.Now().AddDate(0, 0, -30)); xErr != nil {
		e.log.Error(ctx, "flush - 清理过期负载日志失败: "+xErr.Error())
	}
}

// flushServer 处理单个服务器的负载刷盘
func (e *ServerLoadFlushEngine) flushServer(ctx context.Context, srv entity.Server) {
	serverKey := string(bConst.CacheStatusServer.Get(srv.Name))
	data, err := e.rdb.HGetAll(ctx, serverKey).Result()
	if err != nil {
		e.log.Warn(ctx, "flushServer - 读取 Redis 失败，跳过: "+srv.Name)
		return
	}
	if len(data) == 0 {
		// 服务器离线，无数据
		return
	}

	// 解析 TPS
	tps, _ := strconv.ParseFloat(data["tps"], 64)

	// 解析 CPU 信息
	var cpuCores int
	var cpuUsage float64
	if cpuJSON, ok := data["cpu_info"]; ok && cpuJSON != "" {
		var cpuInfo struct {
			Cores        int     `json:"cores"`
			UsagePercent float64 `json:"usage_percent"`
		}
		if err := json.Unmarshal([]byte(cpuJSON), &cpuInfo); err == nil {
			cpuCores = cpuInfo.Cores
			cpuUsage = cpuInfo.UsagePercent
		}
	}

	// 解析内存信息
	var memTotal, memUsed, memFree int64
	if memJSON, ok := data["memory_info"]; ok && memJSON != "" {
		var memInfo struct {
			TotalBytes int64 `json:"total_bytes"`
			UsedBytes  int64 `json:"used_bytes"`
			FreeBytes  int64 `json:"free_bytes"`
		}
		if err := json.Unmarshal([]byte(memJSON), &memInfo); err == nil {
			memTotal = memInfo.TotalBytes
			memUsed = memInfo.UsedBytes
			memFree = memInfo.FreeBytes
		}
	}

	// 解析 JVM 信息
	var jvmMax, jvmUsed int64
	if jvmJSON, ok := data["jvm_info"]; ok && jvmJSON != "" {
		var jvmInfo struct {
			MaxMemoryBytes  int64 `json:"max_memory_bytes"`
			UsedMemoryBytes int64 `json:"used_memory_bytes"`
		}
		if err := json.Unmarshal([]byte(jvmJSON), &jvmInfo); err == nil {
			jvmMax = jvmInfo.MaxMemoryBytes
			jvmUsed = jvmInfo.UsedMemoryBytes
		}
	}

	// 解析时间戳
	var collectedAt int64
	if ts, err := strconv.ParseInt(data["timestamp"], 10, 64); err == nil {
		collectedAt = ts
	}

	// 截断到当前分钟（Asia/Shanghai 时区）
	loc, _ := time.LoadLocation("Asia/Shanghai")
	minuteTime := time.Now().In(loc).Truncate(time.Minute)

	// 构建采样数据
	sample := apiServerLoad.LoadSample{
		TPS:         tps,
		CPUUsagePct: cpuUsage,
		CPUCores:    cpuCores,
		MemTotal:    memTotal,
		MemUsed:     memUsed,
		MemFree:     memFree,
		JVMMemMax:   jvmMax,
		JVMMemUsed:  jvmUsed,
		CollectedAt: collectedAt,
	}

	samplesJSON, err := json.Marshal([]apiServerLoad.LoadSample{sample})
	if err != nil {
		e.log.Error(ctx, "flushServer - 序列化采样数据失败: "+err.Error())
		return
	}

	// 持久化到 DB
	if _, xErr := e.repo.Upsert(ctx, srv.ID, minuteTime, samplesJSON, tps, cpuUsage, memTotal, memUsed, jvmUsed); xErr != nil {
		e.log.Error(ctx, "flushServer - 刷盘失败 ["+srv.Name+"]: "+xErr.Error())
	}
}
