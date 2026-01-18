package http

import (
	"net"
	"net/http"
	"time"
)

// NewHTTPClient creates an HTTP client configured for external API calls.
//
// Configuration:
//   - Proxy: Uses environment variables (HTTP_PROXY, etc.) if set
//   - Dialer.Timeout: TCP connection timeout (shorter than default)
//   - Dialer.KeepAlive: Duration to keep reusable TCP connections alive
//   - MaxIdleConns: Maximum idle connections (100 to prevent exhaustion under heavy load)
//   - IdleConnTimeout: Duration to keep idle connections
//   - TLSHandshakeTimeout: Maximum time for HTTPS handshake
//   - Client.Timeout: Overall request timeout (passed from caller)
//
// Notes:
//   - http.DefaultClient has no timeout, so always use a custom client
//   - Transport is explicitly configured for connection stability and resource management
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
