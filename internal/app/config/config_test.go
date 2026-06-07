package config

import (
	"reflect"
	"testing"
)

// TestParseCORSOrigins は CORS_ALLOWED_ORIGINS env の生文字列パースが
// trim・空要素除去・複数要素対応を正しく行うことを検証します。
func TestParseCORSOrigins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "空文字の場合は nil を返す",
			raw:  "",
			want: nil,
		},
		{
			name: "単一オリジン",
			raw:  "http://localhost:3000",
			want: []string{"http://localhost:3000"},
		},
		{
			name: "カンマ区切り複数オリジン",
			raw:  "http://localhost:3000,https://example.com",
			want: []string{"http://localhost:3000", "https://example.com"},
		},
		{
			name: "前後のスペースを trim する",
			raw:  "  http://a.example.com  ,  http://b.example.com  ",
			want: []string{"http://a.example.com", "http://b.example.com"},
		},
		{
			name: "連続カンマによる空要素を除去する",
			raw:  "a,,b",
			want: []string{"a", "b"},
		},
		{
			name: "空白のみの要素を除去する",
			raw:  "a, ,b",
			want: []string{"a", "b"},
		},
		{
			name: "末尾カンマによる空要素を除去する",
			raw:  "a,b,",
			want: []string{"a", "b"},
		},
		{
			name: "全要素が空白のみなら nil を返す",
			raw:  " , , ",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseCORSOrigins(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCORSOrigins(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

// TestParseBoolString は raw の bool パースとフォールバック動作を検証します。
func TestParseBoolString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		fallback  bool
		wantValue bool
		wantOK    bool
	}{
		{
			name:      "空文字・fallback=false は (false, true)",
			raw:       "",
			fallback:  false,
			wantValue: false,
			wantOK:    true,
		},
		{
			name:      "空文字・fallback=true は (true, true)",
			raw:       "",
			fallback:  true,
			wantValue: true,
			wantOK:    true,
		},
		{
			name:      "raw=true は (true, true)",
			raw:       "true",
			fallback:  false,
			wantValue: true,
			wantOK:    true,
		},
		{
			name:      "raw=false は fallback=true でも (false, true)",
			raw:       "false",
			fallback:  true,
			wantValue: false,
			wantOK:    true,
		},
		{
			name:      "raw=1 は (true, true)",
			raw:       "1",
			fallback:  false,
			wantValue: true,
			wantOK:    true,
		},
		{
			name:      "raw=0 は (false, true)",
			raw:       "0",
			fallback:  true,
			wantValue: false,
			wantOK:    true,
		},
		{
			name:      "不正値は (fallback, false) を返す",
			raw:       "yes",
			fallback:  true,
			wantValue: true,
			wantOK:    false,
		},
		{
			name:      "不正値・fallback=false は (false, false)",
			raw:       "invalid",
			fallback:  false,
			wantValue: false,
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotValue, gotOK := ParseBoolString(tt.raw, tt.fallback)
			if gotValue != tt.wantValue || gotOK != tt.wantOK {
				t.Errorf("ParseBoolString(%q, %v) = (%v, %v), want (%v, %v)",
					tt.raw, tt.fallback, gotValue, gotOK, tt.wantValue, tt.wantOK)
			}
		})
	}
}

// TestParseLogFormat は LOG_FORMAT 明示指定と APP_ENV フォールバックの組み合わせを検証します。
func TestParseLogFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		logFormatRaw string
		appEnv       string
		wantUseJSON  bool
		wantOK       bool
	}{
		{
			name:         "json 明示",
			logFormatRaw: "json",
			appEnv:       "docker-dev",
			wantUseJSON:  true,
			wantOK:       true,
		},
		{
			name:         "text 明示は production でも text",
			logFormatRaw: "text",
			appEnv:       "production",
			wantUseJSON:  false,
			wantOK:       true,
		},
		{
			name:         "大文字・前後空白は無視する",
			logFormatRaw: "  JSON  ",
			appEnv:       "docker-dev",
			wantUseJSON:  true,
			wantOK:       true,
		},
		{
			name:         "未設定 + production は JSON",
			logFormatRaw: "",
			appEnv:       "production",
			wantUseJSON:  true,
			wantOK:       true,
		},
		{
			name:         "未設定 + docker-dev は Text",
			logFormatRaw: "",
			appEnv:       "docker-dev",
			wantUseJSON:  false,
			wantOK:       true,
		},
		{
			name:         "不正値は production フォールバックで JSON + ok=false",
			logFormatRaw: "yaml",
			appEnv:       "production",
			wantUseJSON:  true,
			wantOK:       false,
		},
		{
			name:         "不正値は dev フォールバックで Text + ok=false",
			logFormatRaw: "yaml",
			appEnv:       "docker-dev",
			wantUseJSON:  false,
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotUseJSON, gotOK := ParseLogFormat(tt.logFormatRaw, tt.appEnv)
			if gotUseJSON != tt.wantUseJSON || gotOK != tt.wantOK {
				t.Errorf("ParseLogFormat(%q, %q) = (%v, %v), want (%v, %v)",
					tt.logFormatRaw, tt.appEnv, gotUseJSON, gotOK, tt.wantUseJSON, tt.wantOK)
			}
		})
	}
}
