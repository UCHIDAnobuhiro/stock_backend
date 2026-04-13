package jwtmw

const (
	// EnvKeyJWTSecret はJWT署名シークレットの環境変数キーです。
	EnvKeyJWTSecret = "JWT_SECRET"

	// CookieAuthToken は認証JWTを格納するCookie名です（HttpOnly）。
	CookieAuthToken = "auth_token"

	// CookieCSRFToken はCSRFトークンを格納するCookie名です（非HttpOnly - JSから読み取り可）。
	CookieCSRFToken = "csrf_token"

	// HeaderCSRFToken はCSRFトークンを送信するリクエストヘッダー名です。
	HeaderCSRFToken = "X-CSRF-Token"

	// CookieMaxAge はCookieの有効期限（秒）です。JWTの有効期限と合わせて1時間。
	CookieMaxAge = 3600
)
