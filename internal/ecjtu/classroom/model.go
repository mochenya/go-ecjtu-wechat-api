// Package classroom 查询空闲教室并把上游表单与结果页解析成领域模型。
package classroom

import "fmt"

// Weekday 是空教室查询可选的星期。
type Weekday string

const (
	Monday    Weekday = "星期一"
	Tuesday   Weekday = "星期二"
	Wednesday Weekday = "星期三"
	Thursday  Weekday = "星期四"
	Friday    Weekday = "星期五"
	Saturday  Weekday = "星期六"
	Sunday    Weekday = "星期日"
)

// String 返回星期名称；未知值以 Weekday(值) 表示。
func (w Weekday) String() string {
	switch w {
	case Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday:
		return string(w)
	default:
		return fmt.Sprintf("Weekday(%q)", string(w))
	}
}

// Period 是空教室查询可选的连续两节课。
type Period string

const (
	Period1To2   Period = "1,2"
	Period3To4   Period = "3,4"
	Period5To6   Period = "5,6"
	Period7To8   Period = "7,8"
	Period9To10  Period = "9,10"
	Period11To12 Period = "11,12"
)

// String 返回节次值；未知值以 Period(值) 表示。
func (p Period) String() string {
	switch p {
	case Period1To2, Period3To4, Period5To6, Period7To8, Period9To10, Period11To12:
		return string(p)
	default:
		return fmt.Sprintf("Period(%q)", string(p))
	}
}

// Options 是上游空教室表单提供的选项与浏览器有效默认值。
type Options struct {
	SelectableTerms []string  `json:"selectableTerms"`
	SelectedTerm    string    `json:"selectedTerm"`
	Weeks           []int     `json:"weeks"`
	Weekdays        []Weekday `json:"weekdays"`
	Periods         []Period  `json:"periods"`
	Defaults        Defaults  `json:"defaults"`
}

// Defaults 是表单各下拉框的有效默认选项。
type Defaults struct {
	Week    int     `json:"week"`
	Weekday Weekday `json:"weekday"`
	Period  Period  `json:"period"`
}

// Input 是空教室查询条件；Term、Weekday 与 Period 可留空以使用表单默认值。
type Input struct {
	Term    string  `json:"term"`
	Week    int     `json:"week"`
	Weekday Weekday `json:"weekday"`
	Period  Period  `json:"period"`
}

// Result 是一次空教室查询及其解析结果。
type Result struct {
	Term      string  `json:"term"`
	Week      int     `json:"week"`
	Weekday   Weekday `json:"weekday"`
	Period    Period  `json:"period"`
	Count     int     `json:"count"`
	Rooms     []Room  `json:"rooms"`
	PageCount *int    `json:"-"`
}

// Room 是一间空闲教室。
type Room struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	SeatCount int    `json:"seatCount"`
}
