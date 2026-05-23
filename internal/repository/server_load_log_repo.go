package repository

import (
	"context"
	"encoding/json"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type ServerLoadLogRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewServerLoadLogRepo(db *gorm.DB) *ServerLoadLogRepo {
	return &ServerLoadLogRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "ServerLoadLogRepo"),
	}
}

func (r *ServerLoadLogRepo) Upsert(ctx context.Context, serverID xSnowflake.SnowflakeID, minuteTime time.Time, samples json.RawMessage, tpsAvg, cpuUsageAvg float64, memTotalAvg, memUsedAvg, jvmUsedAvg int64) (*entity.ServerLoadLog, *xError.Error) {
	r.log.Info(ctx, "Upsert - 更新或创建服务器负载日志")
	log := &entity.ServerLoadLog{
		ServerID:    serverID,
		MinuteTime:  minuteTime,
		Samples:     samples,
		TpsAvg:      tpsAvg,
		CpuUsageAvg: cpuUsageAvg,
		MemTotalAvg: memTotalAvg,
		MemUsedAvg:  memUsedAvg,
		JvmUsedAvg:  jvmUsedAvg,
	}
	result := r.db.WithContext(ctx).
		Where("server_id = ? AND minute_time = ?", serverID, minuteTime).
		Assign(map[string]interface{}{
			"samples":       samples,
			"tps_avg":       tpsAvg,
			"cpu_usage_avg": cpuUsageAvg,
			"mem_total_avg": memTotalAvg,
			"mem_used_avg":  memUsedAvg,
			"jvm_used_avg":  jvmUsedAvg,
		}).
		FirstOrCreate(log)
	if result.Error != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "更新或创建服务器负载日志失败", false, result.Error)
	}
	return log, nil
}

func (r *ServerLoadLogRepo) QueryByServerIDAndTimeRange(ctx context.Context, serverID xSnowflake.SnowflakeID, start, end time.Time, page, pageSize int) ([]entity.ServerLoadLog, int64, *xError.Error) {
	r.log.Info(ctx, "QueryByServerIDAndTimeRange - 按服务器ID和时间范围查询负载日志")
	var total int64
	query := r.db.WithContext(ctx).Model(&entity.ServerLoadLog{}).
		Where("server_id = ? AND minute_time BETWEEN ? AND ?", serverID, start, end)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(ctx, xError.DatabaseError, "查询负载日志总数失败", false, err)
	}
	var logs []entity.ServerLoadLog
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("minute_time ASC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(ctx, xError.DatabaseError, "查询负载日志列表失败", false, err)
	}
	return logs, total, nil
}

func (r *ServerLoadLogRepo) DeleteBeforeTime(ctx context.Context, before time.Time) *xError.Error {
	r.log.Info(ctx, "DeleteBeforeTime - 删除指定时间之前的负载日志")
	if err := r.db.WithContext(ctx).
		Where("minute_time < ?", before).
		Delete(&entity.ServerLoadLog{}).Error; err != nil {
		return xError.NewError(ctx, xError.DatabaseError, "删除过期负载日志失败", false, err)
	}
	return nil
}

func (r *ServerLoadLogRepo) GetLatestByServerID(ctx context.Context, serverID xSnowflake.SnowflakeID) (*entity.ServerLoadLog, *xError.Error) {
	r.log.Info(ctx, "GetLatestByServerID - 查询服务器最新负载日志")
	var log entity.ServerLoadLog
	if err := r.db.WithContext(ctx).
		Where("server_id = ?", serverID).
		Order("minute_time DESC").
		Limit(1).
		First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询服务器最新负载日志失败", false, err)
	}
	return &log, nil
}
