package logic

import (
	"context"
	"strconv"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	apiAnnouncement "github.com/frontleaves-mc/frontleaves-plugin/api/announcement"
	bConst "github.com/frontleaves-mc/frontleaves-plugin/internal/constant"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/util"
)

type announcementRepo struct {
	announcement  *repository.AnnouncementRepo
	configRepo    *repository.ConfigRepository
	configLoader  *SchedulerConfigLoader
}

type AnnouncementLogic struct {
	logic
	repo announcementRepo
}

func NewAnnouncementLogic(ctx context.Context) *AnnouncementLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	configRepo := repository.NewConfigRepository()
	configLoader := NewSchedulerConfigLoader(configRepo)
	_ = configLoader.Load(ctx)

	return &AnnouncementLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "AnnouncementLogic"),
		},
		repo: announcementRepo{
			announcement: repository.NewAnnouncementRepo(db),
			configRepo:   configRepo,
			configLoader: configLoader,
		},
	}
}

func (l *AnnouncementLogic) CreateAnnouncement(ctx context.Context, title, content string, annType entity.AnnouncementType, scheduleOrder *int, delaySeconds int) (*apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "CreateAnnouncement - 创建公告")

	announcement := &entity.Announcement{
		Title:         title,
		Content:       content,
		Type:          annType,
		Status:        entity.AnnouncementStatusDraft,
		ScheduleOrder: scheduleOrder,
		DelaySeconds:  delaySeconds,
	}

	if xErr := l.repo.announcement.Create(ctx, announcement); xErr != nil {
		return nil, xErr
	}

	return l.toAnnouncementResponse(announcement), nil
}

func (l *AnnouncementLogic) UpdateAnnouncement(ctx context.Context, id xSnowflake.SnowflakeID, title, content string, annType entity.AnnouncementType, scheduleOrder *int, delaySeconds int) (*apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "UpdateAnnouncement - 更新公告")

	announcement, xErr := l.repo.announcement.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	announcement.Title = title
	announcement.Content = content
	announcement.Type = annType
	announcement.ScheduleOrder = scheduleOrder
	announcement.DelaySeconds = delaySeconds

	if xErr := l.repo.announcement.Update(ctx, announcement); xErr != nil {
		return nil, xErr
	}

	return l.toAnnouncementResponse(announcement), nil
}

func (l *AnnouncementLogic) DeleteAnnouncement(ctx context.Context, id xSnowflake.SnowflakeID) *xError.Error {
	l.log.Info(ctx, "DeleteAnnouncement - 删除公告")
	return l.repo.announcement.Delete(ctx, id)
}

func (l *AnnouncementLogic) GetAnnouncement(ctx context.Context, id xSnowflake.SnowflakeID) (*apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "GetAnnouncement - 查询公告")
	announcement, xErr := l.repo.announcement.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}
	return l.toAnnouncementResponse(announcement), nil
}

func (l *AnnouncementLogic) ListAnnouncements(ctx context.Context, page, pageSize int, annType *int16, status *int16) ([]apiAnnouncement.AnnouncementListItemResponse, int64, *xError.Error) {
	l.log.Info(ctx, "ListAnnouncements - 查询公告列表")

	announcements, total, xErr := l.repo.announcement.List(ctx, page, pageSize, annType, status)
	if xErr != nil {
		return nil, 0, xErr
	}

	var resp []apiAnnouncement.AnnouncementListItemResponse
	for _, a := range announcements {
		resp = append(resp, *l.toAnnouncementListItemResponse(&a))
	}
	return resp, total, nil
}

func (l *AnnouncementLogic) PublishAnnouncement(ctx context.Context, id xSnowflake.SnowflakeID) (*apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "PublishAnnouncement - 发布公告")

	announcement, xErr := l.repo.announcement.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	if announcement.Status == entity.AnnouncementStatusPublished {
		return nil, xError.NewError(nil, xError.ParameterError, "公告已发布", true, nil)
	}

	now := time.Now()
	announcement.Status = entity.AnnouncementStatusPublished
	announcement.PublishedAt = &now

	if xErr := l.repo.announcement.Update(ctx, announcement); xErr != nil {
		return nil, xErr
	}

	return l.toAnnouncementResponse(announcement), nil
}

func (l *AnnouncementLogic) OfflineAnnouncement(ctx context.Context, id xSnowflake.SnowflakeID) (*apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "OfflineAnnouncement - 下线公告")

	announcement, xErr := l.repo.announcement.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	if announcement.Status == entity.AnnouncementStatusDraft {
		return nil, xError.NewError(nil, xError.ParameterError, "草稿状态公告无法下线", true, nil)
	}

	announcement.Status = entity.AnnouncementStatusOffline

	if xErr := l.repo.announcement.Update(ctx, announcement); xErr != nil {
		return nil, xErr
	}

	return l.toAnnouncementResponse(announcement), nil
}

func (l *AnnouncementLogic) GetPublishedGlobalAnnouncements(ctx context.Context) ([]entity.Announcement, *xError.Error) {
	l.log.Info(ctx, "GetPublishedGlobalAnnouncements - 查询已发布全局公告")
	return l.repo.announcement.GetPublishedGlobal(ctx)
}

