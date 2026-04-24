package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) playerRouter(router gin.IRouter) {
	playerHandler := handler.NewPlayerHandler(r.context)

	playerGroup := router.Group("/players")
	playerGroup.Use(middleware.LoginAuth(r.context))
	playerGroup.Use(middleware.Player(r.context))
	{
		playerGroup.GET("/:uuid", playerHandler.GetPlayer)
		playerGroup.GET("", playerHandler.ListPlayers)
	}

	internalGroup := router.Group("/internal/players")
	internalGroup.Use(middleware.LoginAuth(r.context))
	internalGroup.Use(middleware.SuperAdmin(r.context))
	{
		internalGroup.PUT("/:uuid/group", playerHandler.UpdatePlayerGroup)
	}
}
