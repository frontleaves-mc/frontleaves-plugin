package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

	// 启动时执行一次性清理：删除所有服务器状态 hash 中的过期累计字段
	go func() {
		e.log.Info(childCtx, "开始清理过期累计字段...")
		var cursor uint64
		for {
			keys, nextCursor, err := e.rdb.Scan(childCtx, cursor, string(bConst.CacheStatusServer.Get("*")), 100).Result()
			if err != nil {
				e.log.Warn(childCtx, "cleanupStaleCumulativeFields - SCAN 失败: "+err.Error())
				break
			}
			for _, key := range keys {
				fields, err := e.rdb.HGetAll(childCtx, key).Result()
				if err != nil {
					e.log.Warn(childCtx, "cleanupStaleCumulativeFields - HGetAll 失败 ["+key+"]: "+err.Error())
					continue
				}
				var fieldsToDelete []string
				for field := range fields {
					if strings.HasSuffix(field, "_sum") || strings.HasSuffix(field, "_count") {
						fieldsToDelete = append(fieldsToDelete, field)
					}
				}
				if len(fieldsToDelete) > 0 {
					if err := e.rdb.HDel(childCtx, key, fieldsToDelete...).Err(); err != nil {
						e.log.Warn(childCtx, "cleanupStaleCumulativeFields - HDel 失败 ["+key+"]: "+err.Error())
					}
				}
			}
			if nextCursor == 0 {
				break
			}
			cursor = nextCursor
		}
		e.log.Info(childCtx, "过期累计字段清理完成")
	}()

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

// calculateAverage 计算平均值
func calculateAverage(sum float64, count int64) float64 {
	if count <= 0 {
		return 0
	}
	return sum / float64(count)
}

