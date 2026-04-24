package apiAchievement

import (
	"encoding/json"
	"time"
)

type AchievementResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Type            int16           `json:"type"`
	ConditionKey    string          `json:"condition_key"`
	ConditionParams json.RawMessage `json:"condition_params,omitempty"`
	RewardConfig    json.RawMessage `json:"reward_config,omitempty"`
	SortOrder       int             `json:"sort_order"`
	IsActive        bool            `json:"is_active"`
}

type PlayerAchievementResponse struct {
	AchievementResponse
	Status      int16      `json:"status"`
	Progress    int64      `json:"progress"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type AchievementClaimResponse struct {
	AchievementID string `json:"achievement_id"`
	TitleClaimed  bool   `json:"title_claimed"`
}
