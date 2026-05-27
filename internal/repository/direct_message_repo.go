package repository

import (
	"context"
	"time"

	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/google/uuid"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type DirectMessageRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

// Helper types for query results
type ConversationSummary struct {
	PartnerID   uuid.UUID
	PartnerName string
	LastMessage string
	LastMsgAt   time.Time
	UnreadCount int64
}

type UnreadByUser struct {
	SenderID   uuid.UUID
	SenderName string
	Count      int64
}

// NewDirectMessageRepo 创建私信仓库实例
func NewDirectMessageRepo(ctx context.Context) *DirectMessageRepo {
	return &DirectMessageRepo{
		db:  xCtxUtil.MustGetDB(ctx),
		log: xLog.WithName(xLog.NamedREPO, "DirectMessageRepo"),
	}
}

// Create 创建私信记录
func (r *DirectMessageRepo) Create(ctx context.Context, dm *entity.PlayerDirectMessage) *xError.Error {
	r.log.Info(ctx, "Create - 创建私信")
	if err := r.db.WithContext(ctx).Create(dm).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建私信失败", false, err)
	}
	return nil
}

// ListByConversation 查询两个用户之间的私信对话
func (r *DirectMessageRepo) ListByConversation(
	ctx context.Context,
	userID1, userID2 uuid.UUID,
	page, pageSize int,
) ([]entity.PlayerDirectMessage, int64, *xError.Error) {
	r.log.Info(ctx, "ListByConversation - 查询私信对话")

	query := r.db.WithContext(ctx).Model(&entity.PlayerDirectMessage{}).
		Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
			userID1, userID2, userID2, userID1)

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询私信总数失败", false, err)
	}

	var messages []entity.PlayerDirectMessage
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).
		Order("id DESC").Find(&messages).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询私信失败", false, err)
	}

	return messages, total, nil
}

// ListConversations 查询用户的所有会话（每个对话伙伴的最新消息 + 未读数）
func (r *DirectMessageRepo) ListConversations(
	ctx context.Context,
	userID uuid.UUID,
	page, pageSize int,
) ([]ConversationSummary, int64, *xError.Error) {
	r.log.Info(ctx, "ListConversations - 查询用户会话列表")

	// 查询总数
	totalQuery := `
		SELECT COUNT(DISTINCT CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END)
		FROM fp_player_direct_messages
		WHERE (sender_id = ? OR receiver_id = ?)
	`
	var total int64
	if err := r.db.WithContext(ctx).Raw(totalQuery, userID, userID, userID).Scan(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询会话总数失败", false, err)
	}

	// 使用子查询获取每个对话伙伴的最新消息和未读数
	subQuery := `
		SELECT
			dm1.partner_id,
			dm1.partner_name,
			dm1.last_message,
			dm1.last_msg_at,
			COALESCE(unread.unread_count, 0) AS unread_count
		FROM (
			SELECT
				CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END AS partner_id,
				CASE WHEN sender_id = ? THEN receiver_name ELSE sender_name END AS partner_name,
				message AS last_message,
				id AS last_msg_at,
				ROW_NUMBER() OVER (PARTITION BY CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END ORDER BY id DESC) AS rn
			FROM fp_player_direct_messages
			WHERE (sender_id = ? OR receiver_id = ?)
		) dm1
		LEFT JOIN (
			SELECT
				sender_id,
				COUNT(*) AS unread_count
			FROM fp_player_direct_messages
			WHERE receiver_id = ? AND is_read = false
			GROUP BY sender_id
		) unread ON dm1.partner_id = unread.sender_id
		WHERE dm1.rn = 1
		ORDER BY dm1.last_msg_at DESC
		LIMIT ? OFFSET ?
	`

	var conversations []ConversationSummary
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Raw(subQuery, userID, userID, userID, userID, userID, userID, pageSize, offset).Scan(&conversations).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询会话列表失败", false, err)
	}

	return conversations, total, nil
}

// GetUnreadCount 获取接收者的未读消息统计（按发送者分组）
func (r *DirectMessageRepo) GetUnreadCount(
	ctx context.Context,
	receiverID uuid.UUID,
) ([]UnreadByUser, *xError.Error) {
	r.log.Info(ctx, "GetUnreadCount - 查询未读消息统计")

	var unreadList []UnreadByUser
	query := r.db.WithContext(ctx).Model(&entity.PlayerDirectMessage{}).
		Select("sender_id, sender_name, COUNT(*) as count").
		Where("receiver_id = ? AND is_read = false", receiverID).
		Group("sender_id, sender_name")

	if err := query.Scan(&unreadList).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询未读消息统计失败", false, err)
	}

	return unreadList, nil
}

// MarkAsRead 标记指定发送者发给接收者的所有未读消息为已读
func (r *DirectMessageRepo) MarkAsRead(
	ctx context.Context,
	receiverID, senderID uuid.UUID,
) *xError.Error {
	r.log.Info(ctx, "MarkAsRead - 标记私信为已读")

	now := time.Now()
	result := r.db.WithContext(ctx).Model(&entity.PlayerDirectMessage{}).
		Where("receiver_id = ? AND sender_id = ? AND is_read = false", receiverID, senderID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		})

	if err := result.Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "标记私信已读失败", false, err)
	}

	return nil
}

// ListAllForAdmin 管理端分页查询所有私信（支持按发送者/接收者姓名筛选）
func (r *DirectMessageRepo) ListAllForAdmin(
	ctx context.Context,
	page, pageSize int,
	senderName, receiverName string,
) ([]entity.PlayerDirectMessage, int64, *xError.Error) {
	r.log.Info(ctx, "ListAllForAdmin - 管理端分页查询私信")

	query := r.db.WithContext(ctx).Model(&entity.PlayerDirectMessage{})
	if senderName != "" {
		query = query.Where("sender_name LIKE ?", "%"+senderName+"%")
	}
	if receiverName != "" {
		query = query.Where("receiver_name LIKE ?", "%"+receiverName+"%")
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询私信总数失败", false, err)
	}

	var messages []entity.PlayerDirectMessage
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).
		Order("id DESC").Find(&messages).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询私信失败", false, err)
	}

	return messages, total, nil
}