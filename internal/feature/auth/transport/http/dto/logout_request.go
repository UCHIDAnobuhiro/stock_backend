package dto

// LogoutReq represents the request for logout.
type LogoutReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
