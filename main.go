package main

import (
	"context"
	"time"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xMain "github.com/bamboo-services/bamboo-base-go/major/main"
	xReg "github.com/bamboo-services/bamboo-base-go/major/register"
	xGrpcIStream "github.com/bamboo-services/bamboo-base-go/plugins/grpc/interceptor/stream"
	xGrpcIUnary "github.com/bamboo-services/bamboo-base-go/plugins/grpc/interceptor/unary"
	xGrpcRunner "github.com/bamboo-services/bamboo-base-go/plugins/grpc/runner"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/route"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/startup"
	grpcRegister "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/register"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func main() {
	reg := xReg.Register(startup.Init())
	log := xLog.WithName(xLog.NamedMAIN)

	grpcKeepaliveTime := time.Duration(xEnv.GetEnvDuration(bConst.EnvGrpcKeepaliveTime, int64(30*time.Second)))
	grpcKeepaliveTimeout := time.Duration(xEnv.GetEnvDuration(bConst.EnvGrpcKeepaliveTimeout, int64(10*time.Second)))
	grpcKeepaliveMinTime := time.Duration(xEnv.GetEnvDuration(bConst.EnvGrpcKeepaliveMinTime, int64(5*time.Second)))
	grpcKeepaliveMaxIdle := time.Duration(xEnv.GetEnvDuration(bConst.EnvGrpcKeepaliveMaxIdle, int64(5*time.Minute)))

	grpcTask := xGrpcRunner.New(
		xGrpcRunner.WithLogger(xLog.WithName(xLog.NamedGRPC)),
		xGrpcRunner.WithGracefulStopTimeout(30*time.Second),
		xGrpcRunner.WithRegisterService(func(ctx context.Context, server grpc.ServiceRegistrar) {
			grpcRegister.RegisterGRPCServices(ctx, server)
		}),
		xGrpcRunner.WithUnaryInterceptors(
			xGrpcIUnary.ResponseBuilder(),
		),
		xGrpcRunner.WithStreamInterceptors(
			xGrpcIStream.Middleware(),
		),
		xGrpcRunner.WithServerOptions(
			grpc.KeepaliveParams(keepalive.ServerParameters{
				Time:              grpcKeepaliveTime,
				Timeout:           grpcKeepaliveTimeout,
				MaxConnectionIdle: grpcKeepaliveMaxIdle,
			}),
			grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             grpcKeepaliveMinTime,
				PermitWithoutStream: true,
			}),
		),
	)

	xMain.Runner(reg, log, route.NewRoute, grpcTask)
	return
}
