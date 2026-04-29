package apiServerStatus

// ServerStatusResponse 单个服务器状态响应
type ServerStatusResponse struct {
	ServerName    string             `json:"server_name"`
	Online        bool               `json:"online"`
	OnlinePlayers int32              `json:"online_players"`
	Tps           float64            `json:"tps"`
	LastHeartbeat int64              `json:"last_heartbeat"`
	Players       []PlayerStatusInfo `json:"players"`
}

// PlayerStatusInfo 在线玩家信息
type PlayerStatusInfo struct {
	PlayerUUID string `json:"player_uuid"`
	PlayerName string `json:"player_name"`
	WorldName  string `json:"world_name"`
}

// ServerStatusListResponse 所有服务器状态列表
type ServerStatusListResponse struct {
	Servers []ServerStatusResponse `json:"servers"`
}
