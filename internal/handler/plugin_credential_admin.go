package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiPC "github.com/frontleaves-mc/frontleaves-plugin/api/plugin_credential"
)

type PluginCredentialAdminHandler handler

func NewPluginCredentialAdminHandler(ctx context.Context) *PluginCredentialAdminHandler {
	return NewHandler[PluginCredentialAdminHandler](ctx, "PluginCredentialAdminHandler")
}

// CreatePluginCredential 创建插件凭证
//
// @Summary     [超管] 创建插件凭证
// @Description 创建新的插件凭证，生成唯一密钥，返回完整密钥（仅在创建和重置时可见）
// @Tags        超管-插件密钥接口
// @Accept      json
// @Produce     json
// @Param       request  body  apiPC.CreatePluginCredentialRequest  true  "创建插件凭证请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiPC.PluginCredentialResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/plugin-credentials [POST]
func (h *PluginCredentialAdminHandler) CreatePluginCredential(ctx *gin.Context) {
	h.log.Info(ctx, "CreatePluginCredential - 创建插件凭证")

	var req apiPC.CreatePluginCredentialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	resp, xErr := h.service.pluginCredentialLogic.Create(ctx.Request.Context(), req.Name, req.Description)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "创建成功", resp)
}

// UpdatePluginCredential 更新插件凭证
//
// @Summary     [超管] 更新插件凭证描述
// @Description 更新指定插件凭证的描述信息
// @Tags        超管-插件密钥接口
// @Accept      json
// @Produce     json
// @Param       id       path  string                               true  "插件凭证ID"
// @Param       request  body  apiPC.UpdatePluginCredentialRequest  true  "更新插件凭证请求"
// @Success     200  {object}  xBase.BaseResponse{data=apiPC.PluginCredentialResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "插件凭证不存在"
// @Router      /admin/plugin-credentials/:id [PUT]
func (h *PluginCredentialAdminHandler) UpdatePluginCredential(ctx *gin.Context) {
	h.log.Info(ctx, "UpdatePluginCredential - 更新插件凭证")

	id, xErr := h.parsePluginCredentialID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var req apiPC.UpdatePluginCredentialRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "请求参数错误", true, err))
		return
	}

	resp, xErr := h.service.pluginCredentialLogic.UpdateDescription(ctx.Request.Context(), id, req.Description)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "更新成功", resp)
}

// ResetPluginCredentialKey 重置插件密钥
//
// @Summary     [超管] 重置插件密钥
// @Description 重置指定插件凭证的密钥，返回新的完整密钥
// @Tags        超管-插件密钥接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "插件凭证ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiPC.PluginCredentialResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "插件凭证不存在"
// @Router      /admin/plugin-credentials/:id/reset-key [PUT]
func (h *PluginCredentialAdminHandler) ResetPluginCredentialKey(ctx *gin.Context) {
	h.log.Info(ctx, "ResetPluginCredentialKey - 重置插件密钥")

	id, xErr := h.parsePluginCredentialID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	resp, xErr := h.service.pluginCredentialLogic.ResetSecretKey(ctx.Request.Context(), id)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "重置密钥成功", resp)
}

// DeletePluginCredential 删除插件凭证
//
// @Summary     [超管] 删除插件凭证
// @Description 硬删除指定的插件凭证
// @Tags        超管-插件密钥接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "插件凭证ID"
// @Success     200  {object}  xBase.BaseResponse  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "插件凭证不存在"
// @Router      /admin/plugin-credentials/:id [DELETE]
func (h *PluginCredentialAdminHandler) DeletePluginCredential(ctx *gin.Context) {
	h.log.Info(ctx, "DeletePluginCredential - 删除插件凭证")

	id, xErr := h.parsePluginCredentialID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	if xErr := h.service.pluginCredentialLogic.Delete(ctx.Request.Context(), id); xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "删除成功", nil)
}

// GetPluginCredential 查询插件凭证详情
//
// @Summary     [超管] 查询插件凭证详情
// @Description 根据插件凭证 ID 查询详情，密钥脱敏展示
// @Tags        超管-插件密钥接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "插件凭证ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiPC.PluginCredentialResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "插件凭证不存在"
// @Router      /admin/plugin-credentials/:id [GET]
func (h *PluginCredentialAdminHandler) GetPluginCredential(ctx *gin.Context) {
	h.log.Info(ctx, "GetPluginCredential - 查询插件凭证详情")

	id, xErr := h.parsePluginCredentialID(ctx)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	resp, xErr := h.service.pluginCredentialLogic.GetByID(ctx.Request.Context(), id)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", resp)
}

// ListPluginCredentials 查询插件凭证列表
//
// @Summary     [超管] 查询插件凭证列表
// @Description 分页查询插件凭证列表，所有密钥脱敏展示
// @Tags        超管-插件密钥接口
// @Accept      json
// @Produce     json
// @Param       page       query  int  false  "页码"
// @Param       page_size  query  int  false  "每页数量"
// @Success     200  {object}  xBase.BaseResponse{data=apiPC.PluginCredentialListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/plugin-credentials [GET]
func (h *PluginCredentialAdminHandler) ListPluginCredentials(ctx *gin.Context) {
	h.log.Info(ctx, "ListPluginCredentials - 查询插件凭证列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total, xErr := h.service.pluginCredentialLogic.List(ctx.Request.Context(), page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiPC.PluginCredentialListResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *PluginCredentialAdminHandler) parsePluginCredentialID(ctx *gin.Context) (xSnowflake.SnowflakeID, *xError.Error) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, xError.NewError(nil, xError.ParameterError, "无效的插件凭证 ID", true, err)
	}
	return xSnowflake.SnowflakeID(id), nil
}
