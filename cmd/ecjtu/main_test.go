// main 的端到端测试：环境变量 → config → transport → 命令 → 退出码；
// 上游由 httptest 扮演（ECJTU_BASE_URL 注入），gate 与 CI 不发真实请求。
package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/cli"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/config"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

const testOpenID = "oMainTestFakeOpenID0123456789a"

// setEnv 装配一次运行的环境变量；空串表示该变量未设置。
func setEnv(t *testing.T, baseURL, token, timeout string) {
	t.Helper()
	t.Setenv("ECJTU_WEIXIN_TOKEN", token)
	t.Setenv("ECJTU_BASE_URL", baseURL)
	t.Setenv("ECJTU_TIMEOUT", timeout)
	if timeout == "" {
		os.Unsetenv("ECJTU_TIMEOUT")
	}
}

type caseRun struct {
	name            string
	token           string
	timeout         string
	args            []string
	fixture         string
	badBaseURL      bool
	wantCode        int
	wantStdout      []string
	wantStderr      []string
	wantURLContains []string
	wantNoRequest   bool
}

func TestRunEndToEnd(t *testing.T) {
	tests := []caseRun{
		{
			name: "课表JSON", token: testOpenID,
			args:            []string{"course", "--date", "2026-09-14"},
			fixture:         "course-normal-week3-monday.html",
			wantCode:        0,
			wantStdout:      []string{`"date": "2026-09-14"`, `"week": 3`, `"teacher": "教师B"`},
			wantURLContains: []string{"CalendarServlet", "date=2026-09-14", "weiXinID=" + testOpenID},
		},
		{
			name: "成绩文本", token: testOpenID,
			args:            []string{"score", "--term", "2025.2", "--format", "text"},
			fixture:         "score-normal-2025-2.html",
			wantCode:        0,
			wantStdout:      []string{"张三 2025.2（共 15 门）", "职业生涯与发展规划"},
			wantURLContains: []string{"ScoreQuery", "term=2025.2"},
		},
		{
			name: "考试JSON", token: testOpenID,
			args:            []string{"exam", "--term", "2025.1"},
			fixture:         "exam-normal-2025-1.html",
			wantCode:        0,
			wantStdout:      []string{`"term": "2025.1"`, `"count": 3`},
			wantURLContains: []string{"ExamArrangeCl"},
		},
		{
			name: "空教室选项无需token", token: "",
			args: []string{"classroom", "options"}, fixture: "classroom-options-synthetic.html",
			wantCode:        0,
			wantStdout:      []string{`"selectedTerm": "2026.1"`, `"period": "1,2"`},
			wantURLContains: []string{"OtherQueryServlet", "type=1"},
		},
		{
			name: "ECJTU_TIMEOUT生效", token: testOpenID, timeout: "5s",
			args:     []string{"course", "--date", "2026-09-14"},
			fixture:  "course-normal-week3-monday.html",
			wantCode: 0,
		},
		{
			name: "缺token", token: "",
			args: []string{"course"}, fixture: "course-normal-week3-monday.html",
			wantCode:      2,
			wantStderr:    []string{"教务凭证未配置", "ECJTU_WEIXIN_TOKEN"},
			wantNoRequest: true,
		},
		{
			name: "坏baseURL", token: testOpenID, badBaseURL: true,
			args: []string{"course"}, fixture: "course-normal-week3-monday.html",
			wantCode:      2,
			wantStderr:    []string{"ECJTU_BASE_URL 必须是完整的 http(s) 地址"},
			wantNoRequest: true,
		},
		{
			name: "坏timeout", token: testOpenID, timeout: "abc",
			args: []string{"course"}, fixture: "course-normal-week3-monday.html",
			wantCode:      2,
			wantStderr:    []string{"ECJTU_TIMEOUT 必须是正的时间长度"},
			wantNoRequest: true,
		},
		{
			name: "零timeout", token: testOpenID, timeout: "0s",
			args: []string{"course"}, fixture: "course-normal-week3-monday.html",
			wantCode:      2,
			wantStderr:    []string{"ECJTU_TIMEOUT"},
			wantNoRequest: true,
		},
		{
			name: "未绑定", token: testOpenID,
			args: []string{"course"}, fixture: "notbound.html",
			wantCode:   3,
			wantStderr: []string{"教务系统未绑定该微信账号"},
		},
		{
			name: "version不读env", token: "",
			args:          []string{"--version"},
			fixture:       "course-normal-week3-monday.html",
			wantCode:      0,
			wantStdout:    []string{"ecjtu dev", "编译时间 (UTC): unknown", "Go 版本:", "平台:", "编译器:"},
			wantNoRequest: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int
			var lastURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				lastURL = r.URL.String()
				_, _ = w.Write(testsupport.Fixture(t, tt.fixture))
			}))
			defer srv.Close()

			baseURL := srv.URL
			if tt.badBaseURL {
				baseURL = "not a url" // 命中 config 的 URL 校验分支
			}
			setEnv(t, baseURL, tt.token, tt.timeout)

			var stdout, stderr bytes.Buffer
			code := run(context.Background(), tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Fatalf("退出码 = %d，期望 %d\nstdout=%q\nstderr=%q", code, tt.wantCode, stdout.String(), stderr.String())
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout 缺少 %q:\n%s", want, stdout.String())
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(stderr.String(), want) {
					t.Fatalf("stderr 缺少 %q:\n%s", want, stderr.String())
				}
			}
			if tt.wantNoRequest && hits != 0 {
				t.Fatalf("本地校验失败却发出了 %d 次请求", hits)
			}
			for _, want := range tt.wantURLContains {
				if !strings.Contains(lastURL, want) {
					t.Fatalf("请求 URL 缺少 %q: %q", want, lastURL)
				}
			}
			if all := stdout.String() + stderr.String(); strings.Contains(all, testOpenID) {
				t.Fatalf("输出泄漏了完整 openid:\n%s", all)
			}
		})
	}
}

