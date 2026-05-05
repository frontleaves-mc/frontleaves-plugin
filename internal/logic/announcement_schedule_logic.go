package logic

import (
	"context"
	"strconv"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	apiAnnouncementSchedule "github.com/frontleaves-mc/frontleaves-plugin/api/announcement_schedule"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
)

type scheduleRepo struct {
	schedule     *repository.AnnouncementScheduleRepo
	item         *repository.AnnouncementScheduleItemRepo
	announcement *repository.AnnouncementRepo
}

type AnnouncementScheduleLogic struct {
	logic
	repo scheduleRepo
}

func NewAnnouncementScheduleLogic(ctx context.Context) *AnnouncementScheduleLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &AnnouncementScheduleLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "AnnouncementScheduleLogic"),
		},
		repo: scheduleRepo{
			schedule:     repository.NewAnnouncementScheduleRepo(db),
			item:         repository.NewAnnouncementScheduleItemRepo(db),
			announcement: repository.NewAnnouncementRepo(db),
		},
	}
}

// CreateSchedule 创建公告调度
func (l *AnnouncementScheduleLogic) CreateSchedule(ctx context.Context, name string, mode entity.ScheduleMode, intervalSeconds int, items []apiAnnouncementSchedule.ScheduleItemInput) (*apiAnnouncementSchedule.ScheduleResponse, *xError.Error) {
	l.log.Info(ctx, "CreateSchedule - 创建公告调度")

	// 校验调度项不能为空
	if len(items) == 0 {
		return nil, xError.NewError(nil, xError.ParameterError, "调度项不能为空", true, nil)
	}

	// 固定间隔模式需要校验间隔秒数
	if mode == entity.ScheduleModeFixedInterval && intervalSeconds <= 0 {
		return nil, xError.NewError(nil, xError.ParameterError, "固定间隔模式需要设置大于0的间隔秒数", true, nil)
	}

	// 创建调度实体
	schedule := &entity.AnnouncementSchedule{
		Name:            name,
		Mode:            mode,
		IntervalSeconds: intervalSeconds,
		IsActive:        false,
		Status:          entity.AnnouncementStatusDraft,
	}

	if xErr := l.repo.schedule.Create(ctx, schedule); xErr != nil {
		return nil, xErr
	}

	// 创建调度项
	scheduleItems, xErr := l.buildScheduleItems(ctx, schedule.ID, items)
	if xErr != nil {
		return nil, xErr
	}
	if xErr := l.repo.item.CreateItems(ctx, scheduleItems); xErr != nil {
		return nil, xErr
	}

	return l.toScheduleResponse(ctx, schedule, scheduleItems)
}

// UpdateSchedule 更新公告调度
func (l *AnnouncementScheduleLogic) UpdateSchedule(ctx context.Context, id xSnowflake.SnowflakeID, name string, mode entity.ScheduleMode, intervalSeconds int, items []apiAnnouncementSchedule.ScheduleItemInput) (*apiAnnouncementSchedule.ScheduleResponse, *xError.Error) {
	l.log.Info(ctx, "UpdateSchedule - 更新公告调度")

	schedule, xErr := l.repo.schedule.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	// 校验调度项不能为空
	if len(items) == 0 {
		return nil, xError.NewError(nil, xError.ParameterError, "调度项不能为空", true, nil)
	}

	// 固定间隔模式需要校验间隔秒数
	if mode == entity.ScheduleModeFixedInterval && intervalSeconds <= 0 {
		return nil, xError.NewError(nil, xError.ParameterError, "固定间隔模式需要设置大于0的间隔秒数", true, nil)
	}

	// 更新调度字段
	schedule.Name = name
	schedule.Mode = mode
	schedule.IntervalSeconds = intervalSeconds

	if xErr := l.repo.schedule.Update(ctx, schedule); xErr != nil {
		return nil, xErr
	}

	// 删除旧调度项，创建新调度项
	if xErr := l.repo.item.DeleteByScheduleID(ctx, id); xErr != nil {
		return nil, xErr
	}

	scheduleItems, xErr := l.buildScheduleItems(ctx, schedule.ID, items)
	if xErr != nil {
		return nil, xErr
	}
	if xErr := l.repo.item.CreateItems(ctx, scheduleItems); xErr != nil {
		return nil, xErr
	}

	return l.toScheduleResponse(ctx, schedule, scheduleItems)
}

