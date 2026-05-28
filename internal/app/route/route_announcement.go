package route

import (
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) announcementRouter(router gin.IRouter) {
	adminHandler := handler.NewAnnouncementAdminHandler(r.context)
	userHandler := handler.NewAnnouncementUserHandler(r.context)

	adminAnnouncementGroup := router.Group("/admin/announcements")
	adminAnnouncementGroup.Use(middleware.LoginAuth(r.context))
	adminAnnouncementGroup.Use(middleware.Admin(r.context))
	{
		adminAnnouncementGroup.POST("", adminHandler.CreateAnnouncement)
		adminAnnouncementGroup.PUT("/:id", adminHandler.UpdateAnnouncement)
		adminAnnouncementGroup.DELETE("/:id", adminHandler.DeleteAnnouncement)
		adminAnnouncementGroup.GET("", adminHandler.ListAnnouncements)
		adminAnnouncementGroup.GET("/:id", adminHandler.GetAnnouncement)
		adminAnnouncementGroup.POST("/:id/publish", adminHandler.PublishAnnouncement)
		adminAnnouncementGroup.POST("/:id/offline", adminHandler.OfflineAnnouncement)
		adminAnnouncementGroup.GET("/scheduler/config", adminHandler.GetSchedulerConfig)
		adminAnnouncementGroup.PUT("/scheduler/config", adminHandler.UpdateSchedulerConfig)
		adminAnnouncementGroup.POST("/scheduler/enable", adminHandler.EnableScheduler)
		adminAnnouncementGroup.POST("/scheduler/disable", adminHandler.DisableScheduler)
	}

	userGroup := router.Group("/announcements")
	{
		userGroup.GET("", userHandler.ListPublishedAnnouncements)
		userGroup.GET("/:id", userHandler.GetPublishedAnnouncement)
	}
}
