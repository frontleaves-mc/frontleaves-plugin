package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	xCtx "github.com/bamboo-services/bamboo-base-go/defined/context"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	pluginGrpc "github.com/frontleaves-mc/frontleaves-plugin/internal/app/grpc"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
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

		// 异步同步 User + GameProfile 到本地数据库
		go syncUserAndGameProfiles(detachContext(ctx), authClient, userInfo.UserID, log)

		c.Next()
	}
}

// detachContext 从父 context 复制基础设施值（DB、Redis、Snowflake）到独立的新 context，
// 不共享取消通道，确保协程在请求结束后仍可独立运行。
func detachContext(parent context.Context) context.Context {
	detached := context.Background()
	if nodeList := parent.Value(xCtx.RegNodeKey); nodeList != nil {
		detached = context.WithValue(detached, xCtx.RegNodeKey, nodeList)
	}
	return detached
}

func syncUserAndGameProfiles(detachedCtx context.Context, authClient *pluginGrpc.AuthClient, userIDStr string, log *xLog.LogNamedLogger) {
	userID, err := xSnowflake.ParseSnowflakeID(userIDStr)
	if err != nil {
		log.Warn(context.Background(), "解析 UserID 失败: "+err.Error())
		return
	}

	grpcCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, rpcErr := authClient.GetUserInfo(grpcCtx, userIDStr)
	if rpcErr != nil {
		log.Warn(context.Background(), "GetUserInfo 调用失败: "+rpcErr.Error())
		return
	}

	userLogic := logic.NewUserLogic(detachedCtx)
	if xErr := userLogic.Upsert(nil, userID, resp.GetUsername()); xErr != nil {
		log.Warn(context.Background(), "同步 User 失败: "+xErr.Error())
	}

	gpLogic := logic.NewGameProfileLogic(detachedCtx)
	for _, gp := range resp.GetGameProfiles() {
		gpUUID, parseErr := uuid.Parse(gp.GetUuid())
		if parseErr != nil {
			log.Warn(context.Background(), "解析 GameProfile UUID 失败: "+parseErr.Error())
			continue
		}
		if xErr := gpLogic.Upsert(nil, userID, gpUUID, gp.GetUsername()); xErr != nil {
			log.Warn(context.Background(), "同步 GameProfile 失败: "+xErr.Error())
		}
	}
}

func injectUser(c *gin.Context, info *cache.AuthUserInfo) {
	newCtx := context.WithValue(c.Request.Context(), bConst.CtxAuthUserKey, info)
	c.Request = c.Request.WithContext(newCtx)
}
