package http

import (
	"net"
	"net/http"
	"time"
)

// NewHTTPClient は外部API用の HTTP クライアントを生成します。
//
// 主な設定:
//   - Proxy: 環境変数 (HTTP_PROXY など) があれば利用
//   - Dialer.Timeout: TCP接続のタイムアウト (デフォルトより短めに設定)
//   - Dialer.KeepAlive: 再利用可能なTCP接続を保持する時間
//   - MaxIdleConns: 最大アイドル接続数 (大量リクエストでも枯渇しないように100)
//   - IdleConnTimeout: アイドル状態の接続を保持する時間
//   - TLSHandshakeTimeout: HTTPS ハンドシェイクの上限時間
//   - Client.Timeout: リクエスト全体のタイムアウト (呼び出し元から渡す)
//
// ポイント:
//   - http.DefaultClient はタイムアウトが無制限なので、必ずカスタムクライアントを使う
//   - 外部APIとの通信安定性・リソース管理のため、Transport を明示的に設定している
func NewHTTPClient(timeout time.Duration) *http.Client {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	return &http.Client{Timeout: timeout, Transport: t}
}
