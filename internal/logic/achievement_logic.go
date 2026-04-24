package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	apiAchievement "github.com/frontleaves-mc/frontleaves-plugin/api/achievement"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
)

type achievementRepo struct {
	achievement *repository.AchievementRepo
	playerAch   *repository.PlayerAchievementRepo
	claim       *repository.PlayerAchievementClaimRepo
	playerTitle *repository.PlayerTitleRepo
}

type AchievementLogic struct {
	logic
	repo achievementRepo
}

func NewAchievementLogic(ctx context.Context) *AchievementLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &AchievementLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "AchievementLogic"),
		},
		repo: achievementRepo{
			achievement: repository.NewAchievementRepo(db, rdb),
			playerAch:   repository.NewPlayerAchievementRepo(db, rdb),
			claim:       repository.NewPlayerAchievementClaimRepo(db, rdb),
			playerTitle: repository.NewPlayerTitleRepo(db, rdb),
		},
	}
}

func (l *AchievementLogic) CreateAchievement(ctx *gin.Context, name, description string, achType entity.AchievementType, conditionKey string, conditionParams, rewardConfig json.RawMessage, sortOrder int) (*apiAchievement.AchievementResponse, *xError.Error) {
	l.log.Info(ctx, "CreateAchievement - 创建成就")

	ach := &entity.Achievement{
		Name:            name,
		Description:     description,
		Type:            achType,
		ConditionKey:    conditionKey,
		ConditionParams: conditionParams,
		RewardConfig:    rewardConfig,
		IsActive:        true,
		SortOrder:       sortOrder,
	}

	if xErr := l.repo.achievement.Create(ctx.Request.Context(), ach); xErr != nil {
		return nil, xErr
	}

	return l.toAchievementResponse(ach), nil
}

func (l *AchievementLogic) UpdateAchievement(ctx *gin.Context, id xSnowflake.SnowflakeID, name, description string, achType entity.AchievementType, conditionKey string, conditionParams, rewardConfig json.RawMessage, sortOrder int, isActive *bool) (*apiAchievement.AchievementResponse, *xError.Error) {
	l.log.Info(ctx, "UpdateAchievement - 更新成就")

	ach, xErr := l.repo.achievement.GetByID(ctx.Request.Context(), id)
	if xErr != nil {
		return nil, xErr
	}

	ach.Name = name
	ach.Description = description
	ach.Type = achType
	ach.ConditionKey = conditionKey
	ach.ConditionParams = conditionParams
	ach.RewardConfig = rewardConfig
	ach.SortOrder = sortOrder
	if isActive != nil {
		ach.IsActive = *isActive
	}

	if xErr := l.repo.achievement.Update(ctx.Request.Context(), ach); xErr != nil {
		return nil, xErr
	}

	return l.toAchievementResponse(ach), nil
}

func (l *AchievementLogic) DeleteAchievement(ctx *gin.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "DeleteAchievement - 删除成就")
	return l.repo.achievement.Delete(ctx.Request.Context(), id)
}

