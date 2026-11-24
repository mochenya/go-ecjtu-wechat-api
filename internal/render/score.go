package render

import (
	"strconv"
	"strings"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/score"
)

// Score 渲染成绩表：首行是页头回显，列集固定为 课程|代码|学分|期末|重考|重修|性质，
// null 列渲染成 `-`。
func Score(r score.Result) string {
	t := newTable("课程", "代码", "学分", "期末", "重考", "重修", "性质")
	for _, it := range r.Items {
		t.add(it.CourseName, it.CourseCode, creditLabel(it.Credit), it.FinalScore,
			nullableLabel(it.RetakeScore), nullableLabel(it.RepeatScore), it.CourseType)
	}
	return headerLine(deref(r.StudentName), deref(r.Term), "门", r.Count) + "\n" + t.String()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// nullableLabel 把 null 渲染成占位符，避免空列看起来像解析失败。
func nullableLabel(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

// creditLabel 渲染学分：整数补一位小数（对齐上游 `学分:4.0` 的写法），小数保持原值。
func creditLabel(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
