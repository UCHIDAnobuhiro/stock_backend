// Package handler はauthフィーチャーのHTTPハンドラーを提供します。
package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
)

// AuthUsecase は認証操作のユースケースを定義します。
// Goの慣例に従い、インターフェースはプロバイダー（usecase）ではなくコンシューマー（handler）が定義します。
type AuthUsecase interface {
	// Signup は指定されたメールアドレスとパスワードで新規ユーザーを登録します。
	Signup(ctx context.Context, email, password string) error
	// Login はユーザーを認証し、成功時にJWTトークンを返します。
	Login(ctx context.Context, email, password string) (string, error)
}

// AuthHandler は認証操作のHTTPリクエストを処理します。
// AuthUsecaseインターフェースに依存し、JSONリクエスト/レスポンスを処理します。
type AuthHandler struct {
	auth AuthUsecase
}

// NewAuthHandler はAuthHandlerの新しいインスタンスを生成します。
// 依存性注入用のコンストラクタで、外部からAuthUsecaseを注入します。
func NewAuthHandler(auth AuthUsecase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Signup はユーザー登録APIエンドポイントを処理します。
// - リクエストJSONをSignupReqにバインド
// - バリデーションエラー時は400を返却
// - ユーザー作成失敗時（メール重複等）は409を返却
// - 成功時は201を返却
func (h *AuthHandler) Signup(c *gin.Context) {
	var req api.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("signup validation failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request"})
		return
	}
	if err := h.auth.Signup(c.Request.Context(), req.Email, req.Password); err != nil {
		// ユーザー列挙攻撃を防止するため、実際のエラーを公開しない
		slog.Warn("signup failed", "error", err, "email", req.Email, "remote_addr", c.ClientIP())
		c.JSON(http.StatusConflict, api.ErrorResponse{Error: "signup failed"})
		return
	}
	slog.Info("user signup successful", "email", req.Email, "remote_addr", c.ClientIP())
	c.JSON(http.StatusCreated, api.MessageResponse{Message: "ok"})
}

// Login はユーザーログインAPIエンドポイントを処理します。
// - リクエストJSONをLoginReqにバインド
// - バリデーションエラー時は400を返却
// - 認証失敗時は401を返却
// - 認証成功時はJWTトークン付きで200を返却
func (h *AuthHandler) Login(c *gin.Context) {
	var req api.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("login validation failed", "error", err, "remote_addr", c.ClientIP())
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid request"})
		return
	}
	token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// ユーザー列挙攻撃を防止するため、実際のエラーを公開しない
		slog.Warn("login failed", "error", err, "email", req.Email, "remote_addr", c.ClientIP())
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: "invalid email or password"})
		return
	}
	slog.Info("user login successful", "email", req.Email, "remote_addr", c.ClientIP())
	c.JSON(http.StatusOK, api.TokenResponse{Token: token})
}
