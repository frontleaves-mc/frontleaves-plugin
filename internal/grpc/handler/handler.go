package handler

import (
	"context"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic/matrix"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
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
	directMessageLogic  *logic.DirectMessageLogic
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

// economyService 经济系统相关业务逻辑（EconomyTransactionHandler 使用）
type economyService struct {
	transactionLogLogic *logic.TransactionLogLogic
}

// matrixService Matrix 遥测相关业务逻辑（MatrixTelemetryHandler 使用）
type matrixService struct {
	sessionManager *matrix.MatrixSessionManager
	monitorCache   *cache.MatrixMonitorCache
	statRepo       *repository.MatrixStatisticRepo
	warningRepo    *repository.MatrixWarningRepo
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
		directMessageLogic: logic.NewDirectMessageLogic(ctx),
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

// newEconomyService 创建经济系统相关业务逻辑服务组
func newEconomyService(ctx context.Context) *economyService {
	db := xCtxUtil.MustGetDB(ctx)
	return &economyService{
		transactionLogLogic: logic.NewTransactionLogLogic(repository.NewTransactionLogRepo(db)),
	}
}

// newMatrixService 创建 Matrix 遥测相关业务逻辑服务组
func newMatrixService(ctx context.Context, db *gorm.DB, rdb *redis.Client) *matrixService {
	monitorCache := &cache.MatrixMonitorCache{RDB: rdb, TTL: 5 * time.Minute}
	statRepo := repository.NewMatrixStatisticRepo(db)
	warningRepo := repository.NewMatrixWarningRepo(db)
	sessionManager := matrix.NewMatrixSessionManager(ctx, db, rdb, monitorCache, statRepo, warningRepo)
	matrix.SetGlobalMatrixSessionManager(sessionManager)

	return &matrixService{
		sessionManager: sessionManager,
		monitorCache:   monitorCache,
		statRepo:       statRepo,
		warningRepo:    warningRepo,
	}
}
