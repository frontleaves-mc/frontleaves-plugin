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

func (r *PluginCredentialRepo) List(ctx context.Context, page, pageSize int) ([]entity.PluginCredential, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询插件凭证列表")

	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.PluginCredential{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询插件凭证总数失败", false, err)
	}

	var creds []entity.PluginCredential
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&creds).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询插件凭证列表失败", false, err)
	}
	return creds, total, nil
}

func (r *PluginCredentialRepo) Update(ctx context.Context, id xSnowflake.SnowflakeID, updates map[string]interface{}) *xError.Error {
	r.log.Info(ctx, "Update - 更新插件凭证")
	if err := r.db.WithContext(ctx).Model(&entity.PluginCredential{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新插件凭证失败", false, err)
	}
	return nil
}

func (r *PluginCredentialRepo) UpdateSecretKey(ctx context.Context, id xSnowflake.SnowflakeID, newKey string) *xError.Error {
	r.log.Info(ctx, "UpdateSecretKey - 更新插件密钥")
	if err := r.db.WithContext(ctx).Model(&entity.PluginCredential{}).Where("id = ?", id).Update("secret_key", newKey).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新插件密钥失败", false, err)
	}
	return nil
}

func (r *PluginCredentialRepo) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除插件凭证")
	if err := r.db.WithContext(ctx).Delete(&entity.PluginCredential{}, id).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除插件凭证失败", false, err)
	}
	return nil
}
