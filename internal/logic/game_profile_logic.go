package logic

import (
	"context"
	"time"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	apiGameProfile "github.com/frontleaves-mc/frontleaves-plugin/api/game_profile"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type gameProfileRepo struct {
	gameProfile *repository.GameProfileRepo
}

type GameProfileLogic struct {
	logic
	repo gameProfileRepo
}

func NewGameProfileLogic(ctx context.Context) *GameProfileLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)
	return &GameProfileLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "GameProfileLogic"),
		},
		repo: gameProfileRepo{
			gameProfile: repository.NewGameProfileRepo(db),
		},
	}
}

func (l *GameProfileLogic) GetPlayer(ctx context.Context, playerUUID uuid.UUID) (*apiGameProfile.GameProfileResponse, *xError.Error) {
	l.log.Info(ctx, "GetPlayer - 查询玩家信息")
	gp, xErr := l.repo.gameProfile.GetByUUID(ctx, playerUUID)
	if xErr != nil {
		return nil, xErr
	}
	return &apiGameProfile.GameProfileResponse{
		UUID:       gp.UUID.String(),
		Username:   gp.Username,
		GroupName:  gp.GroupName,
		ReportedAt: gp.ReportedAt,
	}, nil
}

func (l *GameProfileLogic) ListPlayers(ctx context.Context, page, pageSize int) ([]apiGameProfile.GameProfileResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListPlayers - 查询玩家列表")
	gps, total, xErr := l.repo.gameProfile.List(ctx, page, pageSize)
	if xErr != nil {
		return nil, 0, xErr
	}
	var resp []apiGameProfile.GameProfileResponse
	for _, p := range gps {
		resp = append(resp, apiGameProfile.GameProfileResponse{
			UUID:       p.UUID.String(),
			Username:   p.Username,
			GroupName:  p.GroupName,
			ReportedAt: p.ReportedAt,
		})
	}
	return resp, total, nil
}

func (l *GameProfileLogic) Upsert(ctx context.Context, userID xSnowflake.SnowflakeID, gpUUID uuid.UUID, name string, groupName string) error {
	l.log.Info(ctx, "Upsert - 同步 GameProfile")
	gp := &entity.GameProfile{
		UserID:     userID,
		UUID:       gpUUID,
		Username:   name,
		GroupName:  groupName,
		ReportedAt: time.Now(),
	}
	if xErr := l.repo.gameProfile.CreateOrUpdate(ctx, gp); xErr != nil {
		l.log.Warn(ctx, "同步 GameProfile 失败: "+xErr.Error())
		return xErr
	}
	return nil
}

func (l *GameProfileLogic) UpdateGroupName(ctx context.Context, playerUUID uuid.UUID, groupName string) error {
	l.log.Info(ctx, "UpdateGroupName - 更新权限组: "+groupName)

	gp, xErr := l.repo.gameProfile.GetByUUID(ctx, playerUUID)
	if xErr != nil {
		return xErr
	}

	gp.GroupName = groupName
	gp.ReportedAt = time.Now()

	if xErr := l.repo.gameProfile.CreateOrUpdate(ctx, gp); xErr != nil {
		return xErr
	}
	return nil
}
