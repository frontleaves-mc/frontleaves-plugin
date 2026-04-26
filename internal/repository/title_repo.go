package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type TitleRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewTitleRepo(db *gorm.DB) *TitleRepo {
	return &TitleRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "TitleRepo"),
	}
}

func (r *TitleRepo) Create(ctx context.Context, title *entity.Title) *xError.Error {
	r.log.Info(ctx, "Create - 创建称号")
	if err := r.db.WithContext(ctx).Create(title).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建称号失败", false, err)
	}
	return nil
}

func (r *TitleRepo) Update(ctx context.Context, title *entity.Title) *xError.Error {
	r.log.Info(ctx, "Update - 更新称号")
	if err := r.db.WithContext(ctx).Save(title).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新称号失败", false, err)
	}
	return nil
}

func (r *TitleRepo) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除称号")
	if err := r.db.WithContext(ctx).Delete(&entity.Title{}, id).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除称号失败", false, err)
	}
	return nil
}

func (r *TitleRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.Title, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询称号")
	var title entity.Title
	if err := r.db.WithContext(ctx).First(&title, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "称号不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询称号失败", false, err)
	}
	return &title, nil
}

func (r *TitleRepo) List(ctx context.Context, page, pageSize int, titleType *int16) ([]entity.Title, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询称号列表")

	query := r.db.WithContext(ctx).Model(&entity.Title{})
	if titleType != nil {
		query = query.Where("type = ?", *titleType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询称号总数失败", false, err)
	}

	var titles []entity.Title
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&titles).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询称号列表失败", false, err)
	}
	return titles, total, nil
}

func (r *TitleRepo) GetByPermissionGroup(ctx context.Context, groupName string) ([]entity.Title, *xError.Error) {
	r.log.Info(ctx, "GetByPermissionGroup - 按权限组查询称号")
	var titles []entity.Title
	if err := r.db.WithContext(ctx).Where("type = ? AND permission_group = ? AND is_active = ?", entity.TitleTypeGroup, groupName, true).Find(&titles).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "按权限组查询称号失败", false, err)
	}
	return titles, nil
}
