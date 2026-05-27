package route

import (
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) dmRouter(router gin.IRouter) {
	dmHandler := handler.NewDirectMessageHandler(r.context)

	userGroup := router.Group("/user/messages/dm")
	userGroup.Use(middleware.LoginAuth(r.context))
	userGroup.Use(middleware.Player(r.context))
	{
		userGroup.POST("", dmHandler.SendDirectMessage)
		userGroup.GET("", dmHandler.ListMyDirectMessages)
		userGroup.GET("/unread", dmHandler.GetUnreadCount)
		userGroup.PUT("/read", dmHandler.MarkAsRead)
		userGroup.GET("/conversations", dmHandler.ListConversations)
	}

	adminGroup := router.Group("/admin/messages/dm")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.Admin(r.context))
	{
		adminGroup.GET("", dmHandler.ListAllDirectMessages)
	}
}
