package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	upstream "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	ecjtu "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/client"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

func TestScoreTermsDiscoversCandidatesAndAvailableTerm(t *testing.T) {
	page := testsupport.Fixture(t, "score-normal-2025-2.html")
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/ScoreQuery" || r.URL.Query().Get("term") != "2025.2" {
			t.Error("成绩查询路径或学期不匹配")
		}
		_, _ = w.Write(page)
	}))
	defer server.Close()

	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ScoreTerms, "2025.2")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"2026.2", "2026.1", "2025.2", "2025.1"}; !reflect.DeepEqual(got.Candidates, want) {
		t.Fatalf("候选学期 = %v，期望 %v", got.Candidates, want)
	}
	if want := []string{"2025.2"}; !reflect.DeepEqual(got.Available, want) {
		t.Fatalf("已确认有成绩的学期 = %v，期望 %v", got.Available, want)
	}
	if got.CheckedTerm == nil || *got.CheckedTerm != "2025.2" {
		t.Fatalf("本次查询学期 = %v，期望 2025.2", got.CheckedTerm)
	}
	if hits.Load() != 1 {
		t.Fatalf("成功查询次数 = %d，期望 1", hits.Load())
	}
}

func TestExamTermsUsesExamPage(t *testing.T) {
	page := testsupport.Fixture(t, "exam-normal-2025-1.html")
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/ExamArrangeCl" || r.URL.Query().Get("term") != "2025.1" {
			t.Error("考试查询路径或学期不匹配")
		}
		_, _ = w.Write(page)
	}))
	defer server.Close()

	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ExamTerms, "2025.1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"2026.2", "2026.1", "2025.2", "2025.1"}; !reflect.DeepEqual(got.Candidates, want) {
		t.Fatalf("候选学期 = %v，期望 %v", got.Candidates, want)
	}
	if want := []string{"2025.1"}; !reflect.DeepEqual(got.Available, want) {
		t.Fatalf("已确认有考试的学期 = %v，期望 %v", got.Available, want)
	}
	if got.CheckedTerm == nil || *got.CheckedTerm != "2025.1" {
		t.Fatalf("本次查询学期 = %v，期望 2025.1", got.CheckedTerm)
	}
	if hits.Load() != 1 {
		t.Fatalf("成功查询次数 = %d，期望 1", hits.Load())
	}
}

func TestTermsSortsYearAndPeriodNumerically(t *testing.T) {
	page := string(testsupport.Fixture(t, "exam-normal-2025-1.html"))
	page = strings.NewReplacer(
		">2026.2</a>", ">2025.1</a>",
		">2026.1</a>", ">2025.10</a>",
		">2025.2</a>", ">2026.2</a>",
		">2025.1</a>", ">2025.2</a>",
	).Replace(page)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(page))
	}))
	defer server.Close()

	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ExamTerms, "2025.1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"2026.2", "2025.10", "2025.2", "2025.1"}; !reflect.DeepEqual(got.Candidates, want) {
		t.Fatalf("候选学期 = %v，期望 %v", got.Candidates, want)
	}
}

func TestInvalidTermIsRejectedBeforeRequest(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		call func() error
	}{
		{"成绩学期发现", func() error { _, err := school.Terms(t.Context(), ecjtu.ScoreTerms, "bogus"); return err }},
		{"考试学期发现", func() error { _, err := school.Terms(t.Context(), ecjtu.ExamTerms, "bogus"); return err }},
		{"成绩查询", func() error { _, err := school.Scores(t.Context(), "bogus"); return err }},
		{"考试查询", func() error { _, err := school.Exams(t.Context(), "bogus"); return err }},
		{"未知来源", func() error { _, err := school.Terms(t.Context(), ecjtu.TermSource(99), "2025.2"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !upstream.IsKind(err, upstream.KindInvalidInput) {
				t.Fatalf("应返回输入错误，实际 %v", err)
			}
		})
	}
	if hits.Load() != 0 {
		t.Fatalf("输入错误不应发送请求，实际 %d 次", hits.Load())
	}
}

func TestMalformedCandidateReturnsParseError(t *testing.T) {
	page := strings.Replace(string(testsupport.Fixture(t, "exam-normal-2025-1.html")), ">2026.2</a>", ">invalid</a>", 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(page))
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ExamTerms, "2025.1")
	if !upstream.IsKind(err, upstream.KindParse) {
		t.Fatalf("学期候选异常应返回解析错误，实际 %v", err)
	}
	if got.Candidates != nil || got.Available != nil || got.CheckedTerm != nil {
		t.Fatalf("失败时不应返回半满结果: %+v", got)
	}
}

