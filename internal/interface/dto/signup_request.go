package dto

// signupReqは/signupのリクエストボディを表す構造体です。
// Ginのbindingタグで入力チェック（必須・メール形式・パスワード長）を行います。
type SignupReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}
