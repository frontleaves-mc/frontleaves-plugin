package logic

import (
	"context"
	"time"

	xError "github.com/bamboo-services/bamboo-base-go/common/error"
	xLog "github.com/bamboo-services/bamboo-base-go/common/log"
	xCtxUtil "github.com/bamboo-services/bamboo-base-go/common/utility/context"
	xSnowflake "github.com/bamboo-services/bamboo-base-go/common/snowflake"
	apiAnnouncement "github.com/frontleaves-mc/frontleaves-plugin/api/announcement"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/entity"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/repository"
	"github.com/frontleaves-mc/frontleaves-plugin/internal/util"
)

type announcementRepo struct {
	announcement *repository.AnnouncementRepo
}

type AnnouncementLogic struct {
	logic
	repo announcementRepo
}

func NewAnnouncementLogic(ctx context.Context) *AnnouncementLogic {
	db := xCtxUtil.MustGetDB(ctx)
	rdb := xCtxUtil.MustGetRDB(ctx)

	return &AnnouncementLogic{
		logic: logic{
			db:  db,
			rdb: rdb,
			log: xLog.WithName(xLog.NamedLOGC, "AnnouncementLogic"),
		},
		repo: announcementRepo{
			announcement: repository.NewAnnouncementRepo(db),
		},
	}
}

func (l *AnnouncementLogic) CreateAnnouncement(ctx context.Context, title, content string, annType entity.AnnouncementType) (*apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "CreateAnnouncement - 创建公告")

	announcement := &entity.Announcement{
		Title:   title,
		Content: content,
		Type:    annType,
		Status:  entity.AnnouncementStatusDraft,
	}

	if xErr := l.repo.announcement.Create(ctx, announcement); xErr != nil {
		return nil, xErr
	}

	return l.toAnnouncementResponse(announcement), nil
}

func (l *AnnouncementLogic) UpdateAnnouncement(ctx context.Context, id xSnowflake.SnowflakeID, title, content string, annType entity.AnnouncementType) (*apiAnnouncement.AnnouncementResponse, *xError.Error) {
	l.log.Info(ctx, "UpdateAnnouncement - 更新公告")

	announcement, xErr := l.repo.announcement.GetByID(ctx, id)
	if xErr != nil {
		return nil, xErr
	}

	announcement.Title = title
	announcement.Content = content
	announcement.Type = annType

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

func (l *AnnouncementLogic) toAnnouncementResponse(a *entity.Announcement) *apiAnnouncement.AnnouncementResponse {
	return &apiAnnouncement.AnnouncementResponse{
		ID:          a.ID.String(),
		Title:       a.Title,
		Content:     a.Content,
		Type:        int16(a.Type),
		Status:      int16(a.Status),
		PublishedAt: a.PublishedAt,
		CreatedAt:   a.CreatedAt,
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
