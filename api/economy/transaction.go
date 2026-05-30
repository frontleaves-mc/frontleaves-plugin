package economy

// TransactionDTO 交易流水响应
type TransactionDTO struct {
	ID            int64  `json:"id"`
	PlayerUUID    string `json:"player_uuid"`
	PlayerName    string `json:"player_name"`
	Amount        int64  `json:"amount"`         // 单位：分
	AmountDisplay string `json:"amount_display"` // 格式化显示
	Type          int16  `json:"type"`
	TypeName      string `json:"type_name"` // "转账"/"管理员操作"
	Counterparty  string `json:"counterparty,omitempty"`
	Operator      string `json:"operator,omitempty"`
	Comment       string `json:"comment,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// TransactionListRequest 交易流水列表请求
type TransactionListRequest struct {
	PlayerUUID string `form:"player_uuid"` // 管理员查指定玩家时需要
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// TransactionListResponse 交易流水列表响应
type TransactionListResponse struct {
	List     []TransactionDTO `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}
