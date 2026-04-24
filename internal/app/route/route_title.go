package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) titleRouter(router gin.IRouter) {
	titleAdminHandler := handler.NewTitleAdminHandler(r.context)
	titlePlayerHandler := handler.NewTitlePlayerHandler(r.context)

	adminGroup := router.Group("/admin/titles")
	{
		adminGroup.POST("", titleAdminHandler.CreateTitle)
		adminGroup.PUT("/:id", titleAdminHandler.UpdateTitle)
		adminGroup.DELETE("/:id", titleAdminHandler.DeleteTitle)
		adminGroup.GET("", titleAdminHandler.ListTitles)
		adminGroup.GET("/:id", titleAdminHandler.GetTitle)
		adminGroup.POST("/:id/assign", titleAdminHandler.AssignTitle)
		adminGroup.DELETE("/:id/assign", titleAdminHandler.RevokeTitle)
	}

	playerGroup := router.Group("/players/:uuid/titles")
	{
		playerGroup.GET("", titlePlayerHandler.GetPlayerTitles)
		playerGroup.PUT("/equip", titlePlayerHandler.EquipTitle)
		playerGroup.DELETE("/equip", titlePlayerHandler.UnequipTitle)
		playerGroup.GET("/equipped", titlePlayerHandler.GetEquippedTitle)
	}
}
