package logic

import (
	"context"
	"fmt"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiDirectMessage "github.com/frontleaves-mc/frontleaves-plugin/api/direct_message"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/sse"
	"github.com/google/uuid"
)

// DirectMessageLogic 私信业务编排层
type DirectMessageLogic struct {
	logic
	repo         *repository.DirectMessageRepo
	dmEmailTimer *DmEmailTimer
}

// NewDirectMessageLogic 创建私信业务逻辑实例
func NewDirectMessageLogic(ctx context.Context) *DirectMessageLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &DirectMessageLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "DirectMessageLogic"),
		},
		repo:         repository.NewDirectMessageRepo(ctx),
		dmEmailTimer: NewDmEmailTimer(),
	}
}

// RecordDirectMessage 记录游戏内私信消息
func (l *DirectMessageLogic) RecordDirectMessage(
	ctx context.Context,
	senderUserID string,
	senderName string,
	receiverName string,
	message string,
	source uint8,
) *xError.Error {
	l.log.Info(ctx, "RecordDirectMessage - 记录游戏内私信")

	// 解析发送者 UUID
	senderUUID, err := uuid.Parse(senderUserID)
	if err != nil {
		return xError.NewError(nil, xError.ParameterError, "无效的发送者 ID", true, err)
	}

	// 通过接收者用户名查找 GameProfile → 获取 UUID
	receiverUUID, err := l.findPlayerUUIDByName(ctx, receiverName)
	if err != nil {
		return xError.NewError(nil, xError.NotFound, "接收者不存在", false, err)
	}

	// 防止自己给自己发私信
	if senderUUID == receiverUUID {
		return xError.NewError(nil, xError.ParameterError, "不能给自己发送私信", true, nil)
	}

	// 构建私信实体
	dm := &entity.PlayerDirectMessage{
		ID:           xSnowflake.GenerateID(bConst.GenePlayerDirectMessage),
		SenderID:     senderUUID,
		ReceiverID:   receiverUUID,
		SenderName:   senderName,
		ReceiverName: receiverName,
		Message:      message,
		Source:       source,
		IsRead:       false,
	}

	// 持久化
	if xErr := l.repo.Create(ctx, dm); xErr != nil {
		l.log.Warn(ctx, "记录私信失败: "+xErr.Error())
		return xErr
	}

	// 构建 SSE 消息并推送
	sseMsg := apiDirectMessage.SSEDirectMessage{
		SenderID:     senderUUID.String(),
		SenderName:   senderName,
		ReceiverID:   receiverUUID.String(),
		ReceiverName: receiverName,
		Message:      message,
		Source:       source,
		IsRead:       false,
	}
	sse.SendDirectMessage(receiverUUID.String(), sseMsg)

	// 检查接收者是否 MC 在线
	online := l.isPlayerMCOnline(ctx, receiverUUID.String())
	if online {
		// 在线 → 推送消息到游戏
		_ = PushPrivateMessageToGame(ctx, senderName, senderUUID.String(), message)
	} else {
		// 离线 → 启动邮件定时器
		snapshot := dmMessageSnapshot{
			SenderName: senderName,
			Message:    message,
			SentAt:     time.Now(),
		}
		l.dmEmailTimer.Schedule(ctx, senderUUID, receiverUUID, senderName, snapshot)
	}

	return nil
}

