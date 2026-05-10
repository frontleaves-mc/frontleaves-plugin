package entity

import (
	"time"

	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/google/uuid"
)

type ServerPlayer struct {
	xModels.BaseEntity
	ServerID   xSnowflake.SnowflakeID `gorm:"not null;index;uniqueIndex:uk_server_player;comment:所属服务器ID"`
	Server     *Server                `gorm:"foreignKey:ServerID;references:ID" json:"-"`
	PlayerUUID uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:uk_server_player;comment:玩家UUID"`
	PlayerName string                 `gorm:"not null;type:varchar(64);comment:玩家名称"`
	WorldName  string                 `gorm:"not null;type:varchar(64);comment:所在世界"`
	Online     bool                   `gorm:"not null;default:false;comment:是否在线"`
	LastSeen   time.Time              `gorm:"not null;type:timestamptz;comment:最后在线时间"`
}

func (_ *ServerPlayer) GetGene() xSnowflake.Gene {
	return bConst.GeneServerPlayer
}
