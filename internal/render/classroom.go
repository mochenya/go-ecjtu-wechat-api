package render

import (
	"strconv"
	"strings"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/classroom"
)

// ClassroomOptions 渲染本次表单实际提供的选项和默认值。
func ClassroomOptions(options classroom.Options) string {
	lines := []string{
		"可选学期：" + strings.Join(options.SelectableTerms, ", "),
		"表单选中学期：" + options.SelectedTerm,
		"可选周次：" + joinInts(options.Weeks),
		"可选星期：" + joinWeekdays(options.Weekdays),
		"可选节次：" + joinPeriods(options.Periods),
		"表单默认值：第" + strconv.Itoa(options.Defaults.Week) + "周，" +
			string(options.Defaults.Weekday) + "，" + string(options.Defaults.Period),
	}
	return strings.Join(lines, "\n")
}

// Classrooms 渲染空教室结果表；roomFilter 非空时在首行附上教室名筛选关键词。
func Classrooms(result classroom.Result, roomFilter string) string {
	t := newTable("教室", "教室类型", "座位数")
	for _, room := range result.Rooms {
		t.add(room.Name, room.Type, strconv.Itoa(room.SeatCount))
	}
	summary := "共 " + strconv.Itoa(result.Count) + " 间"
	if roomFilter != "" {
		summary = "教室含 " + roomFilter + "，" + summary
	}
	head := result.Term + " 第" + strconv.Itoa(result.Week) + "周 " +
		string(result.Weekday) + " " + string(result.Period) + "（" + summary + "）"
	return head + "\n" + t.String()
}

func joinInts(values []int) string {
	items := make([]string, len(values))
	for i, value := range values {
		items[i] = strconv.Itoa(value)
	}
	return strings.Join(items, ", ")
}

func joinWeekdays(values []classroom.Weekday) string {
	items := make([]string, len(values))
	for i, value := range values {
		items[i] = string(value)
	}
	return strings.Join(items, ", ")
}

func joinPeriods(values []classroom.Period) string {
	items := make([]string, len(values))
	for i, value := range values {
		items[i] = string(value)
	}
	return strings.Join(items, ", ")
}
