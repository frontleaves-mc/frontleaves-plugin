package logic

import (
	"context"
	"strconv"
	"strings"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiServer "github.com/frontleaves-mc/frontleaves-plugin/api/server"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type serverRepo struct {
	server *repository.ServerRepo
}

type ServerLogic struct {
	logic
	repo serverRepo
}

func NewServerLogic(ctx context.Context) *ServerLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &ServerLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "ServerLogic"),
		},
		repo: serverRepo{
			server: repository.NewServerRepo(db),
		},
	}
}

func (l *ServerLogic) Create(ctx context.Context, name, displayName, description, address string, sortOrder int) (*apiServer.ServerResponse, *xError.Error) {
	l.log.Info(ctx, "Create - 创建服务器")

	server := &entity.Server{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Address:     address,
		SortOrder:   sortOrder,
		IsPublic:    false,
		IsEnabled:   true,
	}

	if xErr := l.repo.server.Create(ctx, server); xErr != nil {
		return nil, xErr
	}

	return l.toResponse(server), nil
}

func (l *ServerLogic) GetByID(ctx context.Context, id xSnowflake.SnowflakeID) (*apiServer.ServerResponse, *xError.Error) {
	l.log.Info(ctx, "GetByID - 查询服务器")

	server, xErr := l.repo.server.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toResponse(server), nil
}

func (l *ServerLogic) GetByName(ctx context.Context, name string) (*apiServer.ServerResponse, *xError.Error) {
	l.log.Info(ctx, "GetByName - 按名称查询服务器")

	server, xErr := l.repo.server.GetByName(ctx, name)
	if xErr != nil {
		return nil, xErr
	}
	return l.toResponse(server), nil
}

func (l *ServerLogic) Update(ctx context.Context, id xSnowflake.SnowflakeID, displayName, description, address string, sortOrder int) (*apiServer.ServerResponse, *xError.Error) {
	l.log.Info(ctx, "Update - 更新服务器")

	server, xErr := l.repo.server.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	server.DisplayName = displayName
	server.Description = description
	server.Address = address
	server.SortOrder = sortOrder

	if xErr := l.repo.server.Update(ctx, server); xErr != nil {
		return nil, xErr
	}

	return l.toResponse(server), nil
}

func (l *ServerLogic) Delete(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "Delete - 删除服务器")
	return l.repo.server.Delete(ctx, id)
}

