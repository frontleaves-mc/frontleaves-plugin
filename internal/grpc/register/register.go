package register

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix"
	"google.golang.org/grpc"
)

var log = xLog.WithName(xLog.NamedGRPC, "Register")

// RegisterGRPCServices 注册所有 gRPC 服务
//
// 每个服务在 Handler 构造函数中绑定各自的服务级中间件。
func RegisterGRPCServices(ctx context.Context, server grpc.ServiceRegistrar) {
	handler.NewServerStatusHandler(ctx, server)
	handler.NewTitleHandler(ctx, server)
	handler.NewEssentialsPlayerEventHandler(ctx, server)
	queryHandler := handler.NewEssentialsPlayerQueryHandler(ctx, server)
	handler.NewMatrixTelemetryHandler(ctx, server)

	// Essentials 消息推送服务
	messageHandler := handler.NewEssentialsPlayerMessageHandler(ctx, server)
	logic.SetGlobalPushChatFunc(messageHandler.PushChatMessage)

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

	// 创建负载刷盘引擎并启动
	flushEngine := logic.NewServerLoadFlushEngine(ctx)
	flushEngine.Start()

	// 启动 Session 恢复（异步，不阻塞服务启动）
	recoveryAdapter := handler.NewServerQuerierAdapter(queryHandler)
	go matrix.RecoverSessions(ctx, recoveryAdapter)
}
