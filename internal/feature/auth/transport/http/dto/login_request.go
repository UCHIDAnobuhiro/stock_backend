// Package dto はauthフィーチャーのHTTPトランスポート層のデータ転送オブジェクトを定義します。
package dto

// LoginReq は/loginエンドポイントのリクエストボディを表します。
// 必須フィールドとメール形式のバリデーションを含みます。
type LoginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
