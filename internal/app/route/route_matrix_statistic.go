package route

import (
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) matrixStatisticRouter(router gin.IRouter) {
	adminHandler := handler.NewMatrixStatisticAdminHandler(r.context)
	playerHandler := handler.NewMatrixStatisticPlayerHandler(r.context)

	adminGroup := router.Group("/admin/matrix/statistics")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.Admin(r.context))
	{
		adminGroup.GET("/:uuid", adminHandler.GetByUUID)
	}

	playerGroup := router.Group("/matrix/statistics")
	playerGroup.Use(middleware.LoginAuth(r.context))
	playerGroup.Use(middleware.Player(r.context))
	{
		playerGroup.GET("/me", playerHandler.GetMyStatistic)
	}
}