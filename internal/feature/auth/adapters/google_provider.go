// Package adapters はauthフィーチャーのリポジトリ実装を提供します。
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"stock_backend/internal/feature/auth/usecase"
)

// GoogleProvider はusecase.OAuthProviderインターフェースのGoogle実装です。
type GoogleProvider struct {
	cfg *oauth2.Config
	hc  *http.Client
}

var _ usecase.OAuthProvider = (*GoogleProvider)(nil)

// NewGoogleProvider はGoogleProvider の新しいインスタンスを生成します。
func NewGoogleProvider(clientID, clientSecret, redirectURL string, hc *http.Client) *GoogleProvider {
	return &GoogleProvider{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email"},
			Endpoint:     google.Endpoint,
		},
		hc: hc,
	}
}

// AuthorizationURL はPKCE(S256)付きのGoogle認可URLを返します。
func (p *GoogleProvider) AuthorizationURL(state, codeChallenge string) string {
	return p.cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

// ExchangeCode はauthorization codeをユーザー情報に交換します。
// Googleの /oauth2/v3/userinfo エンドポイントでメールアドレスを取得します。
func (p *GoogleProvider) ExchangeCode(ctx context.Context, code, codeVerifier string) (*usecase.OAuthUserInfo, error) {
	tok, err := p.cfg.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("google: code exchange failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("google: failed to build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google: userinfo request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("google: failed to read userinfo response: %w", err)
	}

	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("google: failed to parse userinfo: %w", err)
	}
	if !info.EmailVerified || info.Email == "" {
		return nil, usecase.ErrOAuthEmailUnavailable
	}

	return &usecase.OAuthUserInfo{ProviderUID: info.Sub, Email: info.Email}, nil
}
