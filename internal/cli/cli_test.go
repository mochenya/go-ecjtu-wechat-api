// CLI 端到端测试：从参数解析一路到 stdout/stderr 与退出码，上游由 httptest 假服务器扮演。
package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/cli"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/client"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

// fakeOpenID 是独特假值：任何输出里出现完整串都算泄漏。
const fakeOpenID = "oCLItestFakeOpenID0123456789ab"

type result struct {
	code   int
	stdout string
	stderr string
	path   string
	query  url.Values
	hits   int
}

// stub 扮演上游：按用例返回指定状态码与正文，并记录最后一次请求。
// hits/last 由服务器 goroutine 写入，全部经 mu 保护（-race 下必须）。
type stub struct {
	t   *testing.T
	srv *httptest.Server

	mu   sync.Mutex
	last *http.Request
	hits int
}

func (s *stub) record(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits++
	s.last = r.Clone(context.Background())
}

func (s *stub) snapshot() (int, string, url.Values) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last == nil {
		return s.hits, "", nil
	}
	return s.hits, s.last.URL.Path, s.last.URL.Query()
}

func newStub(t *testing.T, status int, body []byte) *stub {
	t.Helper()
	s := &stub{t: t}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r)
		if len(body) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

// slow 让 handler 在写响应前阻塞，用于超时用例。
func (s *stub) slow(d time.Duration) *stub {
	s.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r)
		select {
		case <-time.After(d):
			_, _ = w.Write([]byte("<html><body>late</body></html>"))
		case <-r.Context().Done():
		}
	})
	return s
}

func (s *stub) assemble() cli.Assembler {
	return func() (cli.Assembly, error) {
		school, err := client.New(client.Config{BaseURL: s.srv.URL, OpenID: fakeOpenID})
		if err != nil {
			return cli.Assembly{}, err
		}
		return cli.Assembly{
			Client:  school,
			Timeout: 5 * time.Second,
		}, nil
	}
}

func (s *stub) assemblePublic() cli.PublicAssembler {
	return func() (cli.Assembly, error) {
		return cli.Assembly{
			PublicClient: client.NewPublic(client.PublicConfig{BaseURL: s.srv.URL}),
			Timeout:      5 * time.Second,
		}, nil
	}
}

// today 是注入的固定时钟：2026-09-14 08:30（Asia/Shanghai）。
func today() time.Time {
	return time.Date(2026, 9, 14, 8, 30, 0, 0, time.FixedZone("CST", 8*3600))
}

func runWith(t *testing.T, assemble cli.Assembler, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cli.Execute(context.Background(), cli.Deps{
		Assemble: assemble,
		Stdout:   &out,
		Stderr:   &errOut,
		BuildInfo: cli.BuildInfo{
			Version:   "test",
			BuildTime: "2026-11-19T00:00:00Z",
			GoVersion: "go-test",
			OS:        "linux",
			Arch:      "amd64",
			Compiler:  "gc",
		},
		Now: today,
	}, args)
	return result{code: code, stdout: out.String(), stderr: errOut.String()}
}

func runPublicWith(t *testing.T, assemble cli.PublicAssembler, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cli.Execute(context.Background(), cli.Deps{
		AssemblePublic: assemble,
		Stdout:         &out,
		Stderr:         &errOut,
		BuildInfo:      cli.BuildInfo{Version: "test"},
		Now:            today,
	}, args)
	return result{code: code, stdout: out.String(), stderr: errOut.String()}
}

func (s *stub) run(t *testing.T, args ...string) result {
	t.Helper()
	res := runWith(t, s.assemble(), args...)
	res.hits, res.path, res.query = s.snapshot()
	return res
}

func (s *stub) runPublic(t *testing.T, args ...string) result {
	t.Helper()
	res := runPublicWith(t, s.assemblePublic(), args...)
	res.hits, res.path, res.query = s.snapshot()
	return res
}

func TestClassroomOptionsUsesPublicTransport(t *testing.T) {
	s := newStub(t, http.StatusOK, testsupport.Fixture(t, "classroom-options-synthetic.html"))
	got := s.runPublic(t, "classroom")
	if got.code != cli.ExitOK {
		t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", got.code, cli.ExitOK, got.stderr)
	}
	for _, want := range []string{`"selectedTerm": "2026.1"`, `"weekday": "星期四"`, `"period": "1,2"`} {
		if !strings.Contains(got.stdout, want) {
			t.Fatalf("stdout 缺少 %q:\n%s", want, got.stdout)
		}
	}
	if got.path != "/OtherQueryServlet" || got.query.Get("type") != "1" || got.query.Get("weiXinID") != "" {
		t.Fatalf("公开选项请求 = %s %v", got.path, got.query)
	}
}

