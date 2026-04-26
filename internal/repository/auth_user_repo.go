package repository

import (
	"context"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/redis/go-redis/v9"
)

// AuthUserInfo 认证用户信息，由 gRPC 验证返回
type AuthUserInfo = cache.AuthUserInfo

// AuthUserRepo 认证数据仓储，封装认证缓存操作
type AuthUserRepo struct {
	cache *cache.AuthUserCache
	log   *xLog.LogNamedLogger
}

func NewAuthUserRepo(rdb *redis.Client) *AuthUserRepo {
	return &AuthUserRepo{
		cache: &cache.AuthUserCache{
			RDB: rdb,
			TTL: time.Minute * 15,
		},
		log: xLog.WithName(xLog.NamedREPO, "AuthUserRepo"),
	}
}

// GetCachedAuth 获取缓存的认证信息（Cache-Aside 读）
func (r *AuthUserRepo) GetCachedAuth(ctx context.Context, accessToken string) (*AuthUserInfo, bool, *xError.Error) {
	tokenMD5 := cache.Md5Hex(accessToken)
	result, err := r.cache.GetAll(ctx, tokenMD5)
	if err != nil {
		r.log.Warn(ctx, "读取认证缓存失败: "+err.Error())
		return nil, false, xError.NewError(ctx, xError.CacheError, "读取认证缓存失败", true, err)
	}
	if len(result) == 0 {
		return nil, false, nil
	}
	info := &cache.AuthUserInfo{
		UserID:   result["user_id"],
		Username: result["username"],
		RoleName: result["role_name"],
		HasBan:   result["has_ban"] == "1",
	}
	return info, true, nil
}

// SetCachedAuth 写入认证缓存
func (r *AuthUserRepo) SetCachedAuth(ctx context.Context, accessToken string, info *AuthUserInfo) *xError.Error {
	tokenMD5 := cache.Md5Hex(accessToken)
	if err := r.cache.SetAllStruct(ctx, tokenMD5, info); err != nil {
		r.log.Warn(ctx, "写入认证缓存失败: "+err.Error())
		return xError.NewError(ctx, xError.CacheError, "写入认证缓存失败", true, err)
	}
	return nil
}

// DeleteCachedAuth 删除认证缓存（用于登出等场景）
func (r *AuthUserRepo) DeleteCachedAuth(ctx context.Context, accessToken string) *xError.Error {
	tokenMD5 := cache.Md5Hex(accessToken)
	if err := r.cache.Delete(ctx, tokenMD5); err != nil {
		r.log.Warn(ctx, "删除认证缓存失败: "+err.Error())
		return xError.NewError(ctx, xError.CacheError, "删除认证缓存失败", true, err)
	}
	return nil
}
