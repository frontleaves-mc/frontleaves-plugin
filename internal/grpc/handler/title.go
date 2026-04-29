package handler

import (
	"context"

	"github.com/google/uuid"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xGrpcMiddle "github.com/bamboo-services/bamboo-base-go/plugins/grpc/middleware"
	xGrpcResult "github.com/bamboo-services/bamboo-base-go/plugins/grpc/result"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/middleware"
	titlepb "github.com/frontleaves-mc/frontleaves-plugin/internal/grpc/gen/title/v1"
	"google.golang.org/grpc"
)

// TitleHandler 称号服务 gRPC Handler
type TitleHandler struct {
	grpcHandler
	titlepb.UnimplementedTitleServiceServer
}

// NewTitleHandler 创建称号服务 gRPC Handler
func NewTitleHandler(ctx context.Context, server grpc.ServiceRegistrar) *TitleHandler {
	base := NewGRPCHandler[grpcHandler](ctx, "TitleHandler")
	h := &TitleHandler{grpcHandler: *base}

	titlepb.RegisterTitleServiceServer(server, h)
	xGrpcMiddle.UseUnary(titlepb.TitleService_ServiceDesc, middleware.UnaryPluginVerify(ctx))

	return h
}

// GetPlayerTitles 查询玩家拥有的所有称号
func (h *TitleHandler) GetPlayerTitles(
	ctx context.Context, req *titlepb.GetPlayerTitlesRequest,
) (*titlepb.GetPlayerTitlesResponse, error) {
	h.log.Info(ctx, "GetPlayerTitles - 查询玩家称号")

	playerUUID, err := uuid.Parse(req.GetPlayerUuid())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "player_uuid 格式无效", true, err)
	}

	playerTitles, xErr := h.service.titleLogic.GetPlayerTitles(ctx, playerUUID)
	if xErr != nil {
		return nil, xErr
	}

	resp := xGrpcResult.SuccessWith[*titlepb.GetPlayerTitlesResponse](ctx, "查询成功")
	resp.Titles = make([]*titlepb.PlayerTitle, len(playerTitles))
	for i, pt := range playerTitles {
		resp.Titles[i] = &titlepb.PlayerTitle{
			TitleId:    pt.ID,
			Name:       pt.Name,
			Description: pt.Description,
			Type:       int32(pt.Type),
			Source:     int32(pt.Source),
			IsEquipped: pt.IsEquipped,
		}
	}

	return resp, nil
}

// GetEquippedTitle 查询玩家当前装备的称号
func (h *TitleHandler) GetEquippedTitle(
	ctx context.Context, req *titlepb.GetEquippedTitleRequest,
) (*titlepb.GetEquippedTitleResponse, error) {
	h.log.Info(ctx, "GetEquippedTitle - 查询装备称号")

	playerUUID, err := uuid.Parse(req.GetPlayerUuid())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "player_uuid 格式无效", true, err)
	}

	equipped, xErr := h.service.titleLogic.GetEquippedTitle(ctx, playerUUID)
	if xErr != nil {
		return nil, xErr
	}

	resp := xGrpcResult.SuccessWith[*titlepb.GetEquippedTitleResponse](ctx, "查询成功")
	if equipped != nil {
		resp.TitleId = equipped.TitleID
		resp.Name = equipped.Name
		resp.Description = equipped.Description
		resp.Type = int32(equipped.Type)
	}

	return resp, nil
}

// EquipTitle 装备称号
func (h *TitleHandler) EquipTitle(
	ctx context.Context, req *titlepb.EquipTitleRequest,
) (*titlepb.EquipTitleResponse, error) {
	h.log.Info(ctx, "EquipTitle - 装备称号")

	playerUUID, err := uuid.Parse(req.GetPlayerUuid())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "player_uuid 格式无效", true, err)
	}

	titleID, err := xSnowflake.ParseSnowflakeID(req.GetTitleId())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "title_id 格式无效", true, err)
	}

	if xErr := h.service.titleLogic.EquipTitle(ctx, playerUUID, titleID); xErr != nil {
		return nil, xErr
	}

	resp := xGrpcResult.SuccessWith[*titlepb.EquipTitleResponse](ctx, "装备成功")
	return resp, nil
}

// UnequipTitle 卸下称号
func (h *TitleHandler) UnequipTitle(
	ctx context.Context, req *titlepb.UnequipTitleRequest,
) (*titlepb.UnequipTitleResponse, error) {
	h.log.Info(ctx, "UnequipTitle - 卸下称号")

	playerUUID, err := uuid.Parse(req.GetPlayerUuid())
	if err != nil {
		return nil, xError.NewError(ctx, xError.ParameterError, "player_uuid 格式无效", true, err)
	}

	if xErr := h.service.titleLogic.UnequipTitle(ctx, playerUUID); xErr != nil {
		return nil, xErr
	}

	resp := xGrpcResult.SuccessWith[*titlepb.UnequipTitleResponse](ctx, "卸下成功")
	return resp, nil
}
