package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewUserRepo(db *gorm.DB, rdb *redis.Client) *UserRepo {
	return &UserRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "UserRepo"),
	}
}

func (r *UserRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.User, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询用户")
	var user entity.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "用户不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询用户失败", false, err)
	}
	return &user, nil
}

func (r *UserRepo) Upsert(ctx context.Context, user *entity.User) *xError.Error {
	r.log.Info(ctx, "Upsert - 创建或更新用户")
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"username"}),
	}).Create(user).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "同步用户失败", false, err)
	}
	return nil
}
