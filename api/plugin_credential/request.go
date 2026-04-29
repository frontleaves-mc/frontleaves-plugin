package apiPC

type CreatePluginCredentialRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type UpdatePluginCredentialRequest struct {
	Description string `json:"description" binding:"required"`
}
