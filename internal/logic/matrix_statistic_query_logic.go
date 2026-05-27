package logic

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/google/uuid"
)

type statisticQueryRepo struct {
	matrixStatistic *repository.MatrixStatisticRepo
}

// StatisticQueryLogic 统计查询 Logic
type StatisticQueryLogic struct {
	logic
	repo statisticQueryRepo
}

// NewStatisticQueryLogic 创建 StatisticQueryLogic 实例
func NewStatisticQueryLogic(ctx context.Context) *StatisticQueryLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &StatisticQueryLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "StatisticQueryLogic"),
		},
		repo: statisticQueryRepo{
			matrixStatistic: repository.NewMatrixStatisticRepo(db),
		},
	}
}

// GetByUUID 按玩家 UUID 查询统计数据
func (l *StatisticQueryLogic) GetByUUID(ctx context.Context, playerUUID uuid.UUID) (*entity.MatrixPlayerStatistic, *xError.Error) {
	l.log.Info(ctx, "GetByUUID - 查询玩家统计数据")
	stat, xErr := l.repo.matrixStatistic.GetByUUID(ctx, playerUUID)
	if xErr != nil {
		return nil, xErr
	}
	if stat == nil {
		return nil, xError.NewError(nil, xError.NotFound, "未找到该玩家统计数据", false, nil)
	}
	return stat, nil
}