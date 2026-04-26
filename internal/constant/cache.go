package bConst

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

type RedisKey string

const (
	CacheAuthUser   RedisKey = "auth:user:%s"      // 认证缓存（md5 token）
	CacheUserEntity RedisKey = "user:entity:%s"    // 用户实体缓存（snowflake ID）
)

func (k RedisKey) Get(args ...interface{}) RedisKey {
	prefix := xEnv.GetEnvString(xEnv.NoSqlPrefix, "tpl:")
	return RedisKey(fmt.Sprintf(prefix+string(k), args...))
}

func (k RedisKey) String() string {
	return string(k)
}
