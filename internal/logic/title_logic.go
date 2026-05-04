package logic

import (
	"context"
	"time"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	apiTitle "github.com/frontleaves-mc/frontleaves-plugin/api/title"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type titleRepo struct {
	title            *repository.TitleRepo
	gameProfileTitle *repository.GameProfileTitleRepo
	gameProfile      *repository.GameProfileRepo
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
			gameProfile:      repository.NewGameProfileRepo(db),
		},
	}
}

func (l *TitleLogic) CreateTitle(ctx context.Context, name, description string, titleType entity.TitleType, permissionGroup *string) (*apiTitle.TitleResponse, *xError.Error) {
	l.log.Info(ctx, "CreateTitle - 创建称号")

	title := &entity.Title{
		Name:            name,
		Description:     description,
		Type:            titleType,
		PermissionGroup: permissionGroup,
		IsActive:        true,
	}

	if xErr := l.repo.title.Create(ctx, title); xErr != nil {
		return nil, xErr
	}

	return l.toTitleResponse(title), nil
}

func (l *TitleLogic) UpdateTitle(ctx context.Context, id xSnowflake.SnowflakeID, name, description string, titleType entity.TitleType, permissionGroup *string, isActive *bool) (*apiTitle.TitleResponse, *xError.Error) {
	l.log.Info(ctx, "UpdateTitle - 更新称号")

	title, xErr := l.repo.title.GetByID(ctx, id)
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

	if xErr := l.repo.title.Update(ctx, title); xErr != nil {
		return nil, xErr
	}

	return l.toTitleResponse(title), nil
}

func (l *TitleLogic) DeleteTitle(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "DeleteTitle - 删除称号")
	return l.repo.title.Delete(ctx, id)
}

func (l *TitleLogic) GetTitle(ctx context.Context, id xSnowflake.SnowflakeID) (*apiTitle.TitleResponse, *xError.Error) {
	l.log.Info(ctx, "GetTitle - 查询称号")
	title, xErr := l.repo.title.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toTitleResponse(title), nil
}

func (l *TitleLogic) ListTitles(ctx context.Context, page, pageSize int, titleType *int16) ([]apiTitle.TitleResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListTitles - 查询称号列表")

	titles, total, xErr := l.repo.title.List(ctx, page, pageSize, titleType)
	if xErr != nil {
		return nil, 0, xErr
	}

	var resp []apiTitle.TitleResponse
	for _, t := range titles {
		resp = append(resp, *l.toTitleResponse(&t))
	}
	return resp, total, nil
}

func (l *TitleLogic) AssignTitleToPlayer(ctx context.Context, titleID xSnowflake.SnowflakeID, playerUUID uuid.UUID) *xError.Error {
	l.log.Info(ctx, "AssignTitleToPlayer - 分配称号给玩家")

	has, xErr := l.repo.gameProfileTitle.HasTitle(ctx, playerUUID, titleID)
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
		GrantedAt:  time.Now(),
		IsEquipped: false,
	}

	return l.repo.gameProfileTitle.Create(ctx, playerTitle)
}

func (l *TitleLogic) RevokeTitleFromPlayer(ctx context.Context, titleID xSnowflake.SnowflakeID, playerUUID uuid.UUID) *xError.Error {
	l.log.Info(ctx, "RevokeTitleFromPlayer - 撤销玩家称号")
	return l.repo.gameProfileTitle.Delete(ctx, playerUUID, titleID)
}

