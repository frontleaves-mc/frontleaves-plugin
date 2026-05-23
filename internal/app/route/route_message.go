package route

import (
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) messageRouter(router gin.IRouter) {
	messageAdminHandler := handler.NewMessageAdminHandler(r.context)
	messageUserHandler := handler.NewMessageUserHandler(r.context)
	chatSSEHandler := handler.NewChatSSEHandler(r.context)
	chatSendHandler := handler.NewChatSendHandler(r.context)

	// 管理端接口
	adminGroup := router.Group("/admin/messages")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.Admin(r.context))
	{
		adminGroup.GET("/chat", messageAdminHandler.ListAllChatHistory)
		adminGroup.GET("/commands", messageAdminHandler.ListAllCommandHistory)
	}

	// 用户端接口
	userGroup := router.Group("/user/messages")
	userGroup.Use(middleware.LoginAuth(r.context))
	userGroup.Use(middleware.Player(r.context))
	{
		userGroup.GET("/chat", messageUserHandler.ListMyChatHistory)
		userGroup.GET("/commands", messageUserHandler.ListMyCommandHistory)
		userGroup.GET("/chat/stream", chatSSEHandler.StreamChat)
		userGroup.POST("/chat", chatSendHandler.SendChatMessage)
	}
}