func TestClassroomSearchUsesFormDefaults(t *testing.T) {
	var paths []string
	var forms []url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("Cookie") != "" || r.URL.Query().Get("weiXinID") != "" {
			t.Errorf("公开请求携带身份信息：cookie=%q query=%v", r.Header.Get("Cookie"), r.URL.Query())
		}
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			forms = append(forms, values)
			_, _ = w.Write(testsupport.Fixture(t, "classroom-results-synthetic.html"))
			return
		}
		_, _ = w.Write(testsupport.Fixture(t, "classroom-options-synthetic.html"))
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	code := cli.Execute(context.Background(), cli.Deps{
		AssemblePublic: func() (cli.Assembly, error) {
			return cli.Assembly{PublicClient: client.NewPublic(client.PublicConfig{BaseURL: srv.URL})}, nil
		},
		Stdout: &out, Stderr: &errOut, BuildInfo: cli.BuildInfo{Version: "test"}, Now: today,
	}, []string{"classroom", "search", "--week", "3"})
	if code != cli.ExitOK {
		t.Fatalf("退出码 = %d，stderr=%q", code, errOut.String())
	}
	if strings.Join(paths, ",") != "/OtherQueryServlet,/KongClassRoomQuery" || len(forms) != 1 {
		t.Fatalf("paths=%v forms=%v", paths, forms)
	}
	want := url.Values{"term_xiala": {"2026.1"}, "weekSpan_xiala": {"3"}, "week_xiala": {"星期四"}, "classSpan_xiala": {"1,2"}}
	if !reflect.DeepEqual(forms[0], want) {
		t.Fatalf("form = %v, want %v", forms[0], want)
	}
	if !strings.Contains(out.String(), `"count": 2`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestClassroomSearchRequiresWeekWithoutRequest(t *testing.T) {
	called := false
	got := runPublicWith(t, func() (cli.Assembly, error) {
		called = true
		return cli.Assembly{}, errors.New("assembler should not be called")
	}, "classroom", "search")
	if got.code != cli.ExitUsage || called ||
		!strings.Contains(got.stderr, "--week 必须是正整数") ||
		!strings.Contains(got.stderr, "ecjtu classroom options") {
		t.Fatalf("code=%d called=%v stderr=%q", got.code, called, got.stderr)
	}
}

func TestClassroomSearchRejectsBlankRoomWithoutRequest(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t\n"} {
		t.Run("raw="+raw, func(t *testing.T) {
			called := false
			got := runPublicWith(t, func() (cli.Assembly, error) {
				called = true
				return cli.Assembly{}, errors.New("assembler should not be called")
			}, "classroom", "search", "--week", "3", "--room", raw)
			if got.code != cli.ExitUsage || called || !strings.Contains(got.stderr, "--room 不能为空白") {
				t.Fatalf("code=%d called=%v stderr=%q", got.code, called, got.stderr)
			}
			if got.stdout != "" {
				t.Fatalf("失败路径不应写 stdout，实得 %q", got.stdout)
			}
		})
	}
}

