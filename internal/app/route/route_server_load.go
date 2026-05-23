package route

import (
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) serverLoadRouter(router gin.IRouter) {
	serverLoadAdminHandler := handler.NewServerLoadAdminHandler(r.context)

	adminGroup := router.Group("/admin/servers/load")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.SuperAdmin(r.context))
	{
		adminGroup.GET("/realtime", serverLoadAdminHandler.GetAllRealtimeLoad)
		adminGroup.GET("/:id/realtime", serverLoadAdminHandler.GetRealtimeLoad)
		adminGroup.GET("/:id/history", serverLoadAdminHandler.GetLoadHistory)
	}
}
