package handler

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
)

type service struct {
	healthLogic           *logic.HealthLogic
	serverStatusLogic     *logic.ServerStatusLogic
	titleLogic            *logic.TitleLogic
	achievementLogic      *logic.AchievementLogic
	pluginCredentialLogic *logic.PluginCredentialLogic
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
			healthLogic:           logic.NewHealthLogic(ctx),
			serverStatusLogic:     logic.NewServerStatusLogic(ctx),
			titleLogic:            logic.NewTitleLogic(ctx),
			achievementLogic:      logic.NewAchievementLogic(ctx),
			pluginCredentialLogic: logic.NewPluginCredentialLogic(ctx),
		},
	}
}

type HealthHandler handler
