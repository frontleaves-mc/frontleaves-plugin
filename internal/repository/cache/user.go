package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
)

const (
	userCacheFieldID       = "id"
	userCacheFieldUpdated  = "updated_at"
	userCacheFieldName     = "username"
	userCacheTimeLayout    = "2006-01-02T15:04:05.999999999Z07:00"
)

// UserCache 用户实体缓存管理器，基于 xCache.Cache 标准类型
type UserCache xCache.Cache

// Get 获取指定字段的值
func (c *UserCache) Get(ctx context.Context, key string, field string) (*string, bool, error) {
	if key == "" {
		return nil, false, fmt.Errorf("用户标识为空")
	}
	if field == "" {
		return nil, false, fmt.Errorf("字段为空")
	}

	value, err := c.RDB.HGet(ctx, bConst.CacheUserEntity.Get(key).String(), field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &value, true, nil
}

// Set 设置指定字段的值
func (c *UserCache) Set(ctx context.Context, key string, field string, value *string) error {
	if key == "" {
		return fmt.Errorf("用户标识为空")
	}
	if field == "" {
		return fmt.Errorf("字段为空")
	}
	if value == nil {
		return nil
	}

	if err := c.RDB.HSet(ctx, bConst.CacheUserEntity.Get(key).String(), field, *value).Err(); err != nil {
		return err
	}
	return c.RDB.Expire(ctx, bConst.CacheUserEntity.Get(key).String(), c.TTL).Err()
}

// GetAll 获取哈希表中的所有字段和值
func (c *UserCache) GetAll(ctx context.Context, key string) (map[string]string, error) {
	if key == "" {
		return nil, fmt.Errorf("用户标识为空")
	}
	return c.RDB.HGetAll(ctx, bConst.CacheUserEntity.Get(key).String()).Result()
}

// GetAllStruct 获取所有字段并映射为 entity.User
func (c *UserCache) GetAllStruct(ctx context.Context, key string) (*entity.User, error) {
	if key == "" {
		return nil, fmt.Errorf("用户标识为空")
	}

	result, err := c.RDB.HGetAll(ctx, bConst.CacheUserEntity.Get(key).String()).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return parseUserFields(result)
}

// SetAll 批量设置多个字段的值
func (c *UserCache) SetAll(ctx context.Context, key string, fields map[string]*string) error {
	if key == "" {
		return fmt.Errorf("用户标识为空")
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

	if err := c.RDB.HSet(ctx, bConst.CacheUserEntity.Get(key).String(), values).Err(); err != nil {
		return err
	}
	return c.RDB.Expire(ctx, bConst.CacheUserEntity.Get(key).String(), c.TTL).Err()
}

// SetAllStruct 批量设置用户实体的缓存字段
func (c *UserCache) SetAllStruct(ctx context.Context, key string, value *entity.User) error {
	if key == "" {
		return fmt.Errorf("用户标识为空")
	}
	if value == nil {
		return nil
	}

	fields := buildUserFields(value)
	if len(fields) == 0 {
		return nil
	}

	if err := c.RDB.HSet(ctx, bConst.CacheUserEntity.Get(key).String(), fields).Err(); err != nil {
		return err
	}
	return c.RDB.Expire(ctx, bConst.CacheUserEntity.Get(key).String(), c.TTL).Err()
}

// Exists 检查指定字段是否存在
func (c *UserCache) Exists(ctx context.Context, key string, field string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("用户标识为空")
	}
	if field == "" {
		return false, fmt.Errorf("字段为空")
	}
	return c.RDB.HExists(ctx, bConst.CacheUserEntity.Get(key).String(), field).Result()
}

// Remove 从缓存中移除指定的字段
func (c *UserCache) Remove(ctx context.Context, key string, fields ...string) error {
	if key == "" {
		return fmt.Errorf("用户标识为空")
	}
	if len(fields) == 0 {
		return nil
	}
	return c.RDB.HDel(ctx, bConst.CacheUserEntity.Get(key).String(), fields...).Err()
}

// Delete 删除用户缓存数据
func (c *UserCache) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("用户标识为空")
	}
	return c.RDB.Del(ctx, bConst.CacheUserEntity.Get(key).String()).Err()
}

func buildUserFields(user *entity.User) map[string]any {
	fields := map[string]any{
		userCacheFieldName: user.Username,
	}
	if !user.ID.IsZero() {
		fields[userCacheFieldID] = user.ID.String()
	}
	if !user.UpdatedAt.IsZero() {
		fields[userCacheFieldUpdated] = user.UpdatedAt.Format(userCacheTimeLayout)
	}
	return fields
}

func parseUserFields(fields map[string]string) (*entity.User, error) {
	user := &entity.User{}
	if value, ok := fields[userCacheFieldID]; ok && value != "" {
		id, err := xSnowflake.ParseSnowflakeID(value)
		if err != nil {
			return nil, fmt.Errorf("字段 %s 解析失败: %w", userCacheFieldID, err)
		}
		user.ID = id
	}
	if value, ok := fields[userCacheFieldUpdated]; ok && value != "" {
		parsed, err := time.Parse(userCacheTimeLayout, value)
		if err != nil {
			return nil, fmt.Errorf("字段 %s 解析失败: %w", userCacheFieldUpdated, err)
		}
		user.UpdatedAt = parsed
	}
	if value, ok := fields[userCacheFieldName]; ok {
		user.Username = value
	}
	return user, nil
}
