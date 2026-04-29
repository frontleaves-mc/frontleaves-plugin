package logic

import (
	"context"

	"github.com/google/uuid"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	constant "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type playerEventRepo struct {
	playerEvent *repository.PlayerEventRepo
}

type PlayerEventLogic struct {
	logic
	repo playerEventRepo
}

func NewPlayerEventLogic(ctx context.Context) *PlayerEventLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &PlayerEventLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "PlayerEventLogic"),
		},
		repo: playerEventRepo{
			playerEvent: repository.NewPlayerEventRepo(db),
		},
	}
}

func (l *PlayerEventLogic) RecordEvent(
	ctx context.Context,
	playerUUID uuid.UUID,
	playerName string,
	serverName string,
	worldName string,
	eventType constant.PlayerEventType,
	content string,
) error {
	l.log.Info(ctx, "RecordEvent - 记录玩家事件")
	event := &entity.PlayerEvent{
		PlayerUUID: playerUUID,
		PlayerName: playerName,
		ServerName: serverName,
		WorldName:  worldName,
		EventType:  eventType,
		Content:    content,
	}
	if xErr := l.repo.playerEvent.Create(ctx, event); xErr != nil {
		l.log.Warn(ctx, "记录玩家事件失败: "+xErr.Error())
		return xErr
	}
	return nil
}
