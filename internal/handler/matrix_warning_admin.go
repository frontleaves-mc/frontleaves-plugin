package handler

import (
	"context"
	"strconv"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xResult "github.com/bamboo-services/bamboo-base-go/major/result"
	apiMatrixWarning "github.com/frontleaves-mc/frontleaves-plugin/api/matrix_warning"
	"github.com/gin-gonic/gin"
)

type MatrixWarningAdminHandler handler

func NewMatrixWarningAdminHandler(ctx context.Context) *MatrixWarningAdminHandler {
	return NewHandler[MatrixWarningAdminHandler](ctx, "MatrixWarningAdminHandler")
}

// ListWarnings 查询警告列表
//
// @Summary     [管理] 查询警告列表
// @Description 多条件筛选查询警告列表，支持分页
// @Tags        管理-Matrix警告接口
// @Accept      json
// @Produce     json
// @Param       page            query  int     false  "页码"
// @Param       page_size       query  int     false  "每页数量"
// @Param       player_uuid     query  string  false  "玩家UUID筛选"
// @Param       warning_type    query  string  false  "警告类型筛选"
// @Param       risk_score_min  query  int     false  "最低风险分数"
// @Param       risk_score_max  query  int     false  "最高风险分数"
// @Param       server_name     query  string  false  "服务器名称筛选"
// @Param       start_time      query  string  false  "开始时间"
// @Param       end_time        query  string  false  "结束时间"
// @Success     200  {object}  xBase.BaseResponse{data=apiMatrixWarning.WarningListResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Router      /admin/matrix/warnings [GET]
func (h *MatrixWarningAdminHandler) ListWarnings(ctx *gin.Context) {
	h.log.Info(ctx, "ListWarnings - 查询警告列表")

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	playerUUID := ctx.Query("player_uuid")
	warningType := ctx.Query("warning_type")
	serverName := ctx.Query("server_name")

	var riskScoreMin *int32
	if r := ctx.Query("risk_score_min"); r != "" {
		v, _ := strconv.Atoi(r)
		tv := int32(v)
		riskScoreMin = &tv
	}

	var riskScoreMax *int32
	if r := ctx.Query("risk_score_max"); r != "" {
		v, _ := strconv.Atoi(r)
		tv := int32(v)
		riskScoreMax = &tv
	}

	if riskScoreMin != nil && riskScoreMax != nil && *riskScoreMin > *riskScoreMax {
		riskScoreMin, riskScoreMax = riskScoreMax, riskScoreMin
	}

	var startTime *time.Time
	if s := ctx.Query("start_time"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil {
			startTime = &t
		}
	}

	var endTime *time.Time
	if e := ctx.Query("end_time"); e != "" {
		t, err := time.Parse(time.RFC3339, e)
		if err == nil {
			endTime = &t
		}
	}

	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		startTime, endTime = endTime, startTime
	}

	warnings, total, xErr := h.service.matrixWarningQueryLogic.ListWarnings(ctx.Request.Context(), playerUUID, warningType, serverName, riskScoreMin, riskScoreMax, startTime, endTime, page, pageSize)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	var list []apiMatrixWarning.WarningListItemResponse
	for _, w := range warnings {
		list = append(list, apiMatrixWarning.WarningListItemResponse{
			ID:          w.ID.String(),
			PlayerUUID:  w.PlayerUUID.String(),
			PlayerName:  w.PlayerName,
			ServerName:  w.ServerName,
			WarningType: w.WarningType,
			Description: w.Description,
			RiskScore:   w.RiskScore,
			CreatedAt:   w.CreatedAt,
		})
	}

	xResult.SuccessHasData(ctx, "查询成功", &apiMatrixWarning.WarningListResponse{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetWarning 查询警告详情
//
// @Summary     [管理] 查询警告详情
// @Description 根据警告 ID 查询警告详情
// @Tags        管理-Matrix警告接口
// @Accept      json
// @Produce     json
// @Param       id  path  string  true  "警告ID"
// @Success     200  {object}  xBase.BaseResponse{data=apiMatrixWarning.WarningDetailResponse}  "成功"
// @Failure     400  {object}  xBase.BaseResponse  "请求参数错误"
// @Failure     404  {object}  xBase.BaseResponse  "警告不存在"
// @Router      /admin/matrix/warnings/:id [GET]
func (h *MatrixWarningAdminHandler) GetWarning(ctx *gin.Context) {
	h.log.Info(ctx, "GetWarning - 查询警告详情")

	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = ctx.Error(xError.NewError(nil, xError.ParameterError, "无效的警告 ID", true, err))
		return
	}

	warning, xErr := h.service.matrixWarningQueryLogic.GetWarningByID(ctx.Request.Context(), id)
	if xErr != nil {
		_ = ctx.Error(xErr)
		return
	}

	detail := apiMatrixWarning.WarningDetailResponse{
		ID:          warning.ID.String(),
		PlayerUUID:  warning.PlayerUUID.String(),
		PlayerName:  warning.PlayerName,
		ServerName:  warning.ServerName,
		WarningType: warning.WarningType,
		Description: warning.Description,
		RiskScore:   warning.RiskScore,
		ContextData: warning.ContextData,
		Record:      warning.Record,
		CreatedAt:   warning.CreatedAt,
	}

	xResult.SuccessHasData(ctx, "查询成功", detail)
}