package bConst

import xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"

const (
	GeneUser xSnowflake.Gene = 31 // 用户

	Demo xSnowflake.Gene = 32

	// 称号系统 + 成就系统
	GeneTitle             xSnowflake.Gene = 33 // 称号定义
	GenePlayerTitle       xSnowflake.Gene = 34 // 玩家称号关联
	GeneAchievement       xSnowflake.Gene = 35 // 成就定义
	GenePlayerAchievement xSnowflake.Gene = 36 // 玩家成就
	GeneAchievementClaim  xSnowflake.Gene = 38 // 成就申领记录
	GenePluginCredential  xSnowflake.Gene = 39 // 插件授权凭证
	GenePlayerEvent       xSnowflake.Gene = 41 // 玩家事件
	GenePlayerChatLog     xSnowflake.Gene = 42 // 玩家聊天日志

	// 公告系统
	GeneAnnouncement xSnowflake.Gene = 43 // 公告

	// 通用配置系统
	GeneConfig xSnowflake.Gene = 44 // 通用键值对配置（复用原 GeneAnnouncementSchedulerConfig）

	// 服务器系统
	GeneServer        xSnowflake.Gene = 46 // 服务器
	GeneServerPlayer  xSnowflake.Gene = 47 // 服务器玩家
	GenePlayerCommand xSnowflake.Gene = 48 // 玩家指令日志
	GeneServerLoadLog xSnowflake.Gene = 49 // 服务器负载采样日志

	// Matrix 系统
	GeneMatrixPlayerStatistic xSnowflake.Gene = 50 // 玩家统计
	GeneMatrixPlayerWarning   xSnowflake.Gene = 51 // 玩家警告日志

	// 私信系统
	GenePlayerDirectMessage xSnowflake.Gene = 52 // 玩家私信

)
