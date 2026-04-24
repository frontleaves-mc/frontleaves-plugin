package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"time"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/redis/go-redis/v9"
)

type AuthUserInfo struct {
	UserID   string `redis:"user_id"`
	Username string `redis:"username"`
	RoleName string `redis:"role_name"`
	HasBan   bool   `redis:"has_ban"`
}

type AuthUserCache struct {
	rdb *redis.Client
	ttl time.Duration
	log *xLog.LogNamedLogger
}

func NewAuthUserCache(rdb *redis.Client) *AuthUserCache {
	return &AuthUserCache{
		rdb: rdb,
		ttl: time.Minute * 15,
		log: xLog.WithName(xLog.NamedREPO, "AuthUserCache"),
	}
}

func (c *AuthUserCache) Get(ctx context.Context, accessToken string) (*AuthUserInfo, bool, error) {
	tokenMD5 := md5Hex(accessToken)
	key := bConst.CacheAuthUser.Get(tokenMD5).String()

	result, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, false, err
	}
	if len(result) == 0 {
		return nil, false, nil
	}

	info := &AuthUserInfo{
		UserID:   result["user_id"],
		Username: result["username"],
		RoleName: result["role_name"],
		HasBan:   result["has_ban"] == "1",
	}
	return info, true, nil
}

func (c *AuthUserCache) Set(ctx context.Context, accessToken string, info *AuthUserInfo) error {
	tokenMD5 := md5Hex(accessToken)
	key := bConst.CacheAuthUser.Get(tokenMD5).String()

	hasBan := "0"
	if info.HasBan {
		hasBan = "1"
	}

	if err := c.rdb.HSet(ctx, key, map[string]interface{}{
		"user_id":  info.UserID,
		"username": info.Username,
		"role_name": info.RoleName,
		"has_ban":  hasBan,
	}).Err(); err != nil {
		return err
	}
	return c.rdb.Expire(ctx, key, c.ttl).Err()
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
