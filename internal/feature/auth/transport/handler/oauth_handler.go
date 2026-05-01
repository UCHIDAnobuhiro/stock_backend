// Package handler はauthフィーチャーのHTTPハンドラーを提供します。
package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
	"stock_backend/internal/feature/auth/usecase"
	"stock_backend/internal/platform/csrf"
)

// OAuthUsecase はOAuth2認証フローのユースケースインターフェースです。
// Goの慣例に従い、インターフェースはプロバイダー（usecase）ではなくコンシューマー（handler）が定義します。
type OAuthUsecase interface {
	BeginAuth(ctx context.Context, provider string) (authURL string, err error)
	HandleCallback(ctx context.Context, provider, code, state string) (token, email string, err error)
}

// OAuthHandler はOAuth2フローのHTTPリクエストを処理します。
type OAuthHandler struct {
	oauth        OAuthUsecase
	secureCookie bool
	frontendURL  string // OAUTH_FRONTEND_REDIRECT_URL: 認証完了後のリダイレクト先
}

// NewOAuthHandler はOAuthHandlerの新しいインスタンスを生成します。
func NewOAuthHandler(oauth OAuthUsecase, secureCookie bool, frontendURL string) *OAuthHandler {
	return &OAuthHandler{
		oauth:        oauth,
		secureCookie: secureCookie,
		frontendURL:  frontendURL,
	}
}

// BeginAuth はOAuth2認可フローを開始します。
// プロバイダーの認可画面へリダイレクトします。
func (h *OAuthHandler) BeginAuth(c *gin.Context) {
	provider := c.Param("provider")
	authURL, err := h.oauth.BeginAuth(c.Request.Context(), provider)
	if err != nil {
		slog.Warn("oauth begin: failed", "provider", provider, "error", err)
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "unsupported provider"})
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

// Callback はOAuth2コールバックを処理します。
// stateの検証・コード交換・ユーザー作成/リンクを行い、JWTとCSRFトークンをCookieにセットして
// フロントエンドURLへリダイレクトします。
func (h *OAuthHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "missing code or state"})
		return
	}

	token, email, err := h.oauth.HandleCallback(c.Request.Context(), provider, code, state)
	if err != nil {
		switch err {
		case usecase.ErrStateNotFound:
			c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: "invalid or expired state"})
		case usecase.ErrOAuthEmailUnavailable:
			c.JSON(http.StatusBadGateway, api.ErrorResponse{Error: "cannot obtain verified email from provider"})
		default:
			slog.Error("oauth callback failed", "provider", provider, "error", err)
			c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "oauth failed"})
		}
		return
	}

	// CSRFトークンを先に生成（失敗した場合はCookieをセットしない → 部分ログイン状態を防止）
	csrfToken, err := csrf.GenerateToken()
	if err != nil {
		slog.Error("failed to generate csrf token", "error", err)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "internal error"})
		return
	}

	slog.Info("oauth login successful", "provider", provider, "email", email)

	// auth_handler.goのLoginと同一パターンでCookieをセット
	// GinのSetSameSiteは直後のSetCookie1回にのみ適用されるため、Cookieごとに毎回呼ぶ必要がある。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", token, 3600, "/", "", h.secureCookie, true)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("csrf_token", csrfToken, 3600, "/", "", h.secureCookie, false)

	c.Redirect(http.StatusFound, h.frontendURL)
}
