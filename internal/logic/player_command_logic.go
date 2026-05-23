package logic

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiMessage "github.com/frontleaves-mc/frontleaves-plugin/api/message"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
)

type playerCommandLogRepo struct {
	playerCommandLog *repository.PlayerCommandLogRepo
}

type PlayerCommandLogic struct {
	logic
	repo playerCommandLogRepo
}

func NewPlayerCommandLogic(ctx context.Context) *PlayerCommandLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &PlayerCommandLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "PlayerCommandLogic"),
		},
		repo: playerCommandLogRepo{
			playerCommandLog: repository.NewPlayerCommandLogRepo(db),
		},
	}
}

// RecordCommand 记录玩家指令日志
func (l *PlayerCommandLogic) RecordCommand(
	ctx context.Context,
	playerUUID uuid.UUID,
	playerName string,
	serverName string,
	worldName string,
	command string,
) error {
	l.log.Info(ctx, "RecordCommand - 记录玩家指令")
	cmdLog := &entity.PlayerCommandLog{
		PlayerUUID: playerUUID,
		PlayerName: playerName,
		ServerName: serverName,
		WorldName:  worldName,
		Command:    command,
	}
	if xErr := l.repo.playerCommandLog.Create(ctx, cmdLog); xErr != nil {
		l.log.Warn(ctx, "记录玩家指令失败: "+xErr.Error())
		return xErr
	}
	return nil
}

// ListCommandHistory 管理端分页查询指令日志
func (l *PlayerCommandLogic) ListCommandHistory(
	ctx context.Context, page, pageSize int,
	playerUUID *uuid.UUID, serverName *string,
) ([]apiMessage.CommandLogResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListCommandHistory - 管理端查询指令日志")
	logs, total, xErr := l.repo.playerCommandLog.ListByPage(ctx, page, pageSize, playerUUID, serverName)
	if xErr != nil {
		return nil, 0, xErr
	}
	return l.toCommandLogResponseList(logs), total, nil
}

// ListMyCommandHistory 用户端查询自己的指令记录
func (l *PlayerCommandLogic) ListMyCommandHistory(
	ctx context.Context, page, pageSize int, playerUUIDs []uuid.UUID,
) ([]apiMessage.CommandLogResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListMyCommandHistory - 用户端查询指令记录")
	logs, total, xErr := l.repo.playerCommandLog.ListByPlayerUUIDs(ctx, page, pageSize, playerUUIDs)
	if xErr != nil {
		return nil, 0, xErr
	}
	return l.toCommandLogResponseList(logs), total, nil
}

func (l *PlayerCommandLogic) toCommandLogResponseList(logs []entity.PlayerCommandLog) []apiMessage.CommandLogResponse {
	result := make([]apiMessage.CommandLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, l.toCommandLogResponse(log))
	}
	return result
}

func (l *PlayerCommandLogic) toCommandLogResponse(log entity.PlayerCommandLog) apiMessage.CommandLogResponse {
	return apiMessage.CommandLogResponse{
		ID:         log.ID.String(),
		PlayerUUID: log.PlayerUUID.String(),
		PlayerName: log.PlayerName,
		ServerName: log.ServerName,
		WorldName:  log.WorldName,
		Command:    log.Command,
	}
}
