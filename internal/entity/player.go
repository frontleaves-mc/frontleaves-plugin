package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Player struct {
	UUID       uuid.UUID `gorm:"type:uuid;primaryKey;comment:玩家UUID" json:"uuid"`
	Username   string    `gorm:"not null;type:varchar(64);comment:MC用户名" json:"username"`
	GroupName  string    `gorm:"not null;type:varchar(64);comment:当前权限组" json:"group_name"`
	ReportedAt time.Time `gorm:"not null;type:timestamptz;comment:最后上报时间" json:"reported_at"`
	CreatedAt  time.Time `gorm:"not null;type:timestamptz;autoCreateTime:milli;comment:创建时间" json:"-"`
	UpdatedAt  time.Time `gorm:"not null;type:timestamptz;autoUpdateTime:milli;comment:更新时间" json:"-"`
}

func (p *Player) BeforeCreate(_ *gorm.DB) error {
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *Player) BeforeUpdate(_ *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}
