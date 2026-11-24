package ecjtu

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// PageKind 是在解析页面内容之前对页面状态的判定。解析器只允许在 PageOK 的页面上运行。
type PageKind int

const (
	// PageOK 是正常业务页面。课表「无课日」也是 PageOK，0 节课是正常结果。
	PageOK PageKind = iota
	// PageNotBound 是「教务处微信平台绑定」表单页（HTTP 200）：凭证故障，不是空数据。
	PageNotBound
	// PageEmpty 是带「暂无…！」文案的空态页，实测仅成绩/考试域出现，属正常空结果。
	PageEmpty
	// PageUpstreamError 是非 2xx 响应或 Tomcat 异常页。
	PageUpstreamError
	// PageUnknown 是无法归类的页面，按上游改版处理。
	PageUnknown
)

// String 返回页面判定的名称。
func (p PageKind) String() string {
	switch p {
	case PageOK:
		return "ok"
	case PageNotBound:
		return "notbound"
	case PageEmpty:
		return "empty"
	case PageUpstreamError:
		return "upstreamerror"
	case PageUnknown:
		return "unknown"
	default:
		return fmt.Sprintf("PageKind(%d)", int(p))
	}
}

// 各域页面的 <title>，由各域 fetch 作为 wantTitle 传入。
const (
	TitleCourse          = "我的日历"
	TitleScore           = "我的成绩"
	TitleExam            = "考试安排"
	TitleClassroomOption = "空闲教室查询"
	TitleClassroomResult = "空闲教室使用情况"
)

const (
	titleNotBound   = "教务处微信平台绑定"
	titleErrPrefix  = "HTTP状态"
	emptyTextMarker = "暂无"
)

// Classify 判定页面身份：状态码只是第一道，内容标记才是主判据。
func Classify(statusCode int, body []byte, wantTitle string) PageKind {
	if statusCode < 200 || statusCode >= 300 {
		return PageUpstreamError
	}
	title, hasTitle := extractTitle(body)
	switch {
	case hasTitle && strings.HasPrefix(title, titleErrPrefix):
		// 上游异常页可能返回 HTTP 200。
		return PageUpstreamError
	case hasTitle && title == titleNotBound:
		return PageNotBound
	case hasTitle && title == wantTitle:
		// 标题与查询域匹配后才能识别空态。
		if bytes.Contains(body, []byte(emptyTextMarker)) {
			return PageEmpty
		}
		return PageOK
	default:
		return PageUnknown
	}
}

// CheckPage 判定响应并把非正常状态转成带分类的 error。PageOK 与 PageEmpty 都返回
// nil error，调用方据 kind 生成 empty:true 的正常结果。
func CheckPage(op string, r *Response, wantTitle string) (PageKind, error) {
	switch kind := Classify(r.Status, r.Body, wantTitle); kind {
	case PageOK, PageEmpty:
		return kind, nil
	case PageNotBound:
		return kind, &Error{Kind: KindNotBound, Op: op,
			Cause: errors.New("openid not bound to the academic affairs platform")}
	case PageUpstreamError:
		// Tomcat 正文一律丢弃，只保留状态码。
		return kind, &Error{Kind: KindUpstream, Op: op, Status: r.Status}
	default:
		return kind, &Error{Kind: KindParse, Op: op,
			Cause: errors.New("unclassifiable page, possible upstream redesign")}
	}
}

func extractTitle(body []byte) (string, bool) {
	z := html.NewTokenizer(bytes.NewReader(body))
	for z.Next() != html.ErrorToken {
		tok := z.Token()
		if tok.Type != html.StartTagToken || tok.Data != "title" {
			continue
		}
		var b strings.Builder
		for z.Next() != html.ErrorToken {
			inner := z.Token()
			if inner.Type == html.EndTagToken {
				return strings.TrimSpace(b.String()), true
			}
			if inner.Type == html.TextToken {
				b.WriteString(inner.String())
			}
		}
		return strings.TrimSpace(b.String()), true
	}
	return "", false
}

// 页头标签，实测为半角冒号（课表页用全角冒号）。
const (
	labelStudent = "姓名:"
	labelTerm    = "当前学期:"
)

var digitsRe = regexp.MustCompile(`[0-9]+`)

// HeadInfo 是成绩与考试页共用的页头信息。指针字段表达「页面没写」：
// 空态页没有 div.right 与学期下拉。
type HeadInfo struct {
	StudentName    *string
	Term           *string
	AvailableTerms []string
}

// HeadInfoFrom 从已解析的页面提取页头信息。取不到不算错误（空态页没有页头），
// 「正常页面必须有页头」由各域解析器把关。
func HeadInfoFrom(doc *goquery.Document) HeadInfo {
	var (
		h       HeadInfo
		current string
	)
	doc.Find("div.right").First().Contents().Each(func(_ int, s *goquery.Selection) {
		n := s.Get(0)
		switch {
		case n.Type == html.TextNode:
			if text := Collapse(n.Data); text != "" {
				current = text
			}
		case n.Type == html.ElementNode && n.Data == "span":
			value := Collapse(s.Text())
			if value == "" {
				return
			}
			switch {
			case strings.HasPrefix(current, labelStudent):
				h.StudentName = &value
			case strings.HasPrefix(current, labelTerm):
				h.Term = &value
			}
		}
	})
	doc.Find("ul.dropdown-menu li a").Each(func(_ int, a *goquery.Selection) {
		if term := Collapse(a.Text()); term != "" {
			h.AvailableTerms = append(h.AvailableTerms, term) // 保持页面顺序
		}
	})
	return h
}

// DeclaredCount 取 div.words 自带的条数（strong 或 mark 内的数字），
// 只用于 --verbose 的一致性警告，不进 JSON。
func DeclaredCount(doc *goquery.Document) (int, bool) {
	var (
		found bool
		n     int
	)
	doc.Find("div.words").First().Find("strong, mark").Each(func(_ int, s *goquery.Selection) {
		if found {
			return
		}
		digits := digitsRe.FindString(Collapse(s.Text()))
		if digits == "" {
			return
		}
		v, err := strconv.Atoi(digits)
		if err != nil {
			return
		}
		n, found = v, true
	})
	return n, found
}

var spaceRe = regexp.MustCompile(`\s+`)

// Collapse 把连续空白折成单个空格并去掉首尾空白。
func Collapse(s string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}