// RecordDirectMessageFromWeb 记录 Web 端发起的私信消息
func (l *DirectMessageLogic) RecordDirectMessageFromWeb(
	ctx context.Context,
	senderUserID string,
	receiverUserID string,
	message string,
) *xError.Error {
	l.log.Info(ctx, "RecordDirectMessageFromWeb - 记录Web端私信")

	// 解析 UUID
	senderUUID, err := uuid.Parse(senderUserID)
	if err != nil {
		return xError.NewError(nil, xError.ParameterError, "无效的发送者 ID", true, err)
	}
	receiverUUID, err := uuid.Parse(receiverUserID)
	if err != nil {
		return xError.NewError(nil, xError.ParameterError, "无效的接收者 ID", true, err)
	}

	// 防止自己给自己发私信
	if senderUUID == receiverUUID {
		return xError.NewError(nil, xError.ParameterError, "不能给自己发送私信", true, nil)
	}

	// 查询发送者和接收者名称
	senderName, err := l.findPlayerNameByUUID(ctx, senderUUID)
	if err != nil {
		return xError.NewError(nil, xError.NotFound, "发送者不存在", false, err)
	}
	receiverName, err := l.findPlayerNameByUUID(ctx, receiverUUID)
	if err != nil {
		return xError.NewError(nil, xError.NotFound, "接收者不存在", false, err)
	}

	// 构建私信实体
	dm := &entity.PlayerDirectMessage{
		ID:           xSnowflake.GenerateID(bConst.GenePlayerDirectMessage),
		SenderID:     senderUUID,
		ReceiverID:   receiverUUID,
		SenderName:   senderName,
		ReceiverName: receiverName,
		Message:      message,
		Source:       2, // Web 来源
		IsRead:       false,
	}

	// 持久化
	if xErr := l.repo.Create(ctx, dm); xErr != nil {
		l.log.Warn(ctx, "记录Web私信失败: "+xErr.Error())
		return xErr
	}

	// 构建 SSE 消息并推送
	sseMsg := apiDirectMessage.SSEDirectMessage{
		SenderID:     senderUUID.String(),
		SenderName:   senderName,
		ReceiverID:   receiverUUID.String(),
		ReceiverName: receiverName,
		Message:      message,
		Source:       2,
		IsRead:       false,
	}
	sse.SendDirectMessage(receiverUUID.String(), sseMsg)

	// 检查接收者是否 MC 在线
	online := l.isPlayerMCOnline(ctx, receiverUUID.String())
	if online {
		_ = PushPrivateMessageToGame(ctx, senderName, senderUUID.String(), message)
	} else {
		snapshot := dmMessageSnapshot{
			SenderName: senderName,
			Message:    message,
			SentAt:     time.Now(),
		}
		l.dmEmailTimer.Schedule(ctx, senderUUID, receiverUUID, senderName, snapshot)
	}

	return nil
}

// ListDirectMessages 查询两个用户之间的私信对话
func (l *DirectMessageLogic) ListDirectMessages(
	ctx context.Context,
	userID string,
	targetUserID string,
	page, pageSize int,
) ([]apiDirectMessage.DirectMessageResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListDirectMessages - 查询私信对话")

	u1, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, xError.NewError(nil, xError.ParameterError, "无效的用户 ID", true, err)
	}
	u2, err := uuid.Parse(targetUserID)
	if err != nil {
		return nil, 0, xError.NewError(nil, xError.ParameterError, "无效的目标用户 ID", true, err)
	}

	messages, total, xErr := l.repo.ListByConversation(ctx, u1, u2, page, pageSize)
	if xErr != nil {
		return nil, 0, xErr
	}

	result := make([]apiDirectMessage.DirectMessageResponse, 0, len(messages))
	for _, msg := range messages {
		result = append(result, l.toDirectMessageResponse(msg))
	}
	return result, total, nil
}

// ListConversations 查询用户的所有会话列表
func (l *DirectMessageLogic) ListConversations(
	ctx context.Context,
	userID string,
	page, pageSize int,
) ([]apiDirectMessage.ConversationResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListConversations - 查询会话列表")

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, xError.NewError(nil, xError.ParameterError, "无效的用户 ID", true, err)
	}

	conversations, total, xErr := l.repo.ListConversations(ctx, uid, page, pageSize)
	if xErr != nil {
		return nil, 0, xErr
	}

	result := make([]apiDirectMessage.ConversationResponse, 0, len(conversations))
	for _, conv := range conversations {
		result = append(result, apiDirectMessage.ConversationResponse{
			UserID:        conv.PartnerID.String(),
			UserName:      conv.PartnerName,
			LastMessage:   conv.LastMessage,
			LastMessageAt: conv.LastMsgAt,
			UnreadCount:   conv.UnreadCount,
		})
	}
	return result, total, nil
}

