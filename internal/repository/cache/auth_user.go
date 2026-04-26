package cache

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"

	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/redis/go-redis/v9"
)

// AuthUserInfo 认证用户信息，由 gRPC 验证返回
type AuthUserInfo struct {
	UserID   string `redis:"user_id"`
	Username string `redis:"username"`
	RoleName string `redis:"role_name"`
	HasBan   bool   `redis:"has_ban"`
}

// AuthUserCache 认证用户缓存管理器，基于 xCache.Cache 标准类型
type AuthUserCache xCache.Cache

// Get 获取指定字段的值
func (c *AuthUserCache) Get(ctx context.Context, key string, field string) (*string, bool, error) {
	if key == "" {
		return nil, false, fmt.Errorf("缓存键为空")
	}
	if field == "" {
		return nil, false, fmt.Errorf("字段为空")
	}

	value, err := c.RDB.HGet(ctx, bConst.CacheAuthUser.Get(key).String(), field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &value, true, nil
}

// Set 设置指定字段的值
func (c *AuthUserCache) Set(ctx context.Context, key string, field string, value *string) error {
	if key == "" {
		return fmt.Errorf("缓存键为空")
	}
	if field == "" {
		return fmt.Errorf("字段为空")
	}
	if value == nil {
		return nil
	}

	if err := c.RDB.HSet(ctx, bConst.CacheAuthUser.Get(key).String(), field, *value).Err(); err != nil {
		return err
	}
	return c.RDB.Expire(ctx, bConst.CacheAuthUser.Get(key).String(), c.TTL).Err()
}

// GetAll 获取哈希表中的所有字段和值
func (c *AuthUserCache) GetAll(ctx context.Context, key string) (map[string]string, error) {
	if key == "" {
		return nil, fmt.Errorf("缓存键为空")
	}
	return c.RDB.HGetAll(ctx, bConst.CacheAuthUser.Get(key).String()).Result()
}

// GetAllStruct 获取所有字段并映射为 AuthUserInfo
func (c *AuthUserCache) GetAllStruct(ctx context.Context, key string) (*AuthUserInfo, error) {
	if key == "" {
		return nil, fmt.Errorf("缓存键为空")
	}

	result, err := c.RDB.HGetAll(ctx, bConst.CacheAuthUser.Get(key).String()).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return parseAuthUserFields(result)
}

// SetAll 批量设置多个字段的值
func (c *AuthUserCache) SetAll(ctx context.Context, key string, fields map[string]*string) error {
	if key == "" {
		return fmt.Errorf("缓存键为空")
	}
	if len(fields) == 0 {
		return nil
	}

	values := make(map[string]any, len(fields))
	for field, value := range fields {
		if field == "" {
			return fmt.Errorf("字段为空")
		}
		if value != nil {
			values[field] = *value
		}
	}

	if err := c.RDB.HSet(ctx, bConst.CacheAuthUser.Get(key).String(), values).Err(); err != nil {
		return err
	}
	return c.RDB.Expire(ctx, bConst.CacheAuthUser.Get(key).String(), c.TTL).Err()
}

// SetAllStruct 批量设置 AuthUserInfo 的缓存字段
func (c *AuthUserCache) SetAllStruct(ctx context.Context, key string, value *AuthUserInfo) error {
	if key == "" {
		return fmt.Errorf("缓存键为空")
	}
	if value == nil {
		return nil
	}

	fields := buildAuthUserFields(value)
	if err := c.RDB.HSet(ctx, bConst.CacheAuthUser.Get(key).String(), fields).Err(); err != nil {
		return err
	}
	return c.RDB.Expire(ctx, bConst.CacheAuthUser.Get(key).String(), c.TTL).Err()
}

// Exists 检查指定字段是否存在
func (c *AuthUserCache) Exists(ctx context.Context, key string, field string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("缓存键为空")
	}
	if field == "" {
		return false, fmt.Errorf("字段为空")
	}
	return c.RDB.HExists(ctx, bConst.CacheAuthUser.Get(key).String(), field).Result()
}

// Remove 从缓存中移除指定的字段
func (c *AuthUserCache) Remove(ctx context.Context, key string, fields ...string) error {
	if key == "" {
		return fmt.Errorf("缓存键为空")
	}
	if len(fields) == 0 {
		return nil
	}
	return c.RDB.HDel(ctx, bConst.CacheAuthUser.Get(key).String(), fields...).Err()
}

// Delete 删除整个认证缓存
func (c *AuthUserCache) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("缓存键为空")
	}
	return c.RDB.Del(ctx, bConst.CacheAuthUser.Get(key).String()).Err()
}

func buildAuthUserFields(info *AuthUserInfo) map[string]any {
	hasBan := "0"
	if info.HasBan {
		hasBan = "1"
	}
	return map[string]any{
		"user_id":   info.UserID,
		"username":  info.Username,
		"role_name": info.RoleName,
		"has_ban":   hasBan,
	}
}

func parseAuthUserFields(fields map[string]string) (*AuthUserInfo, error) {
	info := &AuthUserInfo{
		UserID:   fields["user_id"],
		Username: fields["username"],
		RoleName: fields["role_name"],
		HasBan:   fields["has_ban"] == "1",
	}
	return info, nil
}

func Md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