func (l *ServerLogic) List(ctx context.Context, page, pageSize int) (*apiServer.ServerListResponse, *xError.Error) {
	l.log.Info(ctx, "List - 查询服务器列表")

	servers, total, xErr := l.repo.server.List(ctx, page, pageSize)
	if xErr != nil {
		return nil, xErr
	}

	var list []apiServer.ServerResponse
	for _, s := range servers {
		list = append(list, *l.toResponse(&s))
	}

	return &apiServer.ServerListResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (l *ServerLogic) SetPublic(ctx context.Context, id xSnowflake.SnowflakeID, isPublic bool) (*apiServer.ServerResponse, *xError.Error) {
	l.log.Info(ctx, "SetPublic - 设置服务器公开状态")

	server, xErr := l.repo.server.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	server.IsPublic = isPublic
	if xErr := l.repo.server.Update(ctx, server); xErr != nil {
		return nil, xErr
	}

	return l.toResponse(server), nil
}

func (l *ServerLogic) SetEnabled(ctx context.Context, id xSnowflake.SnowflakeID, isEnabled bool) (*apiServer.ServerResponse, *xError.Error) {
	l.log.Info(ctx, "SetEnabled - 设置服务器启用状态")

	server, xErr := l.repo.server.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	server.IsEnabled = isEnabled
	if xErr := l.repo.server.Update(ctx, server); xErr != nil {
		return nil, xErr
	}

	return l.toResponse(server), nil
}

func (l *ServerLogic) GetOrCreateByName(ctx context.Context, name string) (*entity.Server, *xError.Error) {
	l.log.Info(ctx, "GetOrCreateByName - 获取或创建服务器")

	server, xErr := l.repo.server.GetByName(ctx, name)
	if xErr == nil {
		return server, nil
	}

	if xErr.ErrorCode != xError.ResourceNotFound {
		return nil, xErr
	}

	newServer := &entity.Server{
		Name:        name,
		DisplayName: name,
		IsPublic:    false,
		IsEnabled:   true,
	}

	if xErr := l.repo.server.Create(ctx, newServer); xErr != nil {
		return nil, xErr
	}

	return newServer, nil
}

func (l *ServerLogic) GetPublicServers(ctx context.Context) ([]entity.Server, *xError.Error) {
	l.log.Info(ctx, "GetPublicServers - 查询公开服务器列表")
	var servers []entity.Server
	if err := l.db.WithContext(ctx).Where("is_public = ? AND is_enabled = ?", true, true).Order("sort_order ASC, created_at DESC").Find(&servers).Error; err != nil {
		return nil, xError.NewError(nil, xError.DatabaseError, "查询公开服务器列表失败", false, err)
	}
	return servers, nil
}

func (l *ServerLogic) toResponse(server *entity.Server) *apiServer.ServerResponse {
	return &apiServer.ServerResponse{
		ID:          server.ID.String(),
		Name:        server.Name,
		DisplayName: server.DisplayName,
		Description: server.Description,
		IsPublic:    server.IsPublic,
		IsEnabled:   server.IsEnabled,
		Address:     server.Address,
		SortOrder:   server.SortOrder,
		CreatedAt:   server.CreatedAt,
		UpdatedAt:   server.UpdatedAt,
	}
}

func (l *ServerLogic) GetMyOnlineProfiles(ctx context.Context, userID xSnowflake.SnowflakeID) ([]apiServer.OnlineGameProfileResponse, *xError.Error) {
	l.log.Info(ctx, "GetMyOnlineProfiles - 获取用户在线游戏账号")

	gameProfileRepo := repository.NewGameProfileRepo(l.db)
	profiles, xErr := gameProfileRepo.GetByUserID(ctx, userID)
	if xErr != nil {
		return nil, xErr
	}

	var onlineProfiles []apiServer.OnlineGameProfileResponse
	for _, profile := range profiles {
		playerKey := string(bConst.CacheStatusPlayer.Get(profile.UUID.String()))
		playerData, err := l.rdb.HGetAll(ctx, playerKey).Result()
		if err != nil {
			continue
		}
		if len(playerData) == 0 || playerData["online"] != "true" {
			continue
		}
		onlineProfiles = append(onlineProfiles, apiServer.OnlineGameProfileResponse{
			UUID:       profile.UUID.String(),
			Username:   profile.Username,
			ServerName: playerData["server_name"],
			WorldName:  playerData["world_name"],
		})
	}

	if onlineProfiles == nil {
		onlineProfiles = []apiServer.OnlineGameProfileResponse{}
	}
	return onlineProfiles, nil
}

func (l *ServerLogic) CheckPlayerOnline(ctx context.Context, playerUUID, username string) (*apiServer.PlayerOnlineResponse, *xError.Error) {
	l.log.Info(ctx, "CheckPlayerOnline - 检查玩家在线状态")

	resp := &apiServer.PlayerOnlineResponse{Online: false}

	if playerUUID != "" {
		playerKey := string(bConst.CacheStatusPlayer.Get(playerUUID))
		playerData, err := l.rdb.HGetAll(ctx, playerKey).Result()
		if err != nil {
			return resp, nil
		}
		if len(playerData) > 0 {
			resp.PlayerUUID = playerUUID
			resp.PlayerName = playerData["player_name"]
			if ls, parseErr := strconv.ParseInt(playerData["last_seen"], 10, 64); parseErr == nil {
				resp.LastSeen = ls
			}
			if playerData["online"] == "true" {
				resp.Online = true
				resp.ServerName = playerData["server_name"]
				resp.WorldName = playerData["world_name"]
			}
		}
	} else if username != "" {
		wildcardKey := string(bConst.CacheStatusPlayer.Get("*"))
		keys, err := l.rdb.Keys(ctx, wildcardKey).Result()
		if err != nil {
			return resp, nil
		}

		playerKeyPrefix := string(bConst.CacheStatusPlayer.Get(""))
		for _, key := range keys {
			playerData, err := l.rdb.HGetAll(ctx, key).Result()
			if err != nil || len(playerData) == 0 {
				continue
			}
			if playerData["player_name"] == username {
				resp.PlayerUUID = strings.TrimPrefix(key, playerKeyPrefix)
				resp.PlayerName = playerData["player_name"]
				if ls, parseErr := strconv.ParseInt(playerData["last_seen"], 10, 64); parseErr == nil {
					resp.LastSeen = ls
				}
				if playerData["online"] == "true" {
					resp.Online = true
					resp.ServerName = playerData["server_name"]
					resp.WorldName = playerData["world_name"]
				}
				break
			}
		}
	}

	return resp, nil
}
