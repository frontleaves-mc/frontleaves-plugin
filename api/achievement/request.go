package apiAchievement

import "encoding/json"

type CreateAchievementRequest struct {
	Name             string          `json:"name" binding:"required"`
	Description      string          `json:"description" binding:"required"`
	Type             int16           `json:"type" binding:"required,oneof=1 2 3 4"`
	ConditionKey     string          `json:"condition_key" binding:"required"`
	ConditionParams  json.RawMessage `json:"condition_params,omitempty"`
	RewardConfig     json.RawMessage `json:"reward_config,omitempty"`
	SortOrder        int             `json:"sort_order,omitempty"`
}

type UpdateAchievementRequest struct {
	Name             string          `json:"name" binding:"required"`
	Description      string          `json:"description" binding:"required"`
	Type             int16           `json:"type" binding:"required,oneof=1 2 3 4"`
	ConditionKey     string          `json:"condition_key" binding:"required"`
	ConditionParams  json.RawMessage `json:"condition_params,omitempty"`
	RewardConfig     json.RawMessage `json:"reward_config,omitempty"`
	SortOrder        int             `json:"sort_order,omitempty"`
	IsActive         *bool           `json:"is_active,omitempty"`
}

type GrantAchievementRequest struct {
	PlayerUUID string `json:"player_uuid" binding:"required"`
}
