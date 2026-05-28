package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
)

type ConfigRepository struct {
	log *xLog.LogNamedLogger
}

func NewConfigRepository() *ConfigRepository {
	return &ConfigRepository{
		log: xLog.WithName(xLog.NamedREPO, "ConfigRepository"),
	}
}

// GetByKey 根据命名空间和键获取单个配置值
func (r *ConfigRepository) GetByKey(ctx context.Context, namespace, key string) (string, *xError.Error) {
	r.log.Info(ctx, "GetByKey - 查询配置")
	var config entity.Config
	db := xCtxUtil.MustGetDB(ctx)
	if result := db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).First(&config); result.Error != nil {
		return "", xError.NewError(ctx, xError.DatabaseError, "查询配置失败", false, result.Error)
	}
	return config.Value, nil
}

// BatchGet 根据命名空间批量获取所有配置
func (r *ConfigRepository) BatchGet(ctx context.Context, namespace string) (map[string]string, *xError.Error) {
	r.log.Info(ctx, "BatchGet - 批量查询配置")
	var configs []entity.Config
	db := xCtxUtil.MustGetDB(ctx)
	if result := db.WithContext(ctx).Where("namespace = ?", namespace).Find(&configs); result.Error != nil {
		return nil, xError.NewError(ctx, xError.DatabaseError, "批量查询配置失败", false, result.Error)
	}
	result := make(map[string]string, len(configs))
	for _, c := range configs {
		result[c.Key] = c.Value
	}
	return result, nil
}

// Set 保存配置（存在则更新，不存在则创建）
func (r *ConfigRepository) Set(ctx context.Context, namespace, key, value string) *xError.Error {
	r.log.Info(ctx, "Set - 保存配置")
	db := xCtxUtil.MustGetDB(ctx)
	var config entity.Config
	result := db.WithContext(ctx).Where("namespace = ? AND key = ?", namespace, key).First(&config)
	if result.Error == nil {
		// 存在则更新
		config.Value = value
		if result := db.WithContext(ctx).Save(&config); result.Error != nil {
			return xError.NewError(ctx, xError.DatabaseError, "更新配置失败", false, result.Error)
		}
	} else {
		// 不存在则创建
		config = entity.Config{Namespace: namespace, Key: key, Value: value}
		if result := db.WithContext(ctx).Create(&config); result.Error != nil {
			return xError.NewError(ctx, xError.DatabaseError, "创建配置失败", false, result.Error)
		}
	}
	return nil
}

// BatchSet 批量保存配置
func (r *ConfigRepository) BatchSet(ctx context.Context, namespace string, values map[string]string) *xError.Error {
	r.log.Info(ctx, "BatchSet - 批量保存配置")
	for key, value := range values {
		if xErr := r.Set(ctx, namespace, key, value); xErr != nil {
			return xErr
		}
	}
	return nil
}
