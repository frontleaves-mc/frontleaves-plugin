package middleware

import (
	"context"
	"strings"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	xAsync "github.com/bamboo-services/bamboo-base-go/plugins/async"
	pluginGrpc "github.com/frontleaves-mc/frontleaves-plugin/internal/app/grpc"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/logic"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func LoginAuth(ctx context.Context) gin.HandlerFunc {
	log := xLog.WithName(xLog.NamedMIDE, "LoginAuth")

	authClient := xCtxUtil.MustGet[*pluginGrpc.AuthClient](ctx, bConst.CtxAuthClientKey)
	rdb := xCtxUtil.MustGetRDB(ctx)
	authUserRepo := repository.NewAuthUserRepo(rdb)

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

		// 通过 Repository 层读取缓存
		cachedUser, found, xErr := authUserRepo.GetCachedAuth(c, accessToken)
		if xErr != nil {
			log.Warn(c, "读取认证缓存失败: "+xErr.Error())
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

		userInfo := &repository.AuthUserInfo{
			UserID:   resp.GetUserId(),
			Username: resp.GetUsername(),
			RoleName: resp.GetRoleName(),
			HasBan:   resp.GetHasBan(),
		}

		// 通过 Repository 层写入缓存
		if cacheErr := authUserRepo.SetCachedAuth(c, accessToken, userInfo); cacheErr != nil {
			log.Warn(c, "写入认证缓存失败: "+cacheErr.Error())
		}

		injectUser(c, userInfo)

		// 异步同步 User + GameProfile 到本地数据库
		xAsync.Async(
			c.Request.Context(),
			func(asyncCtx context.Context) {
				syncUserAndGameProfiles(asyncCtx, authClient, userInfo.UserID, log)
			},
			xAsync.WithDebug(),
			xAsync.WithName("ProfileSync"),
		)

		c.Next()
	}
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
	if xErr := userLogic.Upsert(detachedCtx, userID, resp.GetUsername()); xErr != nil {
		log.Warn(context.Background(), "同步 User 失败: "+xErr.Error())
	}

	gpLogic := logic.NewGameProfileLogic(detachedCtx)
	for _, gp := range resp.GetGameProfiles() {
		gpUUID, parseErr := uuid.Parse(gp.GetUuid())
		if parseErr != nil {
			log.Warn(context.Background(), "解析 GameProfile UUID 失败: "+parseErr.Error())
			continue
		}
		if xErr := gpLogic.Upsert(detachedCtx, userID, gpUUID, gp.GetUsername()); xErr != nil {
			log.Warn(context.Background(), "同步 GameProfile 失败: "+xErr.Error())
		}
	}
}

func injectUser(c *gin.Context, info *repository.AuthUserInfo) {
	newCtx := context.WithValue(c.Request.Context(), bConst.CtxAuthUserKey, info)
	c.Request = c.Request.WithContext(newCtx)
}
