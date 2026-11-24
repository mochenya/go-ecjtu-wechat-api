package course

import (
	"strings"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

// testOpenID 是 fixture 里用的假 openid，不是真实凭证。
const testOpenID = "TESTOPENIDXXXXXXXXXXXXXXXXX"

// calendarPage 拼一个最小课表页，只服务结构负例。
func calendarPage(center string, items ...string) []byte {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<title>我的日历</title>\n</head>\n<body>\n")
	if center != "" {
		b.WriteString("<div class=\"center\">\n\t<p>\n\t\t" + center + "\n\t</p>\n</div>\n")
	}
	b.WriteString("<div class=\"top\"><div class=\"calendar\"><ul class=\"rl_info\">\n")
	for _, it := range items {
		b.WriteString("\t<li>\n" + it + "\t</li>\n")
	}
	b.WriteString("</ul></div></div>\n</body>\n</html>\n")
	return []byte(b.String())
}

// courseItem 按实测排版拼出一条课程 li 的内容。
func courseItem(classSpan, name, timeText, location, teacher string) string {
	return "\t\t\t<p>\n" +
		"\t\t\t\t<span class=\"class_span\">" + classSpan + "<br /> </span>\n" +
		"\t\t\t\t" + name + "\n\t\t\t\t<br />\n" +
		"\t\t\t\t时间：" + timeText + "\n\t\t\t\t<br />\n" +
		"\t\t\t\t地点：" + location + "\n\t\t\t\t<br />\n" +
		"\t\t\t\t教师：" + teacher + "\n\t\t\t\t<br />\n" +
		"\t\t\t</p>\n"
}

func TestParseDayWeekdayAndWeek(t *testing.T) {
	tests := []struct {
		name    string
		center  string
		date    string
		weekday int
		week    int
	}{
		{"周一第三周", "2026-09-14 星期一（第3周）", "2026-09-14", 1, 3},
		{"周日属当周", "2026-09-20 星期日（第3周）", "2026-09-20", 7, 3},
		{"星期天写法", "2026-09-20 星期天（第3周）", "2026-09-20", 7, 3},
		{"学期外远日期周次不设上界", "2027-01-01 星期五（第18周）", "2027-01-01", 5, 18},
	}
	q := &Query{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := q.parseDay(calendarPage(tt.center))
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			if got.Date != tt.date || got.Weekday != tt.weekday || got.Week != tt.week {
				t.Fatalf("got date=%q weekday=%d week=%d, 期望 date=%q weekday=%d week=%d",
					got.Date, got.Weekday, got.Week, tt.date, tt.weekday, tt.week)
			}
			if got.Courses == nil {
				t.Fatal("Courses 为 nil：无课日必须输出空切片")
			}
			if len(got.Courses) != 0 {
				t.Fatalf("期望 0 节课，实际 %d", len(got.Courses))
			}
		})
	}
}

func TestParseDayMultiRangeWeeks(t *testing.T) {
	body := calendarPage("2026-09-14 星期一（第3周）",
		courseItem("3-4节", "体育Ⅲ(上课)", "1-8,10-19 3,4", "北区田径场18", "教师A"),
		courseItem("7-8节", "数据库系统原理(实验)", "3 7,8", "软件设计实验室(教26栋403; 教26栋404)", "教师C"),
	)
	got, err := (&Query{}).parseDay(body)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(got.Courses) != 2 {
		t.Fatalf("期望 2 节课，实际 %d", len(got.Courses))
	}
	want := []WeekSpan{{1, 8}, {10, 19}}
	if len(got.Courses[0].Weeks) != len(want) {
		t.Fatalf("多段周次数量不符: got %v want %v", got.Courses[0].Weeks, want)
	}
	for i, w := range want {
		if got.Courses[0].Weeks[i] != w {
			t.Fatalf("Weeks[%d] = %+v, 期望 %+v", i, got.Courses[0].Weeks[i], w)
		}
	}
	if got.Courses[0].TimeText != "1-8,10-19 3,4" {
		t.Fatalf("TimeText 应保留原文，实际 %q", got.Courses[0].TimeText)
	}
	// 单周 Start==End，且地点保留分号列表原文不拆。
	if got.Courses[1].Weeks[0] != (WeekSpan{3, 3}) {
		t.Fatalf("单周周次应为 {3,3}，实际 %+v", got.Courses[1].Weeks[0])
	}
	if !strings.Contains(got.Courses[1].Location, ";") {
		t.Fatal("地点里的分号列表被改动了")
	}
}

