package apiAnnouncement

type CreateAnnouncementRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Type    int16  `json:"type" binding:"required,oneof=1 2"`
}

type UpdateAnnouncementRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	Type    int16  `json:"type" binding:"required,oneof=1 2"`
}

type PublishAnnouncementRequest struct{}

type OfflineAnnouncementRequest struct{}
