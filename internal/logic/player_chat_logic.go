package logic

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiMessage "github.com/frontleaves-mc/frontleaves-plugin/api/message"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
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

// RecordChat 记录聊天消息（游戏内来源）
func (l *PlayerChatLogic) RecordChat(
	ctx context.Context,
	playerUUID uuid.UUID,
	playerName string,
	serverName string,
	worldName string,
	message string,
	userID *xSnowflake.SnowflakeID,
) error {
	l.log.Info(ctx, "RecordChat - 记录聊天消息")
	chatLog := &entity.PlayerChatLog{
		PlayerUUID: &playerUUID,
		PlayerName: playerName,
		ServerName: serverName,
		WorldName:  worldName,
		Message:    message,
		Source:     1,
		UserID:     userID,
	}
	if xErr := l.repo.playerChatLog.Create(ctx, chatLog); xErr != nil {
		l.log.Warn(ctx, "记录聊天消息失败: "+xErr.Error())
		return xErr
	}
	return nil
}

// RecordWebChat 记录 Web 端发送的聊天消息
func (l *PlayerChatLogic) RecordWebChat(
	ctx context.Context,
	senderID xSnowflake.SnowflakeID,
	playerName string,
	playerUUID uuid.UUID,
	message string,
) (*entity.PlayerChatLog, error) {
	l.log.Info(ctx, "RecordWebChat - 记录Web聊天消息")
	chatLog := &entity.PlayerChatLog{
		PlayerName: playerName,
		PlayerUUID: &playerUUID,
		Message:    message,
		Source:     2,
		SenderID:   senderID,
		UserID:     &senderID,
	}
	if xErr := l.repo.playerChatLog.Create(ctx, chatLog); xErr != nil {
		l.log.Warn(ctx, "记录Web聊天消息失败: "+xErr.Error())
		return nil, xErr
	}
	return chatLog, nil
}

// ListChatHistory 管理端分页查询聊天记录
func (l *PlayerChatLogic) ListChatHistory(
	ctx context.Context, page, pageSize int,
	playerUUID *uuid.UUID, serverName *string, source *uint8,
) ([]apiMessage.ChatLogResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListChatHistory - 管理端查询聊天记录")
	logs, total, xErr := l.repo.playerChatLog.ListByPage(ctx, page, pageSize, playerUUID, serverName, source)
	if xErr != nil {
		return nil, 0, xErr
	}
	return l.toChatLogResponseList(logs), total, nil
}

// ListRecentChats 查询最近 N 条聊天记录（SSE 初始化用）
func (l *PlayerChatLogic) ListRecentChats(ctx context.Context, limit int) ([]apiMessage.ChatLogResponse, *xError.Error) {
	l.log.Info(ctx, "ListRecentChats - 查询最近聊天记录")
	logs, xErr := l.repo.playerChatLog.ListRecent(ctx, limit)
	if xErr != nil {
		return nil, xErr
	}
	return l.toChatLogResponseList(logs), nil
}

// ListMyChatHistory 用户端查询自己的聊天记录
func (l *PlayerChatLogic) ListMyChatHistory(
	ctx context.Context, page, pageSize int, playerUUIDs []uuid.UUID, userID xSnowflake.SnowflakeID,
) ([]apiMessage.ChatLogResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListMyChatHistory - 用户端查询聊天记录")
	logs, total, xErr := l.repo.playerChatLog.ListByPlayerUUIDsOrUserID(ctx, page, pageSize, playerUUIDs, userID)
	if xErr != nil {
		return nil, 0, xErr
	}
	return l.toChatLogResponseList(logs), total, nil
}

func (l *PlayerChatLogic) toChatLogResponseList(logs []entity.PlayerChatLog) []apiMessage.ChatLogResponse {
	result := make([]apiMessage.ChatLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, l.toChatLogResponse(log))
	}
	return result
}

func (l *PlayerChatLogic) toChatLogResponse(log entity.PlayerChatLog) apiMessage.ChatLogResponse {
	playerUUIDStr := ""
	if log.PlayerUUID != nil {
		playerUUIDStr = log.PlayerUUID.String()
	}
	resp := apiMessage.ChatLogResponse{
		ID:         log.ID.String(),
		PlayerUUID: playerUUIDStr,
		PlayerName: log.PlayerName,
		ServerName: log.ServerName,
		WorldName:  log.WorldName,
		Message:    log.Message,
		Source:     int(log.Source),
	}
	if log.SenderID != 0 {
		senderIDStr := log.SenderID.String()
		resp.SenderID = &senderIDStr
	}
	if log.UserID != nil && *log.UserID != 0 {
		userIDStr := log.UserID.String()
		resp.UserID = &userIDStr
	}
	return resp
}
