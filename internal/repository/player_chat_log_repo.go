package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
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

// ListByPage 管理端分页查询聊天记录
func (r *PlayerChatLogRepo) ListByPage(
	ctx context.Context, page, pageSize int,
	playerUUID *uuid.UUID, serverName *string, source *uint8,
) ([]entity.PlayerChatLog, int64, *xError.Error) {
	r.log.Info(ctx, "ListByPage - 分页查询聊天记录")

	query := r.db.WithContext(ctx).Model(&entity.PlayerChatLog{})
	if playerUUID != nil {
		query = query.Where("player_uuid = ?", *playerUUID)
	}
	if serverName != nil && *serverName != "" {
		query = query.Where("server_name = ?", *serverName)
	}
	if source != nil {
		query = query.Where("source = ?", *source)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询聊天记录总数失败", false, err)
	}

	var logs []entity.PlayerChatLog
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询聊天记录失败", false, err)
	}

	return logs, total, nil
}

// ListRecent 查询最近 N 条聊天记录（SSE 初始化用）
// 返回结果按时间正序排列（oldest first），方便前端按时间线展示
func (r *PlayerChatLogRepo) ListRecent(ctx context.Context, limit int) ([]entity.PlayerChatLog, *xError.Error) {
	r.log.Info(ctx, "ListRecent - 查询最近聊天记录")
	var logs []entity.PlayerChatLog
	if err := r.db.WithContext(ctx).Order("id DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询最近聊天记录失败", false, err)
	}
	// 反转为时间正序
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}
	return logs, nil
}

// ListByPlayerUUIDs 按玩家 UUID 列表查询聊天记录（用户端）
func (r *PlayerChatLogRepo) ListByPlayerUUIDs(
	ctx context.Context, page, pageSize int, playerUUIDs []uuid.UUID,
) ([]entity.PlayerChatLog, int64, *xError.Error) {
	r.log.Info(ctx, "ListByPlayerUUIDs - 按玩家UUID列表查询聊天记录")

	// 无关联角色时直接返回空结果，避免跳过 WHERE 导致全库查询
	if len(playerUUIDs) == 0 {
		return []entity.PlayerChatLog{}, 0, nil
	}

	query := r.db.WithContext(ctx).Model(&entity.PlayerChatLog{}).Where("player_uuid IN ?", playerUUIDs)

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询聊天记录总数失败", false, err)
	}

	var logs []entity.PlayerChatLog
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询聊天记录失败", false, err)
	}

	return logs, total, nil
}

// ListByPlayerUUIDsOrUserID 按玩家 UUID 列表或用户 ID 查询聊天记录（用户端，包含 Web 发送的消息）
func (r *PlayerChatLogRepo) ListByPlayerUUIDsOrUserID(
	ctx context.Context, page, pageSize int, playerUUIDs []uuid.UUID, userID xSnowflake.SnowflakeID,
) ([]entity.PlayerChatLog, int64, *xError.Error) {
	r.log.Info(ctx, "ListByPlayerUUIDsOrUserID - 按玩家UUID或用户ID查询聊天记录")

	query := r.db.WithContext(ctx).Model(&entity.PlayerChatLog{})
	if len(playerUUIDs) > 0 {
		query = query.Where("player_uuid IN ? OR user_id = ?", playerUUIDs, userID)
	} else {
		query = query.Where("user_id = ?", userID)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询聊天记录总数失败", false, err)
	}

	var logs []entity.PlayerChatLog
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询聊天记录失败", false, err)
	}

	return logs, total, nil
}