func (l *AchievementLogic) GetAchievement(ctx *gin.Context, id xSnowflake.SnowflakeID) (*apiAchievement.AchievementResponse, *xError.Error) {
	l.log.Info(ctx, "GetAchievement - 查询成就")
	ach, xErr := l.repo.achievement.GetByID(ctx.Request.Context(), id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toAchievementResponse(ach), nil
}

func (l *AchievementLogic) ListAchievements(ctx *gin.Context, page, pageSize int, achType *int16) ([]apiAchievement.AchievementResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListAchievements - 查询成就列表")

	achievements, total, xErr := l.repo.achievement.List(ctx.Request.Context(), page, pageSize, achType)
	if xErr != nil {
		return nil, 0, xErr
	}

	var resp []apiAchievement.AchievementResponse
	for _, a := range achievements {
		resp = append(resp, *l.toAchievementResponse(&a))
	}
	return resp, total, nil
}

func (l *AchievementLogic) GrantAchievement(ctx *gin.Context, achievementID xSnowflake.SnowflakeID, playerUUID uuid.UUID) *xError.Error {
	l.log.Info(ctx, "GrantAchievement - 手动授予成就")

	ach, xErr := l.repo.achievement.GetByID(ctx.Request.Context(), achievementID)
	if xErr != nil {
		return xErr
	}

	existing, xErr := l.repo.playerAch.GetByPlayerAndAchievement(ctx.Request.Context(), playerUUID, achievementID)
	if xErr != nil {
		return xErr
	}
	if existing != nil {
		return xError.NewError(nil, xError.ParameterError, "玩家已拥有该成就", true, nil)
	}

	pa := &entity.PlayerAchievement{
		PlayerUUID:    playerUUID,
		AchievementID: achievementID,
		Status:        entity.AchievementStatusCompleted,
		Progress:      1,
	}
	now := time.Now()
	pa.CompletedAt = &now

	if xErr := l.repo.playerAch.Create(ctx.Request.Context(), pa); xErr != nil {
		return xErr
	}

	if len(ach.RewardConfig) > 0 {
		claim := &entity.PlayerAchievementClaim{
			PlayerUUID:    playerUUID,
			AchievementID: achievementID,
			TitleClaimed:  false,
		}
		if xErr := l.repo.claim.Create(ctx.Request.Context(), claim); xErr != nil {
			return xErr
		}
	} else {
		if xErr := l.repo.playerAch.UpdateStatus(ctx.Request.Context(), pa.ID, entity.AchievementStatusClaimed); xErr != nil {
			return xErr
		}
	}

	return nil
}

func (l *AchievementLogic) EvaluateEvent(ctx context.Context, conditionKey string, playerUUID uuid.UUID, value int64) *xError.Error {
	l.log.Info(ctx, "EvaluateEvent - 评估事件触发成就")

	achievements, xErr := l.repo.achievement.GetActiveByConditionKey(ctx, conditionKey)
	if xErr != nil {
		return xErr
	}

	for _, ach := range achievements {
		if ach.Type == entity.AchievementTypeManual {
			continue
		}

		pa, xErr := l.repo.playerAch.GetByPlayerAndAchievement(ctx, playerUUID, ach.ID)
		if xErr != nil {
			return xErr
		}

		if pa != nil && pa.Status != entity.AchievementStatusInProgress {
			continue
		}

		if pa == nil {
			pa = &entity.PlayerAchievement{
				PlayerUUID:    playerUUID,
				AchievementID: ach.ID,
				Status:        entity.AchievementStatusInProgress,
				Progress:      0,
			}
			if xErr := l.repo.playerAch.Create(ctx, pa); xErr != nil {
				return xErr
			}
		}

		completed := false
		switch ach.Type {
		case entity.AchievementTypeStat:
			newProgress := pa.Progress + value
			if xErr := l.repo.playerAch.UpdateProgress(ctx, pa.ID, newProgress); xErr != nil {
				return xErr
			}
			threshold := l.getThreshold(ach.ConditionParams)
			if threshold > 0 && newProgress >= threshold {
				completed = true
			}
		case entity.AchievementTypeEvent:
			if xErr := l.repo.playerAch.UpdateProgress(ctx, pa.ID, 1); xErr != nil {
				return xErr
			}
			completed = true
		case entity.AchievementTypeSpecial:
			threshold := l.getThreshold(ach.ConditionParams)
			if threshold > 0 && value >= threshold {
				if xErr := l.repo.playerAch.UpdateProgress(ctx, pa.ID, value); xErr != nil {
					return xErr
				}
				completed = true
			}
		}

		if completed {
			if xErr := l.repo.playerAch.UpdateStatus(ctx, pa.ID, entity.AchievementStatusCompleted); xErr != nil {
				return xErr
			}

			if len(ach.RewardConfig) > 0 {
				claim := &entity.PlayerAchievementClaim{
					PlayerUUID:    playerUUID,
					AchievementID: ach.ID,
					TitleClaimed:  false,
				}
				if xErr := l.repo.claim.Create(ctx, claim); xErr != nil {
					return xErr
				}
			} else {
				if xErr := l.repo.playerAch.UpdateStatus(ctx, pa.ID, entity.AchievementStatusClaimed); xErr != nil {
					return xErr
				}
			}
		}
	}
	return nil
}

func (l *AchievementLogic) ClaimReward(ctx *gin.Context, playerUUID uuid.UUID, achievementID xSnowflake.SnowflakeID) (*apiAchievement.AchievementClaimResponse, *xError.Error) {
	l.log.Info(ctx, "ClaimReward - 领取成就奖励")

	pa, xErr := l.repo.playerAch.GetByPlayerAndAchievement(ctx.Request.Context(), playerUUID, achievementID)
	if xErr != nil {
		return nil, xErr
	}
	if pa == nil || pa.Status < entity.AchievementStatusCompleted {
		return nil, xError.NewError(nil, xError.ParameterError, "成就尚未完成", true, nil)
	}

	ach, xErr := l.repo.achievement.GetByID(ctx.Request.Context(), achievementID)
	if xErr != nil {
		return nil, xErr
	}

	claim, xErr := l.repo.claim.GetByPlayerAndAchievement(ctx.Request.Context(), playerUUID, achievementID)
	if xErr != nil {
		return nil, xErr
	}

	if claim == nil {
		if xErr := l.repo.playerAch.UpdateStatus(ctx.Request.Context(), pa.ID, entity.AchievementStatusClaimed); xErr != nil {
			return nil, xErr
		}
		return &apiAchievement.AchievementClaimResponse{
			AchievementID: achievementID.String(),
			TitleClaimed:  false,
		}, nil
	}

	var rewardConfig struct {
		TitleID string `json:"title_id"`
	}
	if err := json.Unmarshal(ach.RewardConfig, &rewardConfig); err == nil && rewardConfig.TitleID != "" {
		if !claim.TitleClaimed {
			titleID, _ := strconv.ParseInt(rewardConfig.TitleID, 10, 64)
			has, xErr := l.repo.playerTitle.HasTitle(ctx.Request.Context(), playerUUID, xSnowflake.SnowflakeID(titleID))
			if xErr != nil {
				return nil, xErr
			}
			if !has {
				playerTitle := &entity.PlayerTitle{
					PlayerUUID: playerUUID,
					TitleID:    xSnowflake.SnowflakeID(titleID),
					Source:     entity.TitleSourceAchievement,
					IsEquipped: false,
					GrantedAt:  time.Now(),
				}
				if xErr := l.repo.playerTitle.Create(ctx.Request.Context(), playerTitle); xErr != nil {
					return nil, xErr
				}
			}
			if xErr := l.repo.claim.UpdateTitleClaimed(ctx.Request.Context(), claim.ID, true); xErr != nil {
				return nil, xErr
			}
		}
	}

	if xErr := l.repo.playerAch.UpdateStatus(ctx.Request.Context(), pa.ID, entity.AchievementStatusClaimed); xErr != nil {
		return nil, xErr
	}

	return &apiAchievement.AchievementClaimResponse{
		AchievementID: achievementID.String(),
		TitleClaimed:  claim.TitleClaimed,
	}, nil
}

func (l *AchievementLogic) GetPlayerAchievements(ctx *gin.Context, playerUUID uuid.UUID) ([]apiAchievement.PlayerAchievementResponse, *xError.Error) {
	l.log.Info(ctx, "GetPlayerAchievements - 查询玩家成就列表")

	playerAchievements, xErr := l.repo.playerAch.ListByPlayer(ctx.Request.Context(), playerUUID)
	if xErr != nil {
		return nil, xErr
	}

	var resp []apiAchievement.PlayerAchievementResponse
	for _, pa := range playerAchievements {
		resp = append(resp, apiAchievement.PlayerAchievementResponse{
			AchievementResponse: *l.toAchievementResponse(pa.Achievement),
			Status:              int16(pa.Status),
			Progress:            pa.Progress,
			CompletedAt:         pa.CompletedAt,
		})
	}
	return resp, nil
}

func (l *AchievementLogic) ListPublicAchievements(ctx *gin.Context) ([]apiAchievement.AchievementResponse, *xError.Error) {
	l.log.Info(ctx, "ListPublicAchievements - 查询公开成就列表")

	achievements, xErr := l.repo.achievement.ListActive(ctx.Request.Context())
	if xErr != nil {
		return nil, xErr
	}

	var resp []apiAchievement.AchievementResponse
	for _, a := range achievements {
		resp = append(resp, *l.toAchievementResponse(&a))
	}
	return resp, nil
}

func (l *AchievementLogic) getThreshold(params json.RawMessage) int64 {
	if len(params) == 0 {
		return 0
	}
	var p struct {
		Threshold int64 `json:"threshold"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return 0
	}
	return p.Threshold
}

func (l *AchievementLogic) toAchievementResponse(ach *entity.Achievement) *apiAchievement.AchievementResponse {
	return &apiAchievement.AchievementResponse{
		ID:              ach.ID.String(),
		Name:            ach.Name,
		Description:     ach.Description,
		Type:            int16(ach.Type),
		ConditionKey:    ach.ConditionKey,
		ConditionParams: ach.ConditionParams,
		RewardConfig:    ach.RewardConfig,
		SortOrder:       ach.SortOrder,
		IsActive:        ach.IsActive,
	}
}