func TestParseDaySplitsTrailingStatus(t *testing.T) {
	tests := []struct {
		line       string
		wantName   string
		wantStatus string
	}{
		{"大学英语Ⅲ(上课)", "大学英语Ⅲ", "上课"},
		{"高等数学(A)Ⅰ(考试)", "高等数学(A)Ⅰ", "考试"},
		{"数据库系统原理(实验)", "数据库系统原理", "实验"},
		{"形势与政策", "形势与政策", ""},
		{"创新创业基础(讲座)说明", "创新创业基础(讲座)说明", ""}, // 括号不贴词尾：整体是课名
		{"(考试)", "(考试)", ""},                 // 剥不出课名：整体是课名
		{"体育()", "体育()", ""},                 // 空括号不产生空状态
	}
	q := &Query{}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got, err := q.parseDay(calendarPage("2026-09-14 星期一（第3周）",
				courseItem("1-2节", tt.line, "1-8 1,2", "31-422", "教师B")))
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			c := got.Courses[0]
			if c.Name != tt.wantName || c.Status != tt.wantStatus {
				t.Fatalf("课名/状态 = %q/%q, 期望 %q/%q", c.Name, c.Status, tt.wantName, tt.wantStatus)
			}
		})
	}
}

func TestParseDayIgnoresNonCourseItems(t *testing.T) {
	body := calendarPage("2026-09-14 星期一（第3周）",
		"\t\t\t<p>\n\t\t\t\t<img src=\"imgs/today_schedule/easy1.png\" />\n\t\t\t</p>\n",
		"\t\t\t<p>\n\t\t\t\t时间：1-8 1,2\n\t\t\t</p>\n", // 无 class_span：不是课程条目
		courseItem("1-2节", "大学英语Ⅲ(上课)", "1-8 1,2", "31-422", "教师B"),
	)
	got, err := (&Query{}).parseDay(body)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(got.Courses) != 1 {
		t.Fatalf("非课程行未被忽略，实际 %d 条", len(got.Courses))
	}
}

