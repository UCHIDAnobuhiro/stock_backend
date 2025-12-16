// Package handler provides HTTP handlers for the auth feature.
package handler

import (
	"context"
	"log/slog"
	"net/http"

	"stock_backend/internal/feature/auth/transport/http/dto"
	"stock_backend/internal/feature/auth/usecase"

	"github.com/gin-gonic/gin"
)

// AuthUsecase defines use cases for authentication operations.
// Following Go convention: interfaces are defined by the consumer (handler), not the provider (usecase).
type AuthUsecase interface {
	// Signup registers a new user with the given email and password.
	Signup(ctx context.Context, email, password string) error
	// Login authenticates a user and returns tokens on success.
	Login(ctx context.Context, email, password, userAgent, ipAddress string) (*usecase.LoginResult, error)
	// RefreshToken validates a refresh token and issues new tokens.
	RefreshToken(ctx context.Context, refreshToken, userAgent, ipAddress string) (*usecase.RefreshResult, error)
	// Logout revokes a refresh token.
	Logout(ctx context.Context, refreshToken string) error
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
		// Don't expose the actual error to prevent user enumeration attacks
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
// - Returns 200 with access token and refresh token on successful authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("login validation failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password, userAgent, ipAddress)
	if err != nil {
		// Don't expose the actual error to prevent user enumeration attacks
		slog.Warn("login failed", "error", err, "email", req.Email, "remote_addr", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	slog.Info("user login successful", "email", req.Email, "remote_addr", c.ClientIP())
	c.JSON(http.StatusOK, dto.LoginRes{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    result.TokenType,
	})
}

// Refresh handles the token refresh API endpoint.
// - Binds the request JSON to RefreshReq
// - Returns 400 on validation errors
// - Returns 401 on invalid refresh token
// - Returns 200 with new access token and refresh token on success
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("refresh validation failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	userAgent := c.Request.UserAgent()
	ipAddress := c.ClientIP()

	result, err := h.auth.RefreshToken(c.Request.Context(), req.RefreshToken, userAgent, ipAddress)
	if err != nil {
		slog.Warn("refresh failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	slog.Info("token refresh successful", "remote_addr", c.ClientIP())
	c.JSON(http.StatusOK, dto.RefreshRes{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

// Logout handles the logout API endpoint.
// - Binds the request JSON to LogoutReq
// - Returns 400 on validation errors
// - Returns 200 on success (always returns success to prevent token enumeration)
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("logout validation failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Always return success to prevent token enumeration
	_ = h.auth.Logout(c.Request.Context(), req.RefreshToken)
	slog.Info("user logout", "remote_addr", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
