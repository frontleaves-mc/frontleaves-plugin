package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) achievementRouter(router gin.IRouter) {
	achAdminHandler := handler.NewAchievementAdminHandler(r.context)
	achPlayerHandler := handler.NewAchievementPlayerHandler(r.context)

	adminGroup := router.Group("/admin/achievements")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.Admin(r.context))
	{
		adminGroup.POST("", achAdminHandler.CreateAchievement)
		adminGroup.PUT("/:id", achAdminHandler.UpdateAchievement)
		adminGroup.DELETE("/:id", achAdminHandler.DeleteAchievement)
		adminGroup.GET("", achAdminHandler.ListAchievements)
		adminGroup.GET("/:id", achAdminHandler.GetAchievement)
		adminGroup.POST("/:id/grant", achAdminHandler.GrantAchievement)
	}

	// 公开成就列表
	router.GET("/achievements", achPlayerHandler.ListPublicAchievements)

	// 玩家成就
	playerAchGroup := router.Group("/players/:uuid/achievements")
	playerAchGroup.Use(middleware.LoginAuth(r.context))
	playerAchGroup.Use(middleware.Player(r.context))
	{
		playerAchGroup.GET("", achPlayerHandler.GetPlayerAchievements)
		playerAchGroup.POST("/:achId/claim", achPlayerHandler.ClaimReward)
	}
}