func TestParseDayRejectsMalformedPages(t *testing.T) {
	tests := []struct {
		name  string
		fix   string // 非空则改用 testdata/pages 下的这个 fixture
		body  []byte
		subst string // 错误串必须含的子串
	}{
		{"无 .center（fixture 改版样本）", "course-missing-center.html", nil, "date line not found"},
		{"教师标签缺失（fixture 改版样本）", "course-missing-teacher.html", nil, "missing label teacher"},
		{"时间只有一段（fixture 改版样本）", "course-broken-time.html", nil, "time field has 1 segments"},
		{"无 title 的空文档", "", []byte("<html><body></body></html>"), "date line not found"},
		{"日期行不匹配", "", calendarPage("2026-09-14 星期一 第3周"), "date line does not match"},
		{"日期行半角括号", "", calendarPage("2026-09-14 星期一(第3周)"), "date line does not match"},
		{"class_span 但无 p", "", calendarPage("2026-09-14 星期一（第3周）",
			"<span class=\"class_span\">1-2节</span>大学英语(上课)"), "paragraph not found"},
		{"课名行缺失", "", calendarPage("2026-09-14 星期一（第3周）",
			"\t\t\t<p><span class=\"class_span\">1-2节</span><br />时间：1-8 1,2<br />地点：A<br />教师：B</p>\n"),
			"course name line not found"},
		{"出现未知标签行", "", calendarPage("2026-09-14 星期一（第3周）",
			"\t\t\t<p><span class=\"class_span\">1-2节</span><br />大学英语(上课)<br />时间：1-8 1,2<br />地点：A<br />教师：B<br />教室：C</p>\n"),
			"unexpected line"},
		{"重复标签", "", calendarPage("2026-09-14 星期一（第3周）",
			"\t\t\t<p><span class=\"class_span\">1-2节</span><br />大学英语(上课)<br />时间：1-8 1,2<br />时间：2-3 1,2<br />地点：A<br />教师：B</p>\n"),
			"duplicate label time"},
		{"时间三段", "", calendarPage("2026-09-14 星期一（第3周）",
			courseItem("1-2节", "大学英语(上课)", "1-8 1,2 extra", "A", "B")), "time field has 3 segments"},
		{"周次反区间", "", calendarPage("2026-09-14 星期一（第3周）",
			courseItem("1-2节", "大学英语(上课)", "8-1 1,2", "A", "B")), "week range is reversed"},
		{"周次非数字", "", calendarPage("2026-09-14 星期一（第3周）",
			courseItem("1-2节", "大学英语(上课)", "x 1,2", "A", "B")), "week range start is not a number"},
		{"周次三段界", "", calendarPage("2026-09-14 星期一（第3周）",
			courseItem("1-2节", "大学英语(上课)", "1-2-3 1,2", "A", "B")), "too many bounds"},
		{"节次非数字", "", calendarPage("2026-09-14 星期一（第3周）",
			courseItem("1-2节", "大学英语(上课)", "1-8 a", "A", "B")), "period is not a number"},
		{"节次空列表", "", calendarPage("2026-09-14 星期一（第3周）",
			courseItem("1-2节", "大学英语(上课)", "1-8 ", "A", "B")), "time field has 1 segments"},
	}
	q := &Query{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.body
			if tt.fix != "" {
				body = testsupport.Fixture(t, tt.fix)
			}
			_, err := q.parseDay(body)
			if err == nil {
				t.Fatal("期望报错，实际返回了结果")
			}
			kind, ok := ecjtu.KindOf(err)
			if !ok || kind != ecjtu.KindParse {
				t.Fatalf("分类应为 KindParse，实际 %v (%v)", kind, err)
			}
			if !strings.Contains(err.Error(), tt.subst) {
				t.Fatalf("错误串应含 %q，实际 %q", tt.subst, err.Error())
			}
			// 错误里不得出现页面 markup 与凭证。
			if strings.Contains(err.Error(), "<") || strings.Contains(err.Error(), testOpenID) {
				t.Fatalf("错误串泄露了页面原文或凭证: %q", err.Error())
			}
		})
	}
}

func TestParseDayAllowsEmptyLabelledValues(t *testing.T) {
	// 标签在场而值为空是数据缺失，不是结构改版：照常输出空串。
	body := calendarPage("2026-09-14 星期一（第3周）",
		courseItem("1-2节", "大学英语(上课)", "1-8 1,2", "", ""))
	got, err := (&Query{}).parseDay(body)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if got.Courses[0].Location != "" || got.Courses[0].Teacher != "" {
		t.Fatalf("空值应保留为空串，实际 %q/%q", got.Courses[0].Location, got.Courses[0].Teacher)
	}
}

func TestParseDayRejectsHalfWidthColon(t *testing.T) {
	// 实测标签是全角冒号；半角出现即视为改版，不静默兼容。
	body := calendarPage("2026-09-14 星期一（第3周）",
		"\t\t\t<p><span class=\"class_span\">1-2节</span><br />大学英语(上课)<br />时间:1-8 1,2<br />地点：A<br />教师：B</p>\n")
	if _, err := (&Query{}).parseDay(body); err == nil {
		t.Fatal("半角冒号应判 KindParse")
	}
}
