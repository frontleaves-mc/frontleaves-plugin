package bConst

import xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"

const (
	GeneUser xSnowflake.Gene = 31 // 用户

	Demo xSnowflake.Gene = 32

	// 称号系统 + 成就系统
	GeneTitle              xSnowflake.Gene = 33 // 称号定义
	GenePlayerTitle        xSnowflake.Gene = 34 // 玩家称号关联
	GeneAchievement        xSnowflake.Gene = 35 // 成就定义
	GenePlayerAchievement  xSnowflake.Gene = 36 // 玩家成就
	GeneAchievementClaim    xSnowflake.Gene = 38 // 成就申领记录
	GenePluginCredential    xSnowflake.Gene = 39 // 插件授权凭证
	GenePlayerEvent         xSnowflake.Gene = 41 // 玩家事件
	GenePlayerChatLog       xSnowflake.Gene = 42 // 玩家聊天日志
)
