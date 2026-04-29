package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) pluginCredentialRouter(router gin.IRouter) {
	pluginCredentialAdminHandler := handler.NewPluginCredentialAdminHandler(r.context)

	adminGroup := router.Group("/admin/plugin-credentials")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.SuperAdmin(r.context))
	{
		adminGroup.POST("", pluginCredentialAdminHandler.CreatePluginCredential)
		adminGroup.PUT("/:id", pluginCredentialAdminHandler.UpdatePluginCredential)
		adminGroup.PUT("/:id/reset-key", pluginCredentialAdminHandler.ResetPluginCredentialKey)
		adminGroup.DELETE("/:id", pluginCredentialAdminHandler.DeletePluginCredential)
		adminGroup.GET("", pluginCredentialAdminHandler.ListPluginCredentials)
		adminGroup.GET("/:id", pluginCredentialAdminHandler.GetPluginCredential)
	}
}
