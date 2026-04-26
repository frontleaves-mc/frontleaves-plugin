package apiGameProfile

import "time"

type GameProfileResponse struct {
	UUID       string    `json:"uuid"`
	Username   string    `json:"username"`
	GroupName  string    `json:"group_name"`
	ReportedAt time.Time `json:"reported_at"`
}