func (l *AnnouncementLogic) SetAnnouncementScheduleOrder(ctx context.Context, id xSnowflake.SnowflakeID, order *int, delaySeconds int) *xError.Error {
	l.log.Info(ctx, "SetAnnouncementScheduleOrder - 设置公告调度顺序")

	announcement, xErr := l.repo.announcement.GetByID(ctx, id)
	if xErr != nil {
		return xErr
	}

	announcement.ScheduleOrder = order
	announcement.DelaySeconds = delaySeconds

	if xErr := l.repo.announcement.Update(ctx, announcement); xErr != nil {
		return xErr
	}

	return nil
}

func (l *AnnouncementLogic) ListScheduledAnnouncements(ctx context.Context) ([]apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "ListScheduledAnnouncements - 获取调度公告列表")

	announcements, xErr := l.repo.announcement.GetScheduledAnnouncements(ctx)
	if xErr != nil {
		return nil, xErr
	}

	var resp []apiAnnouncement.AnnouncementResponse
	for _, a := range announcements {
		resp = append(resp, *l.toAnnouncementResponse(&a))
	}
	return resp, nil
}

func (l *AnnouncementLogic) GetSchedulerConfig(ctx context.Context) (*apiAnnouncement.GetSchedulerConfigResponse, *xError.Error) {
	l.log.Info(ctx, "GetSchedulerConfig - 获取公告调度配置")
	snapshot := l.repo.configLoader.Get()
	return &apiAnnouncement.GetSchedulerConfigResponse{
		Mode:            int16(snapshot.Mode),
		IntervalSeconds: snapshot.IntervalSeconds,
		IsEnabled:       snapshot.IsEnabled,
	}, nil
}

func (l *AnnouncementLogic) SaveSchedulerConfig(ctx context.Context, mode int16, intervalSeconds int) *xError.Error {
	l.log.Info(ctx, "SaveSchedulerConfig - 保存公告调度配置")
	if xErr := l.repo.configRepo.Set(ctx, bConst.SchedulerConfigNamespace, bConst.SchedulerConfigMode, strconv.Itoa(int(mode))); xErr != nil {
		return xErr
	}
	if xErr := l.repo.configRepo.Set(ctx, bConst.SchedulerConfigNamespace, bConst.SchedulerConfigIntervalSeconds, strconv.Itoa(intervalSeconds)); xErr != nil {
		return xErr
	}
	if xErr := l.repo.configLoader.Load(ctx); xErr != nil {
		return xErr
	}
	return nil
}

func (l *AnnouncementLogic) EnableScheduler(ctx context.Context) *xError.Error {
	l.log.Info(ctx, "EnableScheduler - 启用公告调度")
	// 先确保 mode 和 interval 存在（如果不存在，loader 会有默认值）
	snapshot := l.repo.configLoader.Get()
	if xErr := l.repo.configRepo.Set(ctx, bConst.SchedulerConfigNamespace, bConst.SchedulerConfigMode, strconv.Itoa(int(snapshot.Mode))); xErr != nil {
		return xErr
	}
	if xErr := l.repo.configRepo.Set(ctx, bConst.SchedulerConfigNamespace, bConst.SchedulerConfigIntervalSeconds, strconv.Itoa(snapshot.IntervalSeconds)); xErr != nil {
		return xErr
	}
	if xErr := l.repo.configRepo.Set(ctx, bConst.SchedulerConfigNamespace, bConst.SchedulerConfigIsEnabled, "true"); xErr != nil {
		return xErr
	}
	if xErr := l.repo.configLoader.Load(ctx); xErr != nil {
		return xErr
	}
	engine := GetGlobalEngine()
	if engine != nil && engine.IsRunning() {
		if xErr := engine.Restart(ctx); xErr != nil {
			return xErr
		}
	}
	return nil
}

func (l *AnnouncementLogic) DisableScheduler(ctx context.Context) *xError.Error {
	l.log.Info(ctx, "DisableScheduler - 停用公告调度")
	if xErr := l.repo.configRepo.Set(ctx, bConst.SchedulerConfigNamespace, bConst.SchedulerConfigIsEnabled, "false"); xErr != nil {
		return xErr
	}
	if xErr := l.repo.configLoader.Load(ctx); xErr != nil {
		return xErr
	}
	engine := GetGlobalEngine()
	if engine != nil {
		_ = engine.Stop()
	}
	return nil
}

func (l *AnnouncementLogic) toAnnouncementResponse(a *entity.Announcement) *apiAnnouncement.AnnouncementResponse {
	return &apiAnnouncement.AnnouncementResponse{
		ID:            a.ID.String(),
		Title:         a.Title,
		Content:       a.Content,
		Type:          int16(a.Type),
		Status:        int16(a.Status),
		ScheduleOrder: a.ScheduleOrder,
		DelaySeconds:  a.DelaySeconds,
		PublishedAt:   a.PublishedAt,
		CreatedAt:     a.CreatedAt,
	}
}

func (l *AnnouncementLogic) toAnnouncementListItemResponse(a *entity.Announcement) *apiAnnouncement.AnnouncementListItemResponse {
	return &apiAnnouncement.AnnouncementListItemResponse{
		ID:          a.ID.String(),
		Title:       a.Title,
		Desc:        util.TruncateDescription(a.Content, 100),
		Type:        int16(a.Type),
		Status:      int16(a.Status),
		PublishedAt: a.PublishedAt,
		CreatedAt:   a.CreatedAt,
	}
}
