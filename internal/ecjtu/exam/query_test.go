package exam_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/exam"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

// testOpenID 是 fixture 与测试共用的假凭证，不是真实值。
const testOpenID = "TESTOPENIDXXXXXXXXXXXXXXXXX"

func TestQueryExamsSendsTerm(t *testing.T) {
	var (
		lastPath  string
		lastQuery url.Values
		hits      int64
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		lastPath, lastQuery = r.URL.Path, r.URL.Query()
		_, _ = w.Write(testsupport.Fixture(t, "exam-normal-2025-1.html"))
	}))
	defer srv.Close()

	got, err := exam.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID)).Exams(t.Context(), "2025.1")
	if err != nil {
		t.Fatalf("Exams 失败: %v", err)
	}
	if lastPath != "/ExamArrangeCl" {
		t.Fatalf("path = %q", lastPath)
	}
	if lastQuery.Get("term") != "2025.1" || lastQuery.Get("weiXinID") != testOpenID {
		t.Fatalf("参数不符: %v", lastQuery)
	}
	if got.Count != 3 || hits != 1 {
		t.Fatalf("count=%d hits=%d，期望 3/1", got.Count, hits)
	}
}

func TestQueryExamsOmitsEmptyTerm(t *testing.T) {
	var lastQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.Query()
		_, _ = w.Write(testsupport.Fixture(t, "exam-normal-2025-1.html"))
	}))
	defer srv.Close()

	if _, err := exam.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID)).Exams(t.Context(), ""); err != nil {
		t.Fatalf("Exams 失败: %v", err)
	}
	if _, ok := lastQuery["term"]; ok {
		t.Fatalf("不应发送 term 参数: %v", lastQuery)
	}
}

func TestQueryExamsEmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(testsupport.Fixture(t, "exam-empty-current.html"))
	}))
	defer srv.Close()

	got, err := exam.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID)).Exams(t.Context(), "2026.1")
	if err != nil {
		t.Fatalf("空态是正常结果，实际报错: %v", err)
	}
	if !got.Empty || got.Count != 0 || got.Items == nil {
		t.Fatalf("期望 empty:true count:0 items:[]，实际 %+v", got)
	}
	if got.Term != nil || got.StudentName != nil || got.AvailableTerms != nil {
		t.Fatalf("空态页页头应为 null，实际 %+v", got)
	}
}

func TestQueryExamsErrorKinds(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		status  int
		want    ecjtu.Kind
	}{
		{"未绑定页", "notbound.html", 200, ecjtu.KindNotBound},
		{"500 上游错误", "tomcat500.html", 500, ecjtu.KindUpstream},
		{"无法归类页面", "unknown.html", 200, ecjtu.KindParse},
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

			got, err := exam.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID)).Exams(t.Context(), "2025.1")
			if err == nil {
				t.Fatalf("应报错，实际 %+v", got)
			}
			if !ecjtu.IsKind(err, tt.want) {
				t.Fatalf("分类应为 %v，实际 %v", tt.want, err)
			}
			if got.Items != nil || got.Term != nil {
				t.Fatalf("报错时不得返回半满结果: %+v", got)
			}
			if strings.Contains(err.Error(), testOpenID) || strings.Contains(err.Error(), "<") {
				t.Fatalf("错误串泄露凭证或页面原文: %q", err.Error())
			}
			if tt.status == 500 && hits != 1 {
				t.Fatalf("拿到响应不得重试，实际 %d 次", hits)
			}
		})
	}
}
