package apiPlayer

type UpdatePlayerGroupRequest struct {
	GroupName string `json:"group_name" binding:"required"`
}
