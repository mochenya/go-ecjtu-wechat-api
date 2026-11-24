package score_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/score"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

// testOpenID 是 fixture 与测试共用的假凭证，不是真实值。
const testOpenID = "TESTOPENIDXXXXXXXXXXXXXXXXX"

func TestQueryScoresSendsTerm(t *testing.T) {
	var (
		lastPath  string
		lastQuery url.Values
		hits      int64
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		lastPath, lastQuery = r.URL.Path, r.URL.Query()
		_, _ = w.Write(testsupport.Fixture(t, "score-normal-2025-2.html"))
	}))
	defer srv.Close()

	q := score.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID))
	got, err := q.Scores(t.Context(), "2025.2")
	if err != nil {
		t.Fatalf("Scores 失败: %v", err)
	}
	if lastPath != "/ScoreQuery" {
		t.Fatalf("path = %q", lastPath)
	}
	if lastQuery.Get("term") != "2025.2" {
		t.Fatalf("term 参数 = %q", lastQuery.Get("term"))
	}
	if lastQuery.Get("weiXinID") != testOpenID {
		t.Fatal("weiXinID 参数不匹配")
	}
	if got.Count != 15 {
		t.Fatalf("count = %d, 期望 15", got.Count)
	}
	if hits != 1 {
		t.Fatalf("成功路径应恰好一次请求，实际 %d", hits)
	}
}

// 省略学期时不发本地猜测：请求里不带 term，由上游落到它的默认学期。
func TestQueryScoresOmitsEmptyTerm(t *testing.T) {
	var lastQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.Query()
		_, _ = w.Write(testsupport.Fixture(t, "score-normal-2025-2.html"))
	}))
	defer srv.Close()

	if _, err := score.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID)).Scores(t.Context(), ""); err != nil {
		t.Fatalf("Scores 失败: %v", err)
	}
	if _, ok := lastQuery["term"]; ok {
		t.Fatalf("不应发送 term 参数: %v", lastQuery)
	}
}

func TestQueryScoresEmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(testsupport.Fixture(t, "score-empty-2026-2.html"))
	}))
	defer srv.Close()

	got, err := score.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID)).Scores(t.Context(), "2026.2")
	if err != nil {
		t.Fatalf("空态是正常结果，实际报错: %v", err)
	}
	if !got.Empty || got.Count != 0 {
		t.Fatalf("期望 empty:true count:0，实际 %+v", got)
	}
	if got.Items == nil {
		t.Fatal("Items 必须是非 nil 空切片")
	}
	// 空态页没有 div.right 与学期下拉：三字段为 null。
	if got.Term != nil || got.StudentName != nil || got.AvailableTerms != nil {
		t.Fatalf("空态页页头应为 null，实际 %+v", got)
	}
}

func TestQueryScoresErrorKinds(t *testing.T) {
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
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write(testsupport.Fixture(t, tt.fixture))
			}))
			defer srv.Close()

			got, err := score.NewQuery(ecjtu.NewTransport(srv.URL, testOpenID)).Scores(t.Context(), "2025.2")
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
		})
	}
}
