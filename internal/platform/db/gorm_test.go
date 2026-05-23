package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// TestBuildDSN_TCP はTCP接続用のDSN文字列が正しく生成されることを検証します。
func TestBuildDSN_TCP(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User:     "testuser",
		Password: "testpass",
		Name:     "testdb",
		Host:     "localhost",
		Port:     "5432",
	}

	dsn := BuildDSN(cfg)

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	if dsn != expected {
		t.Errorf("expected DSN %q, got %q", expected, dsn)
	}
}

// TestBuildDSN_CloudSQL はCloud SQL Unixソケット接続用のDSN文字列が正しく生成されることを検証します。
func TestBuildDSN_CloudSQL(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User:         "testuser",
		Password:     "testpass",
		Name:         "testdb",
		InstanceName: "project:region:instance",
	}

	dsn := BuildDSN(cfg)

	expected := "host=/cloudsql/project:region:instance user=testuser password=testpass dbname=testdb"
	if dsn != expected {
		t.Errorf("expected DSN %q, got %q", expected, dsn)
	}
}

// TestBuildDSN_CloudSQLTakesPrecedence はInstanceNameとHost/Portが両方設定されている場合にInstanceNameが優先されることを検証します。
func TestBuildDSN_CloudSQLTakesPrecedence(t *testing.T) {
	t.Parallel()

	// When both InstanceName and Host/Port are set, InstanceName takes precedence
	cfg := Config{
		User:         "testuser",
		Password:     "testpass",
		Name:         "testdb",
		Host:         "localhost",
		Port:         "5432",
		InstanceName: "project:region:instance",
	}

	dsn := BuildDSN(cfg)

	// Should use Cloud SQL format, not TCP
	if dsn == "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable" {
		t.Error("expected Cloud SQL DSN format, but got TCP format")
	}
	expected := "host=/cloudsql/project:region:instance user=testuser password=testpass dbname=testdb"
	if dsn != expected {
		t.Errorf("expected DSN %q, got %q", expected, dsn)
	}
}

// TestBuildDSN_EscapesSpecialCharacters は値に空白・シングルクオート・バックスラッシュを含む場合に
// libpq 仕様に従って DSN が正しくエスケープされることを検証します。
func TestBuildDSN_EscapesSpecialCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      Config
		expected string
	}{
		{
			name: "password with space",
			cfg: Config{
				User:     "u",
				Password: "p@ss word",
				Name:     "d",
				Host:     "h",
				Port:     "5432",
			},
			expected: "host=h port=5432 user=u password='p@ss word' dbname=d sslmode=disable",
		},
		{
			name: "password with single quote and backslash",
			cfg: Config{
				User:     "u",
				Password: `p'a\ss`,
				Name:     "d",
				Host:     "h",
				Port:     "5432",
			},
			expected: `host=h port=5432 user=u password='p\'a\\ss' dbname=d sslmode=disable`,
		},
		{
			name: "empty password is quoted",
			cfg: Config{
				User:     "u",
				Password: "",
				Name:     "d",
				Host:     "h",
				Port:     "5432",
			},
			expected: "host=h port=5432 user=u password='' dbname=d sslmode=disable",
		},
		{
			name: "user with equals sign",
			cfg: Config{
				User:     "us=er",
				Password: "p",
				Name:     "d",
				Host:     "h",
				Port:     "5432",
			},
			expected: "host=h port=5432 user='us=er' password=p dbname=d sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BuildDSN(tt.cfg)
			if got != tt.expected {
				t.Errorf("BuildDSN mismatch\n want: %q\n  got: %q", tt.expected, got)
			}
		})
	}
}

