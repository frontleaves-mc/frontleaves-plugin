package apiTitle

type CreateTitleRequest struct {
	Name            string  `json:"name" binding:"required"`
	Description     string  `json:"description" binding:"required"`
	Type            int16   `json:"type" binding:"required,oneof=1 2 3"`
	PermissionGroup *string `json:"permission_group,omitempty"`
}

type UpdateTitleRequest struct {
	Name            string  `json:"name" binding:"required"`
	Description     string  `json:"description" binding:"required"`
	Type            int16   `json:"type" binding:"required,oneof=1 2 3"`
	PermissionGroup *string `json:"permission_group,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

type AssignTitleRequest struct {
	PlayerUUID string `json:"player_uuid" binding:"required"`
}

type EquipTitleRequest struct {
	TitleID string `json:"title_id" binding:"required"`
}
