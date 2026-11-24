package render

import (
	"strconv"
	"strings"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/course"
)

// weekdayCN 把 weekday 投影回中文（int 是数据，中文是视图）。
var weekdayCN = [...]string{"", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六", "星期日"}

// Course 渲染课表：首行是页面回显的日期与周次，随后是固定列集 节次|课程|时间|地点|教师。
// 「课程」列把 name 与 status 合成可读单元，不新增数据；无课日由调用方决定文案。
func Course(s course.Schedule) string {
	day := ""
	if s.Weekday >= 1 && s.Weekday <= 7 {
		day = " " + weekdayCN[s.Weekday] // 越界的星期不猜，直接省略
	}
	head := strings.TrimSpace(s.Date+day) + "（第" + strconv.Itoa(s.Week) + "周）"

	t := newTable("节次", "课程", "时间", "地点", "教师")
	for _, c := range s.Courses {
		t.add(periodsLabel(c.Periods), nameWithStatus(c), c.TimeText, c.Location, c.Teacher)
	}
	return head + "\n" + t.String()
}

// nameWithStatus 复现上游页面的课名写法：括号状态非空才追加。
func nameWithStatus(c course.Course) string {
	if c.Status == "" {
		return c.Name
	}
	return c.Name + "(" + c.Status + ")"
}

// periodsLabel 把节次序列渲染成 `3-4节`；不连续的段用逗号分隔。
func periodsLabel(periods []int) string {
	if len(periods) == 0 {
		return ""
	}
	var b strings.Builder
	start := periods[0]
	prev := start
	flush := func(from, to int) {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(strconv.Itoa(from))
		if to > from {
			b.WriteString("-" + strconv.Itoa(to))
		}
	}
	for _, p := range periods[1:] {
		if p == prev+1 {
			prev = p
			continue
		}
		flush(start, prev)
		start, prev = p, p
	}
	flush(start, prev)
	b.WriteString("节")
	return b.String()
}
