package main

import (
	"context"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xMain "github.com/bamboo-services/bamboo-base-go/major/main"
	xReg "github.com/bamboo-services/bamboo-base-go/major/register"
	xGrpcIStream "github.com/bamboo-services/bamboo-base-go/plugins/grpc/interceptor/stream"
	xGrpcIUnary "github.com/bamboo-services/bamboo-base-go/plugins/grpc/interceptor/unary"
	xGrpcRunner "github.com/bamboo-services/bamboo-base-go/plugins/grpc/runner"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/route"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/startup"
	grpcRegister "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/register"
	"google.golang.org/grpc"
)

func main() {
	reg := xReg.Register(startup.Init())
	log := xLog.WithName(xLog.NamedMAIN)

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
	)

	xMain.Runner(reg, log, route.NewRoute, grpcTask)
	return
}
