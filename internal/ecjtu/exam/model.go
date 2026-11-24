// Package exam 是考试域：抓取「考试安排」页面、判定页面身份、解析成分学期考试安排。
package exam

import "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"

// ExamItem 是一条考试安排。
type ExamItem struct {
	CourseName string `json:"courseName"`
	ExamWeek   int    `json:"examWeek"`  // 与日期无稳定对应（实测 4 月 30 日标 19 周），照抄
	ExamDate   string `json:"examDate"`  // 归一 2026-07-07
	DayOfWeek  int    `json:"dayOfWeek"` // 1=周一 … 7=周日
	StartTime  string `json:"startTime"` // 14:30
	EndTime    string `json:"endTime"`
	TimeText   string `json:"timeText"` // 原文，改版时不丢信息
	Location   string `json:"location"`
	CourseType string `json:"courseType"`
	ClassName  string `json:"className"` // 含 【小2班】 全文照抄
	Headcount  int    `json:"headcount"`
	Note       string `json:"note,omitempty"` // 红色 div trim 后文本，可空
}

// Result 是考试域的响应容器，与成绩域共享同一形状。
type Result = ecjtu.Result[ExamItem]
