package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) titleRouter(router gin.IRouter) {
	titleAdminHandler := handler.NewTitleAdminHandler(r.context)
	titleGameProfileHandler := handler.NewTitleGameProfileHandler(r.context)

	adminGroup := router.Group("/admin/titles")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.Admin(r.context))
	{
		adminGroup.POST("", titleAdminHandler.CreateTitle)
		adminGroup.PUT("/:id", titleAdminHandler.UpdateTitle)
		adminGroup.DELETE("/:id", titleAdminHandler.DeleteTitle)
		adminGroup.GET("", titleAdminHandler.ListTitles)
		adminGroup.GET("/:id", titleAdminHandler.GetTitle)
		adminGroup.POST("/:id/assign", titleAdminHandler.AssignTitle)
		adminGroup.DELETE("/:id/assign", titleAdminHandler.RevokeTitle)
	}

	gameProfileGroup := router.Group("/game-profiles/:uuid/titles")
	gameProfileGroup.Use(middleware.LoginAuth(r.context))
	gameProfileGroup.Use(middleware.Player(r.context))
	{
		gameProfileGroup.GET("", titleGameProfileHandler.GetPlayerTitles)
		gameProfileGroup.PUT("/equip", titleGameProfileHandler.EquipTitle)
		gameProfileGroup.DELETE("/equip", titleGameProfileHandler.UnequipTitle)
		gameProfileGroup.GET("/equipped", titleGameProfileHandler.GetEquippedTitle)
	}
}
