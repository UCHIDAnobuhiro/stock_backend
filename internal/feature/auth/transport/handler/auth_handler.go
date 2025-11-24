package handler

import (
	"net/http"

	"stock_backend/internal/feature/auth/transport/http/dto"
	"stock_backend/internal/feature/auth/usecase"

	"github.com/gin-gonic/gin"
)

// AuthHandlerは認証関連のHTTPリクエストを処理します。
// Usecase層のAuthUsecaseに依存し、JSONリクエストを受けてレスポンスを返す責務を持ちます。
type AuthHandler struct {
	auth usecase.AuthUsecase
}

// NewAuthHandlerはAuthHandlerの新しいインスタンスを返します。
// DI用のコンストラクタであり、外部から AuthUsecase を注入します。
func NewAuthHandler(auth usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Signupは新規ユーザー登録APIです。
// - リクエストJSONをsignupReqにバインド
// - バリデーションエラー時は400を返す
// - ユーザー作成失敗（例:重複メール）の場合は409を返す
// - 成功時は201を返す
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

// LoginはログインAPIです。
// - リクエストJSONをLoginReqにバインド
// - バリデーションエラー時は400を返す
// - 認証失敗時は401を返す
// - 認証成功時はJWTを発行して200を返す
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
