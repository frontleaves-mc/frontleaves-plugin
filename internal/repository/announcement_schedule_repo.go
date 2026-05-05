package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type AnnouncementScheduleRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewAnnouncementScheduleRepo(db *gorm.DB) *AnnouncementScheduleRepo {
	return &AnnouncementScheduleRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "AnnouncementScheduleRepo"),
	}
}

func (r *AnnouncementScheduleRepo) Create(ctx context.Context, schedule *entity.AnnouncementSchedule) *xError.Error {
	r.log.Info(ctx, "Create - 创建公告调度")
	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建公告调度失败", false, err)
	}
	return nil
}

func (r *AnnouncementScheduleRepo) Update(ctx context.Context, schedule *entity.AnnouncementSchedule) *xError.Error {
	r.log.Info(ctx, "Update - 更新公告调度")
	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新公告调度失败", false, err)
	}
	return nil
}

func (r *AnnouncementScheduleRepo) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除公告调度")
	if err := r.db.WithContext(ctx).Delete(&entity.AnnouncementSchedule{}, id).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除公告调度失败", false, err)
	}
	return nil
}

func (r *AnnouncementScheduleRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.AnnouncementSchedule, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询公告调度")
	var schedule entity.AnnouncementSchedule
	if err := r.db.WithContext(ctx).First(&schedule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "公告调度不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询公告调度失败", false, err)
	}
	return &schedule, nil
}

func (r *AnnouncementScheduleRepo) List(ctx context.Context, page, pageSize int) ([]entity.AnnouncementSchedule, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询公告调度列表")

	query := r.db.WithContext(ctx).Model(&entity.AnnouncementSchedule{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询公告调度总数失败", false, err)
	}

	var schedules []entity.AnnouncementSchedule
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&schedules).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询公告调度列表失败", false, err)
	}
	return schedules, total, nil
}

func (r *AnnouncementScheduleRepo) GetActiveSchedule(ctx context.Context) (*entity.AnnouncementSchedule, *xError.Error) {
	r.log.Info(ctx, "GetActiveSchedule - 查询活动调度")
	var schedule entity.AnnouncementSchedule
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).First(&schedule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询活动调度失败", false, err)
	}
	return &schedule, nil
}

func (r *AnnouncementScheduleRepo) SetActiveSchedule(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "SetActiveSchedule - 设置活动调度")
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.AnnouncementSchedule{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			return err
		}
		if err := tx.Model(&entity.AnnouncementSchedule{}).Where("id = ?", id).Update("is_active", true).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return xError.NewError(nil, xError.DatabaseError, "设置活动调度失败", false, err)
	}
	return nil
}
