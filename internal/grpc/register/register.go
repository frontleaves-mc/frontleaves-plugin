package register

import (
	"context"

	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/handler"
	"google.golang.org/grpc"
)

// RegisterGRPCServices 注册所有 gRPC 服务
//
// 每个服务在 Handler 构造函数中绑定各自的服务级中间件。
func RegisterGRPCServices(ctx context.Context, server grpc.ServiceRegistrar) {
	handler.NewServerStatusHandler(ctx, server)
	handler.NewTitleHandler(ctx, server)
}
