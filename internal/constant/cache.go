package bConst

import (
	"fmt"

	xEnv "github.com/bamboo-services/bamboo-base-go/defined/env"
)

type RedisKey string

const (
	CacheAuthUser          RedisKey = "auth:user:%s"                // 认证缓存（md5 token）
	CacheUserEntity        RedisKey = "user:entity:%s"              // 用户实体缓存（snowflake ID）
	CacheStatusPlayer      RedisKey = "status:player:%s"            // 玩家在线状态
	CacheStatusServer      RedisKey = "status:server:%s"            // 服务器状态
	CacheStatusServerPlayers RedisKey = "status:server:%s:players"  // 服务器在线玩家集合
)

func (k RedisKey) Get(args ...interface{}) RedisKey {
	prefix := xEnv.GetEnvString(xEnv.NoSqlPrefix, "tpl:")
	return RedisKey(fmt.Sprintf(prefix+string(k), args...))
}

func (k RedisKey) String() string {
	return string(k)
}
