package middleware

import (
	"context"

	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/utility"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryPluginVerify 创建插件认证中间件
//
// 从 gRPC metadata 中提取 plugin-name 和 plugin-secret-key，
// 调用 PluginCredentialLogic 进行验证。
// 此中间件应作为 per-service 中间件绑定，而非全局拦截器。
func UnaryPluginVerify(mainCtx context.Context) grpc.UnaryServerInterceptor {
	log := xLog.WithName(xLog.NamedMIDE, "UnaryPluginVerify")

	return func(
		ctx context.Context, req any,
		info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (any, error) {
		log.Info(ctx, "验证插件身份")

		// 从 metadata 提取插件名称
		pluginName, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
		if xErr != nil {
			return nil, xError.NewError(ctx, xError.Unauthorized, "缺少 plugin-name", true)
		}

		// 从 metadata 提取插件密钥
		secretKey, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginSecretKey)
		if xErr != nil {
			return nil, xError.NewError(ctx, xError.Unauthorized, "缺少 plugin-secret-key", true)
		}

		// 调用 Logic 层认证
		pluginCredLogic := logic.NewPluginCredentialLogic(mainCtx)
		_, xErr = pluginCredLogic.Authenticate(ctx, pluginName, secretKey)
		if xErr != nil {
			log.Warn(ctx, "插件认证失败: "+pluginName)
			return nil, xErr
		}

		log.Info(ctx, "插件认证通过: "+pluginName)
		return handler(ctx, req)
	}
}

// StreamPluginVerify 创建流式插件认证中间件
//
// 从 gRPC stream metadata 中提取 plugin-name 和 plugin-secret-key，
// 调用 PluginCredentialLogic 进行验证。
// 流式拦截器直接返回 gRPC status 错误，不使用 xError。
func StreamPluginVerify(mainCtx context.Context) grpc.StreamServerInterceptor {
	log := xLog.WithName(xLog.NamedMIDE, "StreamPluginVerify")

	return func(
		srv interface{}, ss grpc.ServerStream,
		info *grpc.StreamServerInfo, handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()
		log.Info(ctx, "验证插件身份")

		// 从 metadata 提取插件名称
		pluginName, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginName)
		if xErr != nil {
			return status.Error(codes.Unauthenticated, "缺少 plugin-name")
		}

		// 从 metadata 提取插件密钥
		secretKey, xErr := xGrpcUtil.ExtractMetadata(ctx, bConst.MetadataPluginSecretKey)
		if xErr != nil {
			return status.Error(codes.Unauthenticated, "缺少 plugin-secret-key")
		}

		// 调用 Logic 层认证
		pluginCredLogic := logic.NewPluginCredentialLogic(mainCtx)
		_, xErr = pluginCredLogic.Authenticate(ctx, pluginName, secretKey)
		if xErr != nil {
			log.Warn(ctx, "插件认证失败: "+pluginName)
			return status.Error(codes.Unauthenticated, "插件认证失败")
		}

		log.Info(ctx, "插件认证通过: "+pluginName)
		return handler(srv, ss)
	}
}
