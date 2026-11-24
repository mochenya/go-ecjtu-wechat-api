package exam

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

// testOpenID 是 fixture 里用的假 openid，不是真实凭证。
const testOpenID = "TESTOPENIDXXXXXXXXXXXXXXXXX"

// examPage 拼一个最小考试页；withHead=false 用来模拟页头缺失的改版页。
func examPage(withHead bool, rows ...string) []byte {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<title>考试安排</title>\n</head>\n<body>\n")
	if withHead {
		b.WriteString("<div class=\"right\">\n姓名:\n<span>张三</span>\n<br />\n当前学期:\n<span>2025.2</span>\n</div>\n")
		b.WriteString("<ul class=\"dropdown-menu\"><li><a href=\"#\">2026.2</a></li><li><a href=\"#\">2025.2</a></li></ul>\n")
		b.WriteString("<div class=\"words\">您好！本学期你共有<mark>1</mark>门考试安排。</div>\n")
	}
	for _, row := range rows {
		b.WriteString("<div class=\"row \"><div class=\"col-xs-12\">" + row + "</div></div>\n")
	}
	b.WriteString("</body>\n</html>\n")
	return []byte(b.String())
}

// field 是一条「标签 + 取值」，取值元素按实测可能是 u 或 span。
func field(label, tag, value string) string {
	return label + "\n<" + tag + ">" + value + "</" + tag + ">\n<br />\n"
}

// examItemBlock 按实测顺序拼出 div.text 与同级的 div.course。
func examItemBlock(course string, fields ...string) string {
	return "<div class=\"text\">" + strings.Join(fields, "") + "</div>" +
		"<div class=\"course\"><mark>\n" + course + "\n</mark></div>"
}

func TestParseExamsFields(t *testing.T) {
	body := examPage(true, examItemBlock("离散数学",
		field("考试周次:", "u", "9"),
		field("考试时间:", "u", "2026年04月30日(星期四)14:00-16:00"),
		field("考试地点:", "u", "进贤1-202"),
		field("课程性质:", "span", "必修课"),
		field("班级名称:", "span", "离散数学(20252-2)【小1班】"),
		field("考试人数:", "span", "32"),
	))
	got, err := (&Query{}).parseExams(body)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	item := got.Items[0]
	want := ExamItem{
		CourseName: "离散数学", ExamWeek: 9, ExamDate: "2026-04-30", DayOfWeek: 4,
		StartTime: "14:00", EndTime: "16:00", TimeText: "2026年04月30日(星期四)14:00-16:00",
		Location: "进贤1-202", CourseType: "必修课", ClassName: "离散数学(20252-2)【小1班】", Headcount: 32,
	}
	if item != want {
		t.Fatalf("考试条目 = %+v，期望 %+v", item, want)
	}
}

func TestParseExamsAnchorsByLabelNotPosition(t *testing.T) {
	// 按 u/span 出现下标取值在字段换序时会整体错位，这里作回归。
	body := examPage(true, examItemBlock("C语言程序设计",
		field("考试人数:", "span", "29"),
		field("考试地点:", "u", "进贤1-502"),
		field("课程性质:", "span", "必修课"),
		field("考试周次:", "u", "19"),
		field("班级名称:", "span", "C语言程序设计(20251-5)【小2班】"),
		field("考试时间:", "u", "2026年01月08日(星期四)14:00-16:00"),
	))
	got, err := (&Query{}).parseExams(body)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	item := got.Items[0]
	if item.Headcount != 29 || item.Location != "进贤1-502" || item.ExamWeek != 19 ||
		item.ExamDate != "2026-01-08" || item.DayOfWeek != 4 {
		t.Fatalf("字段换序后取值错位: %+v", item)
	}
}

func TestParseExamsNote(t *testing.T) {
	const timeField = "<u>2026年07月07日(星期二)14:30-16:30" +
		"<br/><div style='color:red;font-weight:bold;text-align:center;'>    教室在进贤校区</div></u>" + "\n<br />\n"
	body := examPage(true, examItemBlock("高等数学(A)Ⅱ",
		field("考试周次:", "u", "19"),
		"考试时间:\n"+timeField,
		field("考试地点:", "u", "进贤1-308"),
		field("课程性质:", "span", "必修课"),
		field("班级名称:", "span", "高等数学(A)Ⅱ(20252-38)【小1班】"),
		field("考试人数:", "span", "31"),
	))
	got, err := (&Query{}).parseExams(body)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	item := got.Items[0]
	if item.Note != "教室在进贤校区" {
		t.Fatalf("备注应取红色 div 并 trim，实际 %q", item.Note)
	}
	if item.TimeText != "2026年07月07日(星期二)14:30-16:30" {
		t.Fatalf("备注文本污染了 timeText，实际 %q", item.TimeText)
	}
	if item.ExamDate != "2026-07-07" || item.StartTime != "14:30" || item.EndTime != "16:30" {
		t.Fatalf("时间归一化失败: %+v", item)
	}
}