func (l *TitleLogic) EquipTitle(ctx context.Context, playerUUID uuid.UUID, titleID xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "EquipTitle - 装备称号")

	has, xErr := l.repo.gameProfileTitle.HasTitle(ctx, playerUUID, titleID)
	if xErr != nil {
		return xErr
	}

	// 虚拟称号需要先创建 GameProfileTitle 记录才能装备
	if !has {
		title, xErr := l.repo.title.GetByID(ctx, titleID)
		if xErr != nil {
			return xErr
		}

		var source entity.TitleSource
		switch title.Type {
		case entity.TitleTypeFree:
			source = entity.TitleSourceAuto
		case entity.TitleTypeGroup:
			source = entity.TitleSourceGroup
		default:
			return xError.NewError(nil, xError.ParameterError, "玩家未拥有该称号", true, nil)
		}

		playerTitle := &entity.GameProfileTitle{
			GameProfileUUID: playerUUID,
			TitleID:    titleID,
			Source:     source,
			GrantedAt:  time.Now(),
			IsEquipped: false,
		}
		if xErr := l.repo.gameProfileTitle.Create(ctx, playerTitle); xErr != nil {
			return xErr
		}
	}

	return l.repo.gameProfileTitle.EquipTitle(ctx, l.db, playerUUID, titleID)
}

func (l *TitleLogic) UnequipTitle(ctx context.Context, playerUUID uuid.UUID) *xError.Error {
	l.log.Info(ctx, "UnequipTitle - 卸下称号")
	return l.repo.gameProfileTitle.UnequipTitle(ctx, l.db, playerUUID)
}

func (l *TitleLogic) GetPlayerTitles(ctx context.Context, playerUUID uuid.UUID) ([]apiTitle.PlayerTitleResponse, *xError.Error) {
	l.log.Info(ctx, "GetPlayerTitles - 查询玩家拥有的称号（含虚拟授予）")

	titleMap := make(map[xSnowflake.SnowflakeID]apiTitle.PlayerTitleResponse)

	// 1. 免费称号（虚拟授予）
	freeTitles, xErr := l.repo.title.GetActiveFreeTitles(ctx)
	if xErr != nil {
		return nil, xErr
	}
	for _, t := range freeTitles {
		titleMap[t.ID] = apiTitle.PlayerTitleResponse{
			TitleResponse: *l.toTitleResponse(&t),
			Source:        int16(entity.TitleSourceAuto),
			IsEquipped:    false,
			GrantedAt:     time.Time{},
		}
	}

	// 2. 权限组称号（虚拟授予）
	profile, xErr := l.repo.gameProfile.GetByUUID(ctx, playerUUID)
	if xErr == nil && profile.GroupName != "" {
		groupTitles, xErr := l.repo.title.GetActiveGroupTitlesByGroupName(ctx, profile.GroupName)
		if xErr != nil {
			return nil, xErr
		}
		for _, t := range groupTitles {
			titleMap[t.ID] = apiTitle.PlayerTitleResponse{
				TitleResponse: *l.toTitleResponse(&t),
				Source:        int16(entity.TitleSourceGroup),
				IsEquipped:    false,
				GrantedAt:     time.Time{},
			}
		}
	}

	// 3. GameProfileTitle 表中的专属记录
	playerTitles, xErr := l.repo.gameProfileTitle.GetByGameProfileUUID(ctx, playerUUID)
	if xErr != nil {
		return nil, xErr
	}
	for _, pt := range playerTitles {
		if pt.Title != nil {
			titleMap[pt.TitleID] = apiTitle.PlayerTitleResponse{
				TitleResponse: *l.toTitleResponse(pt.Title),
				Source:        int16(pt.Source),
				IsEquipped:    pt.IsEquipped,
				GrantedAt:     pt.GrantedAt,
			}
		}
	}

	var resp []apiTitle.PlayerTitleResponse
	for _, t := range titleMap {
		resp = append(resp, t)
	}
	return resp, nil
}

func (l *TitleLogic) GetEquippedTitle(ctx context.Context, playerUUID uuid.UUID) (*apiTitle.EquippedTitleResponse, *xError.Error) {
	l.log.Info(ctx, "GetEquippedTitle - 查询装备的称号")

	playerTitle, xErr := l.repo.gameProfileTitle.GetEquippedByGameProfileUUID(ctx, playerUUID)
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
	l.log.Info(ctx, "MatchGroupTitle - 已废弃，权限组称号现通过虚拟授予获取")
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
