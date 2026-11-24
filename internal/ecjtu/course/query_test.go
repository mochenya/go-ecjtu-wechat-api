package course_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/course"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

// testOpenID 是 fixture 与测试共用的假凭证，不是真实值。
const testOpenID = "TESTOPENIDXXXXXXXXXXXXXXXXX"

func TestQueryDaySuccess(t *testing.T) {
	body := testsupport.Fixture(t, "course-normal-week3-monday.html")
	var gotPath, gotDate, gotOpenID, gotCookie, gotUA string
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		gotPath = r.URL.Path
		gotDate = r.URL.Query().Get("date")
		gotOpenID = r.URL.Query().Get("weiXinID")
		gotCookie = r.Header.Get("Cookie")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Set-Cookie", "JSESSIONID=DEADBEEF; Path=/weixin")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	q := course.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID))
	got, err := q.Day(t.Context(), time.Date(2026, 9, 14, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("Day 失败: %v", err)
	}
	if len(got.Courses) != 3 {
		t.Fatalf("期望 3 节课，实际 %d", len(got.Courses))
	}
	if gotPath != "/CalendarServlet" {
		t.Fatalf("path = %q, 期望 /CalendarServlet", gotPath)
	}
	if gotDate != "2026-09-14" {
		t.Fatalf("date 参数 = %q, 期望 2026-09-14", gotDate)
	}
	if gotOpenID != testOpenID {
		t.Fatal("weiXinID 参数不匹配")
	}
	if gotUA == "" || !strings.Contains(gotUA, "MicroMessenger") {
		t.Fatalf("未发送微信 UA: %q", gotUA)
	}
	if hits != 1 {
		t.Fatalf("成功路径应恰好一次请求，实际 %d", hits)
	}

	// 禁 cookie jar：上游发来的 JSESSIONID 不得在后续请求里回送。
	if _, err := q.Day(t.Context(), time.Date(2026, 9, 15, 0, 0, 0, 0, time.Local)); err != nil {
		t.Fatalf("第二次 Day 失败: %v", err)
	}
	if gotCookie != "" {
		t.Fatalf("第二次请求带回了 cookie: %q", gotCookie)
	}
}

func TestQueryDayPageKinds(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		status  int
		want    ecjtu.Kind
	}{
		{"未绑定页是凭证故障不是空课表（缺陷 1）", "notbound.html", 200, ecjtu.KindNotBound},
		{"500 上游错误", "tomcat500.html", 500, ecjtu.KindUpstream},
		{"无法归类页面按改版处理", "unknown.html", 200, ecjtu.KindParse},
		{"课表域出现空态文案按改版处理", "course-unexpected-empty.html", 200, ecjtu.KindParse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				w.WriteHeader(tt.status)
				_, _ = w.Write(testsupport.Fixture(t, tt.fixture))
			}))
			defer srv.Close()

			q := course.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID))
			got, err := q.Day(t.Context(), time.Date(2026, 9, 14, 0, 0, 0, 0, time.Local))
			if err == nil {
				t.Fatalf("应报错，实际返回 %v", got)
			}
			if got.Date != "" || got.Courses != nil {
				t.Fatalf("报错时不得返回半满结果: %+v", got)
			}
			kind, ok := ecjtu.KindOf(err)
			if !ok || kind != tt.want {
				t.Fatalf("分类 = %v (ok=%v), 期望 %v", kind, ok, tt.want)
			}
			msg := err.Error()
			if strings.Contains(msg, testOpenID) || strings.Contains(msg, "<") {
				t.Fatalf("错误串泄露凭证或页面原文: %q", msg)
			}
			if tt.want == ecjtu.KindUpstream && !strings.Contains(msg, "500") {
				t.Fatalf("上游错误应保留状态码: %q", msg)
			}
			if tt.want == ecjtu.KindUpstream && hits != 1 {
				t.Fatalf("拿到响应不得重试，实际请求 %d 次", hits)
			}
		})
	}
}

func TestQueryDayContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("取消的 ctx 不应到达处理器")
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	q := course.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID))
	_, err := q.Day(ctx, time.Date(2026, 9, 14, 0, 0, 0, 0, time.Local))
	if err == nil {
		t.Fatal("期望 KindTransport 错误")
	}
	kind, ok := ecjtu.KindOf(err)
	if !ok || kind != ecjtu.KindTransport {
		t.Fatalf("分类 = %v, 期望 transport", kind)
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("错误应可追溯到 ctx 取消: %v", err)
	}
	if strings.Contains(err.Error(), testOpenID) {
		t.Fatalf("错误串泄露凭证: %q", err.Error())
	}
}

func TestQueryDayRejectsZeroDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("零值日期不应发起请求")
	}))
	defer srv.Close()

	q := course.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID))
	_, err := q.Day(t.Context(), time.Time{})
	if err == nil {
		t.Fatal("零值日期应报错")
	}
	if kind, ok := ecjtu.KindOf(err); !ok || kind != ecjtu.KindInvalidInput {
		t.Fatalf("分类 = %v (ok=%v)，期望 invalidinput", kind, ok)
	}
}
