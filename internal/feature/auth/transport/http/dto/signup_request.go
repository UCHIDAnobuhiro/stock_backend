// Package dto はauthフィーチャーのHTTPトランスポート層のデータ転送オブジェクトを定義します。
package dto

// SignupReq は/signupエンドポイントのリクエストボディを表します。
// Ginのバインディングタグによるバリデーション（必須、メール形式、パスワード長）を使用します。
type SignupReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}
