package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type AnnouncementRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewAnnouncementRepo(db *gorm.DB) *AnnouncementRepo {
	return &AnnouncementRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "AnnouncementRepo"),
	}
}

func (r *AnnouncementRepo) Create(ctx context.Context, announcement *entity.Announcement) *xError.Error {
	r.log.Info(ctx, "Create - 创建公告")
	if err := r.db.WithContext(ctx).Create(announcement).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建公告失败", false, err)
	}
	return nil
}

func (r *AnnouncementRepo) Update(ctx context.Context, announcement *entity.Announcement) *xError.Error {
	r.log.Info(ctx, "Update - 更新公告")
	if err := r.db.WithContext(ctx).Save(announcement).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新公告失败", false, err)
	}
	return nil
}

func (r *AnnouncementRepo) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除公告")
	if err := r.db.WithContext(ctx).Delete(&entity.Announcement{}, id).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除公告失败", false, err)
	}
	return nil
}

func (r *AnnouncementRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.Announcement, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询公告")
	var announcement entity.Announcement
	if err := r.db.WithContext(ctx).First(&announcement, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "公告不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询公告失败", false, err)
	}
	return &announcement, nil
}

func (r *AnnouncementRepo) List(ctx context.Context, page, pageSize int, annType *int16, status *int16) ([]entity.Announcement, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询公告列表")

	query := r.db.WithContext(ctx).Model(&entity.Announcement{})
	if annType != nil {
		query = query.Where("type = ?", *annType)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询公告总数失败", false, err)
	}

	var announcements []entity.Announcement
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&announcements).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询公告列表失败", false, err)
	}
	return announcements, total, nil
}

func (r *AnnouncementRepo) GetPublishedGlobal(ctx context.Context) ([]entity.Announcement, *xError.Error) {
	r.log.Info(ctx, "GetPublishedGlobal - 查询已发布全局公告")
	var announcements []entity.Announcement
	if err := r.db.WithContext(ctx).
		Where("type = ? AND status = ?", entity.AnnouncementTypeGlobal, entity.AnnouncementStatusPublished).
		Find(&announcements).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询已发布全局公告失败", false, err)
	}
	return announcements, nil
}