func TestClassroomSearchFiltersByRoomName(t *testing.T) {
	// 故意把其他楼栋的教室类型写成「多媒体教室」、座位数写成 31，验证 --room 只匹配教室名。
	resultPage := []byte(`<!doctype html>
<html>
<head><title>空闲教室使用情况</title></head>
<body>
<div class="text">共有 <mark>3</mark> 间空闲教室</div>
<div class="tb">
  <table>
    <tr><th>教室</th><th>教室类型</th><th>座位数</th></tr>
    <tr><td>31-A101</td><td>普通教室</td><td>120</td></tr>
    <tr><td>教31栋422</td><td>多媒体教室</td><td>90</td></tr>
    <tr><td>教一-203</td><td>多媒体教室</td><td>31</td></tr>
  </table>
</div>
</body>
</html>`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write(resultPage)
			return
		}
		_, _ = w.Write(testsupport.Fixture(t, "classroom-options-synthetic.html"))
	}))
	defer srv.Close()

	assemble := func() (cli.Assembly, error) {
		return cli.Assembly{PublicClient: client.NewPublic(client.PublicConfig{BaseURL: srv.URL})}, nil
	}

	tests := []struct {
		name          string
		args          []string
		wantStdout    []string
		notWantStdout []string
		notWantStderr []string
	}{
		{
			name:          "按楼栋子串筛选且不误匹配座位数/json",
			args:          []string{"classroom", "search", "--week", "3", "--room", " 31 "},
			wantStdout:    []string{`"count": 2`, `"31-A101"`, `"教31栋422"`},
			notWantStdout: []string{`"教一-203"`, `"room":`},
		},
		{
			name:          "忽略大小写匹配/json",
			args:          []string{"classroom", "search", "--week", "3", "--room", "31-a"},
			wantStdout:    []string{`"count": 1`, `"31-A101"`},
			notWantStdout: []string{`"教31栋422"`, `"教一-203"`},
		},
		{
			name:          "不匹配教室类型字段/json",
			args:          []string{"classroom", "search", "--week", "3", "--room", "多媒体"},
			wantStdout:    []string{`"count": 0`, `"rooms": []`},
			notWantStdout: []string{`"教31栋422"`, `"教一-203"`},
		},
		{
			name:          "筛选命中/text",
			args:          []string{"classroom", "search", "--week", "3", "--room", "31", "--format", "text"},
			wantStdout:    []string{"2026.1 第3周 星期四 1,2（教室含 31，共 2 间）", "31-A101", "教31栋422"},
			notWantStdout: []string{"教一-203"},
		},
		{
			name:       "筛选零命中/text",
			args:       []string{"classroom", "search", "--week", "3", "--room", "99", "--format", "text"},
			wantStdout: []string{"该条件下没有教室名包含「99」的空闲教室。"},
		},
		{
			name:          "verbose筛选后不误报条数不一致",
			args:          []string{"classroom", "search", "--week", "3", "--room", "31-a", "--verbose"},
			wantStdout:    []string{`"count": 1`, `"31-A101"`},
			notWantStderr: []string{"warn=count-mismatch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runPublicWith(t, assemble, tt.args...)
			if got.code != cli.ExitOK {
				t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", got.code, cli.ExitOK, got.stderr)
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(got.stdout, want) {
					t.Fatalf("stdout 缺少 %q:\n%s", want, got.stdout)
				}
			}
			for _, bad := range tt.notWantStdout {
				if strings.Contains(got.stdout, bad) {
					t.Fatalf("stdout 不应包含 %q:\n%s", bad, got.stdout)
				}
			}
			for _, bad := range tt.notWantStderr {
				if strings.Contains(got.stderr, bad) {
					t.Fatalf("stderr 不应包含 %q:\n%s", bad, got.stderr)
				}
			}
		})
	}
}

