package main

import (
	"testing"

	"stock_backend/internal/feature/auth"
	jwtmw "stock_backend/internal/platform/jwt"
)

// clearEnv は設定検証に関わる環境変数をすべて空にし、テストを決定的にする。
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		jwtmw.EnvKeyJWTSecret,
		auth.EnvKeyPasswordPepper,
		"COOKIE_SECURE",
		"APP_ENV",
		"CORS_ALLOWED_ORIGINS",
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"GOOGLE_REDIRECT_URL",
		"GITHUB_CLIENT_ID",
		"GITHUB_CLIENT_SECRET",
		"GITHUB_REDIRECT_URL",
		"OAUTH_FRONTEND_REDIRECT_URL",
	} {
		t.Setenv(k, "")
	}
}

func TestLoadServerConfig(t *testing.T) {
	t.Run("JWT_SECRET 未設定はエラー", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(auth.EnvKeyPasswordPepper, "pepper")
		if _, err := loadServerConfig(); err == nil {
			t.Fatal("expected error when JWT_SECRET is missing, got nil")
		}
	})

	t.Run("PASSWORD_PEPPER 未設定はエラー", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(jwtmw.EnvKeyJWTSecret, "secret")
		if _, err := loadServerConfig(); err == nil {
			t.Fatal("expected error when PASSWORD_PEPPER is missing, got nil")
		}
	})

	t.Run("必須のみ設定で成功・デフォルト適用", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(jwtmw.EnvKeyJWTSecret, "secret")
		t.Setenv(auth.EnvKeyPasswordPepper, "pepper")

		cfg, err := loadServerConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.oauth != nil {
			t.Errorf("oauth should be nil when no provider configured, got %+v", cfg.oauth)
		}
		if cfg.secureCookie {
			t.Error("secureCookie should default to false without APP_ENV=production")
		}
		if len(cfg.corsOrigins) != 1 || cfg.corsOrigins[0] != "http://localhost:3000" {
			t.Errorf("corsOrigins should default to localhost:3000, got %v", cfg.corsOrigins)
		}
	})

	t.Run("APP_ENV=production で secureCookie が true", func(t *testing.T) {
		clearEnv(t)
		t.Setenv(jwtmw.EnvKeyJWTSecret, "secret")
		t.Setenv(auth.EnvKeyPasswordPepper, "pepper")
		t.Setenv("APP_ENV", "production")

		cfg, err := loadServerConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.secureCookie {
			t.Error("secureCookie should be true when APP_ENV=production")
		}
	})
}

func TestRun_ReturnsOneWhenDBConfigInvalid(t *testing.T) {
	clearEnv(t)
	t.Setenv(jwtmw.EnvKeyJWTSecret, "secret")
	t.Setenv(auth.EnvKeyPasswordPepper, "pepper")
	t.Setenv("DB_USER", "")

	if got := run(); got != 1 {
		t.Errorf("run() = %d, want 1", got)
	}
}

func TestLoadOAuthConfig(t *testing.T) {
	t.Run("プロバイダ未設定は無効(nil)", func(t *testing.T) {
		clearEnv(t)
		cfg, err := loadOAuthConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Errorf("expected nil config when OAuth disabled, got %+v", cfg)
		}
	})

	t.Run("frontend URL 欠落はエラー", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "gid")
		if _, err := loadOAuthConfig(); err == nil {
			t.Fatal("expected error when OAUTH_FRONTEND_REDIRECT_URL is missing")
		}
	})

	t.Run("Google secret 欠落はエラー", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "gid")
		t.Setenv("OAUTH_FRONTEND_REDIRECT_URL", "https://app.example.com")
		t.Setenv("GOOGLE_REDIRECT_URL", "https://api.example.com/cb")
		if _, err := loadOAuthConfig(); err == nil {
			t.Fatal("expected error when GOOGLE_CLIENT_SECRET is missing")
		}
	})

	t.Run("Google redirect 欠落はエラー", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "gid")
		t.Setenv("GOOGLE_CLIENT_SECRET", "gsec")
		t.Setenv("OAUTH_FRONTEND_REDIRECT_URL", "https://app.example.com")
		if _, err := loadOAuthConfig(); err == nil {
			t.Fatal("expected error when GOOGLE_REDIRECT_URL is missing")
		}
	})

	t.Run("Google 完全設定で google のみ生成", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "gid")
		t.Setenv("GOOGLE_CLIENT_SECRET", "gsec")
		t.Setenv("GOOGLE_REDIRECT_URL", "https://api.example.com/cb")
		t.Setenv("OAUTH_FRONTEND_REDIRECT_URL", "https://app.example.com")

		cfg, err := loadOAuthConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil || cfg.google == nil {
			t.Fatalf("expected google config, got %+v", cfg)
		}
		if cfg.github != nil {
			t.Error("github should be nil when not configured")
		}
	})

	t.Run("GitHub secret 欠落はエラー", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("GITHUB_CLIENT_ID", "hid")
		t.Setenv("GITHUB_REDIRECT_URL", "https://api.example.com/cb")
		t.Setenv("OAUTH_FRONTEND_REDIRECT_URL", "https://app.example.com")
		if _, err := loadOAuthConfig(); err == nil {
			t.Fatal("expected error when GITHUB_CLIENT_SECRET is missing")
		}
	})

	t.Run("両プロバイダ完全設定で両方生成", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("GOOGLE_CLIENT_ID", "gid")
		t.Setenv("GOOGLE_CLIENT_SECRET", "gsec")
		t.Setenv("GOOGLE_REDIRECT_URL", "https://api.example.com/google/cb")
		t.Setenv("GITHUB_CLIENT_ID", "hid")
		t.Setenv("GITHUB_CLIENT_SECRET", "hsec")
		t.Setenv("GITHUB_REDIRECT_URL", "https://api.example.com/github/cb")
		t.Setenv("OAUTH_FRONTEND_REDIRECT_URL", "https://app.example.com")

		cfg, err := loadOAuthConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil || cfg.google == nil || cfg.github == nil {
			t.Fatalf("expected both providers, got %+v", cfg)
		}
	})
}
