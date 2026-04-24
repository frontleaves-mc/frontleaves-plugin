package middleware

import (
	"context"
	"strings"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	pluginGrpc "github.com/frontleaves-mc/frontleaves-plugin/internal/app/grpc"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/gin-gonic/gin"
)

func LoginAuth(ctx context.Context) gin.HandlerFunc {
	log := xLog.WithName(xLog.NamedMIDE, "LoginAuth")

	authClient := xCtxUtil.MustGet[*pluginGrpc.AuthClient](ctx, bConst.CtxAuthClientKey)
	rdb := xCtxUtil.MustGetRDB(ctx)
	authUserCache := cache.NewAuthUserCache(rdb)

	return func(c *gin.Context) {
		log.Info(c, "验证用户登录状态")

		// 提取 Bearer Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			xResult.AbortError(c, xError.ParameterEmpty, "需要访问令牌", nil)
			return
		}
		accessToken := strings.TrimPrefix(authHeader, "Bearer ")
		if accessToken == authHeader {
			xResult.AbortError(c, xError.ParameterEmpty, "访问令牌格式无效", nil)
			return
		}

		// 缓存优先
		cachedUser, found, err := authUserCache.Get(c, accessToken)
		if err != nil {
			log.Warn(c, "读取认证缓存失败: "+err.Error())
		}
		if found && cachedUser != nil {
			if cachedUser.HasBan {
				xResult.AbortError(c, xError.Forbidden, "用户已被封禁", nil)
				return
			}
			injectUser(c, cachedUser)
			c.Next()
			return
		}

		// gRPC 远程验证
		resp, rpcErr := authClient.ValidateToken(c.Request.Context(), accessToken)
		if rpcErr != nil {
			xResult.AbortError(c, xError.Unauthorized, xError.ErrMessage("认证失败: "+rpcErr.Error()), nil)
			return
		}

		userInfo := &cache.AuthUserInfo{
			UserID:   resp.GetUserId(),
			Username: resp.GetUsername(),
			RoleName: resp.GetRoleName(),
			HasBan:   resp.GetHasBan(),
		}

		// 写入本地缓存
		if cacheErr := authUserCache.Set(c, accessToken, userInfo); cacheErr != nil {
			log.Warn(c, "写入认证缓存失败: "+cacheErr.Error())
		}

		injectUser(c, userInfo)
		c.Next()
	}
}

func injectUser(c *gin.Context, info *cache.AuthUserInfo) {
	newCtx := context.WithValue(c.Request.Context(), bConst.CtxAuthUserKey, info)
	c.Request = c.Request.WithContext(newCtx)
}
