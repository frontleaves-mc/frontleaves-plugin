package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) serverRouter(router gin.IRouter) {
	serverAdminHandler := handler.NewServerAdminHandler(r.context)

	adminGroup := router.Group("/admin/servers")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.SuperAdmin(r.context))
	{
		adminGroup.POST("", serverAdminHandler.CreateServer)
		adminGroup.GET("", serverAdminHandler.ListServers)
		adminGroup.GET("/:id", serverAdminHandler.GetServer)
		adminGroup.PUT("/:id", serverAdminHandler.UpdateServer)
		adminGroup.DELETE("/:id", serverAdminHandler.DeleteServer)
		adminGroup.PUT("/:id/public", serverAdminHandler.SetServerPublic)
		adminGroup.PUT("/:id/enabled", serverAdminHandler.SetServerEnabled)
	}
}
