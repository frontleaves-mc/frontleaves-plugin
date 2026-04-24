package middleware

import (
	"context"
	"slices"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/gin-gonic/gin"
)

func RequireRole(ctx context.Context, allowedRoles ...string) gin.HandlerFunc {
	log := xLog.WithName(xLog.NamedMIDE, "RequireRole")

	return func(c *gin.Context) {
		log.Info(c, "检查用户角色权限")

		userInfo, ok := c.Request.Context().Value(bConst.CtxAuthUserKey).(*cache.AuthUserInfo)
		if !ok || userInfo == nil {
			xResult.AbortError(c, xError.Unauthorized, "未获取到用户信息", nil)
			return
		}

		if !slices.Contains(allowedRoles, userInfo.RoleName) {
			xResult.AbortError(c, xError.PermissionDenied, "权限不足", nil)
			return
		}

		c.Next()
	}
}

func Admin(ctx context.Context) gin.HandlerFunc {
	return RequireRole(ctx, "SUPER_ADMIN", "ADMIN")
}

func SuperAdmin(ctx context.Context) gin.HandlerFunc {
	return RequireRole(ctx, "SUPER_ADMIN")
}

func Player(ctx context.Context) gin.HandlerFunc {
	return RequireRole(ctx, "PLAYER", "ADMIN", "SUPER_ADMIN")
}
