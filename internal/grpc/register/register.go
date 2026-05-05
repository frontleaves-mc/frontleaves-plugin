package register

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"google.golang.org/grpc"
)

var log = xLog.WithName(xLog.NamedGRPC, "Register")

// RegisterGRPCServices 注册所有 gRPC 服务
//
// 每个服务在 Handler 构造函数中绑定各自的服务级中间件。
func RegisterGRPCServices(ctx context.Context, server grpc.ServiceRegistrar) {
	handler.NewServerStatusHandler(ctx, server)
	handler.NewTitleHandler(ctx, server)
	announcementHandler := handler.NewAnnouncementHandler(ctx, server)

	// 创建调度引擎并注册为全局单例
	engine := logic.NewSchedulerEngine(
		logic.NewAnnouncementScheduleLogic(ctx),
		logic.NewAnnouncementLogic(ctx),
		announcementHandler.PushAnnouncement,
	)
	logic.SetGlobalEngine(engine)

	// 从数据库恢复活动调度（启动时自动恢复）
	if xErr := engine.RecoverFromDatabase(ctx); xErr != nil {
		log.Error(ctx, "RegisterGRPCServices - 调度引擎恢复失败: "+xErr.Error())
	}
}
