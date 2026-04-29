package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type PlayerEventRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewPlayerEventRepo(db *gorm.DB) *PlayerEventRepo {
	return &PlayerEventRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "PlayerEventRepo"),
	}
}

func (r *PlayerEventRepo) Create(ctx context.Context, event *entity.PlayerEvent) *xError.Error {
	r.log.Info(ctx, "Create - 创建玩家事件")
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建玩家事件失败", false, err)
	}
	return nil
}