// TestPassword_Masking は Password 型がログ・文字列化・JSON シリアライズのいずれの経路でも
// 平文を露出せず "***" にマスクされることを検証します。
func TestPassword_Masking(t *testing.T) {
	t.Parallel()

	const secret = "super-secret"
	p := Password(secret)

	t.Run("String", func(t *testing.T) {
		t.Parallel()
		if got := p.String(); got != "***" {
			t.Errorf("String() = %q, want %q", got, "***")
		}
	})

	t.Run("fmt %v and %s do not leak", func(t *testing.T) {
		t.Parallel()
		for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
			got := fmt.Sprintf(verb, p)
			if strings.Contains(got, secret) {
				t.Errorf("fmt %q leaked secret: %q", verb, got)
			}
		}
	})

	t.Run("embedded in Config via %+v does not leak", func(t *testing.T) {
		t.Parallel()
		cfg := Config{User: "u", Password: p, Name: "d"}
		got := fmt.Sprintf("%+v", cfg)
		if strings.Contains(got, secret) {
			t.Errorf("Config format leaked secret: %q", got)
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(b) != `"***"` {
			t.Errorf("MarshalJSON = %s, want %q", b, `"***"`)
		}
		cfg := Config{User: "u", Password: p, Name: "d"}
		cb, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(string(cb), secret) {
			t.Errorf("Config JSON leaked secret: %s", cb)
		}
	})

	t.Run("slog structured logging does not leak", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		logger.Info("connecting", "password", p)
		if strings.Contains(buf.String(), secret) {
			t.Errorf("slog leaked secret: %s", buf.String())
		}
	})

	t.Run("explicit string conversion still exposes value", func(t *testing.T) {
		t.Parallel()
		// DSN 構築など実値が必要な場面では明示的変換で取得できる
		if string(p) != secret {
			t.Errorf("string(p) = %q, want %q", string(p), secret)
		}
	})
}

// TestConnectWithRetry_SuccessOnFirstTry は初回接続成功時にリトライせずDBを返すことを検証します。
func TestConnectWithRetry_SuccessOnFirstTry(t *testing.T) {
	t.Parallel()

	mockDB := &gorm.DB{}
	opener := func(dsn string) (*gorm.DB, error) {
		return mockDB, nil
	}

	db, err := ConnectWithRetry("test-dsn", 5*time.Second, opener)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db != mockDB {
		t.Error("expected mock DB to be returned")
	}
}

// TestConnectWithRetry_RetriesOnFailure は接続失敗時にリトライして最終的に成功することを検証します。
func TestConnectWithRetry_RetriesOnFailure(t *testing.T) {
	// Not parallel because this test takes time due to retry sleeps

	mockDB := &gorm.DB{}
	attemptCount := 0

	opener := func(dsn string) (*gorm.DB, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, errors.New("connection refused")
		}
		return mockDB, nil
	}

	// Use a timeout that allows for 2 retries (retry interval is 3 seconds)
	db, err := ConnectWithRetry("test-dsn", 10*time.Second, opener)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db != mockDB {
		t.Error("expected mock DB to be returned")
	}
	if attemptCount != 3 {
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}
}

// TestConnectWithRetry_TimeoutAfterRetries はタイムアウト後にエラーが返されることを検証します。
func TestConnectWithRetry_TimeoutAfterRetries(t *testing.T) {
	t.Parallel()

	attemptCount := 0
	opener := func(dsn string) (*gorm.DB, error) {
		attemptCount++
		return nil, errors.New("connection refused")
	}

	// Very short timeout - should fail quickly
	_, err := ConnectWithRetry("test-dsn", 100*time.Millisecond, opener)

	if err == nil {
		t.Fatal("expected error after timeout, got nil")
	}
	if attemptCount == 0 {
		t.Error("expected at least one connection attempt")
	}
}

