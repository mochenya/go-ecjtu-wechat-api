package ecjtu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeOpenID 为 28 位测试假值（与真实 openid 同形，绝不用于真实请求）。
const fakeOpenID = "oFAKE01234567890123456789012"

type hitLog struct {
	mu      sync.Mutex
	count   int
	headers []http.Header
	queries []url.Values
}

func (h *hitLog) record(r *http.Request) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.count++
	h.headers = append(h.headers, r.Header.Clone())
	h.queries = append(h.queries, r.URL.Query())
	return h.count
}

func (h *hitLog) total() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.count
}

// newTestTransport 起 httptest 服务器；handler 收到的 n 是当前第几次命中。
// 退避注入为即时返回：测试不得真实等待。
func newTestTransport(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, n int)) (*Transport, *httptest.Server, *hitLog) {
	t.Helper()
	hl := &hitLog{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, hl.record(r))
	}))
	t.Cleanup(srv.Close)
	tr := NewTransport(srv.URL, fakeOpenID)
	tr.backoff = []time.Duration{time.Millisecond, 2 * time.Millisecond}
	tr.sleep = func(context.Context, time.Duration) error { return nil }
	return tr, srv, hl
}

// failConnection 无响应直接断开连接，触发连接层错误。
func failConnection(w http.ResponseWriter, _ *http.Request, _ int) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "no hijacker", http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = conn.Close()
}

