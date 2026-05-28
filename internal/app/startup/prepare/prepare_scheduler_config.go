package prepare

import (
	"context"
	"strconv"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

// PrepareSchedulerConfig 检查并初始化公告调度配置
// 服务启动时执行，确保所有必需的调度配置键存在并包含有效值
func PrepareSchedulerConfig(ctx context.Context) {
	log := xLog.WithName(xLog.NamedINIT, "PrepareSchedulerConfig")
	repo := repository.NewConfigRepository()

	// 定义必需配置及其默认值
	defaults := map[string]string{
		bConst.SchedulerConfigMode:            strconv.Itoa(int(1)), // FixedInterval
		bConst.SchedulerConfigIntervalSeconds: "60",
		bConst.SchedulerConfigIsEnabled:       "false",
	}

	// 批量检查，缺失则插入默认值
	for key, defaultValue := range defaults {
		value, xErr := repo.GetByKey(ctx, bConst.SchedulerConfigNamespace, key)
		if xErr != nil {
			log.Error(ctx, "查询配置失败 ["+key+"]: "+xErr.Error())
			continue
		}
		if value == "" {
			log.Info(ctx, "配置键缺失，插入默认值 ["+bConst.SchedulerConfigNamespace+"/"+key+"] = "+defaultValue)
			if xErr := repo.Set(ctx, bConst.SchedulerConfigNamespace, key, defaultValue); xErr != nil {
				log.Error(ctx, "插入默认配置失败 ["+key+"]: "+xErr.Error())
			}
		}
	}
}
