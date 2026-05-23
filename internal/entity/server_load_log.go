package entity

import (
	"encoding/json"
	"time"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

// ServerLoadLog 服务器负载采样日志（按分钟聚合）
type ServerLoadLog struct {
	xModels.BaseEntity
	ServerID    xSnowflake.SnowflakeID `gorm:"not null;index:idx_server_minute,unique:uk_server_minute;comment:服务器ID" json:"server_id"` // 服务器ID
	Server      *Server                `gorm:"foreignKey:ServerID;references:ID" json:"server,omitempty"`                                  // 关联服务器
	MinuteTime  time.Time              `gorm:"not null;type:timestamptz;index:idx_server_minute,unique:uk_server_minute;comment:分钟时间戳" json:"minute_time"` // 分钟时间戳
	Samples     json.RawMessage        `gorm:"type:jsonb;not null;default:'[]';comment:采样数据JSON数组" json:"samples"`                    // 采样数据JSON数组
	TpsAvg      float64                `gorm:"type:double precision;comment:TPS均值" json:"tps_avg"`                                      // TPS均值
	CpuUsageAvg float64                `gorm:"type:double precision;comment:CPU使用率均值" json:"cpu_usage_avg"`                            // CPU使用率均值
	MemTotalAvg int64                  `gorm:"comment:内存总量均值(字节)" json:"mem_total_avg"`                                              // 内存总量均值(字节)
	MemUsedAvg  int64                  `gorm:"comment:内存已用均值(字节)" json:"mem_used_avg"`                                               // 内存已用均值(字节)
	JvmUsedAvg  int64                  `gorm:"comment:JVM已用内存均值(字节)" json:"jvm_used_avg"`                                             // JVM已用内存均值(字节)
}

func (_ *ServerLoadLog) GetGene() xSnowflake.Gene {
	return bConst.GeneServerLoadLog
}
