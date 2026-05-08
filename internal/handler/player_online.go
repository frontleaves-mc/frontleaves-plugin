package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type PlayerOnlineHandler handler

func NewPlayerOnlineHandler(ctx context.Context) *PlayerOnlineHandler {
	return NewHandler[PlayerOnlineHandler](ctx, "PlayerOnlineHandler")
}

// GetMyOnlineProfiles 获取当前用户在线游戏账号
//
// @Summary     [用户] 获取在线游戏账号
// @Description 查询当前登录用户所有在线的游戏账号
// @Tags        玩家在线查询接口
// @Accept      json
// @Produce     json
// @Success     200  {object}  xBase.BaseResponse{data=[]apiServer.OnlineGameProfileResponse}  "成功"
// @Failure     401  {object}  xBase.BaseResponse  "未登录"
// @Router      /servers/game-profiles/online/mine [GET]
func (h *PlayerOnlineHandler) GetMyOnlineProfiles(ctx *gin.Context) {
	h.log.Info(ctx, "GetMyOnlineProfiles - 获取当前用户在线游戏账号")

	userInfo, ok := ctx.Request.Context().Value(bConst.CtxAuthUserKey).(*repository.AuthUserInfo)
	if !ok || userInfo == nil {
		_ = ctx.Error(xError.NewError(nil, xError.Unauthorized, "未登录", true))
		return
	}

	userID, err := strconv.ParseInt(userInfo.UserID, 10, 64)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的用户 ID", true, err))
		return
	}

	onlineProfiles, xErr := h.service.serverLogic.GetMyOnlineProfiles(ctx.Request.Context(), xSnowflake.SnowflakeID(userID))
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", onlineProfiles)
}

// CheckPlayerOnline 检查玩家在线状态
//
// @Summary     [用户] 检查玩家在线状态
// @Description 按 UUID 或用户名检查指定玩家是否在线
// @Tags        玩家在线查询接口
// @Accept      json
// @Produce     json
// @Param       uuid      query  string  false  "玩家 UUID"
// @Param       username  query  string  false  "玩家用户名"
// @Success     200  {object}  xBase.BaseResponse{data=apiServer.PlayerOnlineResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "参数错误"
// @Router      /servers/players/online/check [GET]
func (h *PlayerOnlineHandler) CheckPlayerOnline(ctx *gin.Context) {
	h.log.Info(ctx, "CheckPlayerOnline - 检查玩家在线状态")

	playerUUID := ctx.Query("uuid")
	username := ctx.Query("username")

	if playerUUID == "" && username == "" {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请提供 uuid 或 username 参数", true))
		return
	}

	resp, xErr := h.service.serverLogic.CheckPlayerOnline(ctx.Request.Context(), playerUUID, username)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", resp)
}
