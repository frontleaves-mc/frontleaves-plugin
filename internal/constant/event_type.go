package bConst

type PlayerEventType uint

const (
	PlayerEventKick  PlayerEventType = 1 // 玩家被踢出
	PlayerEventDeath PlayerEventType = 2 // 玩家死亡
)
