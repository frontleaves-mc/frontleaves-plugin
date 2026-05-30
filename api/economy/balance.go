package economy

// BalanceDTO 余额查询响应
type BalanceDTO struct {
	PlayerUUID    string `json:"player_uuid"`
	Balance       int64  `json:"balance"`         // 单位：分
	BalanceDisplay string `json:"balance_display"` // 格式化显示，如 "100.50"
	Currency      string `json:"currency"`
}
