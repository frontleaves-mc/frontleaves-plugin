package handler

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/redis/go-redis/v9"
)

// grpcHandler gRPC Handler 基类 — 仅包含共享字段
type grpcHandler struct {
	name string
	log  *xLog.LogNamedLogger
	rdb  *redis.Client
}

// essentialsService 服务器状态相关业务逻辑（EssentialsPlayerEventHandler/EssentialsPlayerQueryHandler 使用）
type essentialsService struct {
	gameProfileLogic    *logic.GameProfileLogic
	playerEventLogic    *logic.PlayerEventLogic
	playerChatLogic     *logic.PlayerChatLogic
	playerCommandLogic  *logic.PlayerCommandLogic
	serverLogic         *logic.ServerLogic
	serverPlayerLogic   *logic.ServerPlayerLogic
}

// statusService 服务器状态相关业务逻辑（ServerStatusHandler 精简版使用）
type statusService struct {
	serverLogic       *logic.ServerLogic
	serverPlayerLogic *logic.ServerPlayerLogic
}

// titleService 称号相关业务逻辑（TitleHandler 使用）
type titleService struct {
	titleLogic       *logic.TitleLogic
	gameProfileLogic *logic.GameProfileLogic
}

// IGRPCHandler gRPC Handler 泛型约束接口
type IGRPCHandler interface {
	~struct {
		name string
		log  *xLog.LogNamedLogger
		rdb  *redis.Client
	}
}

// NewGRPCHandler 泛型 gRPC Handler 构造函数（仅初始化共享基类字段）
func NewGRPCHandler[T IGRPCHandler](ctx context.Context, handlerName string) *T {
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &T{
		name: handlerName,
		log:  xLog.WithName(xLog.NamedGRPC, handlerName),
		rdb:  rdb,
	}
}

// newStatusService 创建服务器状态相关业务逻辑服务组（精简版）
func newStatusService(ctx context.Context) *statusService {
	return &statusService{
		serverLogic:       logic.NewServerLogic(ctx),
		serverPlayerLogic: logic.NewServerPlayerLogic(ctx),
	}
}

// newEssentialsService 创建服务器状态相关业务逻辑服务组
func newEssentialsService(ctx context.Context) *essentialsService {
	return &essentialsService{
		gameProfileLogic:   logic.NewGameProfileLogic(ctx),
		playerEventLogic:   logic.NewPlayerEventLogic(ctx),
		playerChatLogic:    logic.NewPlayerChatLogic(ctx),
		playerCommandLogic: logic.NewPlayerCommandLogic(ctx),
		serverLogic:        logic.NewServerLogic(ctx),
		serverPlayerLogic:  logic.NewServerPlayerLogic(ctx),
	}
}

// newTitleService 创建称号相关业务逻辑服务组
func newTitleService(ctx context.Context) *titleService {
	return &titleService{
		titleLogic:       logic.NewTitleLogic(ctx),
		gameProfileLogic: logic.NewGameProfileLogic(ctx),
	}
}
