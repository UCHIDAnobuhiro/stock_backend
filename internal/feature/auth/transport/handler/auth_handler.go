// Package handler はauthフィーチャーのHTTPハンドラーを提供します。
package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"stock_backend/internal/api"
	"stock_backend/internal/platform/csrf"
	"stock_backend/internal/platform/ratelimit"
)

// AuthUsecase は認証操作のユースケースを定義します。
// Goの慣例に従い、インターフェースはプロバイダー（usecase）ではなくコンシューマー（handler）が定義します。
type AuthUsecase interface {
	// Signup は指定されたメールアドレスとパスワードで新規ユーザーを登録し、作成されたユーザーIDを返します。
	Signup(ctx context.Context, email, password string) (uint, error)
	// Login はユーザーを認証し、成功時にJWTトークンを返します。
	Login(ctx context.Context, email, password string) (string, error)
}

// PostSignupHook はサインアップ成功後に呼び出されるフックのインターフェースです。
type PostSignupHook interface {
	OnUserCreated(ctx context.Context, userID uint) error
}

// ログインのメールベースレートリミット設定
const (
	loginEmailLimit  = 5                // 15分間のメールアドレスあたりの最大ログイン試行回数
	loginEmailWindow = 15 * time.Minute // メールベースレートリミットのウィンドウ
)

// AuthHandler は認証操作のHTTPリクエストを処理します。
// AuthUsecaseインターフェースに依存し、JSONリクエスト/レスポンスを処理します。
type AuthHandler struct {
	auth         AuthUsecase
	limiter      *ratelimit.Limiter
	secureCookie bool
	postHooks    []PostSignupHook
}

// NewAuthHandler はAuthHandlerの新しいインスタンスを生成します。
// 依存性注入用のコンストラクタで、外部からAuthUsecaseとレートリミッターを注入します。
// secureCookie が true の場合、Secure属性付きのCookieを設定します（本番環境用）。
// postHooks にはサインアップ後に実行するフックを任意で渡せます。
func NewAuthHandler(auth AuthUsecase, limiter *ratelimit.Limiter, secureCookie bool, postHooks ...PostSignupHook) *AuthHandler {
	return &AuthHandler{auth: auth, limiter: limiter, secureCookie: secureCookie, postHooks: postHooks}
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
	userID, err := h.auth.Signup(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// ユーザー列挙攻撃を防止するため、実際のエラーを公開しない
		slog.Warn("signup failed", "error", err, "email", req.Email, "remote_addr", c.ClientIP())
		c.JSON(http.StatusConflict, api.ErrorResponse{Error: "signup failed"})
		return
	}
	for _, hook := range h.postHooks {
		if err := hook.OnUserCreated(c.Request.Context(), userID); err != nil {
			slog.Error("post-signup hook failed", "error", err, "userID", userID)
			c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "signup failed"})
			return
		}
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

	// メールベースのレートリミットチェック
	key := fmt.Sprintf("rl:login:email:%s", strings.ToLower(req.Email))
	result := h.limiter.Allow(c.Request.Context(), key, loginEmailLimit, loginEmailWindow)
	if !result.Allowed {
		slog.Warn("login rate limit exceeded",
			"type", "email",
			"email", req.Email,
			"remote_addr", c.ClientIP(),
		)
		c.Header("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
		c.JSON(http.StatusTooManyRequests, api.ErrorResponse{Error: "too many requests"})
		return
	}

	token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		// ユーザー列挙攻撃を防止するため、実際のエラーを公開しない
		slog.Warn("login failed", "error", err, "email", req.Email, "remote_addr", c.ClientIP())
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: "invalid email or password"})
		return
	}

	// auth_token: httpOnly Cookie（JavaScriptから読み取り不可 → XSS対策）
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", token, 3600, "/", "", h.secureCookie, true)

	// csrf_token: 非httpOnly Cookie（JavaScriptが読み取りX-CSRF-Tokenヘッダーにセット → CSRF対策）
	csrfToken, err := csrf.GenerateToken()
	if err != nil {
		slog.Error("failed to generate csrf token", "error", err)
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Error: "internal error"})
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("csrf_token", csrfToken, 3600, "/", "", h.secureCookie, false)

	slog.Info("user login successful", "email", req.Email, "remote_addr", c.ClientIP())
	c.JSON(http.StatusOK, api.MessageResponse{Message: "ok"})
}

// Logout はauth_tokenとcsrf_tokenのCookieを削除してログアウトします。
// 期限切れトークンでも動作するよう認証不要のルートに配置します。
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", "", -1, "/", "", h.secureCookie, true)

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("csrf_token", "", -1, "/", "", h.secureCookie, false)

	c.JSON(http.StatusOK, api.MessageResponse{Message: "ok"})
}
