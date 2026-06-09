package authhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/UCHIDAnobuhiro/stock-backend/internal/api"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/feature/auth"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/transport/csrf"
	"github.com/UCHIDAnobuhiro/stock-backend/internal/transport/httpx"
)

// OAuthUsecase はOAuth2認証フローのユースケースインターフェースです。
// Goの慣例に従い、インターフェースはプロバイダー（usecase）ではなくコンシューマー（handler）が定義します。
type OAuthUsecase interface {
	BeginAuth(ctx context.Context, provider string) (authURL string, err error)
	HandleCallback(ctx context.Context, provider, code, state string) (token string, err error)
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
func (h *OAuthHandler) BeginAuth(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	authURL, err := h.oauth.BeginAuth(r.Context(), provider)
	if err != nil {
		slog.Warn("oauth begin: failed", "provider", provider, "error", err)
		httpx.WriteJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "unsupported provider"})
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback はOAuth2コールバックを処理します。
// stateの検証・コード交換・ユーザー作成/リンクを行い、JWTとCSRFトークンをCookieにセットして
// フロントエンドURLへリダイレクトします。
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		httpx.WriteJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "missing code or state"})
		return
	}

	token, err := h.oauth.HandleCallback(r.Context(), provider, code, state)
	if err != nil {
		if errors.Is(err, auth.ErrStateNotFound) {
			httpx.WriteJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "invalid or expired state"})
		} else if errors.Is(err, auth.ErrOAuthEmailUnavailable) {
			httpx.WriteJSON(w, http.StatusBadGateway, api.ErrorResponse{Error: "cannot obtain verified email from provider"})
		} else if errors.Is(err, auth.ErrUnknownProvider) {
			httpx.WriteJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: "unsupported provider"})
		} else {
			slog.Error("oauth callback failed", "provider", provider, "error", err)
			httpx.WriteJSON(w, http.StatusInternalServerError, api.ErrorResponse{Error: "oauth failed"})
		}
		return
	}

	// CSRFトークンを先に生成（失敗した場合はCookieをセットしない → 部分ログイン状態を防止）
	csrfToken, err := csrf.GenerateToken()
	if err != nil {
		slog.Error("failed to generate csrf token", "error", err)
		httpx.WriteJSON(w, http.StatusInternalServerError, api.ErrorResponse{Error: "internal error"})
		return
	}

	slog.Info("oauth login successful", "provider", provider)

	// handler.go の Login と同一パターンで Cookie をセット
	setAuthCookie(w, "auth_token", token, 3600, h.secureCookie, true)
	setAuthCookie(w, "csrf_token", csrfToken, 3600, h.secureCookie, false)

	http.Redirect(w, r, h.frontendURL, http.StatusFound)
}
