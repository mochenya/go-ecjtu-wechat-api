package course

// WeekSpan 是一个连续的上课周区间；单周时 Start==End。
type WeekSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Schedule 是某一天的课表。无课日仍是正常结果：Courses 为空切片而非 null。
type Schedule struct {
	Date    string   `json:"date"`    // 页面回显的日期，形如 2026-09-14
	Weekday int      `json:"weekday"` // 1=周一 … 7=周日
	Week    int      `json:"week"`    // 上游自学期首日起线性外推，照抄，不设上界
	Courses []Course `json:"courses"`
}

// Course 是一节课的安排。
type Course struct {
	Name     string     `json:"name"`
	Status   string     `json:"status,omitempty"` // 上课/实验/考试；词尾括号对剥不出则为空并省略
	TimeText string     `json:"timeText"`         // 原文 1-8,10-19 3,4
	Weeks    []WeekSpan `json:"weeks"`
	Periods  []int      `json:"periods"`
	Location string     `json:"location"` // 原始串，分号列表不拆
	Teacher  string     `json:"teacher"`
}