func TestExitCodeMatrix(t *testing.T) {
	fixture := func(name string) []byte { return testsupport.Fixture(t, name) }
	tests := []struct {
		name       string
		status     int
		body       []byte
		args       []string
		wantCode   int
		wantStdout []string // 必须出现在 stdout 的片段
		wantStderr []string // 必须出现在 stderr 的片段
		wantHits   int      // 上游请求次数（本地校验类用例必须为 0）
	}{
		{"课表正常/json", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "--date", "2026-09-14"}, 0,
			[]string{`"date": "2026-09-14"`, `"weekday": 1`, `"week": 3`, `"teacher": "教师B"`}, nil, 1},
		{"课表正常/text", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "--date", "2026-09-14", "--format", "text"}, 0,
			[]string{"2026-09-14 星期一（第3周）", "节次", "大学英语Ⅲ(上课)"}, nil, 1},
		{"无课日/json", 200, fixture("course-noclass-sunday.html"),
			[]string{"course", "--date", "2026-09-13"}, 0,
			[]string{`"courses": []`}, nil, 1},
		{"无课日/text", 200, fixture("course-noclass-sunday.html"),
			[]string{"course", "--date", "2026-09-13", "--format", "text"}, 0,
			[]string{"当天无课。"}, nil, 1},
		{"成绩正常/json", 200, fixture("score-normal-2025-2.html"),
			[]string{"score", "--term", "2025.2"}, 0,
			[]string{`"term": "2025.2"`, `"studentName": "张三"`, `"count": 15`, `"courseName": "高等数学(A)Ⅱ"`}, nil, 1},
		{"成绩正常/text", 200, fixture("score-normal-2025-2.html"),
			[]string{"score", "--term", "2025.2", "--format", "text"}, 0,
			[]string{"张三 2025.2（共 15 门）", "职业生涯与发展规划", "学科任选课"}, nil, 1},
		{"成绩空态/json", 200, fixture("score-empty-2026-2.html"),
			[]string{"score", "--term", "2026.2"}, 0,
			[]string{`"empty": true`, `"count": 0`, `"items": []`, `"term": null`}, nil, 1},
		{"成绩空态/text", 200, fixture("score-empty-2026-2.html"),
			[]string{"score", "--term", "2026.2", "--format", "text"}, 0,
			[]string{"该学期无成绩。"}, nil, 1},
		{"考试正常/json", 200, fixture("exam-normal-2025-2.html"),
			[]string{"exam", "--term", "2025.2"}, 0,
			[]string{`"examDate": "2026-07-07"`, `"headcount": 31`, `"note": "进贤校区"`}, nil, 1},
		{"考试正常/text", 200, fixture("exam-normal-2025-2.html"),
			[]string{"exam", "--term", "2025.2", "--format", "text"}, 0,
			[]string{"张三 2025.2（共 5 场）", "备注：进贤校区"}, nil, 1},
		{"考试空态/json", 200, fixture("exam-empty-current.html"),
			[]string{"exam"}, 0,
			[]string{`"empty": true`}, nil, 1},

		{"坏日期/不发请求", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "--date", "2026-9-14"}, 2, nil, []string{"--date 必须是 YYYY-MM-DD"}, 0},
		{"不存在的日历日", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "--date", "2026-02-30"}, 2, nil, []string{"--date 必须是 YYYY-MM-DD"}, 0},
		{"坏学期", 200, fixture("score-normal-2025-2.html"),
			[]string{"score", "--term", "2025-2"}, 2, nil, []string{"--term 必须是 YYYY.N"}, 0},
		{"坏格式", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "--format", "yaml"}, 2, nil, []string{"--format 只接受 json 或 text"}, 0},
		{"未知flag", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "--nope"}, 2, nil, []string{"unknown flag"}, 0},
		{"多余位置参数", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "2026-09-14"}, 2, nil, []string{"unknown command"}, 0},
		{"缺子命令", 200, fixture("course-normal-week3-monday.html"),
			[]string{}, 2, nil, []string{"缺少子命令"}, 0},
		{"负timeout", 200, fixture("course-normal-week3-monday.html"),
			[]string{"course", "--timeout", "-1s"}, 2, nil, []string{"--timeout 必须是正的时间长度"}, 0},

		{"未绑定", 200, fixture("notbound.html"),
			[]string{"course"}, 3, nil, []string{"教务系统未绑定该微信账号"}, 1},
		{"未绑定/成绩域", 200, fixture("notbound.html"),
			[]string{"score"}, 3, nil, []string{"请在公众号里重新绑定"}, 1},
		{"上游500", 500, fixture("tomcat500.html"),
			[]string{"course"}, 4, nil, []string{"教务系统暂时不可用（上游返回 500）"}, 1},
		{"状态码撒谎的异常页", 200, fixture("tomcat500.html"),
			[]string{"course"}, 4, nil, []string{"教务系统暂时不可用（上游返回 200）"}, 1},

		{"改版页/无center", 200, fixture("course-missing-center.html"),
			[]string{"course"}, 5, nil, []string{"教务页面结构与预期不符"}, 1},
		{"改版页/无法归类", 200, fixture("unknown.html"),
			[]string{"course"}, 5, nil, []string{"教务页面结构与预期不符"}, 1},
		{"改版页/课表出现空态文案", 200, fixture("course-unexpected-empty.html"),
			[]string{"course"}, 5, nil, []string{"教务页面结构与预期不符"}, 1},
		{"改版页/考试标签缺失", 200, fixture("exam-missing-label.html"),
			[]string{"exam"}, 5, nil, []string{"教务页面结构与预期不符"}, 1},
		{"改版页/考试时间形态异常", 200, fixture("exam-bad-time.html"),
			[]string{"exam"}, 5, nil, []string{"教务页面结构与预期不符"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t, tt.status, tt.body)
			got := s.run(t, tt.args...)
			if got.code != tt.wantCode {
				t.Fatalf("退出码 = %d，期望 %d\nstdout=%q\nstderr=%q", got.code, tt.wantCode, got.stdout, got.stderr)
			}
			for _, want := range tt.wantStdout {
				if !strings.Contains(got.stdout, want) {
					t.Fatalf("stdout 缺少 %q:\n%s", want, got.stdout)
				}
			}
			for _, want := range tt.wantStderr {
				if !strings.Contains(got.stderr, want) {
					t.Fatalf("stderr 缺少 %q:\n%s", want, got.stderr)
				}
			}
			if got.hits != tt.wantHits {
				t.Fatalf("上游请求次数 = %d，期望 %d", got.hits, tt.wantHits)
			}
			if tt.wantCode != 0 && got.stdout != "" {
				t.Fatalf("失败路径不应写 stdout，实得 %q", got.stdout)
			}
			assertNoLeak(t, got)
		})
	}
}

