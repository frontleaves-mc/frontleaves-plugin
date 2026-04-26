package logic

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	apiTitle "github.com/frontleaves-mc/frontleaves-plugin/api/title"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/gin-gonic/gin"
)

type titleRepo struct {
	title       *repository.TitleRepo
	gameProfileTitle *repository.GameProfileTitleRepo
}

type TitleLogic struct {
	logic
	repo titleRepo
}

func NewTitleLogic(ctx context.Context) *TitleLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &TitleLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "TitleLogic"),
		},
		repo: titleRepo{
			title:            repository.NewTitleRepo(db),
			gameProfileTitle: repository.NewGameProfileTitleRepo(db),
		},
	}
}

func (l *TitleLogic) CreateTitle(ctx *gin.Context, name, description string, titleType entity.TitleType, permissionGroup *string) (*apiTitle.TitleResponse, *xError.Error) {
	l.log.Info(ctx, "CreateTitle - 创建称号")

	title := &entity.Title{
		Name:            name,
		Description:     description,
		Type:            titleType,
		PermissionGroup: permissionGroup,
		IsActive:        true,
	}

	if xErr := l.repo.title.Create(ctx.Request.Context(), title); xErr != nil {
		return nil, xErr
	}

	return l.toTitleResponse(title), nil
}

func (l *TitleLogic) UpdateTitle(ctx *gin.Context, id xSnowflake.SnowflakeID, name, description string, titleType entity.TitleType, permissionGroup *string, isActive *bool) (*apiTitle.TitleResponse, *xError.Error) {
	l.log.Info(ctx, "UpdateTitle - 更新称号")

	title, xErr := l.repo.title.GetByID(ctx.Request.Context(), id)
	if xErr != nil {
		return nil, xErr
	}

	title.Name = name
	title.Description = description
	title.Type = titleType
	title.PermissionGroup = permissionGroup
	if isActive != nil {
		title.IsActive = *isActive
	}

	if xErr := l.repo.title.Update(ctx.Request.Context(), title); xErr != nil {
		return nil, xErr
	}

	return l.toTitleResponse(title), nil
}

func (l *TitleLogic) DeleteTitle(ctx *gin.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "DeleteTitle - 删除称号")
	return l.repo.title.Delete(ctx.Request.Context(), id)
}

func (l *TitleLogic) GetTitle(ctx *gin.Context, id xSnowflake.SnowflakeID) (*apiTitle.TitleResponse, *xError.Error) {
	l.log.Info(ctx, "GetTitle - 查询称号")
	title, xErr := l.repo.title.GetByID(ctx.Request.Context(), id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toTitleResponse(title), nil
}

func (l *TitleLogic) ListTitles(ctx *gin.Context, page, pageSize int, titleType *int16) ([]apiTitle.TitleResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListTitles - 查询称号列表")

	titles, total, xErr := l.repo.title.List(ctx.Request.Context(), page, pageSize, titleType)
	if xErr != nil {
		return nil, 0, xErr
	}

	var resp []apiTitle.TitleResponse
	for _, t := range titles {
		resp = append(resp, *l.toTitleResponse(&t))
	}
	return resp, total, nil
}

func (l *TitleLogic) AssignTitleToPlayer(ctx *gin.Context, titleID xSnowflake.SnowflakeID, playerUUID uuid.UUID) *xError.Error {
	l.log.Info(ctx, "AssignTitleToPlayer - 分配称号给玩家")

	has, xErr := l.repo.gameProfileTitle.HasTitle(ctx.Request.Context(), playerUUID, titleID)
	if xErr != nil {
		return xErr
	}
	if has {
		return xError.NewError(nil, xError.ParameterError, "玩家已拥有该称号", true, nil)
	}

	playerTitle := &entity.GameProfileTitle{
		GameProfileUUID: playerUUID,
		TitleID:    titleID,
		Source:     entity.TitleSourceAdmin,
		IsEquipped: false,
	}

	return l.repo.gameProfileTitle.Create(ctx.Request.Context(), playerTitle)
}

func (l *TitleLogic) RevokeTitleFromPlayer(ctx *gin.Context, titleID xSnowflake.SnowflakeID, playerUUID uuid.UUID) *xError.Error {
	l.log.Info(ctx, "RevokeTitleFromPlayer - 撤销玩家称号")
	return l.repo.gameProfileTitle.Delete(ctx.Request.Context(), playerUUID, titleID)
}

func (l *TitleLogic) EquipTitle(ctx *gin.Context, playerUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "EquipTitle - 装备称号")

	has, xErr := l.repo.gameProfileTitle.HasTitle(ctx.Request.Context(), playerUUID, titleID)
	if xErr != nil {
		return xErr
	}
	if !has {
		return xError.NewError(nil, xError.ParameterError, "玩家未拥有该称号", true, nil)
	}

	return l.repo.gameProfileTitle.EquipTitle(ctx.Request.Context(), l.db, playerUUID, titleID)
}

