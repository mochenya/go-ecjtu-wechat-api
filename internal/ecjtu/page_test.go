package ecjtu_test

import (
	"errors"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		status    int
		wantTitle string
		want      ecjtu.PageKind
	}{
		{"课表正常页判OK", "calendar_normal.html", 200, ecjtu.TitleCourse, ecjtu.PageOK},
		{"课表无课日不是空态", "calendar_noclass.html", 200, ecjtu.TitleCourse, ecjtu.PageOK},
		{"成绩空态页", "score_empty.html", 200, ecjtu.TitleScore, ecjtu.PageEmpty},
		{"考试空态页", "exam_empty.html", 200, ecjtu.TitleExam, ecjtu.PageEmpty},
		{"未绑定页优先于域期望", "notbound.html", 200, ecjtu.TitleCourse, ecjtu.PageNotBound},
		{"500状态码判上游错误", "tomcat500.html", 500, ecjtu.TitleCourse, ecjtu.PageUpstreamError},
		{"200但title是HTTP状态仍判上游错误", "tomcat500.html", 200, ecjtu.TitleCourse, ecjtu.PageUpstreamError},
		{"无法归类的标题判未知", "unknown.html", 200, ecjtu.TitleCourse, ecjtu.PageUnknown},
		{"域不匹配判未知", "calendar_normal.html", 200, ecjtu.TitleScore, ecjtu.PageUnknown},
		{"空态文案不跨域生效", "score_empty.html", 200, ecjtu.TitleExam, ecjtu.PageUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ecjtu.Classify(tt.status, testsupport.Fixture(t, tt.file), tt.wantTitle)
			if got != tt.want {
				t.Fatalf("Classify(%s, %d, %q) = %v, 期望 %v", tt.file, tt.status, tt.wantTitle, got, tt.want)
			}
		})
	}
}

func TestClassifyMissingTitleIsUnknown(t *testing.T) {
	if got := ecjtu.Classify(200, []byte("<html><body>no title</body></html>"), ecjtu.TitleCourse); got != ecjtu.PageUnknown {
		t.Fatalf("无 title 页面应判未知，实际 %v", got)
	}
}

func TestCheckPage(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		status    int
		wantTitle string
		wantKind  ecjtu.PageKind
		wantErr   error // nil 表示不应产生 error
	}{
		{"正常页无错误", "calendar_normal.html", 200, ecjtu.TitleCourse, ecjtu.PageOK, nil},
		{"空态页无错误", "score_empty.html", 200, ecjtu.TitleScore, ecjtu.PageEmpty, nil},
		{"未绑定映射KindNotBound", "notbound.html", 200, ecjtu.TitleCourse, ecjtu.PageNotBound, &ecjtu.Error{Kind: ecjtu.KindNotBound}},
		{"上游错误映射KindUpstream", "tomcat500.html", 500, ecjtu.TitleCourse, ecjtu.PageUpstreamError, &ecjtu.Error{Kind: ecjtu.KindUpstream}},
		{"未知页映射KindParse", "unknown.html", 200, ecjtu.TitleCourse, ecjtu.PageUnknown, &ecjtu.Error{Kind: ecjtu.KindParse}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ecjtu.Response{Status: tt.status, Body: testsupport.Fixture(t, tt.file), Attempts: 1}
			kind, err := ecjtu.CheckPage("course.Day", r, tt.wantTitle)
			if kind != tt.wantKind {
				t.Fatalf("kind = %v, 期望 %v", kind, tt.wantKind)
			}
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("不应产生 error: %v", err)
				}
				return
			}
			var e *ecjtu.Error
			if !errors.As(err, &e) {
				t.Fatalf("应返回 *ecjtu.Error，实际 %T: %v", err, err)
			}
			want := tt.wantErr.(*ecjtu.Error)
			if e.Kind != want.Kind {
				t.Fatalf("Kind = %v, 期望 %v", e.Kind, want.Kind)
			}
			if e.Op != "course.Day" {
				t.Fatalf("Op = %q", e.Op)
			}
			if want.Kind == ecjtu.KindUpstream && e.Status != 500 {
				t.Fatalf("Status = %d, 期望 500", e.Status)
			}
		})
	}
}

func TestPageKindString(t *testing.T) {
	tests := []struct {
		in   ecjtu.PageKind
		want string
	}{
		{ecjtu.PageOK, "ok"},
		{ecjtu.PageNotBound, "notbound"},
		{ecjtu.PageEmpty, "empty"},
		{ecjtu.PageUpstreamError, "upstreamerror"},
		{ecjtu.PageUnknown, "unknown"},
		{ecjtu.PageKind(42), "PageKind(42)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("PageKind(%d).String() = %q, 期望 %q", int(tt.in), got, tt.want)
		}
	}
}
