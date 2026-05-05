package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type AnnouncementScheduleItemRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewAnnouncementScheduleItemRepo(db *gorm.DB) *AnnouncementScheduleItemRepo {
	return &AnnouncementScheduleItemRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "AnnouncementScheduleItemRepo"),
	}
}

func (r *AnnouncementScheduleItemRepo) CreateItems(ctx context.Context, items []entity.AnnouncementScheduleItem) *xError.Error {
	r.log.Info(ctx, "CreateItems - 批量创建调度项")
	if err := r.db.WithContext(ctx).Create(&items).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "批量创建调度项失败", false, err)
	}
	return nil
}

func (r *AnnouncementScheduleItemRepo) DeleteByScheduleID(ctx context.Context, scheduleID xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "DeleteByScheduleID - 按调度ID删除调度项")
	if err := r.db.WithContext(ctx).Where("schedule_id = ?", scheduleID).Delete(&entity.AnnouncementScheduleItem{}).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "按调度ID删除调度项失败", false, err)
	}
	return nil
}

func (r *AnnouncementScheduleItemRepo) GetByScheduleID(ctx context.Context, scheduleID xSnowflake.SnowflakeID) ([]entity.AnnouncementScheduleItem, *xError.Error) {
	r.log.Info(ctx, "GetByScheduleID - 按调度ID查询调度项")
	var items []entity.AnnouncementScheduleItem
	if err := r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Order("sort_order ASC").
		Find(&items).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "按调度ID查询调度项失败", false, err)
	}
	return items, nil
}

func (r *AnnouncementScheduleItemRepo) GetByScheduleIDWithAnnouncement(ctx context.Context, scheduleID xSnowflake.SnowflakeID) ([]entity.AnnouncementScheduleItem, *xError.Error) {
	r.log.Info(ctx, "GetByScheduleIDWithAnnouncement - 按调度ID查询调度项（含公告信息）")
	var items []entity.AnnouncementScheduleItem
	if err := r.db.WithContext(ctx).
		Where("schedule_id = ?", scheduleID).
		Order("sort_order ASC").
		Find(&items).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "按调度ID查询调度项失败", false, err)
	}
	return items, nil
}