// assertNoLeak 检查输出里没有完整 openid，也没有上游原文与 query 泄漏。
func assertNoLeak(t *testing.T, got result) {
	t.Helper()
	joined := got.stdout + got.stderr
	for _, forbidden := range []string{fakeOpenID, "weiXinID", "jsessionid", "JSESSIONID", "<title", "Tomcat", "暂无"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("输出泄漏了 %q:\n%s", forbidden, joined)
		}
	}
}

func TestUpstreamPathsAndParams(t *testing.T) {
	tests := []struct {
		args     []string
		fixture  string
		wantPath string
		wantDate string
		wantTerm string
	}{
		{[]string{"course", "--date", "2026-09-20"}, "course-normal-week3-monday.html", "/CalendarServlet", "2026-09-20", ""},
		{[]string{"course"}, "course-noclass-sunday.html", "/CalendarServlet", "2026-09-14", ""}, // 缺省用注入时钟的「今天」
		{[]string{"score", "--term", "2025.1"}, "score-normal-2025-1.html", "/ScoreQuery", "", "2025.1"},
		{[]string{"exam", "--term", "2025.1"}, "exam-normal-2025-1.html", "/ExamArrangeCl", "", "2025.1"},
	}
	for _, tt := range tests {
		s := newStub(t, 200, testsupport.Fixture(t, tt.fixture))
		got := s.run(t, tt.args...)
		if got.code != 0 {
			t.Fatalf("退出码 = %d，期望 0（stderr=%q）", got.code, got.stderr)
		}
		if got.path != tt.wantPath {
			t.Fatalf("请求路径 = %q，期望 %q", got.path, tt.wantPath)
		}
		if tt.wantDate != "" && got.query.Get("date") != tt.wantDate {
			t.Fatalf("date 参数 = %q，期望 %q", got.query.Get("date"), tt.wantDate)
		}
		if got.query.Get("term") != tt.wantTerm {
			t.Fatalf("term 参数 = %q，期望 %q", got.query.Get("term"), tt.wantTerm)
		}
		if got.query.Get("weiXinID") != fakeOpenID {
			t.Fatalf("weiXinID 参数未正确映射: %q", got.query.Get("weiXinID"))
		}
	}
}

func TestNoTermSendsNoTermParam(t *testing.T) {
	s := newStub(t, 200, testsupport.Fixture(t, "score-normal-2025-2.html"))
	got := s.run(t, "score")
	if _, ok := got.query["term"]; ok {
		t.Fatalf("省略 --term 时不应发送 term: %v", got.query)
	}
}

func TestVerboseWritesDiagnosticsToStderrOnly(t *testing.T) {
	s := newStub(t, 200, testsupport.Fixture(t, "course-normal-week3-monday.html"))
	got := s.run(t, "course", "--date", "2026-09-14", "--verbose")
	if got.code != 0 {
		t.Fatalf("退出码 = %d，stderr=%q", got.code, got.stderr)
	}
	var v any
	if err := json.Unmarshal([]byte(got.stdout), &v); err != nil {
		t.Fatalf("--verbose 污染了 stdout: %v\n%s", err, got.stdout)
	}
	lines := strings.Split(strings.TrimSpace(got.stderr), "\n")
	if len(lines) != 1 {
		t.Fatalf("stderr 应只有一行诊断，实得:\n%s", got.stderr)
	}
	fields := strings.Fields(lines[0])
	wantPrefix := []string{"phase=fetch", "domain=course", "op=course.Day", "status=200", "kind=ok", "attempts=1"}
	for i, want := range wantPrefix {
		if i >= len(fields) || fields[i] != want {
			t.Fatalf("诊断行格式漂移: %q，第 %d 段期望 %q", lines[0], i, want)
		}
	}
	if !strings.HasSuffix(fields[len(fields)-1], "ms=0") && !strings.Contains(fields[len(fields)-1], "ms=") {
		t.Fatalf("诊断行缺少 ms=: %q", lines[0])
	}
}

