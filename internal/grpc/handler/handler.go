package handler

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/redis/go-redis/v9"
)

// grpcService gRPC 服务的业务逻辑处理层
type grpcService struct {
	titleLogic            *logic.TitleLogic
	gameProfileLogic      *logic.GameProfileLogic
	pluginCredentialLogic *logic.PluginCredentialLogic
}

// grpcHandler gRPC Handler 基类
type grpcHandler struct {
	name    string
	log     *xLog.LogNamedLogger
	service *grpcService
	rdb     *redis.Client
}

// IGRPCHandler gRPC Handler 泛型约束接口
type IGRPCHandler interface {
	~struct {
		name    string
		log     *xLog.LogNamedLogger
		service *grpcService
		rdb     *redis.Client
	}
}

// NewGRPCHandler 泛型 gRPC Handler 构造函数
func NewGRPCHandler[T IGRPCHandler](ctx context.Context, handlerName string) *T {
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &T{
		name: handlerName,
		log:  xLog.WithName(xLog.NamedGRPC, handlerName),
		service: &grpcService{
			titleLogic:            logic.NewTitleLogic(ctx),
			gameProfileLogic:      logic.NewGameProfileLogic(ctx),
			pluginCredentialLogic: logic.NewPluginCredentialLogic(ctx),
		},
		rdb: rdb,
	}
}
