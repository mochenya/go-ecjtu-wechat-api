package exam

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

// 字段标签，实测为半角冒号。解析按标签锚定、不依赖 u/span 的出现下标，
// 上游增加字段不会让取值整体错位。
const (
	labelWeek       = "考试周次:"
	labelTime       = "考试时间:"
	labelLocation   = "考试地点:"
	labelCourseType = "课程性质:"
	labelClassName  = "班级名称:"
	labelHeadcount  = "考试人数:"
)

// timeRe 匹配实测的固定时间形态 `2026年07月07日(星期二)14:30-16:30`（半角括号）。
var timeRe = regexp.MustCompile(`^(20\d\d)年(\d\d)月(\d\d)日\(星期([一二三四五六日天])\)(\d\d:\d\d)-(\d\d:\d\d)$`)

var weekdayValues = map[rune]int{'一': 1, '二': 2, '三': 3, '四': 4, '五': 5, '六': 6, '日': 7, '天': 7}

// errParse 构造解析类错误：消息只含结构性描述符与下标，绝不内插页面文本
// （错误里不得出现上游 HTML 原文与学生姓名）。
func errParse(format string, a ...any) error {
	return &ecjtu.Error{Kind: ecjtu.KindParse, Op: opExams, Cause: fmt.Errorf(format, a...)}
}

func (q *Query) parseExams(body []byte) (Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Result{}, errParse("html document is malformed")
	}
	head := ecjtu.HeadInfoFrom(doc)
	if head.StudentName == nil || head.Term == nil || len(head.AvailableTerms) == 0 {
		return Result{}, errParse("business page without head info")
	}
	items, err := parseItems(doc)
	if err != nil {
		return Result{}, err
	}
	res := Result{
		Term:           head.Term,
		StudentName:    head.StudentName,
		AvailableTerms: head.AvailableTerms,
		Count:          len(items), // 取解析条数，不用页面自带数字
		Items:          items,
	}
	if n, ok := ecjtu.DeclaredCount(doc); ok {
		res.PageCount = &n
	}
	return res, nil
}

// parseItems 遍历考试条目行，容器判据是 div.row > div.col-xs-12。
func parseItems(doc *goquery.Document) ([]ExamItem, error) {
	var (
		items []ExamItem
		pErr  error
	)
	doc.Find("div.row").Each(func(i int, row *goquery.Selection) {
		if pErr != nil {
			return
		}
		col := row.ChildrenFiltered("div.col-xs-12").First()
		if col.Length() == 0 {
			pErr = errParse("item %d: column container not found", i)
			return
		}
		item, err := parseItem(i, col)
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
		items = []ExamItem{}
	}
	return items, nil
}

func parseItem(i int, col *goquery.Selection) (ExamItem, error) {
	text := col.Find("div.text").First()
	if text.Length() == 0 {
		return ExamItem{}, errParse("item %d: text block not found", i)
	}
	// 课程名在同级的 div.course > mark 里，不在 div.text 内。
	courseMark := col.Find("div.course mark").First()
	if courseMark.Length() == 0 {
		return ExamItem{}, errParse("item %d: course name mark not found", i)
	}
	var item ExamItem
	item.CourseName = ecjtu.Collapse(courseMark.Text())
	if item.CourseName == "" {
		return ExamItem{}, errParse("item %d: course name is empty", i)
	}

	var (
		seen    = map[string]bool{}
		current string
		pErr    error
	)
	text.Contents().Each(func(_ int, s *goquery.Selection) {
		if pErr != nil {
			return
		}
		n := s.Get(0)
		switch {
		case n.Type == html.TextNode:
			if label := lastLabel(ecjtu.Collapse(n.Data)); label != "" {
				current = label
			}
		case n.Type == html.ElementNode && (n.Data == "u" || n.Data == "span"):
			pErr = collect(i, current, seen, s, &item)
		}
	})
	if pErr != nil {
		return ExamItem{}, pErr
	}
	for _, label := range []string{labelWeek, labelTime, labelLocation, labelCourseType, labelClassName, labelHeadcount} {
		if !seen[label] {
			return ExamItem{}, errParse("item %d: missing field %s", i, fieldName(label))
		}
	}
	if err := normalizeTime(i, item.TimeText, &item); err != nil {
		return ExamItem{}, err
	}
	return item, nil
}

// collect 按当前标签收一个值：标签缺失或重复都属于结构改版。
func collect(i int, label string, seen map[string]bool, s *goquery.Selection, item *ExamItem) error {
	if label == "" {
		return errParse("item %d: value without label", i)
	}
	if seen[label] {
		return errParse("item %d: duplicate field %s", i, fieldName(label))
	}
	seen[label] = true
	return assign(i, label, s, item)
}

// assign 把标签对应的值写进条目。时间字段额外拆出红色备注 div。
func assign(i int, label string, s *goquery.Selection, item *ExamItem) error {
	switch label {
	case labelWeek:
		n, err := strconv.Atoi(ecjtu.Collapse(s.Text()))
		if err != nil {
			return errParse("item %d: exam week is not a number", i)
		}
		item.ExamWeek = n
	case labelTime:
		// <u> 里除了时间原文还嵌着红色备注 div：两者分别取，互不污染。
		red := s.Find("div").First()
		if red.Length() > 0 {
			item.Note = ecjtu.Collapse(red.Text())
			red.Remove()
		}
		item.TimeText = ecjtu.Collapse(s.Text())
	case labelLocation:
		item.Location = ecjtu.Collapse(s.Text())
	case labelCourseType:
		item.CourseType = ecjtu.Collapse(s.Text())
	case labelClassName:
		item.ClassName = ecjtu.Collapse(s.Text())
	case labelHeadcount:
		n, err := strconv.Atoi(ecjtu.Collapse(s.Text()))
		if err != nil {
			return errParse("item %d: headcount is not a number", i)
		}
		item.Headcount = n
	default:
		return errParse("item %d: unknown field", i)
	}
	return nil
}

// normalizeTime 从时间原文归一出日期、星期与起止时刻；不匹配即 KindParse。
func normalizeTime(i int, text string, item *ExamItem) error {
	m := timeRe.FindStringSubmatch(text)
	if m == nil {
		return errParse("item %d: exam time text does not match", i)
	}
	runes := []rune(m[4])
	weekday, ok := weekdayValues[runes[0]]
	if !ok || len(runes) != 1 {
		return errParse("item %d: exam weekday is out of range", i)
	}
	date := m[1] + "-" + m[2] + "-" + m[3]
	if _, err := time.Parse(time.RFC3339, date+"T"+m[5]+":00Z"); err != nil {
		return errParse("item %d: exam date or time is not a real instant", i)
	}
	item.ExamDate, item.DayOfWeek, item.StartTime, item.EndTime = date, weekday, m[5], m[6]
	return nil
}

// lastLabel 返回文本里最后出现的字段标签：一个文本节点挤进多个标签时，紧邻其后取值元素的是最后一个。
func lastLabel(text string) string {
	best, at := "", -1
	for _, label := range []string{labelWeek, labelTime, labelLocation, labelCourseType, labelClassName, labelHeadcount} {
		if idx := strings.LastIndex(text, label); idx > at {
			best, at = label, idx
		}
	}
	return best
}

func fieldName(label string) string {
	switch label {
	case labelWeek:
		return "exam-week"
	case labelTime:
		return "exam-time"
	case labelLocation:
		return "location"
	case labelCourseType:
		return "course-type"
	case labelClassName:
		return "class-name"
	case labelHeadcount:
		return "headcount"
	default:
		return "unknown"
	}
}
