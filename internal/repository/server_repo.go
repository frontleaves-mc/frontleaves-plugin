package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type ServerRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewServerRepo(db *gorm.DB) *ServerRepo {
	return &ServerRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "ServerRepo"),
	}
}

func (r *ServerRepo) Create(ctx context.Context, server *entity.Server) *xError.Error {
	r.log.Info(ctx, "Create - 创建服务器")
	if err := r.db.WithContext(ctx).Create(server).Error; err != nil {
		return xError.NewError(ctx, xError.DatabaseError, "创建服务器失败", false, err)
	}
	return nil
}

func (r *ServerRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.Server, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询服务器")
	var server entity.Server
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&server).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(ctx, xError.ResourceNotFound, "服务器不存在", true, err)
		}
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询服务器失败", false, err)
	}
	return &server, nil
}

func (r *ServerRepo) GetByName(ctx context.Context, name string) (*entity.Server, *xError.Error) {
	r.log.Info(ctx, "GetByName - 按名称查询服务器")
	var server entity.Server
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&server).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(ctx, xError.ResourceNotFound, "服务器不存在", true, err)
		}
		return nil, xError.NewError(ctx, xError.DatabaseError, "按名称查询服务器失败", false, err)
	}
	return &server, nil
}

func (r *ServerRepo) Update(ctx context.Context, server *entity.Server) *xError.Error {
	r.log.Info(ctx, "Update - 更新服务器")
	if err := r.db.WithContext(ctx).Save(server).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新服务器失败", false, err)
	}
	return nil
}

func (r *ServerRepo) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除服务器")
	if err := r.db.WithContext(ctx).Delete(&entity.Server{}, id).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除服务器失败", false, err)
	}
	return nil
}

func (r *ServerRepo) List(ctx context.Context, page, pageSize int) ([]entity.Server, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询服务器列表")
	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.Server{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询服务器总数失败", false, err)
	}
	var servers []entity.Server
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("sort_order ASC, created_at DESC").Find(&servers).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询服务器列表失败", false, err)
	}
	return servers, total, nil
}
