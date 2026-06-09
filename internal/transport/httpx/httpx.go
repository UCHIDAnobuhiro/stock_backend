// Package httpx は net/http ハンドラー向けの共通ユーティリティを提供します。
// Gin の c.JSON / c.ShouldBindJSON / c.ClientIP に相当する処理を、
// 標準ライブラリベースで再実装し、各ハンドラーから利用します。
package httpx

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// validate は構造体タグによるバリデーションを行うシングルトンです。
// api パッケージの型は `binding:"..."` タグ（Gin 由来）を持つため、
// validator のタグ名を "binding" に切り替えて既存タグをそのまま利用します。
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.SetTagName("binding")
	return v
}

// WriteJSON は status コードと共に v を JSON としてレスポンスへ書き込みます。
// Gin の c.JSON 相当です。エンコードに失敗した場合はステータス設定後のため
// それ以上の回復はできず、呼び出し側の責務として v は常にエンコード可能であることを前提とします。
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// DecodeAndValidate はリクエストボディを JSON として dst にデコードし、
// `binding` タグに基づくバリデーションを実行します。Gin の c.ShouldBindJSON 相当です。
// デコードまたはバリデーションに失敗した場合はエラーを返します。
func DecodeAndValidate(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return validate.Struct(dst)
}

// ClientIP はリクエスト元のIPアドレスを返します。
// Gin の SetTrustedProxies(nil) + c.ClientIP() と同様に、X-Forwarded-For 等の
// プロキシヘッダーは信頼せず、TCP接続元（RemoteAddr）のホスト部のみを返します。
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// ポートが付与されていない場合は RemoteAddr をそのまま返す。
		return r.RemoteAddr
	}
	return host
}
