package repository

import (
	"context"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepo struct {
	db    *gorm.DB
	cache *cache.UserCache
	log   *xLog.LogNamedLogger
}

func NewUserRepo(db *gorm.DB, rdb *redis.Client) *UserRepo {
	return &UserRepo{
		db: db,
		cache: &cache.UserCache{
			RDB: rdb,
			TTL: time.Minute * 15,
		},
		log: xLog.WithName(xLog.NamedREPO, "UserRepo"),
	}
}

func (r *UserRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.User, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询用户")

	// Cache-Aside: 先查缓存
	cached, err := r.cache.GetAllStruct(ctx, id.String())
	if err != nil {
		r.log.Warn(ctx, "读取用户缓存失败: "+err.Error())
	}
	if cached != nil {
		return cached, nil
	}

	// 缓存未命中，查数据库
	var user entity.User
	if dbErr := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; dbErr != nil {
		if dbErr == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "用户不存在", false, dbErr)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询用户失败", false, dbErr)
	}

	// 回写缓存
	if cacheErr := r.cache.SetAllStruct(ctx, id.String(), &user); cacheErr != nil {
		r.log.Warn(ctx, "回写用户缓存失败: "+cacheErr.Error())
	}

	return &user, nil
}

func (r *UserRepo) Upsert(ctx context.Context, user *entity.User) *xError.Error {
	r.log.Info(ctx, "Upsert - 创建或更新用户")

	// Write-Through: 先写数据库
	if dbErr := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"username"}),
	}).Create(user).Error; dbErr != nil {
		return xError.NewError(nil, xError.DatabaseError, "同步用户失败", false, dbErr)
	}

	// 同步更新缓存
	if cacheErr := r.cache.SetAllStruct(ctx, user.ID.String(), user); cacheErr != nil {
		r.log.Warn(ctx, "同步用户缓存失败: "+cacheErr.Error())
	}

	return nil
}

// DeleteCache 主动删除用户缓存（供 Logic 层在需要时调用）
func (r *UserRepo) DeleteCache(ctx context.Context, id string) error {
	return r.cache.Delete(ctx, id)
}
