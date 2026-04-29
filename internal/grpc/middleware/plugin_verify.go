package middleware

import (
	"context"

	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xGrpcUtil "github.com/bamboo-services/bamboo-base-go/plugins/grpc/utility"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"google.golang.org/grpc"
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
