package entity

import (
	"time"

	"github.com/google/uuid"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
)

// 交易类型常量
const (
	TransactionTypeTransfer int16 = 1 // 转账交易
	TransactionTypeAdmin    int16 = 2 // 管理员操作
)

// TransactionLog 交易流水实体，记录所有经济系统资金变动。
type TransactionLog struct {
	ID               xSnowflake.SnowflakeID `gorm:"primaryKey;autoIncrement:false;comment:交易流水ID" json:"id"`                      // 交易流水ID
	PlayerUUID       uuid.UUID              `gorm:"type:uuid;not null;index;comment:玩家UUID" json:"player_uuid"`                   // 玩家UUID
	PlayerName       string                 `gorm:"type:varchar(64);not null;comment:玩家用户名" json:"player_name"`                   // 玩家用户名
	Amount           int64                  `gorm:"not null;default:0;comment:交易金额(单位:分)" json:"amount"`                          // 交易金额(单位:分)
	Type             int16                  `gorm:"not null;default:0;index;comment:交易类型(1=转账,2=管理员)" json:"type"`                // 交易类型(1=转账,2=管理员)
	CounterpartyUUID *uuid.UUID             `gorm:"type:uuid;default:null;comment:对方UUID" json:"counterparty_uuid,omitempty"`     // 对方UUID
	CounterpartyName string                 `gorm:"type:varchar(64);default:'';comment:对方用户名" json:"counterparty_name,omitempty"` // 对方用户名
	OperatorUUID     *uuid.UUID             `gorm:"type:uuid;default:null;comment:操作者UUID" json:"operator_uuid,omitempty"`        // 操作者UUID
	OperatorName     string                 `gorm:"type:varchar(64);default:'';comment:操作者用户名" json:"operator_name,omitempty"`    // 操作者用户名
	Comment          string                 `gorm:"type:text;default:'';comment:备注" json:"comment,omitempty"`                      // 备注
	IdempotencyKey   string                 `gorm:"type:varchar(255);uniqueIndex;not null;comment:幂等键" json:"idempotency_key"`     // 幂等键
	CreatedAt        time.Time              `gorm:"autoCreateTime;not null;comment:创建时间" json:"created_at"`                       // 创建时间
}

// TableName 返回数据库表名。
func (TransactionLog) TableName() string {
	return "fp_transaction_logs"
}

// GetGene 返回 xSnowflake.Gene，用于标识该实体在 ID 生成时使用的基因类型。
func (_ *TransactionLog) GetGene() xSnowflake.Gene {
	return bConst.GeneTransactionLog
}
