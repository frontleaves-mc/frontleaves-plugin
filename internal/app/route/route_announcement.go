package route

import (
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	"github.com/gin-gonic/gin"
)

func (r *route) announcementRouter(router gin.IRouter) {
	adminHandler := handler.NewAnnouncementAdminHandler(r.context)
	userHandler := handler.NewAnnouncementUserHandler(r.context)
	scheduleAdminHandler := handler.NewAnnouncementScheduleAdminHandler(r.context)

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
	}

	adminScheduleGroup := router.Group("/admin/announcements/schedules")
	adminScheduleGroup.Use(middleware.LoginAuth(r.context))
	adminScheduleGroup.Use(middleware.Admin(r.context))
	{
		adminScheduleGroup.POST("", scheduleAdminHandler.CreateSchedule)
		adminScheduleGroup.PUT("/:id", scheduleAdminHandler.UpdateSchedule)
		adminScheduleGroup.DELETE("/:id", scheduleAdminHandler.DeleteSchedule)
		adminScheduleGroup.GET("", scheduleAdminHandler.ListSchedules)
		adminScheduleGroup.GET("/:id", scheduleAdminHandler.GetSchedule)
		adminScheduleGroup.POST("/:id/activate", scheduleAdminHandler.ActivateSchedule)
		adminScheduleGroup.POST("/:id/deactivate", scheduleAdminHandler.DeactivateSchedule)
	}

	userGroup := router.Group("/announcements")
	{
		userGroup.GET("", userHandler.ListPublishedAnnouncements)
		userGroup.GET("/:id", userHandler.GetPublishedAnnouncement)
	}
}