func TestVerboseErrorLine(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		args     []string
		wantKind string
	}{
		{"未绑定", "notbound.html", []string{"course"}, "notbound"},
		{"改版", "unknown.html", []string{"course"}, "parse"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newStub(t, 200, testsupport.Fixture(t, tt.body))
			got := s.run(t, append(tt.args, "--verbose")...)
			if !strings.Contains(got.stderr, "phase=error kind="+tt.wantKind) {
				t.Fatalf("stderr 缺少 phase=error kind=%s:\n%s", tt.wantKind, got.stderr)
			}
			assertNoLeak(t, got)
		})
	}
}

func TestCountMismatchWarning(t *testing.T) {
	s := newStub(t, 200, testsupport.Fixture(t, "score-no-rows.html"))
	got := s.run(t, "score", "--term", "2025.2", "--verbose")
	if got.code != 0 {
		t.Fatalf("退出码 = %d，期望 0（stderr=%q）", got.code, got.stderr)
	}
	if !strings.Contains(got.stderr, "warn=count-mismatch domain=score declared=15 parsed=0") {
		t.Fatalf("stderr 缺少条数不一致告警:\n%s", got.stderr)
	}
	if !strings.Contains(got.stdout, `"count": 0`) {
		t.Fatalf("stdout 应仍给出 count: 0:\n%s", got.stdout)
	}
	assertNoLeak(t, got)
}

func TestQuietHasNoWarning(t *testing.T) {
	s := newStub(t, 200, testsupport.Fixture(t, "score-no-rows.html"))
	got := s.run(t, "score", "--term", "2025.2")
	if got.stderr != "" {
		t.Fatalf("默认不应有任何 stderr 输出:\n%s", got.stderr)
	}
}

// 端口不通归为 KindTransport，退出码 4。预算压到 300ms，
// 让第一次退避就被 ctx 打断，gate 不做真实长等待。
func TestConnectionRefusedExits4(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := dead.URL
	dead.Close()
	s := &stub{}
	s.srv = &httptest.Server{Listener: nil, URL: url}
	got := s.run(t, "course", "--timeout", "300ms")
	if got.code != cli.ExitUpstream {
		t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", got.code, cli.ExitUpstream, got.stderr)
	}
	if !strings.Contains(got.stderr, "无法连接教务系统") {
		t.Fatalf("stderr 缺少网络文案:\n%s", got.stderr)
	}
	assertNoLeak(t, got)
}

func TestTimeoutBudgetProducesExit4(t *testing.T) {
	s := newStub(t, 200, nil).slow(2 * time.Second)
	got := s.run(t, "course", "--timeout", "150ms")
	if got.code != cli.ExitUpstream {
		t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", got.code, cli.ExitUpstream, got.stderr)
	}
	if !strings.Contains(got.stderr, "无法连接教务系统") {
		t.Fatalf("stderr 缺少超时文案:\n%s", got.stderr)
	}
	assertNoLeak(t, got)
}

// 凭证也没配时，非法输入要报输入错，不能把「日期写错」念成「凭证未配置」。
func TestInputBeatsConfigError(t *testing.T) {
	noCred := func() (cli.Assembly, error) {
		return cli.Assembly{}, cli.ConfigError("ECJTU_WEIXIN_TOKEN", errors.New("ECJTU_WEIXIN_TOKEN is not set"))
	}
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"course", "--date", "2026-13-99"}, "--date 必须是 YYYY-MM-DD"},
		{[]string{"score", "--term", "bogus"}, "--term 必须是 YYYY.N"},
		{[]string{"exam", "--term", "2025.2"}, "教务凭证未配置"},
	}
	for _, tt := range tests {
		got := runWith(t, noCred, tt.args...)
		if got.code != cli.ExitUsage {
			t.Fatalf("%v 退出码 = %d，期望 %d", tt.args, got.code, cli.ExitUsage)
		}
		if !strings.Contains(got.stderr, tt.want) {
			t.Fatalf("%v 的 stderr 缺少 %q:\n%s", tt.args, tt.want, got.stderr)
		}
	}
}

