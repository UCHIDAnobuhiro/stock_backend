package http

import (
	"net"
	"net/http"
	"time"
)

// NewHTTPClient は外部API呼び出し用に設定されたHTTPクライアントを作成します。
//
// 設定:
//   - Proxy: 環境変数（HTTP_PROXYなど）が設定されている場合に使用
//   - Dialer.Timeout: TCP接続タイムアウト（デフォルトより短い）
//   - Dialer.KeepAlive: 再利用可能なTCP接続の維持期間
//   - MaxIdleConns: 最大アイドル接続数（高負荷時の枯渇防止のため100）
//   - IdleConnTimeout: アイドル接続の維持期間
//   - TLSHandshakeTimeout: HTTPSハンドシェイクの最大時間
//   - Client.Timeout: リクエスト全体のタイムアウト（呼び出し元から渡される）
//
// 注意:
//   - http.DefaultClientにはタイムアウトがないため、常にカスタムクライアントを使用すること
//   - Transportは接続の安定性とリソース管理のために明示的に設定
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
