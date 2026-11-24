package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string // 空串表示期望成功
	}{
		{
			name: "仅 token 时取默认值",
			env:  map[string]string{EnvOpenID: " oABC123 "},
			want: Config{OpenID: "oABC123", Timeout: DefaultTimeout},
		},
		{
			name:    "缺失 token 报错",
			env:     map[string]string{EnvOpenID: "   "},
			wantErr: EnvOpenID + " is not set",
		},
		{
			name: "httptest 注入 base url 与超时",
			env:  map[string]string{EnvOpenID: "oABC", EnvBaseURL: "http://127.0.0.1:9999/weixin", EnvTimeout: "3s"},
			want: Config{OpenID: "oABC", BaseURL: "http://127.0.0.1:9999/weixin", Timeout: 3 * time.Second},
		},
		{
			name:    "非法 base url 报错",
			env:     map[string]string{EnvOpenID: "oABC", EnvBaseURL: "ftp://x/y"},
			wantErr: EnvBaseURL,
		},
		{
			name:    "base url 缺 scheme 报错",
			env:     map[string]string{EnvOpenID: "oABC", EnvBaseURL: "jwxt.ecjtu.edu.cn"},
			wantErr: EnvBaseURL,
		},
		{
			name:    "超时非 duration 报错",
			env:     map[string]string{EnvOpenID: "oABC", EnvTimeout: "abc"},
			wantErr: EnvTimeout,
		},
		{
			name:    "超时非正数报错",
			env:     map[string]string{EnvOpenID: "oABC", EnvTimeout: "0s"},
			wantErr: EnvTimeout,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			for _, k := range []string{EnvOpenID, EnvBaseURL, EnvTimeout} {
				if _, ok := tt.env[k]; !ok {
					t.Setenv(k, "")
				}
			}

			got, err := Load()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("期望成功，实际报错: %v", err)
				}
				if got != tt.want {
					t.Fatal("配置不匹配")
				}
				return
			}
			if err == nil {
				t.Fatal("期望报错，实际成功")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("错误 %q 应包含 %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfigTimeoutIsTotalBudgetDocumented(t *testing.T) {
	// 防漂移护栏：DefaultTimeout 的语义是「整命令总预算」，改动需要同步 conventions.md。
	if DefaultTimeout != 15*time.Second {
		t.Fatalf("DefaultTimeout 被改动，请同步 conventions.md: %v", DefaultTimeout)
	}
}

func TestLoadPublicDoesNotRequireOpenID(t *testing.T) {
	t.Setenv(EnvOpenID, "")
	t.Setenv(EnvBaseURL, "http://127.0.0.1:9999/weixin")
	t.Setenv(EnvTimeout, "2s")

	got, err := LoadPublic()
	if err != nil {
		t.Fatal(err)
	}
	want := PublicConfig{BaseURL: "http://127.0.0.1:9999/weixin", Timeout: 2 * time.Second}
	if got != want {
		t.Fatalf("public config = %#v, want %#v", got, want)
	}
}
