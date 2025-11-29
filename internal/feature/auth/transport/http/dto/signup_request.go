// Package dto defines data transfer objects for the auth feature's HTTP transport layer.
package dto

// SignupReq represents the request body for the /signup endpoint.
// It uses Gin's binding tags for validation (required, email format, password length).
type SignupReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}
