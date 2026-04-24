package repository

import (
	"context"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type AchievementRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewAchievementRepo(db *gorm.DB, rdb *redis.Client) *AchievementRepo {
	return &AchievementRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "AchievementRepo"),
	}
}

func (r *AchievementRepo) Create(ctx context.Context, ach *entity.Achievement) *xError.Error {
	r.log.Info(ctx, "Create - 创建成就")
	if err := r.db.WithContext(ctx).Create(ach).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建成就失败", false, err)
	}
	return nil
}

func (r *AchievementRepo) Update(ctx context.Context, ach *entity.Achievement) *xError.Error {
	r.log.Info(ctx, "Update - 更新成就")
	if err := r.db.WithContext(ctx).Save(ach).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "更新成就失败", false, err)
	}
	return nil
}

func (r *AchievementRepo) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	r.log.Info(ctx, "Delete - 删除成就")
	if err := r.db.WithContext(ctx).Delete(&entity.Achievement{}, id).Error; err != nil {
		return xError.NewError(nil, xError.DatabaseError, "删除成就失败", false, err)
	}
	return nil
}

func (r *AchievementRepo) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*entity.Achievement, *xError.Error) {
	r.log.Info(ctx, "GetByID - 查询成就")
	var ach entity.Achievement
	if err := r.db.WithContext(ctx).First(&ach, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "成就不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询成就失败", false, err)
	}
	return &ach, nil
}

func (r *AchievementRepo) List(ctx context.Context, page, pageSize int, achType *int16) ([]entity.Achievement, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询成就列表")

	query := r.db.WithContext(ctx).Model(&entity.Achievement{})
	if achType != nil {
		query = query.Where("type = ?", *achType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询成就总数失败", false, err)
	}

	var achievements []entity.Achievement
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("sort_order ASC, created_at DESC").Find(&achievements).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询成就列表失败", false, err)
	}
	return achievements, total, nil
}

func (r *AchievementRepo) GetActiveByConditionKey(ctx context.Context, conditionKey string) ([]entity.Achievement, *xError.Error) {
	r.log.Info(ctx, "GetActiveByConditionKey - 按条件标识查询活跃成就")
	var achievements []entity.Achievement
	if err := r.db.WithContext(ctx).Where("condition_key = ? AND is_active = ?", conditionKey, true).Find(&achievements).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "按条件标识查询成就失败", false, err)
	}
	return achievements, nil
}

func (r *AchievementRepo) ListActive(ctx context.Context) ([]entity.Achievement, *xError.Error) {
	r.log.Info(ctx, "ListActive - 查询所有活跃成就")
	var achievements []entity.Achievement
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("sort_order ASC, created_at DESC").Find(&achievements).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询活跃成就列表失败", false, err)
	}
	return achievements, nil
}
