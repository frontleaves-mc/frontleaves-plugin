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
	"github.com/gin-gonic/gin"
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
			gameProfile: repository.NewGameProfileRepo(db, rdb),
		},
	}
}

func (l *GameProfileLogic) GetPlayer(ctx *gin.Context, playerUUID uuid.UUID) (*apiGameProfile.GameProfileResponse, *xError.Error) {
	l.log.Info(ctx, "GetPlayer - 查询玩家信息")
	gp, xErr := l.repo.gameProfile.GetByUUID(ctx.Request.Context(), playerUUID)
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

func (l *GameProfileLogic) UpdatePlayerGroup(ctx *gin.Context, playerUUID uuid.UUID, username, groupName string) *xError.Error {
	l.log.Info(ctx, "UpdatePlayerGroup - 更新玩家权限组")
	gp := &entity.GameProfile{
		UUID:       playerUUID,
		Username:   username,
		GroupName:  groupName,
		ReportedAt: time.Now(),
	}
	if xErr := l.repo.gameProfile.CreateOrUpdate(ctx.Request.Context(), gp); xErr != nil {
		return xErr
	}
	return nil
}

func (l *GameProfileLogic) ListPlayers(ctx *gin.Context, page, pageSize int) ([]apiGameProfile.GameProfileResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListPlayers - 查询玩家列表")
	gps, total, xErr := l.repo.gameProfile.List(ctx.Request.Context(), page, pageSize)
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

func (l *GameProfileLogic) Upsert(ctx *gin.Context, userID xSnowflake.SnowflakeID, gpUUID uuid.UUID, name string) error {
	l.log.Info(ctx, "Upsert - 同步 GameProfile")
	gp := &entity.GameProfile{
		UserID:     userID,
		UUID:       gpUUID,
		Username:   name,
		GroupName:  "PLAYER",
		ReportedAt: time.Now(),
	}
	if xErr := l.repo.gameProfile.CreateOrUpdate(ctx.Request.Context(), gp); xErr != nil {
		l.log.Warn(ctx, "同步 GameProfile 失败: "+xErr.Error())
		return xErr
	}
	return nil
}