// flushServer 处理单个服务器的负载刷盘
func (e *ServerLoadFlushEngine) flushServer(ctx context.Context, srv entity.Server) {
	serverKey := string(bConst.CacheStatusServer.Get(srv.Name))

	// 读取瞬时字段（常量字段和元数据）
	data, err := e.rdb.HGetAll(ctx, serverKey).Result()
	if err != nil {
		e.log.Warn(ctx, "flushServer - 读取 Redis 失败，跳过: "+srv.Name)
		return
	}
	if len(data) == 0 {
		// 服务器离线，无数据
		return
	}

	// 使用 Watch + TxPipeline 原子性读取累计字段并删除
	var tpsSum, cpuUsageSum, memUsedSum, jvmUsedSum float64
	var tpsCount, cpuUsageCount, memUsedCount, jvmUsedCount int64

	err = e.rdb.Watch(ctx, func(tx *redis.Tx) error {
		pipe := tx.TxPipeline()

		// 读取累计字段
		tpsSumCmd := pipe.HGet(ctx, serverKey, "tps_sum")
		tpsCountCmd := pipe.HGet(ctx, serverKey, "tps_count")
		cpuSumCmd := pipe.HGet(ctx, serverKey, "cpu_usage_sum")
		cpuCountCmd := pipe.HGet(ctx, serverKey, "cpu_usage_count")
		memSumCmd := pipe.HGet(ctx, serverKey, "mem_used_sum")
		memCountCmd := pipe.HGet(ctx, serverKey, "mem_used_count")
		jvmSumCmd := pipe.HGet(ctx, serverKey, "jvm_used_sum")
		jvmCountCmd := pipe.HGet(ctx, serverKey, "jvm_used_count")

		// 删除累计字段
		pipe.HDel(ctx, serverKey,
			"tps_sum", "tps_count",
			"cpu_usage_sum", "cpu_usage_count",
			"mem_used_sum", "mem_used_count",
			"jvm_used_sum", "jvm_used_count",
		)

		_, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}

		// 解析结果
		if val := tpsSumCmd.Val(); val != "" {
			tpsSum, _ = strconv.ParseFloat(val, 64)
		}
		if val := tpsCountCmd.Val(); val != "" {
			tpsCount, _ = strconv.ParseInt(val, 10, 64)
		}
		if val := cpuSumCmd.Val(); val != "" {
			cpuUsageSum, _ = strconv.ParseFloat(val, 64)
		}
		if val := cpuCountCmd.Val(); val != "" {
			cpuUsageCount, _ = strconv.ParseInt(val, 10, 64)
		}
		if val := memSumCmd.Val(); val != "" {
			memUsedSum, _ = strconv.ParseFloat(val, 64)
		}
		if val := memCountCmd.Val(); val != "" {
			memUsedCount, _ = strconv.ParseInt(val, 10, 64)
		}
		if val := jvmSumCmd.Val(); val != "" {
			jvmUsedSum, _ = strconv.ParseFloat(val, 64)
		}
		if val := jvmCountCmd.Val(); val != "" {
			jvmUsedCount, _ = strconv.ParseInt(val, 10, 64)
		}

		return nil
	}, serverKey)

	if err != nil {
		e.log.Warn(ctx, "flushServer - Watch 事务失败 ["+srv.Name+"]: "+err.Error())
		return
	}

	// 解析 CPU 信息（常量字段）
	var cpuCores int
	if cpuJSON, ok := data["cpu_info"]; ok && cpuJSON != "" {
		var cpuInfo struct {
			Cores        int     `json:"cores"`
			UsagePercent float64 `json:"usage_percent"`
		}
		if err := json.Unmarshal([]byte(cpuJSON), &cpuInfo); err == nil {
			cpuCores = cpuInfo.Cores
		}
	}

	// 解析内存信息（常量字段：TotalBytes）
	var memTotal int64
	if memJSON, ok := data["memory_info"]; ok && memJSON != "" {
		var memInfo struct {
			TotalBytes int64 `json:"total_bytes"`
			UsedBytes  int64 `json:"used_bytes"`
			FreeBytes  int64 `json:"free_bytes"`
		}
		if err := json.Unmarshal([]byte(memJSON), &memInfo); err == nil {
			memTotal = memInfo.TotalBytes
		}
	}

	// 解析 JVM 信息（常量字段：MaxMemoryBytes）
	var jvmMax int64
	if jvmJSON, ok := data["jvm_info"]; ok && jvmJSON != "" {
		var jvmInfo struct {
			MaxMemoryBytes  int64 `json:"max_memory_bytes"`
			UsedMemoryBytes int64 `json:"used_memory_bytes"`
		}
		if err := json.Unmarshal([]byte(jvmJSON), &jvmInfo); err == nil {
			jvmMax = jvmInfo.MaxMemoryBytes
		}
	}

	// 截断到当前分钟（Asia/Shanghai 时区）
	loc, _ := time.LoadLocation("Asia/Shanghai")
	minuteTime := time.Now().In(loc).Truncate(time.Minute)

	// 计算均值
	tpsAvg := calculateAverage(tpsSum, tpsCount)
	cpuUsageAvg := calculateAverage(cpuUsageSum, cpuUsageCount)
	memUsedAvg := calculateAverage(memUsedSum, memUsedCount)
	jvmUsedAvg := calculateAverage(jvmUsedSum, jvmUsedCount)

	// 构建采样数据
	var samplesJSON []byte
	if tpsCount == 0 {
		// 零心跳分钟，保存空数组
		samplesJSON = []byte("[]")
	} else {
		sample := apiServerLoad.LoadSample{
			TPS:         tpsAvg,
			CPUUsagePct: cpuUsageAvg,
			CPUCores:    cpuCores,
			MemTotal:    memTotal,
			MemUsed:     int64(memUsedAvg),
			MemFree:     memTotal - int64(memUsedAvg),
			JVMMemMax:   jvmMax,
			JVMMemUsed:  int64(jvmUsedAvg),
			CollectedAt: minuteTime.UnixMilli(),
		}

		var err error
		samplesJSON, err = json.Marshal([]apiServerLoad.LoadSample{sample})
		if err != nil {
			e.log.Error(ctx, "flushServer - 序列化采样数据失败: "+err.Error())
			return
		}
	}

	// 持久化到 DB
	if _, xErr := e.repo.Upsert(ctx, srv.ID, minuteTime, samplesJSON, tpsAvg, cpuUsageAvg, memTotal, int64(memUsedAvg), int64(jvmUsedAvg)); xErr != nil {
		e.log.Error(ctx, "flushServer - 刷盘失败 ["+srv.Name+"]: "+xErr.Error())
	}
}