func TestMissingTokenExits2(t *testing.T) {
	assemble := func() (cli.Assembly, error) {
		return cli.Assembly{}, cli.ConfigError("ECJTU_WEIXIN_TOKEN", errors.New("ECJTU_WEIXIN_TOKEN is not set"))
	}
	got := runWith(t, assemble, "course")
	if got.code != cli.ExitUsage {
		t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", got.code, cli.ExitUsage, got.stderr)
	}
	if !strings.Contains(got.stderr, "教务凭证未配置") || !strings.Contains(got.stderr, "ECJTU_WEIXIN_TOKEN") {
		t.Fatalf("stderr 缺少配置文案:\n%s", got.stderr)
	}
}

func TestHelpAndVersionNeedNoCredentials(t *testing.T) {
	failing := func() (cli.Assembly, error) {
		return cli.Assembly{}, cli.ConfigError("ECJTU_WEIXIN_TOKEN", errors.New("ECJTU_WEIXIN_TOKEN is not set"))
	}
	for _, tt := range []struct {
		args    []string
		want    []string
		notWant []string
	}{
		{[]string{"--help"}, []string{"course", "score", "exam", "classroom", "--verbose", "ECJTU_WEIXIN_TOKEN", "退出码", "默认 15s"}, []string{"--format"}},
		{[]string{"course", "--help"}, []string{"--date", "--format", "ECJTU_WEIXIN_TOKEN", "2026-09-14"}, nil},
		{[]string{"score", "--help"}, []string{"--term", "--format", "ECJTU_WEIXIN_TOKEN", "2025.2"}, nil},
		{[]string{"exam", "--help"}, []string{"--term", "--format", "ECJTU_WEIXIN_TOKEN", "考试安排"}, nil},
		{[]string{"classroom", "--help"}, []string{"options", "search", "无需配置 ECJTU_WEIXIN_TOKEN", "等同于 ecjtu classroom options"}, nil},
		{[]string{"classroom", "options", "--help"}, []string{"无需配置 ECJTU_WEIXIN_TOKEN", "ecjtu classroom options --format text"}, nil},
		{[]string{"classroom", "search", "--help"}, []string{"--week", "--room", "必填", "1,2 / 3,4 / 5,6 / 7,8 / 9,10 / 11,12", "无需配置 ECJTU_WEIXIN_TOKEN"}, nil},
		{[]string{"--version"}, []string{"ecjtu test"}, nil},
		{[]string{"completion", "--help"}, []string{"bash", "zsh"}, nil},
	} {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			got := runWith(t, failing, tt.args...)
			if got.code != 0 {
				t.Fatalf("退出码 = %d，期望 0\nstderr=%q", got.code, got.stderr)
			}
			for _, want := range tt.want {
				if !strings.Contains(got.stdout, want) {
					t.Fatalf("help 输出缺少 %q:\n%s", want, got.stdout)
				}
			}
			for _, bad := range tt.notWant {
				if strings.Contains(got.stdout, bad) {
					t.Fatalf("help 输出不应包含 %q:\n%s", bad, got.stdout)
				}
			}
		})
	}
}

func TestNonQueryCommandsRejectFormatFlag(t *testing.T) {
	for _, args := range [][]string{
		{"--format", "text"},
		{"completion", "bash", "--format", "text"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			s := newStub(t, 200, testsupport.Fixture(t, "course-normal-week3-monday.html"))
			got := s.run(t, args...)
			if got.code != cli.ExitUsage {
				t.Fatalf("退出码 = %d，期望 %d\nstderr=%q", got.code, cli.ExitUsage, got.stderr)
			}
			if !strings.Contains(got.stderr, "unknown flag: --format") {
				t.Fatalf("stderr 应报告 unknown flag --format:\n%s", got.stderr)
			}
			if got.hits != 0 {
				t.Fatalf("非查询命令拒绝 --format 时不应发请求，实发 %d 次", got.hits)
			}
		})
	}
}
