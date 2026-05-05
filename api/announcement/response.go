package apiAnnouncement

import "time"

type AnnouncementResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Type        int16      `json:"type"`
	Status      int16      `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AnnouncementListItemResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Desc        string     `json:"desc"`
	Type        int16      `json:"type"`
	Status      int16      `json:"status"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