// DeleteSchedule 删除公告调度
func (l *AnnouncementScheduleLogic) DeleteSchedule(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "DeleteSchedule - 删除公告调度")

	schedule, xErr := l.repo.schedule.GetByID(ctx, id)
	if xErr != nil {
		return xErr
	}

	// 活动调度不允许删除
	if schedule.IsActive {
		return xError.NewError(nil, xError.ParameterError, "活动调度无法删除，请先停用", true, nil)
	}

	// 先删除调度项，再删除调度
	if xErr := l.repo.item.DeleteByScheduleID(ctx, id); xErr != nil {
		return xErr
	}

	return l.repo.schedule.Delete(ctx, id)
}

// GetSchedule 查询公告调度
func (l *AnnouncementScheduleLogic) GetSchedule(ctx context.Context, id xSnowflake.SnowflakeID) (*apiAnnouncementSchedule.ScheduleResponse, *xError.Error) {
	l.log.Info(ctx, "GetSchedule - 查询公告调度")

	schedule, xErr := l.repo.schedule.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	items, xErr := l.repo.item.GetByScheduleID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	return l.toScheduleResponse(ctx, schedule, items)
}

// ListSchedules 查询公告调度列表
func (l *AnnouncementScheduleLogic) ListSchedules(ctx context.Context, page, pageSize int) ([]apiAnnouncementSchedule.ScheduleResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListSchedules - 查询公告调度列表")

	schedules, total, xErr := l.repo.schedule.List(ctx, page, pageSize)
	if xErr != nil {
		return nil, 0, xErr
	}

	var resp []apiAnnouncementSchedule.ScheduleResponse
	for _, s := range schedules {
		items, xErr := l.repo.item.GetByScheduleID(ctx, s.ID)
		if xErr != nil {
			return nil, 0, xErr
		}

		scheduleResp, xErr := l.toScheduleResponse(ctx, &s, items)
		if xErr != nil {
			return nil, 0, xErr
		}
		resp = append(resp, *scheduleResp)
	}

	return resp, total, nil
}

// ActivateSchedule 激活公告调度
func (l *AnnouncementScheduleLogic) ActivateSchedule(ctx context.Context, id xSnowflake.SnowflakeID) (*apiAnnouncementSchedule.ScheduleResponse, *xError.Error) {
	l.log.Info(ctx, "ActivateSchedule - 激活公告调度")

	schedule, xErr := l.repo.schedule.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	// 校验调度至少有一个调度项
	items, xErr := l.repo.item.GetByScheduleID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}
	if len(items) == 0 {
		return nil, xError.NewError(nil, xError.ParameterError, "调度至少需要包含一个调度项才能激活", true, nil)
	}

	// 设置为活动调度
	if xErr := l.repo.schedule.SetActiveSchedule(ctx, id); xErr != nil {
		return nil, xErr
	}

	// 更新调度状态为已发布
	schedule.IsActive = true
	schedule.Status = entity.AnnouncementStatusPublished
	if xErr := l.repo.schedule.Update(ctx, schedule); xErr != nil {
		return nil, xErr
	}

	// 启动调度引擎
	if engine := GetGlobalEngine(); engine != nil {
		if xErr := engine.Start(ctx, id); xErr != nil {
			l.log.Error(ctx, "ActivateSchedule - 调度引擎启动失败: "+xErr.Error())
		}
	}

	return l.toScheduleResponse(ctx, schedule, items)
}

