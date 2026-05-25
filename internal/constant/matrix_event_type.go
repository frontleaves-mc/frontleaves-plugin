package bConst

// MatrixEventType Matrix 遥测事件类型
type MatrixEventType uint

const (
	MatrixEventPlayerJoin      MatrixEventType = 1  // 玩家加入
	MatrixEventPlayerQuit      MatrixEventType = 2  // 玩家退出
	MatrixEventTelemetryTick   MatrixEventType = 3  // 心跳快照
	MatrixEventBlockBreak      MatrixEventType = 4  // 方块破坏
	MatrixEventBlockPlace      MatrixEventType = 5  // 方块放置
	MatrixEventEntityKill      MatrixEventType = 6  // 实体击杀
	MatrixEventEntityDamage    MatrixEventType = 7  // 实体伤害
	MatrixEventPlayerDamage    MatrixEventType = 8  // 玩家受伤
	MatrixEventPlayerDeath     MatrixEventType = 9  // 玩家死亡
	MatrixEventItemDrop        MatrixEventType = 10 // 物品丢弃
	MatrixEventItemPickup      MatrixEventType = 11 // 物品拾取
	MatrixEventInventoryAction MatrixEventType = 12 // 背包操作
	MatrixEventPlayerChat      MatrixEventType = 13 // 玩家聊天
	MatrixEventPlayerCommand   MatrixEventType = 14 // 玩家指令
	MatrixEventPlayerToggle    MatrixEventType = 15 // 玩家切换状态
	MatrixEventTeleport        MatrixEventType = 16 // 玩家传送
	MatrixEventRespawn         MatrixEventType = 17 // 玩家重生
	MatrixEventGameModeChange  MatrixEventType = 18 // 游戏模式变更
)
