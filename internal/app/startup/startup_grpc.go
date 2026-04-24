package startup

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	pluginGrpc "github.com/frontleaves-mc/frontleaves-plugin/internal/app/grpc"
)

func (r *reg) grpcAuthClientInit(ctx context.Context) (any, error) {
	log := xLog.WithName(xLog.NamedINIT)
	log.Debug(ctx, "正在初始化 gRPC 认证客户端...")

	client, err := pluginGrpc.NewAuthClient(ctx)
	if err != nil {
		return nil, err
	}

	log.Info(ctx, "gRPC 认证客户端初始化成功")
	return client, nil
}
