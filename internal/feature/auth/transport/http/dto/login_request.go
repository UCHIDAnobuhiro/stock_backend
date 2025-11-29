// Package dto defines data transfer objects for the auth feature's HTTP transport layer.
package dto

// LoginReq represents the request body for the /login endpoint.
// It includes validation for required fields and email format.
type LoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
