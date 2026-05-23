package middleware

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/utility"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryPluginVerify 创建插件认证中间件
//
// 从 gRPC metadata 中提取 plugin-secret-key，调用 PluginCredentialLogic 进行验证。
// plugin-name 仅用于日志标识，不参与认证判断。
func UnaryPluginVerify(mainCtx context.Context) grpc.UnaryServerInterceptor {
	log := xLog.WithName(xLog.NamedMIDE, "UnaryPluginVerify")

	return func(
		ctx context.Context, req any,
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		pluginName, _ := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
		logCtx := "未知插件"
		if pluginName != "" {
			logCtx = pluginName
		}
		log.Info(ctx, "验证插件身份: "+logCtx)

		secretKey, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginSecretKey)
		if xErr != nil {
			return nil, xError.NewError(ctx, xError.Unauthorized, "缺少 plugin-secret-key", true)
		}

		pluginCredLogic := logic.NewPluginCredentialLogic(mainCtx)
		_, xErr = pluginCredLogic.Authenticate(ctx, secretKey)
		if xErr != nil {
			log.Warn(ctx, "插件认证失败: "+logCtx)
			return nil, xErr
		}

		log.Info(ctx, "插件认证通过: "+logCtx)
		return handler(ctx, req)
	}
}

// StreamPluginVerify 创建流式插件认证中间件
//
// 从 gRPC stream metadata 中提取 plugin-secret-key，调用 PluginCredentialLogic 进行验证。
// plugin-name 仅用于日志标识，不参与认证判断。
func StreamPluginVerify(mainCtx context.Context) grpc.StreamServerInterceptor {
	log := xLog.WithName(xLog.NamedMIDE, "StreamPluginVerify")

	return func(
		srv interface{}, ss grpc.ServerStream,
		info *grpc.StreamServerInfo, handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		pluginName, _ := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
		logCtx := "未知插件"
		if pluginName != "" {
			logCtx = pluginName
		}
		log.Info(ctx, "验证插件身份: "+logCtx)

		secretKey, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginSecretKey)
		if xErr != nil {
			return status.Error(codes.Unauthenticated, "缺少 plugin-secret-key")
		}

		pluginCredLogic := logic.NewPluginCredentialLogic(mainCtx)
		_, xErr = pluginCredLogic.Authenticate(ctx, secretKey)
		if xErr != nil {
			log.Warn(ctx, "插件认证失败: "+logCtx)
			return status.Error(codes.Unauthenticated, "插件认证失败")
		}

		log.Info(ctx, "插件认证通过: "+logCtx)
		return handler(srv, ss)
	}
}
