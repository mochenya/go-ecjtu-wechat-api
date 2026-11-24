package render

import (
	"strconv"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/exam"
)

// Exam 渲染考试表：首行是页头回显，列集固定为 日期|时间|周次|地点|课程|性质|人数；
// 备注非空时在行下方缩进一行，班级名称不进表（太长），只留在 JSON 里。
func Exam(r exam.Result) string {
	const courseCol = 4
	t := newTable("日期", "时间", "周次", "地点", "课程", "性质", "人数")
	for _, it := range r.Items {
		t.add(it.ExamDate, it.StartTime+"-"+it.EndTime, strconv.Itoa(it.ExamWeek),
			it.Location, it.CourseName, it.CourseType, strconv.Itoa(it.Headcount))
		if it.Note != "" {
			t.note(courseCol, "备注："+it.Note)
		}
	}
	return headerLine(deref(r.StudentName), deref(r.Term), "场", r.Count) + "\n" + t.String()
}
