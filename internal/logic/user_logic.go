package logic

import (
	"context"

	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xModels "github.com/bamboo-services/bamboo-base-go/major/models"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type userRepo struct {
	user *repository.UserRepo
}

type UserLogic struct {
	logic
	repo userRepo
}

func NewUserLogic(ctx context.Context) *UserLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &UserLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "UserLogic"),
		},
		repo: userRepo{
			user: repository.NewUserRepo(db, rdb),
		},
	}
}

func (l *UserLogic) Upsert(ctx context.Context, userID xSnowflake.SnowflakeID, username string) error {
	l.log.Info(ctx, "Upsert - 同步用户信息")
	user := &entity.User{
		BaseEntity: xModels.BaseEntity{ID: userID},
		Username:   username,
	}
	if xErr := l.repo.user.Upsert(ctx, user); xErr != nil {
		l.log.Warn(ctx, "同步用户失败: "+xErr.Error())
		return xErr
	}
	return nil
}
