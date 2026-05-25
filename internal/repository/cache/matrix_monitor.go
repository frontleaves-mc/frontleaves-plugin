package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	xCache "github.com/bamboo-services/bamboo-base-go/major/cache"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/redis/go-redis/v9"
)

const matrixMonitorTTL = 5 * time.Minute

// MatrixMonitorCache Matrix 反作弊监控快照缓存管理器
type MatrixMonitorCache xCache.Cache

// Set 设置监控快照的多个字段
func (c *MatrixMonitorCache) Set(ctx context.Context, sessionKey string, fields map[string]any) error {
	if sessionKey == "" {
		return fmt.Errorf("sessionKey 为空")
	}
	if len(fields) == 0 {
		return nil
	}

	key := bConst.CacheMatrixPlayerMonitor.Get(sessionKey).String()
	if err := c.RDB.HSet(ctx, key, fields).Err(); err != nil {
		return err
	}
	return c.RDB.Expire(ctx, key, matrixMonitorTTL).Err()
}

// Get 获取监控快照的指定字段
func (c *MatrixMonitorCache) Get(ctx context.Context, sessionKey string, field string) (string, bool, error) {
	if sessionKey == "" {
		return "", false, fmt.Errorf("sessionKey 为空")
	}
	if field == "" {
		return "", false, fmt.Errorf("字段为空")
	}

	key := bConst.CacheMatrixPlayerMonitor.Get(sessionKey).String()
	value, err := c.RDB.HGet(ctx, key, field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", false, nil
		}
		return "", false, err
	}
	return value, true, nil
}

// GetAll 获取监控快照的所有字段
func (c *MatrixMonitorCache) GetAll(ctx context.Context, sessionKey string) (map[string]string, error) {
	if sessionKey == "" {
		return nil, fmt.Errorf("sessionKey 为空")
	}
	key := bConst.CacheMatrixPlayerMonitor.Get(sessionKey).String()
	return c.RDB.HGetAll(ctx, key).Result()
}

// Delete 删除监控快照
func (c *MatrixMonitorCache) Delete(ctx context.Context, sessionKey string) error {
	if sessionKey == "" {
		return fmt.Errorf("sessionKey 为空")
	}
	key := bConst.CacheMatrixPlayerMonitor.Get(sessionKey).String()
	return c.RDB.Del(ctx, key).Err()
}
