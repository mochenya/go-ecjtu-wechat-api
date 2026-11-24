package score

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

// 页面文本标签，实测均为半角冒号。
const (
	labelFinal  = "期末成绩:"
	labelRetake = "重考成绩:"
	labelRepeat = "重修成绩:"

	bracketOpen  = "【"
	bracketClose = "】"
	creditOpen   = "(学分:"
)

// creditRe 锚定行尾的 `(学分:X)`：实测样本零反例，只认这一形态，不匹配即 KindParse。
var creditRe = regexp.MustCompile(`^(.*)\(学分:([0-9]+(?:\.[0-9]+)?)\)$`)

// errParse 构造解析类错误：消息只含结构性描述符与下标，绝不内插页面文本
// （错误里不得出现上游 HTML 原文与学生姓名）。
func errParse(format string, a ...any) error {
	return &ecjtu.Error{Kind: ecjtu.KindParse, Op: opScores, Cause: fmt.Errorf(format, a...)}
}

func (q *Query) parseScores(body []byte) (Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Result{}, errParse("html document is malformed")
	}
	head := ecjtu.HeadInfoFrom(doc)
	if head.StudentName == nil || head.Term == nil || len(head.AvailableTerms) == 0 {
		// 能归类为正常业务页却没有页头，说明页面结构与实测骨架已不一致。
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
		Count:          len(items), // count 取解析条数，不用页面自带数字
		Items:          items,
	}
	if n, ok := ecjtu.DeclaredCount(doc); ok {
		res.PageCount = &n
	}
	return res, nil
}

// parseItems 遍历成绩条目行，容器判据是 div.row > div.col-xs-12。
func parseItems(doc *goquery.Document) ([]ScoreItem, error) {
	var (
		items []ScoreItem
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
		items = []ScoreItem{}
	}
	return items, nil
}

func parseItem(i int, col *goquery.Selection) (ScoreItem, error) {
	text := col.Find("div.text").First()
	if text.Length() == 0 {
		return ScoreItem{}, errParse("item %d: text block not found", i)
	}
	courseSel := text.Find("span.course").First()
	if courseSel.Length() == 0 {
		return ScoreItem{}, errParse("item %d: course span not found", i)
	}
	var item ScoreItem
	name, code, credit, err := parseCourseText(i, ecjtu.Collapse(courseSel.Text()))
	if err != nil {
		return ScoreItem{}, err
	}
	item.CourseName, item.CourseCode, item.Credit = name, code, credit

	grade := text.Find("div.grade").First()
	if grade.Length() == 0 {
		return ScoreItem{}, errParse("item %d: grade block not found", i)
	}
	if err := parseGrades(i, grade, &item); err != nil {
		return ScoreItem{}, err
	}
	flag := text.Find("span.flag").First()
	if flag.Length() == 0 {
		return ScoreItem{}, errParse("item %d: flag span not found", i)
	}
	item.Flag = ecjtu.Collapse(flag.Text())
	courseType := col.Find("div.type mark").First()
	if courseType.Length() == 0 {
		return ScoreItem{}, errParse("item %d: course type mark not found", i)
	}
	item.CourseType = ecjtu.Collapse(courseType.Text())
	return item, nil
}

// parseCourseText 从 `【主修】【1508100012】高等数学(A)Ⅱ(学分:4.0)` 取课名、代码与学分：
// 先剥离学分后缀，再取最后一个 `】` 之后的全部剩余作课名，因此保留 `(A)Ⅱ`。
func parseCourseText(i int, text string) (name, code string, credit float64, err error) {
	m := creditRe.FindStringSubmatch(text)
	if m == nil {
		return "", "", 0, errParse("item %d: credit suffix not found", i)
	}
	credit, err = strconv.ParseFloat(m[2], 64)
	if err != nil {
		return "", "", 0, errParse("item %d: credit is not a number", i)
	}
	body := m[1]
	cut := strings.LastIndex(body, bracketClose)
	if cut < 0 || !strings.HasPrefix(body, bracketOpen) {
		return "", "", 0, errParse("item %d: course code brackets not found", i)
	}
	code = body[strings.LastIndex(body[:cut], bracketOpen)+len(bracketOpen) : cut]
	if !isAllDigits(code) {
		// 实测代码是纯数字；不是数字说明最后一个括号组不是代码（例如只有一组标记）。
		return "", "", 0, errParse("item %d: course code is not numeric", i)
	}
	name = ecjtu.Collapse(body[cut+len(bracketClose):])
	if name == "" {
		return "", "", 0, errParse("item %d: course name is empty", i)
	}
	return name, code, credit, nil
}

// parseGrades 按标签锚定取三列成绩：标签文本与其后的 span.score 配对、不依赖下标，
// 所以上游调换列顺序或插入新列都不会取错。
func parseGrades(i int, grade *goquery.Selection, item *ScoreItem) error {
	var (
		current   string
		seen      = map[string]bool{}
		dup       string
		unlabeled string
	)
	grade.Contents().Each(func(_ int, s *goquery.Selection) {
		n := s.Get(0)
		switch {
		case n.Type == html.TextNode:
			if label := lastLabel(ecjtu.Collapse(n.Data)); label != "" {
				current = label
			}
		case s.Is("span.score"):
			if current == "" && unlabeled == "" {
				unlabeled = "unlabeled"
				return
			}
			if seen[current] && dup == "" {
				dup = current
				return
			}
			if seen[current] {
				return
			}
			seen[current] = true
			value := ecjtu.Collapse(s.Text())
			switch current {
			case labelFinal:
				item.FinalScore = value
			case labelRetake:
				item.RetakeScore = nilIfEmpty(value)
			case labelRepeat:
				item.RepeatScore = nilIfEmpty(value)
			}
		}
	})
	if unlabeled != "" {
		return errParse("item %d: score column without label", i)
	}
	if dup != "" {
		return errParse("item %d: duplicate grade column %s", i, columnName(dup))
	}
	for _, label := range []string{labelFinal, labelRetake, labelRepeat} {
		if !seen[label] {
			return errParse("item %d: missing grade column %s", i, columnName(label))
		}
	}
	return nil
}

// lastLabel 返回文本里最后出现的列标签：一个文本节点挤进多个标签时，紧邻其后 span 的是最后一个。
func lastLabel(text string) string {
	best, at := "", -1
	for _, label := range []string{labelFinal, labelRetake, labelRepeat} {
		if i := strings.LastIndex(text, label); i > at {
			best, at = label, i
		}
	}
	return best
}

func columnName(label string) string {
	switch label {
	case labelFinal:
		return "final"
	case labelRetake:
		return "retake"
	case labelRepeat:
		return "repeat"
	default:
		return "unknown"
	}
}

func isAllDigits(s string) bool {
	return s != "" && !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

// nilIfEmpty 把空 span 表达成 null（未发生的重考/重修列是空 span）。
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
