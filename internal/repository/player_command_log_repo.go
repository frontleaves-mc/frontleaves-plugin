package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerCommandLogRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewPlayerCommandLogRepo(db *gorm.DB) *PlayerCommandLogRepo {
	return &PlayerCommandLogRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "PlayerCommandLogRepo"),
	}
}

func (r *PlayerCommandLogRepo) Create(ctx context.Context, cmdLog *entity.PlayerCommandLog) *xError.Error {
	r.log.Info(ctx, "Create - 创建指令日志")
	if err := r.db.WithContext(ctx).Create(cmdLog).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建指令日志失败", false, err)
	}
	return nil
}

// ListByPage 管理端分页查询指令日志
func (r *PlayerCommandLogRepo) ListByPage(
	ctx context.Context, page, pageSize int,
	playerUUID *uuid.UUID, serverName *string,
) ([]entity.PlayerCommandLog, int64, *xError.Error) {
	r.log.Info(ctx, "ListByPage - 分页查询指令日志")

	query := r.db.WithContext(ctx).Model(&entity.PlayerCommandLog{})
	if playerUUID != nil {
		query = query.Where("player_uuid = ?", *playerUUID)
	}
	if serverName != nil && *serverName != "" {
		query = query.Where("server_name = ?", *serverName)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询指令日志总数失败", false, err)
	}

	var logs []entity.PlayerCommandLog
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询指令日志失败", false, err)
	}

	return logs, total, nil
}

// ListByPlayerUUIDs 按玩家 UUID 列表查询指令日志（用户端）
func (r *PlayerCommandLogRepo) ListByPlayerUUIDs(
	ctx context.Context, page, pageSize int, playerUUIDs []uuid.UUID,
) ([]entity.PlayerCommandLog, int64, *xError.Error) {
	r.log.Info(ctx, "ListByPlayerUUIDs - 按玩家UUID列表查询指令日志")

	// 无关联角色时直接返回空结果，避免跳过 WHERE 导致全库查询
	if len(playerUUIDs) == 0 {
		return []entity.PlayerCommandLog{}, 0, nil
	}

	query := r.db.WithContext(ctx).Model(&entity.PlayerCommandLog{}).Where("player_uuid IN ?", playerUUIDs)

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询指令日志总数失败", false, err)
	}

	var logs []entity.PlayerCommandLog
	offset := (page - 1) * pageSize
	if err := query.Session(&gorm.Session{}).Offset(offset).Limit(pageSize).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询指令日志失败", false, err)
	}

	return logs, total, nil
}