func TestGetSendsWeChatHeadersAndOpenIDQuery(t *testing.T) {
	tr, _, hl := newTestTransport(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		if r.URL.Path != "/CalendarServlet" {
			t.Errorf("path = %q, 期望 /CalendarServlet", r.URL.Path)
		}
		fmt.Fprint(w, "<html>ok</html>")
	})

	resp, err := tr.Get(context.Background(), "course.Day", "CalendarServlet", url.Values{"date": {"2026-09-14"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Attempts != 1 || resp.Status != 200 || string(resp.Body) != "<html>ok</html>" {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Elapsed <= 0 {
		t.Errorf("Elapsed 应为正数: %v", resp.Elapsed)
	}

	q := hl.queries[0]
	if q.Get("weiXinID") != fakeOpenID { // 参数拼写只允许出现在 transport 映射处
		t.Errorf("weiXinID 参数未正确携带")
	}
	if q.Get("date") != "2026-09-14" {
		t.Errorf("date 参数丢失: %v", q)
	}
	if ua := hl.headers[0].Get("User-Agent"); !strings.Contains(ua, "MicroMessenger") {
		t.Errorf("未携带微信 UA: %s", ua)
	}
}

func TestGetRetriesConnectionFailuresUpToMaxAttempts(t *testing.T) {
	var waits []time.Duration
	tr, _, hl := newTestTransport(t, func(w http.ResponseWriter, r *http.Request, n int) {
		if n < 3 {
			failConnection(w, r, n)
			return
		}
		fmt.Fprint(w, "done")
	})
	tr.sleep = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}

	resp, err := tr.Get(context.Background(), "course.Day", "CalendarServlet", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Attempts != 3 || hl.total() != 3 {
		t.Fatalf("attempts = %d, hits = %d, 期望 3/3", resp.Attempts, hl.total())
	}
	if len(waits) != 2 || waits[0] != time.Millisecond || waits[1] != 2*time.Millisecond {
		t.Fatalf("退避序列错误: %v", waits)
	}
}

func TestGetStopsAfterMaxAttempts(t *testing.T) {
	tr, _, hl := newTestTransport(t, failConnection)

	_, err := tr.Get(context.Background(), "course.Day", "CalendarServlet", nil)
	if err == nil {
		t.Fatal("期望错误")
	}
	if hl.total() != maxAttempts {
		t.Fatalf("hits = %d, 期望 %d", hl.total(), maxAttempts)
	}
	k, ok := KindOf(err)
	if !ok || k != KindTransport {
		t.Fatalf("分类 = %v,%v，期望 transport,true", k, ok)
	}
	// 错误串不得出现完整 openid（脱敏是这层的职责）。
	if strings.Contains(err.Error(), fakeOpenID) {
		t.Fatalf("error 泄露 openid: %v", err)
	}
}

func TestGetDoesNotRetryHTTPResponses(t *testing.T) {
	tr, _, hl := newTestTransport(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "<html><title>HTTP状态 500</title></html>")
	})

	resp, err := tr.Get(context.Background(), "course.Day", "CalendarServlet", nil)
	if err != nil {
		t.Fatalf("拿到响应不应报错（归类是 Classify 的事）: %v", err)
	}
	if resp.Status != 500 || hl.total() != 1 {
		t.Fatalf("status=%d hits=%d，期望 500/1", resp.Status, hl.total())
	}
}

func TestGetSharesContextBudgetAcrossAttempts(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	tr, _, hl := newTestTransport(t, func(w http.ResponseWriter, r *http.Request, _ int) {
		select {
		case <-release:
			fmt.Fprint(w, "late")
		case <-r.Context().Done():
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := tr.Get(ctx, "course.Day", "CalendarServlet", nil)
	if err == nil {
		t.Fatal("预算耗尽应报错")
	}
	if k, _ := KindOf(err); k != KindTransport {
		t.Fatalf("分类 = %v，期望 transport", k)
	}
	if got := hl.total(); got != 1 {
		t.Fatalf("ctx 耗尽后不应重试: hits = %d", got)
	}
}

func TestGetNeverSendsCookies(t *testing.T) {
	tr, _, hl := newTestTransport(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		w.Header().Add("Set-Cookie", "JSESSIONID=0123456789ABCDEF0123456789ABCDEF; Path=/weixin; HttpOnly")
		fmt.Fprint(w, "ok")
	})

	for range 2 {
		if _, err := tr.Get(context.Background(), "course.Day", "CalendarServlet", nil); err != nil {
			t.Fatal(err)
		}
	}
	for i, h := range hl.headers {
		if c := h.Get("Cookie"); c != "" {
			t.Fatalf("第 %d 次请求携带了 cookie: %s", i+1, c)
		}
	}
}

func TestGetRejectsOversizedBody(t *testing.T) {
	tr, _, hl := newTestTransport(t, func(w http.ResponseWriter, _ *http.Request, _ int) {
		_, _ = io.WriteString(w, strings.Repeat("x", 2048))
	})
	tr.maxBodyBytes = 64

	_, err := tr.Get(context.Background(), "course.Day", "CalendarServlet", nil)
	if err == nil {
		t.Fatal("超限响应体应报错")
	}
	if k, _ := KindOf(err); k != KindParse {
		t.Fatalf("分类 = %v，期望 parse", k)
	}
	if hl.total() != 1 {
		t.Fatalf("超限已拿到响应，不应重试: hits = %d", hl.total())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewTransportWithConfigUsesCustomRoundTripper(t *testing.T) {
	var cookies []string
	var hits int
	tr := NewTransportWithConfig(TransportConfig{
		BaseURL: "https://mock.invalid/weixin",
		OpenID:  fakeOpenID,
		RoundTripper: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			hits++
			cookies = append(cookies, r.Header.Get("Cookie"))
			if r.URL.Query().Get("weiXinID") != fakeOpenID {
				t.Errorf("weiXinID 未携带: %q", r.URL.Query().Get("weiXinID"))
			}
			if !strings.Contains(r.Header.Get("User-Agent"), "MicroMessenger") {
				t.Errorf("未携带默认微信 UA: %q", r.Header.Get("User-Agent"))
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			resp.Header.Add("Set-Cookie", "JSESSIONID=SECRET; Path=/weixin")
			return resp, nil
		}),
	})

	for range 2 {
		resp, err := tr.Get(context.Background(), "course.Day", "CalendarServlet", nil)
		if err != nil {
			t.Fatalf("Get 失败: %v", err)
		}
		if string(resp.Body) != "ok" {
			t.Fatalf("Body = %q，期望 ok", resp.Body)
		}
	}
	if hits != 2 {
		t.Fatalf("自定义 RoundTripper 命中次数 = %d，期望 2", hits)
	}
	for i, c := range cookies {
		if c != "" {
			t.Fatalf("第 %d 次请求泄漏了 cookie: %q", i+1, c)
		}
	}
}

func TestPublicTransportOmitsIdentityAndDoesNotReuseCookies(t *testing.T) {
	var requests []*http.Request
	tr := NewPublicTransportWithConfig(PublicTransportConfig{
		BaseURL: "https://mock.invalid/weixin",
		RoundTripper: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			requests = append(requests, r.Clone(context.Background()))
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ok")),
			}
			resp.Header.Add("Set-Cookie", "JSESSIONID=SECRET; Path=/weixin")
			return resp, nil
		}),
	})

	if _, err := tr.GetPublic(context.Background(), "classroom.Options", "OtherQueryServlet", url.Values{"type": {"1"}, "weiXinID": {"should-not-send"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.GetPublic(context.Background(), "classroom.Options", "OtherQueryServlet", nil); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	for _, r := range requests {
		if got := r.URL.Query().Get("weiXinID"); got != "" {
			t.Fatalf("public request carries weiXinID=%q", got)
		}
		if got := r.Header.Get("Cookie"); got != "" {
			t.Fatalf("public request carries cookie=%q", got)
		}
	}
}

func TestPostPublicDoesNotRetry(t *testing.T) {
	hits := 0
	tr := NewPublicTransportWithConfig(PublicTransportConfig{
		BaseURL: "https://mock.invalid/weixin",
		RoundTripper: roundTripFunc(func(*http.Request) (*http.Response, error) {
			hits++
			return nil, errors.New("connection reset")
		}),
	})

	_, err := tr.PostPublic(context.Background(), "classroom.Search", "KongClassRoomQuery", url.Values{"weekSpan_xiala": {"3"}})
	if err == nil {
		t.Fatal("POST 失败应返回错误")
	}
	if hits != 1 {
		t.Fatalf("POST retries = %d, want 1", hits)
	}
}
