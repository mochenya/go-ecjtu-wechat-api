// Package score 是成绩域：抓取「我的成绩」页面、判定页面身份、解析成分学期成绩。
package score

import "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"

// ScoreItem 是一条成绩记录。
type ScoreItem struct {
	CourseName  string  `json:"courseName"`
	CourseCode  string  `json:"courseCode"`
	Credit      float64 `json:"credit"` // 0.0 是合法值，故不加 omit
	FinalScore  string  `json:"finalScore"`
	RetakeScore *string `json:"retakeScore"` // 空 span → null
	RepeatScore *string `json:"repeatScore"`
	Flag        string  `json:"flag"`       // 主修（HTML class=flag）
	CourseType  string  `json:"courseType"` // 必修课/学科任选课等
}

// Result 是成绩域的响应容器，与考试域共享同一形状。
type Result = ecjtu.Result[ScoreItem]
