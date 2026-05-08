package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) serverStatusRouter(router gin.IRouter) {
	serverStatusHandler := handler.NewServerStatusHandler(r.context)
	playerOnlineHandler := handler.NewPlayerOnlineHandler(r.context)

	serverGroup := router.Group("/servers")
	serverGroup.Use(middleware.LoginAuth(r.context))
	{
		serverGroup.GET("/status", serverStatusHandler.ListServerStatus)
		serverGroup.POST("/:name/refresh", serverStatusHandler.RefreshServerStatus)
		serverGroup.GET("/game-profiles/online/mine", playerOnlineHandler.GetMyOnlineProfiles)
		serverGroup.GET("/players/online/check", playerOnlineHandler.CheckPlayerOnline)
	}
}
