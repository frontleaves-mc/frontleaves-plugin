package economy

// BalanceDTO 余额查询响应
type BalanceDTO struct {
	PlayerUUID    string `json:"player_uuid"`
	Balance       int64  `json:"balance"`         // 单位：分
	BalanceDisplay string `json:"balance_display"` // 格式化显示，如 "100.50"
	Currency      string `json:"currency"`
}

// AdjustBalanceRequest 管理员调整玩家余额请求
type AdjustBalanceRequest struct {
	PlayerUUID string  `json:"player_uuid" binding:"required"` // 玩家UUID
	Operation  string  `json:"operation" binding:"required"`   // 操作类型：add/remove/set/reset
	Amount     float64 `json:"amount"`                         // 操作金额（单位：元，reset时忽略）
}

// AdjustBalanceResponse 管理员调整玩家余额响应
type AdjustBalanceResponse struct {
	PlayerUUID        string `json:"player_uuid"`         // 玩家UUID
	NewBalance        int64  `json:"new_balance"`          // 操作后新余额（单位：分）
	NewBalanceDisplay string `json:"new_balance_display"` // 格式化显示
	Currency          string `json:"currency"`             // 货币类型
}