// TestConfig_Validate_TCP_Success は TCP 接続で必須項目が揃っていればエラーにならないことを検証します。
func TestConfig_Validate_TCP_Success(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User:     "u",
		Password: "p",
		Name:     "d",
		Host:     "h",
		Port:     "5432",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestConfig_Validate_CloudSQL_Success は Cloud SQL 接続で Host/Port が空でもエラーにならないことを検証します。
func TestConfig_Validate_CloudSQL_Success(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User:         "u",
		Name:         "d",
		InstanceName: "project:region:instance",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestConfig_Validate_EmptyPasswordAllowed はパスワード未設定でもエラーにならないことを検証します。
func TestConfig_Validate_EmptyPasswordAllowed(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User: "u",
		Name: "d",
		Host: "h",
		Port: "5432",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestConfig_Validate_MissingRequired は必須項目が欠けている場合にエラーが返ることを検証します。
// wantVars: エラーメッセージに含まれるべき変数名
// notWantVars: エラーメッセージに含まれてはいけない変数名
func TestConfig_Validate_MissingRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         Config
		wantVars    []string
		notWantVars []string
	}{
		{
			name:        "all empty (TCP)",
			cfg:         Config{},
			wantVars:    []string{"DB_USER", "DB_NAME", "DB_HOST", "DB_PORT"},
			notWantVars: nil,
		},
		{
			name:        "missing user",
			cfg:         Config{Name: "d", Host: "h", Port: "5432"},
			wantVars:    []string{"DB_USER"},
			notWantVars: []string{"DB_NAME", "DB_HOST", "DB_PORT"},
		},
		{
			name:        "missing name",
			cfg:         Config{User: "u", Host: "h", Port: "5432"},
			wantVars:    []string{"DB_NAME"},
			notWantVars: []string{"DB_USER", "DB_HOST", "DB_PORT"},
		},
		{
			name:        "missing host/port (TCP)",
			cfg:         Config{User: "u", Name: "d"},
			wantVars:    []string{"DB_HOST", "DB_PORT"},
			notWantVars: []string{"DB_USER", "DB_NAME"},
		},
		{
			name:        "CloudSQL missing user",
			cfg:         Config{Name: "d", InstanceName: "proj:reg:inst"},
			wantVars:    []string{"DB_USER"},
			notWantVars: []string{"DB_NAME", "DB_HOST", "DB_PORT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, v := range tt.wantVars {
				if !strings.Contains(err.Error(), v) {
					t.Errorf("expected error to mention %q, got %q", v, err.Error())
				}
			}
			for _, v := range tt.notWantVars {
				if strings.Contains(err.Error(), v) {
					t.Errorf("error should not mention %q, got %q", v, err.Error())
				}
			}
		})
	}
}

// TestConfig_Validate_WhitespaceOnly は空白文字のみの値が未設定と同じく弾かれることを検証します。
func TestConfig_Validate_WhitespaceOnly(t *testing.T) {
	t.Parallel()

	cfg := Config{
		User: "   ",
		Name: "\t",
		Host: "h",
		Port: "5432",
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, v := range []string{"DB_USER", "DB_NAME"} {
		if !strings.Contains(err.Error(), v) {
			t.Errorf("expected error to mention %q, got %q", v, err.Error())
		}
	}
}

// TestLoadConfigFromEnv は環境変数からデータベース設定が正しく読み込まれることを検証します。
func TestLoadConfigFromEnv(t *testing.T) {
	// Note: Not running in parallel since we're modifying environment variables
	// Set environment variables for the test
	t.Setenv("DB_USER", "envuser")
	t.Setenv("DB_PASSWORD", "envpass")
	t.Setenv("DB_NAME", "envdb")
	t.Setenv("DB_HOST", "envhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("INSTANCE_CONNECTION_NAME", "")

	cfg := LoadConfigFromEnv()

	if cfg.User != "envuser" {
		t.Errorf("expected User 'envuser', got %q", cfg.User)
	}
	if string(cfg.Password) != "envpass" {
		t.Errorf("expected Password 'envpass', got %q", string(cfg.Password))
	}
	if cfg.Name != "envdb" {
		t.Errorf("expected Name 'envdb', got %q", cfg.Name)
	}
	if cfg.Host != "envhost" {
		t.Errorf("expected Host 'envhost', got %q", cfg.Host)
	}
	if cfg.Port != "5432" {
		t.Errorf("expected Port '5432', got %q", cfg.Port)
	}
}
