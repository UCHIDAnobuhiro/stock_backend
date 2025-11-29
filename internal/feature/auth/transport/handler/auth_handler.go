// Package handler provides HTTP handlers for the auth feature.
package handler

import (
	"net/http"

	"stock_backend/internal/feature/auth/transport/http/dto"
	"stock_backend/internal/feature/auth/usecase"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles HTTP requests for authentication operations.
// It depends on the AuthUsecase from the usecase layer and handles JSON requests/responses.
type AuthHandler struct {
	auth usecase.AuthUsecase
}

// NewAuthHandler creates a new AuthHandler instance.
// This is a constructor for dependency injection, injecting AuthUsecase from outside.
func NewAuthHandler(auth usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Signup handles the user registration API endpoint.
// - Binds the request JSON to SignupReq
// - Returns 400 on validation errors
// - Returns 409 on user creation failure (e.g., duplicate email)
// - Returns 201 on success
func (h *AuthHandler) Signup(c *gin.Context) {
	var req dto.SignupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.auth.Signup(req.Email, req.Password); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "ok"})
}

// Login handles the user login API endpoint.
// - Binds the request JSON to LoginReq
// - Returns 400 on validation errors
// - Returns 401 on authentication failure
// - Returns 200 with JWT token on successful authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}
