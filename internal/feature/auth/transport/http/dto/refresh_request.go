package dto

// RefreshReq represents the request for token refresh.
type RefreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshRes represents the response for a successful token refresh.
type RefreshRes struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}
