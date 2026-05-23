package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiServerLoad "github.com/frontleaves-mc/frontleaves-plugin/api/server_load"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type serverLoadRepo struct {
	loadLog *repository.ServerLoadLogRepo
	server  *repository.ServerRepo
}

// ServerLoadLogic 服务器负载业务逻辑
type ServerLoadLogic struct {
	logic
	repo serverLoadRepo
}

// NewServerLoadLogic 创建服务器负载 Logic
func NewServerLoadLogic(ctx context.Context) *ServerLoadLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &ServerLoadLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "ServerLoadLogic"),
		},
		repo: serverLoadRepo{
			loadLog: repository.NewServerLoadLogRepo(db),
			server:  repository.NewServerRepo(db),
		},
	}
}

// GetRealtimeAll 获取所有启用服务器的实时负载数据
func (l *ServerLoadLogic) GetRealtimeAll(ctx context.Context) ([]apiServerLoad.ServerRealtimeLoadResponse, *xError.Error) {
	l.log.Info(ctx, "GetRealtimeAll - 查询所有服务器实时负载")

	var servers []entity.Server
	if err := l.db.WithContext(ctx).Where("is_enabled = ?", true).Find(&servers).Error; err != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询启用服务器列表失败", false, err)
	}

	list := make([]apiServerLoad.ServerRealtimeLoadResponse, 0, len(servers))
	for _, srv := range servers {
		resp := l.parseRedisLoadData(ctx, srv.ID, srv.Name, srv.DisplayName)
		if resp != nil {
			list = append(list, *resp)
		}
	}

	return list, nil
}

// GetRealtimeByServerID 按 ServerID 获取单台服务器实时负载
func (l *ServerLoadLogic) GetRealtimeByServerID(ctx context.Context, serverID xSnowflake.SnowflakeID) (*apiServerLoad.ServerRealtimeLoadResponse, *xError.Error) {
	l.log.Info(ctx, "GetRealtimeByServerID - 查询服务器实时负载: "+serverID.String())

	server, xErr := l.repo.server.GetByID(ctx, serverID)
	if xErr != nil {
		return nil, xError.NewError(ctx, xError.ResourceNotFound, "服务器不存在", true, xErr)
	}

	resp := l.parseRedisLoadData(ctx, server.ID, server.Name, server.DisplayName)
	return resp, nil
}

// GetHistoryByServerID 查询服务器历史负载趋势
func (l *ServerLoadLogic) GetHistoryByServerID(ctx context.Context, serverID xSnowflake.SnowflakeID, start, end time.Time, page, pageSize int) (*apiServerLoad.ServerLoadHistoryResponse, *xError.Error) {
	l.log.Info(ctx, "GetHistoryByServerID - 查询服务器历史负载: "+serverID.String())

	server, xErr := l.repo.server.GetByID(ctx, serverID)
	if xErr != nil {
		return nil, xError.NewError(ctx, xError.ResourceNotFound, "服务器不存在", true, xErr)
	}

	logs, _, xErr := l.repo.loadLog.QueryByServerIDAndTimeRange(ctx, serverID, start, end, page, pageSize)
	if xErr != nil {
		return nil, xErr
	}

	records := make([]apiServerLoad.LoadHistoryRecord, 0, len(logs))
	for _, log := range logs {
		var samples []apiServerLoad.LoadSample
		if err := json.Unmarshal(log.Samples, &samples); err != nil {
			l.log.Warn(ctx, "解析采样数据失败，跳过: "+serverID.String())
			continue
		}
		records = append(records, apiServerLoad.LoadHistoryRecord{
			MinuteTime:  log.MinuteTime.Format(time.RFC3339),
			Samples:     samples,
			TpsAvg:      log.TpsAvg,
			CpuUsageAvg: log.CpuUsageAvg,
			MemUsedAvg:  log.MemUsedAvg,
			JvmUsedAvg:  log.JvmUsedAvg,
		})
	}

	return &apiServerLoad.ServerLoadHistoryResponse{
		ServerID:    serverID.Int64(),
		ServerName:  server.Name,
		DisplayName: server.DisplayName,
		Records:     records,
	}, nil
}

// parseRedisLoadData 从 Redis 解析单台服务器实时负载
func (l *ServerLoadLogic) parseRedisLoadData(ctx context.Context, serverID xSnowflake.SnowflakeID, serverName, displayName string) *apiServerLoad.ServerRealtimeLoadResponse {
	resp := &apiServerLoad.ServerRealtimeLoadResponse{
		ServerID:    serverID.Int64(),
		ServerName:  serverName,
		DisplayName: displayName,
		Online:      false,
	}

	serverKey := string(bConst.CacheStatusServer.Get(serverName))
	data, err := l.rdb.HGetAll(ctx, serverKey).Result()
	if err != nil || len(data) == 0 {
		// 离线状态，返回零值
		return resp
	}

	// TPS
	if tps, parseErr := strconv.ParseFloat(data["tps"], 64); parseErr == nil {
		resp.TPS = tps
	}

	// Timestamp & Online 判定
	if ts, parseErr := strconv.ParseInt(data["timestamp"], 10, 64); parseErr == nil {
		resp.LastHeartbeat = ts
		resp.Online = time.Now().UnixMilli()-ts < heartbeatTimeout.Milliseconds()
	}

	// CPU Info
	if cpuRaw, ok := data["cpu_info"]; ok && cpuRaw != "" {
		var cpu struct {
			Cores        int     `json:"cores"`
			UsagePercent float64 `json:"usage_percent"`
		}
		if err := json.Unmarshal([]byte(cpuRaw), &cpu); err == nil {
			resp.CPUInfo = &apiServerLoad.CPUInfo{
				Cores:        cpu.Cores,
				UsagePercent: cpu.UsagePercent,
			}
		}
	}

	// Memory Info
	if memRaw, ok := data["memory_info"]; ok && memRaw != "" {
		var mem struct {
			TotalBytes int64 `json:"total_bytes"`
			UsedBytes  int64 `json:"used_bytes"`
			FreeBytes  int64 `json:"free_bytes"`
		}
		if err := json.Unmarshal([]byte(memRaw), &mem); err == nil {
			resp.MemoryInfo = &apiServerLoad.MemoryInfo{
				TotalBytes: mem.TotalBytes,
				UsedBytes:  mem.UsedBytes,
				FreeBytes:  mem.FreeBytes,
			}
		}
	}

	// JVM Info
	if jvmRaw, ok := data["jvm_info"]; ok && jvmRaw != "" {
		var jvm struct {
			MaxMemoryBytes  int64 `json:"max_memory_bytes"`
			UsedMemoryBytes int64 `json:"used_memory_bytes"`
		}
		if err := json.Unmarshal([]byte(jvmRaw), &jvm); err == nil {
			resp.JVMInfo = &apiServerLoad.JVMInfo{
				MaxMemoryBytes:  jvm.MaxMemoryBytes,
				UsedMemoryBytes: jvm.UsedMemoryBytes,
			}
		}
	}

	return resp
}
