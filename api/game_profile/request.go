package apiGameProfile

type UpdateGameProfileGroupRequest struct {
	GroupName string `json:"group_name" binding:"required"`
}
