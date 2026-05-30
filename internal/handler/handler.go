package handler

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type service struct {
	healthLogic               *logic.HealthLogic
	serverStatusLogic         *logic.ServerStatusLogic
	serverLogic               *logic.ServerLogic
	titleLogic                *logic.TitleLogic
	achievementLogic          *logic.AchievementLogic
	announcementLogic         *logic.AnnouncementLogic
	pluginCredentialLogic     *logic.PluginCredentialLogic
	playerChatLogic           *logic.PlayerChatLogic
	playerCommandLogic        *logic.PlayerCommandLogic
	gameProfileLogic          *logic.GameProfileLogic
	serverLoadLogic           *logic.ServerLoadLogic
	matrixWarningQueryLogic   *logic.WarningQueryLogic
	matrixStatisticQueryLogic *logic.StatisticQueryLogic
	directMessageLogic        *logic.DirectMessageLogic
	transactionLogLogic       *logic.TransactionLogLogic
}

type handler struct {
	name    string
	log     *xLog.LogNamedLogger
	service *service
}

type IHandler interface {
	~struct {
		name    string
		log     *xLog.LogNamedLogger
		service *service
	}
}

func NewHandler[T IHandler](ctx context.Context, handlerName string) *T {
	return &T{
		name: handlerName,
		log:  xLog.WithName(xLog.NamedCONT, handlerName),
		service: &service{
			healthLogic:               logic.NewHealthLogic(ctx),
			serverStatusLogic:         logic.NewServerStatusLogic(ctx),
			serverLogic:               logic.NewServerLogic(ctx),
			titleLogic:                logic.NewTitleLogic(ctx),
			achievementLogic:          logic.NewAchievementLogic(ctx),
			announcementLogic:         logic.NewAnnouncementLogic(ctx),
			pluginCredentialLogic:     logic.NewPluginCredentialLogic(ctx),
			playerChatLogic:           logic.NewPlayerChatLogic(ctx),
			playerCommandLogic:        logic.NewPlayerCommandLogic(ctx),
			gameProfileLogic:          logic.NewGameProfileLogic(ctx),
			serverLoadLogic:           logic.NewServerLoadLogic(ctx),
			matrixWarningQueryLogic:   logic.NewWarningQueryLogic(ctx),
			matrixStatisticQueryLogic: logic.NewStatisticQueryLogic(ctx),
			directMessageLogic:        logic.NewDirectMessageLogic(ctx),
			transactionLogLogic:       logic.NewTransactionLogLogic(repository.NewTransactionLogRepo(xCtxUtil.MustGetDB(ctx))),
		},
	}
}

type HealthHandler handler
