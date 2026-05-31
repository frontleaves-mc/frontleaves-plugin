package route

import (
	"github.com/frontleaves-mc/frontleaves-plugin/internal/app/middleware"
	handler "github.com/frontleaves-mc/frontleaves-plugin/internal/handler"
	"github.com/gin-gonic/gin"
)

func (r *route) economyRouter(router gin.IRouter) {
	transactionHandler := handler.NewEconomyTransactionHandler(r.context)
	balanceHandler := handler.NewEconomyBalanceHandler(r.context)
	auditHandler := handler.NewEconomyAuditHandler(r.context)

	// 管理端接口
	adminGroup := router.Group("/admin/economy")
	adminGroup.Use(middleware.LoginAuth(r.context))
	adminGroup.Use(middleware.Admin(r.context))
	{
		adminGroup.GET("/transactions", transactionHandler.ListPlayerTransactions)
		adminGroup.GET("/audit-logs", auditHandler.ListAdminAuditLogs)
		adminGroup.GET("/balance", balanceHandler.GetPlayerBalance)
		adminGroup.POST("/balance/adjust", balanceHandler.AdjustPlayerBalance)
	}

	// 用户端接口
	userGroup := router.Group("/user/economy")
	userGroup.Use(middleware.LoginAuth(r.context))
	{
		userGroup.GET("/transactions", transactionHandler.ListMyTransactions)
		userGroup.GET("/balance", balanceHandler.GetMyBalance)
	}
}