func (l *TitleLogic) UnequipTitle(ctx *gin.Context, playerUUID uuid.UUID) *xError.Error {
	l.log.Info(ctx, "UnequipTitle - 卸下称号")
	return l.repo.gameProfileTitle.UnequipTitle(ctx.Request.Context(), l.db, playerUUID)
}

func (l *TitleLogic) GetPlayerTitles(ctx *gin.Context, playerUUID uuid.UUID) ([]apiTitle.PlayerTitleResponse, *xError.Error) {
	l.log.Info(ctx, "GetPlayerTitles - 查询玩家拥有的称号")

	playerTitles, xErr := l.repo.gameProfileTitle.GetByGameProfileUUID(ctx.Request.Context(), playerUUID)
	if xErr != nil {
		return nil, xErr
	}

	var resp []apiTitle.PlayerTitleResponse
	for _, pt := range playerTitles {
		resp = append(resp, apiTitle.PlayerTitleResponse{
			TitleResponse: *l.toTitleResponse(pt.Title),
			Source:        int16(pt.Source),
			IsEquipped:    pt.IsEquipped,
			GrantedAt:     pt.CreatedAt,
		})
	}
	return resp, nil
}

func (l *TitleLogic) GetEquippedTitle(ctx *gin.Context, playerUUID uuid.UUID) (*apiTitle.EquippedTitleResponse, *xError.Error) {
	l.log.Info(ctx, "GetEquippedTitle - 查询装备的称号")

	playerTitle, xErr := l.repo.gameProfileTitle.GetEquippedByGameProfileUUID(ctx.Request.Context(), playerUUID)
	if xErr != nil {
		return nil, xErr
	}
	if playerTitle == nil {
		return nil, nil
	}

	return &apiTitle.EquippedTitleResponse{
		TitleID:     playerTitle.TitleID.String(),
		Name:        playerTitle.Title.Name,
		Description: playerTitle.Title.Description,
		Type:        int16(playerTitle.Title.Type),
	}, nil
}

func (l *TitleLogic) MatchGroupTitle(ctx context.Context, playerUUID uuid.UUID, groupName string) *xError.Error {
	l.log.Info(ctx, "MatchGroupTitle - 匹配权限组称号")

	titles, xErr := l.repo.title.GetByPermissionGroup(ctx, groupName)
	if xErr != nil {
		return xErr
	}

	for _, title := range titles {
		has, xErr := l.repo.gameProfileTitle.HasTitle(ctx, playerUUID, title.ID)
		if xErr != nil {
			return xErr
		}
		if !has {
			playerTitle := &entity.GameProfileTitle{
				GameProfileUUID: playerUUID,
				TitleID:    title.ID,
				Source:     entity.TitleSourceGroup,
				IsEquipped: false,
			}
			if xErr := l.repo.gameProfileTitle.Create(ctx, playerTitle); xErr != nil {
				return xErr
			}
		}
	}
	return nil
}

func (l *TitleLogic) toTitleResponse(title *entity.Title) *apiTitle.TitleResponse {
	resp := &apiTitle.TitleResponse{
		ID:          title.ID.String(),
		Name:        title.Name,
		Description: title.Description,
		Type:        int16(title.Type),
		IsActive:    title.IsActive,
		CreatedAt:   title.CreatedAt,
	}
	if title.PermissionGroup != nil {
		resp.PermissionGroup = title.PermissionGroup
	}
	return resp
}
