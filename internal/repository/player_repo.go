package repository

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PlayerRepo struct {
	db  *gorm.DB
	rdb *redis.Client
	log *xLog.LogNamedLogger
}

func NewPlayerRepo(db *gorm.DB, rdb *redis.Client) *PlayerRepo {
	return &PlayerRepo{
		db:  db,
		rdb: rdb,
		log: xLog.WithName(xLog.NamedREPO, "PlayerRepo"),
	}
}

func (r *PlayerRepo) GetByUUID(ctx context.Context, playerUUID uuid.UUID) (*entity.Player, *xError.Error) {
	r.log.Info(ctx, "GetByUUID - 查询玩家信息")

	var player entity.Player
	if err := r.db.WithContext(ctx).Where("uuid = ?", playerUUID).First(&player).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, xError.NewError(nil, xError.NotFound, "玩家不存在", false, err)
		}
		return nil, xError.NewError(nil, xError.DatabaseError, "查询玩家失败", false, err)
	}
	return &player, nil
}

func (r *PlayerRepo) CreateOrUpdate(ctx context.Context, player *entity.Player) *xError.Error {
	r.log.Info(ctx, "CreateOrUpdate - 创建或更新玩家")

	result := r.db.WithContext(ctx).Where("uuid = ?", player.UUID).
		Assign(map[string]interface{}{
			"username":    player.Username,
			"group_name":  player.GroupName,
			"reported_at": player.ReportedAt,
		}).
		FirstOrCreate(player)
	if result.Error != nil {
		return xError.NewError(nil, xError.DatabaseError, "创建或更新玩家失败", false, result.Error)
	}
	return nil
}

func (r *PlayerRepo) List(ctx context.Context, page, pageSize int) ([]entity.Player, int64, *xError.Error) {
	r.log.Info(ctx, "List - 查询玩家列表")

	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.Player{}).Count(&total).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家总数失败", false, err)
	}

	var players []entity.Player
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&players).Error; err != nil {
		return nil, 0, xError.NewError(nil, xError.DatabaseError, "查询玩家列表失败", false, err)
	}
	return players, total, nil
}
