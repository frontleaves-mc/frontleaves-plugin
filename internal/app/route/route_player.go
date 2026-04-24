package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) playerRouter(router gin.IRouter) {
	playerHandler := handler.NewPlayerHandler(r.context)

	playerGroup := router.Group("/players")
	{
		playerGroup.GET("/:uuid", playerHandler.GetPlayer)
		playerGroup.GET("", playerHandler.ListPlayers)
	}

	internalGroup := router.Group("/internal/players")
	{
		internalGroup.PUT("/:uuid/group", playerHandler.UpdatePlayerGroup)
	}
}
