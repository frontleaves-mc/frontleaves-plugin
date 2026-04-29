package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type PlayerChatLogRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewPlayerChatLogRepo(db *gorm.DB) *PlayerChatLogRepo {
	return &PlayerChatLogRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "PlayerChatLogRepo"),
	}
}

func (r *PlayerChatLogRepo) Create(ctx context.Context, chatLog *entity.PlayerChatLog) *xError.Error {
	r.log.Info(ctx, "Create - 创建聊天记录")
	if err := r.db.WithContext(ctx).Create(chatLog).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建聊天记录失败", false, err)
	}
	return nil
}
