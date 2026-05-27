package route

import (
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) matrixWarningRouter(router gin.IRouter) {
	adminHandler := handler.NewMatrixWarningAdminHandler(r.context)

	adminGroup := router.Group("/admin/matrix/warnings")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.Admin(r.context))
	{
		adminGroup.GET("", adminHandler.ListWarnings)
		adminGroup.GET("/:id", adminHandler.GetWarning)
	}
}