// 把 cli 的中文文案与 config 的变量名绑在一起：本包同时 import 两者，是唯一交叉点，
// 由它守住「文案不串号、不把所有配置错误都念成凭证未配置」。
func TestConfigMessagesBindEnvNames(t *testing.T) {
	tests := []struct {
		name       string
		constant   string
		mention    string
		notMention string
	}{
		{"凭证", config.EnvOpenID, "ECJTU_WEIXIN_TOKEN", "ECJTU_BASE_URL"},
		{"地址", config.EnvBaseURL, "ECJTU_BASE_URL", "ECJTU_WEIXIN_TOKEN"},
		{"超时", config.EnvTimeout, "ECJTU_TIMEOUT", "ECJTU_BASE_URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cli.ConfigError(tt.constant, errors.New("boom"))
			msg := cli.UserMessage(err)
			if !strings.Contains(msg, tt.mention) {
				t.Fatalf("%s 的文案应点名 %s，实得 %q", tt.constant, tt.mention, msg)
			}
			if strings.Contains(msg, tt.notMention) {
				t.Fatalf("%s 的文案串号到了 %s：%q", tt.constant, tt.notMention, msg)
			}
			if cli.ExitCode(err) != cli.ExitUsage {
				t.Fatalf("%s 的配置错误退出码应为 %d", tt.constant, cli.ExitUsage)
			}
		})
	}
}

func TestFlagTimeoutOverridesEnv(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
		_, _ = w.Write(testsupport.Fixture(t, "course-normal-week3-monday.html"))
	}))
	defer srv.Close()
	setEnv(t, srv.URL, testOpenID, "30s")

	var stdout, stderr bytes.Buffer
	started := time.Now()
	code := run(context.Background(), []string{"course", "--timeout", "120ms"}, &stdout, &stderr)
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("--timeout 未覆盖 ECJTU_TIMEOUT，实际耗时 %v", elapsed)
	}
	if code != cli.ExitUpstream {
		t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", code, cli.ExitUpstream, stderr.String())
	}
}

func TestHelpNeedsNoCredentials(t *testing.T) {
	setEnv(t, "", "", "")
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"--help"}, &stdout, &stderr); code != cli.ExitOK {
		t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", code, cli.ExitOK, stderr.String())
	}
	for _, want := range []string{"course", "score", "exam"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help 缺少 %q:\n%s", want, stdout.String())
		}
	}
}

func TestClassroomSearchNeedsNoTokenAndSendsNoIdentity(t *testing.T) {
	var mu sync.Mutex
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		if r.Header.Get("Cookie") != "" || r.URL.Query().Get("weiXinID") != "" ||
			r.URL.Query().Get("code") != "" || r.URL.Query().Get("state") != "" {
			t.Errorf("匿名查询携带了身份字段：cookie=%q query=%v", r.Header.Get("Cookie"), r.URL.Query())
		}
		if r.URL.Path == "/OtherQueryServlet" {
			_, _ = w.Write(testsupport.Fixture(t, "classroom-options-synthetic.html"))
			return
		}
		if r.URL.Path == "/KongClassRoomQuery" && r.Method == http.MethodPost {
			_, _ = w.Write(testsupport.Fixture(t, "classroom-results-synthetic.html"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	setEnv(t, server.URL, "", "")

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"classroom", "search", "--week", "3"}, &stdout, &stderr)
	if code != cli.ExitOK {
		t.Fatalf("退出码 = %d\nstdout=%q\nstderr=%q", code, stdout.String(), stderr.String())
	}
	mu.Lock()
	gotHits := hits
	mu.Unlock()
	if gotHits != 2 {
		t.Fatalf("上游请求数 = %d，期望 GET options + POST search 共 2 次", gotHits)
	}
	if !strings.Contains(stdout.String(), `"count": 2`) || !strings.Contains(stdout.String(), `"weekday": "星期四"`) {
		t.Fatalf("stdout 未包含查询结果与表单默认值：%s", stdout.String())
	}
}
