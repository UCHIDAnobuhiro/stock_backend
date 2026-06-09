package jwt

import "context"

// ctxKey は context へ値を格納するための非公開キー型です。
// 文字列キーの衝突を避けるため、パッケージ固有の型を使用します。
type ctxKey int

const (
	// ctxKeyUserID は認証済みユーザーIDを context に格納するためのキーです。
	ctxKeyUserID ctxKey = iota
	// ctxKeyAuthSource は認証方式（"cookie" または "bearer"）を context に格納するためのキーです。
	// CSRFミドルウェアがBearer認証時にCSRFチェックをスキップするために使用します。
	ctxKeyAuthSource
)

// AuthSourceCookie / AuthSourceBearer は認証方式を表す値です。
const (
	AuthSourceCookie = "cookie"
	AuthSourceBearer = "bearer"
)

// WithUserID は context に認証済みユーザーIDを格納した新しい context を返します。
// 認証ミドルウェア（AuthRequired）が使用するほか、テストでの認証状態の注入にも利用できます。
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// withAuthSource は context に認証方式を格納した新しい context を返します。
func withAuthSource(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, ctxKeyAuthSource, source)
}

// UserIDFromContext は context から認証済みユーザーIDを取り出します。
// 認証ミドルウェア（AuthRequired）を通過したリクエストでのみ ok=true を返します。
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(ctxKeyUserID).(int64)
	return userID, ok
}

// AuthSourceFromContext は context から認証方式を取り出します。
// 未設定の場合は空文字列を返します。
func AuthSourceFromContext(ctx context.Context) string {
	source, _ := ctx.Value(ctxKeyAuthSource).(string)
	return source
}
