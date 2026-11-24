package course

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

// 页面文本标记。全角冒号是上游写法，遇到半角即视为改版，不做兼容。
const (
	labelTime     = "时间："
	labelLocation = "地点："
	labelTeacher  = "教师："
	statusOpen    = "("
	statusClose   = ")"
)

// centerRe 匹配 div.center > p 的整行：`2026-09-14 星期一（第3周）`；
// 允许「星期日」与「星期天」两种写法。
var centerRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) 星期([一二三四五六日天])（第(\d+)周）$`)

var weekdayValues = map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '日': 7, '天': 7}

// errParse 构造解析类错误。消息只含结构性事实（标签名与下标），绝不内插页面文本：
// 错误与日志里不得出现上游 HTML 原文。
func errParse(format string, a ...any) error {
	return &ecjtu.Error{Kind: ecjtu.KindParse, Op: opDay, Cause: fmt.Errorf(format, a...)}
}

// parseDay 把课表页面变成结构化数据（纯函数，无 I/O）；
// 结构不符预期一律返回 KindParse，不返回半满结果。
func (q *Query) parseDay(body []byte) (Schedule, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Schedule{}, errParse("html document is malformed")
	}
	head, err := parseHead(doc)
	if err != nil {
		return Schedule{}, err
	}
	courses, err := parseItems(doc)
	if err != nil {
		return Schedule{}, err
	}
	head.Courses = courses // 恒非 nil：无课日输出 []
	return head, nil
}

func parseHead(doc *goquery.Document) (Schedule, error) {
	center := doc.Find("div.center > p").First()
	if center.Length() == 0 {
		return Schedule{}, errParse("div.center date line not found")
	}
	m := centerRe.FindStringSubmatch(ecjtu.Collapse(center.Text()))
	if m == nil {
		return Schedule{}, errParse("div.center date line does not match")
	}
	runes := []rune(m[2])
	weekday, ok := weekdayValues[runes[0]]
	if !ok || len(runes) != 1 {
		return Schedule{}, errParse("div.center weekday is out of range")
	}
	week, err := strconv.Atoi(m[3])
	if err != nil {
		return Schedule{}, errParse("div.center week number is out of range")
	}
	return Schedule{Date: m[1], Weekday: weekday, Week: week}, nil
}

// parseItems 遍历课程行：li 含 span.class_span 才是课程条目，装饰用的
// today_schedule/easy*.png 一律忽略，因此无课日自然得到 0 条。
func parseItems(doc *goquery.Document) ([]Course, error) {
	var (
		items []Course
		pErr  error
	)
	doc.Find("ul.rl_info li").Each(func(i int, li *goquery.Selection) {
		if pErr != nil {
			return
		}
		span := li.Find("span.class_span")
		if span.Length() == 0 {
			return // 非课程行
		}
		item, err := parseItem(i, li, span)
		if err != nil {
			pErr = err
			return
		}
		items = append(items, item)
	})
	if pErr != nil {
		return nil, pErr
	}
	if items == nil {
		items = []Course{}
	}
	return items, nil
}

// parseItem 解析一条课程：先按 <br> 切成行，再按标签锚定取值。
// 不用位置下标，上游插入一行不会影响其余字段。
func parseItem(i int, li, span *goquery.Selection) (Course, error) {
	p := li.Find("p").First()
	if p.Length() == 0 {
		return Course{}, errParse("item %d: paragraph not found", i)
	}
	lines := itemLines(p, span)
	if len(lines) == 0 {
		return Course{}, errParse("item %d: paragraph is empty", i)
	}
	if isLabelled(lines[0]) {
		return Course{}, errParse("item %d: course name line not found", i)
	}

	var (
		c      Course
		seen   = map[string]bool{}
		labels = []struct {
			prefix string
			dst    *string
		}{{labelTime, &c.TimeText}, {labelLocation, &c.Location}, {labelTeacher, &c.Teacher}}
	)
	c.Name, c.Status = splitStatus(lines[0])
	for _, line := range lines[1:] {
		matched := false
		for _, l := range labels {
			if !strings.HasPrefix(line, l.prefix) {
				continue
			}
			if seen[l.prefix] {
				return Course{}, errParse("item %d: duplicate label %s", i, labelName(l.prefix))
			}
			seen[l.prefix], matched = true, true
			*l.dst = ecjtu.Collapse(strings.TrimPrefix(line, l.prefix))
			break
		}
		if !matched {
			return Course{}, errParse("item %d: unexpected line after the name", i)
		}
	}
	for _, l := range labels {
		if !seen[l.prefix] {
			return Course{}, errParse("item %d: missing label %s", i, labelName(l.prefix))
		}
	}
	weeks, periods, err := parseTimeText(i, c.TimeText)
	if err != nil {
		return Course{}, err
	}
	c.Weeks, c.Periods = weeks, periods
	return c, nil
}

func labelName(prefix string) string {
	switch prefix {
	case labelTime:
		return "time"
	case labelLocation:
		return "location"
	case labelTeacher:
		return "teacher"
	default:
		return "unknown"
	}
}

// isLabelled 报告该行是否以某个字段标签开头（用于识别课名行缺失）。
func isLabelled(line string) bool {
	return strings.HasPrefix(line, labelTime) ||
		strings.HasPrefix(line, labelLocation) ||
		strings.HasPrefix(line, labelTeacher)
}

// itemLines 把 p 的内容按 <br> 切成行，跳过 class_span（其「1-2节」与 periods 重复）。
func itemLines(p, span *goquery.Selection) []string {
	var (
		lines   []string
		current strings.Builder
		skip    *html.Node
	)
	if span.Length() > 0 {
		skip = span.Get(0)
	}
	if p.Length() > 0 {
		flatten(&lines, &current, p.Get(0), skip)
	}
	lines = append(lines, current.String())

	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if l = ecjtu.Collapse(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// flatten 以文档序收集文本节点，遇到 <br> 换行；skip 子树整体忽略。
func flatten(lines *[]string, current *strings.Builder, n, skip *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c == skip:
			continue
		case c.Type == html.ElementNode && c.Data == "br":
			*lines = append(*lines, current.String())
			current.Reset()
		case c.Type == html.TextNode:
			current.WriteString(c.Data)
		default:
			flatten(lines, current, c, skip)
		}
	}
}

// splitStatus 剥离课名词尾的最后一个括号对作为课程状态：`高等数学(A)Ⅰ(考试)` →
// 名 `高等数学(A)Ⅰ`、状态 `考试`。括号必须贴着词尾且非空，否则整行都是课名。
func splitStatus(line string) (name, status string) {
	if !strings.HasSuffix(line, statusClose) {
		return line, ""
	}
	open := strings.LastIndex(line, statusOpen)
	if open <= 0 || open == len(line)-len(statusClose)-1 { // 无左括号，或括号内容为空
		return line, ""
	}
	return ecjtu.Collapse(line[:open]), ecjtu.Collapse(line[open+len(statusOpen) : len(line)-len(statusClose)])
}

// parseTimeText 拆 `1-8,10-19 3,4`：空白分两段，周次段按逗号分段、段内按连号分区间。
func parseTimeText(i int, text string) ([]WeekSpan, []int, error) {
	fields := strings.Fields(text)
	if len(fields) != 2 {
		return nil, nil, errParse("item %d: time field has %d segments, want 2", i, len(fields))
	}
	weeks, err := parseWeeks(i, fields[0])
	if err != nil {
		return nil, nil, err
	}
	periods, err := parseIntList(i, fields[1])
	if err != nil {
		return nil, nil, err
	}
	return weeks, periods, nil
}

func parseWeeks(i int, field string) ([]WeekSpan, error) {
	parts := strings.Split(field, ",")
	spans := make([]WeekSpan, 0, len(parts))
	for _, part := range parts {
		bounds := strings.Split(part, "-")
		if len(bounds) > 2 {
			return nil, errParse("item %d: week range has too many bounds", i)
		}
		start, err := strconv.Atoi(bounds[0])
		if err != nil {
			return nil, errParse("item %d: week range start is not a number", i)
		}
		end := start
		if len(bounds) == 2 {
			end, err = strconv.Atoi(bounds[1])
			if err != nil {
				return nil, errParse("item %d: week range end is not a number", i)
			}
		}
		if start > end {
			return nil, errParse("item %d: week range is reversed", i)
		}
		spans = append(spans, WeekSpan{Start: start, End: end})
	}
	return spans, nil
}

func parseIntList(i int, field string) ([]int, error) {
	parts := strings.Split(field, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, errParse("item %d: period is not a number", i)
		}
		out = append(out, n)
	}
	return out, nil
}
