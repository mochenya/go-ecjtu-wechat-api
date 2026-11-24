package ecjtu

// Result 是成绩与考试共用的结果。
// 查询成功时 Items 非 nil；空态页的 Term、StudentName 和 AvailableTerms 为 nil。
type Result[T any] struct {
	Term           *string  `json:"term"`
	StudentName    *string  `json:"studentName"`
	AvailableTerms []string `json:"availableTerms"`
	Empty          bool     `json:"empty"`
	Count          int      `json:"count"`
	Items          []T      `json:"items"`

	// PageCount 是页面 div.words 自带的条数，只服务 --verbose 的一致性警告，不进 JSON。
	PageCount *int `json:"-"`
}