// DeactivateSchedule 停用公告调度
func (l *AnnouncementScheduleLogic) DeactivateSchedule(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "DeactivateSchedule - 停用公告调度")

	schedule, xErr := l.repo.schedule.GetByID(ctx, id)
	if xErr != nil {
		return xErr
	}

	// 校验调度当前处于活动状态
	if !schedule.IsActive {
		return xError.NewError(nil, xError.ParameterError, "调度当前未处于活动状态", true, nil)
	}

	// 清除活动调度（设置为 0 表示无活动调度）
	if xErr := l.repo.schedule.SetActiveSchedule(ctx, 0); xErr != nil {
		return xErr
	}

	// 停止调度引擎
	if engine := GetGlobalEngine(); engine != nil {
		_ = engine.Stop()
	}

	return nil
}

// GetActiveScheduleWithItems 获取活动调度及其调度项
func (l *AnnouncementScheduleLogic) GetActiveScheduleWithItems(ctx context.Context) (*entity.AnnouncementSchedule, []entity.AnnouncementScheduleItem, *xError.Error) {
	l.log.Info(ctx, "GetActiveScheduleWithItems - 获取活动调度及其调度项")

	schedule, xErr := l.repo.schedule.GetActiveSchedule(ctx)
	if xErr != nil {
		return nil, nil, xErr
	}
	if schedule == nil {
		return nil, nil, nil
	}

	items, xErr := l.repo.item.GetByScheduleID(ctx, schedule.ID)
	if xErr != nil {
		return nil, nil, xErr
	}

	return schedule, items, nil
}

// buildScheduleItems 将输入的调度项转换为实体列表
func (l *AnnouncementScheduleLogic) buildScheduleItems(ctx context.Context, scheduleID xSnowflake.SnowflakeID, items []apiAnnouncementSchedule.ScheduleItemInput) ([]entity.AnnouncementScheduleItem, *xError.Error) {
	var scheduleItems []entity.AnnouncementScheduleItem
	for _, input := range items {
		// 解析 AnnouncementID 字符串为 SnowflakeID
		parsed, err := strconv.ParseInt(input.AnnouncementID, 10, 64)
		if err != nil {
			return nil, xError.NewError(nil, xError.ParameterError, xError.ErrMessage("公告ID格式无效: "+input.AnnouncementID), true, err)
		}

		// 校验公告是否存在
		_, xErr := l.repo.announcement.GetByID(ctx, xSnowflake.SnowflakeID(parsed))
		if xErr != nil {
			return nil, xError.NewError(nil, xError.ParameterError, xError.ErrMessage("公告不存在: "+input.AnnouncementID), true, nil)
		}

		scheduleItems = append(scheduleItems, entity.AnnouncementScheduleItem{
			ScheduleID:     scheduleID,
			AnnouncementID: xSnowflake.SnowflakeID(parsed),
			SortOrder:      input.SortOrder,
			DelaySeconds:   input.DelaySeconds,
		})
	}
	return scheduleItems, nil
}

// toScheduleResponse 将调度实体和调度项转换为响应
func (l *AnnouncementScheduleLogic) toScheduleResponse(ctx context.Context, schedule *entity.AnnouncementSchedule, items []entity.AnnouncementScheduleItem) (*apiAnnouncementSchedule.ScheduleResponse, *xError.Error) {
	resp := &apiAnnouncementSchedule.ScheduleResponse{
		ID:              schedule.ID.String(),
		Name:            schedule.Name,
		Mode:            int16(schedule.Mode),
		IntervalSeconds: schedule.IntervalSeconds,
		IsActive:        schedule.IsActive,
		Status:          int16(schedule.Status),
		Items:           []apiAnnouncementSchedule.ScheduleItemResponse{},
		CreatedAt:       schedule.CreatedAt,
	}

	for _, item := range items {
		// 查询公告标题
		announcementTitle := ""
		announcement, xErr := l.repo.announcement.GetByID(ctx, item.AnnouncementID)
		if xErr == nil && announcement != nil {
			announcementTitle = announcement.Title
		}

		resp.Items = append(resp.Items, apiAnnouncementSchedule.ScheduleItemResponse{
			AnnouncementID:    item.AnnouncementID.String(),
			AnnouncementTitle: announcementTitle,
			SortOrder:         item.SortOrder,
			DelaySeconds:      item.DelaySeconds,
		})
	}

	return resp, nil
}
