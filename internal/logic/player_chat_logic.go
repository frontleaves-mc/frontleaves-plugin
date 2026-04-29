package logic

import (
	"context"

	"github.com/google/uuid"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type playerChatLogRepo struct {
	playerChatLog *repository.PlayerChatLogRepo
}

type PlayerChatLogic struct {
	logic
	repo playerChatLogRepo
}

func NewPlayerChatLogic(ctx context.Context) *PlayerChatLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &PlayerChatLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "PlayerChatLogic"),
		},
		repo: playerChatLogRepo{
			playerChatLog: repository.NewPlayerChatLogRepo(db),
		},
	}
}

func (l *PlayerChatLogic) RecordChat(
	ctx context.Context,
	playerUUID uuid.UUID,
	playerName string,
	serverName string,
	worldName string,
	message string,
) error {
	l.log.Info(ctx, "RecordChat - 记录聊天消息")
	chatLog := &entity.PlayerChatLog{
		PlayerUUID: playerUUID,
		PlayerName: playerName,
		ServerName: serverName,
		WorldName:  worldName,
		Message:    message,
	}
	if xErr := l.repo.playerChatLog.Create(ctx, chatLog); xErr != nil {
		l.log.Warn(ctx, "记录聊天消息失败: "+xErr.Error())
		return xErr
	}
	return nil
}