func TestParseExamsEmptyNoteOmitted(t *testing.T) {
	res, err := (&Query{}).parseExams(testsupport.Fixture(t, "exam-normal-2025-1.html"))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if res.Items[0].Note != "" {
		t.Fatalf("空白备注 div 应归一为空串，实际 %q", res.Items[0].Note)
	}
	if res.Items[1].Note == "" {
		t.Fatal("非空备注应保留")
	}
	encoded, err := json.Marshal(res.Items[0])
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}
	if strings.Contains(string(encoded), "note") {
		t.Fatalf("空备注应被 omit，实际 %s", encoded)
	}
}

func TestParseExamsRejectsMalformedPages(t *testing.T) {
	good := []string{
		field("考试周次:", "u", "19"),
		field("考试时间:", "u", "2026年07月07日(星期二)14:30-16:30"),
		field("考试地点:", "u", "进贤1-308"),
		field("课程性质:", "span", "必修课"),
		field("班级名称:", "span", "课名(20252-38)【小1班】"),
		field("考试人数:", "span", "31"),
	}
	tests := []struct {
		name  string
		fix   string
		body  []byte
		subst string
	}{
		{"字段标签缺失（fixture 改版样本）", "exam-missing-label.html", nil, "missing field headcount"},
		{"时间括号变全角（fixture 改版样本）", "exam-bad-time.html", nil, "exam time text does not match"},
		{"页头缺失", "", examPage(false, examItemBlock("课名", good...)), "without head info"},
		{"无 div.row 的改版页", "", examPage(true), "count"}, // 有页头无行：不报错，单独断言
		{"周次非数字", "", examPage(true, examItemBlock("课名",
			field("考试周次:", "u", "第十周"), good[1], good[2], good[3], good[4], good[5])), "exam week is not a number"},
		{"人数非数字", "", examPage(true, examItemBlock("课名",
			good[0], good[1], good[2], good[3], good[4], field("考试人数:", "span", "若干"))), "headcount is not a number"},
		{"日期不存在", "", examPage(true, examItemBlock("课名",
			good[0], field("考试时间:", "u", "2026年02月30日(星期一)14:30-16:30"),
			good[2], good[3], good[4], good[5])), "not a real instant"},
		{"无标签的取值", "", examPage(true, examItemBlock("课名",
			"<span>必修课</span>", good[0], good[1], good[2], good[4], good[5])), "value without label"},
		{"字段重复", "", examPage(true, examItemBlock("课名",
			append(append([]string{}, good...), field("考试地点:", "u", "进贤9-999"))...)), "duplicate field location"},
		{"课程名缺失", "", examPage(true, "<div class=\"text\">"+strings.Join(good, "")+"</div>"), "course name mark not found"},
	}
	q := &Query{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.body
			if tt.fix != "" {
				body = testsupport.Fixture(t, tt.fix)
			}
			got, err := q.parseExams(body)
			if tt.subst == "count" {
				// 行缺失不升级为错误，只留 verbose 警告；empty 仍为 false。
				if err != nil {
					t.Fatalf("无行页不应报错: %v", err)
				}
				if got.Empty || got.Count != 0 || got.Items == nil {
					t.Fatalf("期望 empty:false count:0 items:[]，实际 %+v", got)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望报错，实际 %+v", got)
			}
			if !ecjtu.IsKind(err, ecjtu.KindParse) {
				t.Fatalf("分类应为 KindParse，实际 %v", err)
			}
			if !strings.Contains(err.Error(), tt.subst) {
				t.Fatalf("错误串应含 %q，实际 %q", tt.subst, err.Error())
			}
			if strings.Contains(err.Error(), "<") || strings.Contains(err.Error(), testOpenID) {
				t.Fatalf("错误串泄露了页面原文或凭证: %q", err.Error())
			}
		})
	}
}
