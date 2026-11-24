package ecjtu

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestKindString(t *testing.T) {
	tests := []struct {
		in   Kind
		want string
	}{
		{KindTransport, "transport"},
		{KindUpstream, "upstream"},
		{KindNotBound, "notbound"},
		{KindParse, "parse"},
		{KindInvalidInput, "invalidinput"},
		{Kind(42), "Kind(42)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, 期望 %q", int(tt.in), got, tt.want)
		}
	}
}

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{"上游错误带状态码", &Error{Kind: KindUpstream, Op: "course.Day", Status: 500},
			"course.Day: upstream returned 500"},
		{"有原因时拼接原因", &Error{Kind: KindTransport, Op: "score.Query",
			Cause: errors.New("dial tcp: lookup failed")}, "score.Query: dial tcp: lookup failed"},
		{"无原因时回落到分类", &Error{Kind: KindParse, Op: "exam.Query"}, "exam.Query: parse"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, 期望 %q", got, tt.want)
			}
		})
	}
}

func TestKindOf(t *testing.T) {
	inner := &Error{Kind: KindNotBound, Op: "course.Day"}
	wrapped := fmt.Errorf("cli: %w", inner)

	if k, ok := KindOf(wrapped); !ok || k != KindNotBound {
		t.Fatalf("KindOf(wrapped) = %v,%v，期望 notbound,true", k, ok)
	}
	if _, ok := KindOf(errors.New("plain")); ok {
		t.Fatal("普通 error 不应命中分类")
	}
	if _, ok := KindOf(nil); ok {
		t.Fatal("nil 不应命中分类")
	}
}

func TestMaskOpenID(t *testing.T) {
	openid := "oFAKE01234567890123456789012"
	got := MaskOpenID("request failed for openid="+openid, openid)
	if strings.Contains(got, openid) {
		t.Fatalf("openid 未被脱敏: %s", got)
	}
	if !strings.Contains(got, "oFAK***") {
		t.Fatalf("期望前 4 位加 ***，实际: %s", got)
	}

	if got := MaskOpenID("id=ab1", "ab1"); got != "id=***" {
		t.Fatalf("过短 openid 应整体替换为 ***，实际: %s", got)
	}
	if got := MaskOpenID("untouched", ""); got != "untouched" {
		t.Fatalf("空 openid 不应改动文本: %s", got)
	}
}

func TestDropSessionIDs(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"URL 内嵌形态", "/weixin/CalendarServlet;jsessionid=8925010F05624E12E314227A69DA0EC3?weiXinID=abc"},
		{"响应头形态", "Set-Cookie: JSESSIONID=8925010F05624E12E314227A69DA0EC3; Path=/weixin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DropSessionIDs(tt.in)
			if strings.Contains(got, "8925010F05624E12E314227A69DA0EC3") {
				t.Fatalf("jsessionid 未被清除: %s", got)
			}
			if !strings.Contains(got, "jsessionid=***") {
				t.Fatalf("期望替换为占位符，实际: %s", got)
			}
		})
	}
}

func TestSanitizeComposition(t *testing.T) {
	openid := "oFAKE01234567890123456789012"
	in := fmt.Sprintf("GET /weixin/ScoreQuery;jsessionid=ABCDEF0123456789ABCDEF0123456789?weiXinID=%s failed", openid)
	got := Sanitize(in, openid)
	if strings.Contains(got, openid) || strings.Contains(got, "ABCDEF0123456789") {
		t.Fatalf("Sanitize 后仍含敏感值: %s", got)
	}
}
