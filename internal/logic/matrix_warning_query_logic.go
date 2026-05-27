package logic

import (
	"context"
	"time"

	"gorm.io/gorm"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type warningQueryRepo struct {
	matrixWarning *repository.MatrixWarningRepo
}

// WarningQueryLogic 警告查询 Logic
type WarningQueryLogic struct {
	logic
	repo warningQueryRepo
}

// NewWarningQueryLogic 创建 WarningQueryLogic 实例
func NewWarningQueryLogic(ctx context.Context) *WarningQueryLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &WarningQueryLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "WarningQueryLogic"),
		},
		repo: warningQueryRepo{
			matrixWarning: repository.NewMatrixWarningRepo(db),
		},
	}
}

// ListWarnings 多条件筛选查询警告列表
func (l *WarningQueryLogic) ListWarnings(ctx context.Context, playerUUID, warningType, serverName string, riskScoreMin, riskScoreMax *int32, startTime, endTime *time.Time, page, pageSize int) ([]*entity.MatrixPlayerWarning, int64, *xError.Error) {
	l.log.Info(ctx, "ListWarnings - 查询警告列表")
	return l.repo.matrixWarning.ListWithFilter(ctx, playerUUID, warningType, serverName, riskScoreMin, riskScoreMax, startTime, endTime, page, pageSize)
}

// GetWarningByID 按 ID 查询警告详情
func (l *WarningQueryLogic) GetWarningByID(ctx context.Context, id int64) (*entity.MatrixPlayerWarning, *xError.Error) {
	l.log.Info(ctx, "GetWarningByID - 查询警告详情")
	var warning entity.MatrixPlayerWarning
	if err := l.db.WithContext(ctx).Where("id = ?", id).First(&warning).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "警告记录不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询警告详情失败", false, err)
	}
	return &warning, nil
}