func TestBlankCandidateIsExcluded(t *testing.T) {
	page := strings.Replace(string(testsupport.Fixture(t, "exam-normal-2025-1.html")), ">2026.2</a>", ">  </a>", 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(page))
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ExamTerms, "2025.1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"2026.1", "2025.2", "2025.1"}; !reflect.DeepEqual(got.Candidates, want) {
		t.Fatalf("候选学期 = %v，期望 %v", got.Candidates, want)
	}
}

func TestEmptyPagesDoNotInventCandidatesOrDefaultTerm(t *testing.T) {
	tests := []struct {
		name    string
		source  ecjtu.TermSource
		fixture string
		term    string
	}{
		{"成绩默认学期", ecjtu.ScoreTerms, "score-empty-2026-2.html", ""},
		{"成绩指定学期", ecjtu.ScoreTerms, "score-empty-2026-2.html", "2026.2"},
		{"考试默认学期", ecjtu.ExamTerms, "exam-empty-current.html", ""},
		{"考试指定学期", ecjtu.ExamTerms, "exam-empty-current.html", "2026.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := testsupport.Fixture(t, tt.fixture)
			var hits atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if got := r.URL.Query().Get("term"); got != tt.term || (tt.term == "" && r.URL.Query().Has("term")) {
					t.Errorf("请求学期 = %q，期望 %q", got, tt.term)
				}
				_, _ = w.Write(page)
			}))
			defer server.Close()
			school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			got, err := school.Terms(t.Context(), tt.source, tt.term)
			if err != nil {
				t.Fatal(err)
			}
			if got.Candidates != nil || got.Available == nil || len(got.Available) != 0 {
				t.Fatalf("空态不应编造候选或有数据学期: %+v", got)
			}
			if tt.term == "" && got.CheckedTerm != nil {
				t.Fatalf("省略学期的空态不能推断查询学期: %q", *got.CheckedTerm)
			}
			if tt.term != "" && (got.CheckedTerm == nil || *got.CheckedTerm != tt.term) {
				t.Fatalf("指定查询学期应记录参数 %q，实际 %v", tt.term, got.CheckedTerm)
			}
			if hits.Load() != 1 {
				t.Fatalf("成功查询次数 = %d，期望 1", hits.Load())
			}
		})
	}
}

func TestNormalPageWithoutItemsHasNoAvailableTerm(t *testing.T) {
	page := testsupport.Fixture(t, "score-no-rows.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(page)
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ScoreTerms, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Candidates) == 0 || len(got.Available) != 0 || got.CheckedTerm == nil || *got.CheckedTerm != "2025.2" {
		t.Fatalf("正常页零条不应视为有成绩: %+v", got)
	}
}

func TestTermDiscoveryReturnsUpstreamFailure(t *testing.T) {
	page := testsupport.Fixture(t, "notbound.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(page)
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ScoreTerms, "2025.2")
	if !upstream.IsKind(err, upstream.KindNotBound) {
		t.Fatalf("应返回未绑定错误，实际 %v", err)
	}
	if got.Candidates != nil || got.Available != nil || got.CheckedTerm != nil {
		t.Fatalf("失败时不应返回半满结果: %+v", got)
	}
}

func TestTermDiscoveryJSONUsesProjectFieldNames(t *testing.T) {
	page := testsupport.Fixture(t, "score-empty-2026-2.html")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(page)
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := school.Terms(t.Context(), ecjtu.ScoreTerms, "2026.2")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"candidates":null,"available":[],"checkedTerm":"2026.2"}`
	if string(data) != want {
		t.Fatalf("学期发现 JSON = %s，期望 %s", data, want)
	}
}

func TestTermDiscoveryDoesNotCacheResults(t *testing.T) {
	populated := testsupport.Fixture(t, "score-normal-2025-2.html")
	empty := testsupport.Fixture(t, "score-empty-2026-2.html")
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			_, _ = w.Write(populated)
			return
		}
		_, _ = w.Write(empty)
	}))
	defer server.Close()
	school, err := ecjtu.New(ecjtu.Config{OpenID: "TESTOPENIDXXXXXXXXXXXXXXXXX", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	first, err := school.Terms(t.Context(), ecjtu.ScoreTerms, "2025.2")
	if err != nil {
		t.Fatal(err)
	}
	second, err := school.Terms(t.Context(), ecjtu.ScoreTerms, "2026.2")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Available) != 1 || first.Available[0] != "2025.2" || len(second.Available) != 0 || second.Candidates != nil {
		t.Fatalf("各次结果应来自各自页面: first=%+v second=%+v", first, second)
	}
	if hits.Load() != 2 {
		t.Fatalf("两次调用应分别抓取，实际 %d 次", hits.Load())
	}
}
