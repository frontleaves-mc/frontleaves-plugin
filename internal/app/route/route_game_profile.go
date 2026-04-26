package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) gameProfileRouter(router gin.IRouter) {
	gameProfileHandler := handler.NewGameProfileHandler(r.context)

	gameProfileGroup := router.Group("/game-profiles")
	gameProfileGroup.Use(middleware.LoginAuth(r.context))
	gameProfileGroup.Use(middleware.Player(r.context))
	{
		gameProfileGroup.GET("/:uuid", gameProfileHandler.GetGameProfile)
		gameProfileGroup.GET("", gameProfileHandler.ListGameProfiles)
	}
}
