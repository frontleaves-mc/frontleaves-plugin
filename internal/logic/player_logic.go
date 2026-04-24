package logic

import (
	"context"
	"time"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiPlayer "github.com/frontleaves-mc/frontleaves-plugin/api/player"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
)

type playerRepo struct {
	player *repository.PlayerRepo
}

type PlayerLogic struct {
	logic
	repo playerRepo
}

func NewPlayerLogic(ctx context.Context) *PlayerLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &PlayerLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "PlayerLogic"),
		},
		repo: playerRepo{
			player: repository.NewPlayerRepo(db, rdb),
		},
	}
}

func (l *PlayerLogic) GetPlayer(ctx *gin.Context, playerUUID uuid.UUID) (*apiPlayer.PlayerResponse, *xError.Error) {
	l.log.Info(ctx, "GetPlayer - 查询玩家信息")

	player, xErr := l.repo.player.GetByUUID(ctx.Request.Context(), playerUUID)
	if xErr != nil {
		return nil, xErr
	}

	return &apiPlayer.PlayerResponse{
		UUID:       player.UUID.String(),
		Username:   player.Username,
		GroupName:  player.GroupName,
		ReportedAt: player.ReportedAt,
	}, nil
}

func (l *PlayerLogic) UpdatePlayerGroup(ctx *gin.Context, playerUUID uuid.UUID, username, groupName string) *xError.Error {
	l.log.Info(ctx, "UpdatePlayerGroup - 更新玩家权限组")

	player := &entity.Player{
		UUID:       playerUUID,
		Username:   username,
		GroupName:  groupName,
		ReportedAt: time.Now(),
	}

	if xErr := l.repo.player.CreateOrUpdate(ctx.Request.Context(), player); xErr != nil {
		return xErr
	}
	return nil
}

func (l *PlayerLogic) ListPlayers(ctx *gin.Context, page, pageSize int) ([]apiPlayer.PlayerResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListPlayers - 查询玩家列表")

	players, total, xErr := l.repo.player.List(ctx.Request.Context(), page, pageSize)
	if xErr != nil {
		return nil, 0, xErr
	}

	var resp []apiPlayer.PlayerResponse
	for _, p := range players {
		resp = append(resp, apiPlayer.PlayerResponse{
			UUID:       p.UUID.String(),
			Username:   p.Username,
			GroupName:  p.GroupName,
			ReportedAt: p.ReportedAt,
		})
	}
	return resp, total, nil
}
