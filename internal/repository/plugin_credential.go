package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"gorm.io/gorm"
)

type PluginCredentialRepo struct {
	db  *gorm.DB
	log *xLog.LogNamedLogger
}

func NewPluginCredentialRepo(db *gorm.DB) *PluginCredentialRepo {
	return &PluginCredentialRepo{
		db:  db,
		log: xLog.WithName(xLog.NamedREPO, "PluginCredentialRepo"),
	}
}

func (r *PluginCredentialRepo) GetByName(ctx context.Context, name string) (*entity.PluginCredential, *xError.Error) {
	r.log.Info(ctx, "GetByName - 查询插件凭证")

	var cred entity.PluginCredential
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&cred).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(ctx, xError.ResourceNotFound, "插件凭证不存在", true, err)
		}
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询插件凭证失败", false, err)
	}
	return &cred, nil
}

func (r *PluginCredentialRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.PluginCredential, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询插件凭证")

	var cred entity.PluginCredential
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&cred).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(ctx, xError.ResourceNotFound, "插件凭证不存在", true, err)
		}
		return nil, xError.NewError(ctx, xError.DatabaseError, "查询插件凭证失败", false, err)
	}
	return &cred, nil
}

func (r *PluginCredentialRepo) Create(ctx context.Context, cred *entity.PluginCredential) *xError.Error {
	r.log.Info(ctx, "Create - 创建插件凭证")

	if err := r.db.WithContext(ctx).Create(cred).Error; err != nil {
		return xError.NewError(ctx, xError.DatabaseError, "创建插件凭证失败", false, err)
	}
	return nil
}
