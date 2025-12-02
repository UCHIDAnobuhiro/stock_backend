// Package handler provides HTTP handlers for the auth feature.
package handler

import (
	"context"
	"log/slog"
	"net/http"

	"stock_backend/internal/feature/auth/transport/http/dto"

	"github.com/gin-gonic/gin"
)

// AuthUsecase defines use cases for authentication operations.
// Following Go convention: interfaces are defined by the consumer (handler), not the provider (usecase).
type AuthUsecase interface {
	// Signup registers a new user with the given email and password.
	Signup(ctx context.Context, email, password string) error
	// Login authenticates a user and returns a JWT token on success.
	Login(ctx context.Context, email, password string) (string, error)
}

// AuthHandler handles HTTP requests for authentication operations.
// It depends on the AuthUsecase interface and handles JSON requests/responses.
type AuthHandler struct {
	auth AuthUsecase
}

// NewAuthHandler creates a new AuthHandler instance.
// This is a constructor for dependency injection, injecting AuthUsecase from outside.
func NewAuthHandler(auth AuthUsecase) *AuthHandler {
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
		slog.Warn("signup validation failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.auth.Signup(c.Request.Context(), req.Email, req.Password); err != nil {
		slog.Warn("signup failed", "error", err, "email", req.Email, "remote_addr", c.ClientIP())
		c.JSON(http.StatusConflict, gin.H{"error": "signup failed"})
		return
	}
	slog.Info("user signup successful", "email", req.Email, "remote_addr", c.ClientIP())
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
		slog.Warn("login validation failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// Don't log the actual error message to avoid leaking user existence info
		slog.Warn("login failed", "email", req.Email, "remote_addr", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	slog.Info("user login successful", "email", req.Email, "remote_addr", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"token": token})
}
