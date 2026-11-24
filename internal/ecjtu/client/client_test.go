package client_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ecjtu "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/client"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

func TestCourseUsesConfiguredUserAgent(t *testing.T) {
	page := testsupport.Fixture(t, "course-normal-week3-monday.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("weiXinID") != "TESTOPENIDXXXXXXXXXXXXXXXXX" {
			t.Error("openid 不匹配")
		}
		if r.Header.Get("User-Agent") != "ecjtu-test/1.0" {
			t.Errorf("UA = %q", r.Header.Get("User-Agent"))
		}
		if r.URL.Query().Get("date") != "2026-09-14" {
			t.Error("日期不匹配")
		}
		_, _ = w.Write(page)
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL, UserAgent: "ecjtu-test/1.0"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := school.Course(t.Context(), time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Courses) != 3 {
		t.Fatalf("课程数量 = %d", len(result.Courses))
	}
}

func TestQueriesShareOpenIDAndDefaultUserAgent(t *testing.T) {
	pages := map[string][]byte{
		"/ScoreQuery":    testsupport.Fixture(t, "score-normal-2025-1.html"),
		"/ExamArrangeCl": testsupport.Fixture(t, "exam-normal-2025-1.html"),
	}
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Query().Get("weiXinID") != "TESTOPENIDXXXXXXXXXXXXXXXXX" {
			t.Error("openid 不匹配")
		}
		if r.URL.Query().Get("term") != "2025.1" {
			t.Errorf("学期 = %q", r.URL.Query().Get("term"))
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "MicroMessenger") {
			t.Errorf("默认 UA = %q", r.Header.Get("User-Agent"))
		}
		page, ok := pages[r.URL.Path]
		if !ok {
			t.Errorf("未知路径 %q", r.URL.Path)
			return
		}
		_, _ = w.Write(page)
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	scores, err := school.Scores(t.Context(), "2025.1")
	if err != nil {
		t.Fatal(err)
	}
	if scores.Empty || len(scores.Items) == 0 {
		t.Fatal("成绩结果为空")
	}
	exams, err := school.Exams(t.Context(), "2025.1")
	if err != nil {
		t.Fatal(err)
	}
	if exams.Empty || len(exams.Items) == 0 {
		t.Fatal("考试结果为空")
	}
	if hits != 2 {
		t.Fatalf("请求次数 = %d", hits)
	}
}

func TestNewRequiresOpenID(t *testing.T) {
	_, err := ecjtu.New(ecjtu.Config{})
	if err == nil {
		t.Fatal("缺少 openid 应返回错误")
	}
}

func TestUserAgentAppliesToAllQueries(t *testing.T) {
	pages := map[string][]byte{
		"/CalendarServlet": testsupport.Fixture(t, "course-normal-week3-monday.html"),
		"/ScoreQuery":      testsupport.Fixture(t, "score-normal-2025-1.html"),
		"/ExamArrangeCl":   testsupport.Fixture(t, "exam-normal-2025-1.html"),
	}
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if got := r.Header.Get("User-Agent"); got != "custom-ecjtu/1.0" {
			t.Errorf("%s 的 UA = %q", r.URL.Path, got)
		}
		_, _ = w.Write(pages[r.URL.Path])
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL, UserAgent: "custom-ecjtu/1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := school.Course(t.Context(), time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := school.Scores(t.Context(), "2025.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := school.Exams(t.Context(), "2025.1"); err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Fatalf("请求次数 = %d", hits)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHTTPTransportAppliesToQueries(t *testing.T) {
	pages := map[string][]byte{
		"/weixin/CalendarServlet": testsupport.Fixture(t, "course-normal-week3-monday.html"),
		"/weixin/ScoreQuery":      testsupport.Fixture(t, "score-normal-2025-1.html"),
		"/weixin/ExamArrangeCl":   testsupport.Fixture(t, "exam-normal-2025-1.html"),
	}
	var hits int
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		hits++
		if r.URL.Query().Get("weiXinID") != "TESTOPENIDXXXXXXXXXXXXXXXXX" {
			t.Errorf("%s 缺少 openid", r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "MicroMessenger") {
			t.Errorf("%s 未携带默认微信 UA: %q", r.URL.Path, r.Header.Get("User-Agent"))
		}
		body, ok := pages[r.URL.Path]
		if !ok {
			t.Fatalf("未知路径 %q", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil
	})

	school, err := ecjtu.New(ecjtu.Config{
		OpenID:        "TESTOPENIDXXXXXXXXXXXXXXXXX",
		HTTPTransport: rt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := school.Course(t.Context(), time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := school.Scores(t.Context(), "2025.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := school.Exams(t.Context(), "2025.1"); err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Fatalf("自定义 HTTPTransport 命中次数 = %d，期望 3", hits)
	}
}
