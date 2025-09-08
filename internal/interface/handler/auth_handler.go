package handler

import (
	"net/http"

	"todo_backend/internal/usecase"

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

// signupReqは/signupのリクエストボディを表す構造体です。
// Ginのbindingタグで入力チェック（必須・メール形式・パスワード長）を行います。
type signupReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// Signupは新規ユーザー登録APIです。
// - リクエストJSONをsignupReqにバインド
// - バリデーションエラー時は400を返す
// - ユーザー作成失敗（例:重複メール）の場合は409を返す
// - 成功時は201を返す
func (h *AuthHandler) Signup(c *gin.Context) {
	var req signupReq
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

// loginReqは/loginのリクエストボディを表す構造体です。
// バリデーションとして必須チェックを行います。
type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginはログインAPIです。
// - リクエストJSONをloginReqにバインド
// - バリデーションエラー時は400を返す
// - 認証失敗時は401を返す
// - 認証成功時はJWTを発行して200を返す
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
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
