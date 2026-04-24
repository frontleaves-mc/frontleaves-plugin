package handler

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
)

type service struct {
	healthLogic      *logic.HealthLogic
	playerLogic      *logic.PlayerLogic
	titleLogic       *logic.TitleLogic
	achievementLogic *logic.AchievementLogic
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
			healthLogic:      logic.NewHealthLogic(ctx),
			playerLogic:      logic.NewPlayerLogic(ctx),
			titleLogic:       logic.NewTitleLogic(ctx),
			achievementLogic: logic.NewAchievementLogic(ctx),
		},
	}
}

type HealthHandler handler