// GetUnreadCount 获取用户的未读消息统计
func (l *DirectMessageLogic) GetUnreadCount(
	ctx context.Context,
	userID string,
) (*apiDirectMessage.UnreadCountResponse, *xError.Error) {
	l.log.Info(ctx, "GetUnreadCount - 查询未读消息统计")

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, xError.NewError(nil, xError.ParameterError, "无效的用户 ID", true, err)
	}

	unreadList, xErr := l.repo.GetUnreadCount(ctx, uid)
	if xErr != nil {
		return nil, xErr
	}

	var totalCount int
	details := make([]apiDirectMessage.UnreadCountByUser, 0, len(unreadList))
	for _, item := range unreadList {
		totalCount += int(item.Count)
		details = append(details, apiDirectMessage.UnreadCountByUser{
			UserID:   item.SenderID.String(),
			UserName: item.SenderName,
			Count:    item.Count,
		})
	}

	return &apiDirectMessage.UnreadCountResponse{
		Total:  totalCount,
		Detail: details,
	}, nil
}

// MarkAsRead 标记指定发送者的未读消息为已读
func (l *DirectMessageLogic) MarkAsRead(
	ctx context.Context,
	receiverID string,
	senderID string,
) *xError.Error {
	l.log.Info(ctx, "MarkAsRead - 标记私信已读")

	rID, err := uuid.Parse(receiverID)
	if err != nil {
		return xError.NewError(nil, xError.ParameterError, "无效的接收者 ID", true, err)
	}
	sID, err := uuid.Parse(senderID)
	if err != nil {
		return xError.NewError(nil, xError.ParameterError, "无效的发送者 ID", true, err)
	}

	return l.repo.MarkAsRead(ctx, rID, sID)
}

// ListAllForAdmin 管理端分页查询所有私信
func (l *DirectMessageLogic) ListAllForAdmin(
	ctx context.Context,
	page, pageSize int,
	senderName, receiverName string,
) ([]apiDirectMessage.DirectMessageResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListAllForAdmin - 管理端查询私信")

	messages, total, xErr := l.repo.ListAllForAdmin(ctx, page, pageSize, senderName, receiverName)
	if xErr != nil {
		return nil, 0, xErr
	}

	result := make([]apiDirectMessage.DirectMessageResponse, 0, len(messages))
	for _, msg := range messages {
		result = append(result, l.toDirectMessageResponse(msg))
	}
	return result, total, nil
}

// --- 内部辅助方法 ---

// toDirectMessageResponse 将实体映射为响应 DTO
func (l *DirectMessageLogic) toDirectMessageResponse(msg entity.PlayerDirectMessage) apiDirectMessage.DirectMessageResponse {
	return apiDirectMessage.DirectMessageResponse{
		ID:           msg.ID.Int64(),
		SenderID:     msg.SenderID.String(),
		SenderName:   msg.SenderName,
		ReceiverID:   msg.ReceiverID.String(),
		ReceiverName: msg.ReceiverName,
		Message:      msg.Message,
		Source:       msg.Source,
		IsRead:       msg.IsRead,
		ReadAt:       msg.ReadAt,
	}
}

// findPlayerUUIDByName 通过用户名查找玩家的 UUID
func (l *DirectMessageLogic) findPlayerUUIDByName(ctx context.Context, name string) (uuid.UUID, error) {
	var gp entity.GameProfile
	if err := l.db.WithContext(ctx).Where("username = ?", name).First(&gp).Error; err != nil {
		return uuid.Nil, fmt.Errorf("用户名 %s 不存在: %w", name, err)
	}
	return gp.UUID, nil
}

// findPlayerNameByUUID 通过 UUID 查找玩家用户名
func (l *DirectMessageLogic) findPlayerNameByUUID(ctx context.Context, id uuid.UUID) (string, error) {
	var gp entity.GameProfile
	if err := l.db.WithContext(ctx).Where("uuid = ?", id).First(&gp).Error; err != nil {
		return "", fmt.Errorf("UUID %s 对应的玩家不存在: %w", id, err)
	}
	return gp.Username, nil
}

// isPlayerMCOnline 检查玩家是否在 MC 服务器在线（通过 Redis 缓存判断）
func (l *DirectMessageLogic) isPlayerMCOnline(ctx context.Context, playerUUID string) bool {
	playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
	playerData, err := l.rdb.HGetAll(ctx, playerKey).Result()
	if err != nil {
		return false
	}
	if len(playerData) == 0 {
		return false
	}
	return playerData["online"] == "true"
}
