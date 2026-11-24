package score

import (
	"strings"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

// testOpenID 是 fixture 里用的假 openid，不是真实凭证。
const testOpenID = "TESTOPENIDXXXXXXXXXXXXXXXXX"

// scorePage 拼一个最小成绩页，只服务列顺序与标签变体等结构负例。
func scorePage(rows ...string) []byte {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<title>我的成绩</title>\n</head>\n<body>\n")
	b.WriteString("<div class=\"right\">\n姓名:\n<span>张三</span>\n<br />\n当前学期:\n<span>2025.2</span>\n</div>\n")
	b.WriteString("<ul class=\"dropdown-menu\"><li><a href=\"#\">2026.2</a></li><li><a href=\"#\">2025.2</a></li></ul>\n")
	b.WriteString("<div class=\"words\">您好！本学期当前你共有<strong>1</strong>门考试成绩。</div>\n")
	for _, row := range rows {
		b.WriteString("<div class=\"row \"><div class=\"col-xs-12\">" + row + "</div></div>\n")
	}
	b.WriteString("</body>\n</html>\n")
	return []byte(b.String())
}

// scoreItemBlock 按实测排版拼出一条成绩的 div.text。
func scoreItemBlock(courseText string, grades ...[2]string) string {
	var b strings.Builder
	b.WriteString("<div class=\"text\"><span class=\"course\">" + courseText + "</span><div class=\"grade\">")
	for _, g := range grades {
		b.WriteString(g[0] + "\n<span class=\"score\">" + g[1] + "</span><br />")
	}
	b.WriteString("<span class=\"flag\">主修</span></div></div><div class=\"type\"><span class=\"require\"><mark>必修课</mark> </span></div>")
	return b.String()
}

func TestParseScoresKeepsParenthesisedCourseName(t *testing.T) {
	// 非贪婪正则容易在第一个左括号处截断含括号课名，这里作回归。
	tests := []struct {
		courseText string
		wantName   string
		wantCode   string
		wantCredit float64
	}{
		{"【主修】【1508100012】高等数学(A)Ⅱ(学分:4.0)", "高等数学(A)Ⅱ", "1508100012", 4.0},
		{"【主修】【1508100011】高等数学(A)Ⅰ(考试)(学分:6.0)", "高等数学(A)Ⅰ(考试)", "1508100011", 6.0},
		{"【主修】【1500190090】专业导论(学分:0.0)", "专业导论", "1500190090", 0.0}, // 0.0 是合法值
		{"【重修】【1521101430】数据结构(学分:3.5)", "数据结构", "1521101430", 3.5},
	}
	q := &Query{}
	for _, tt := range tests {
		t.Run(tt.courseText, func(t *testing.T) {
			res, err := q.parseScores(scorePage(scoreItemBlock(tt.courseText,
				[2]string{"期末成绩:", "86"}, [2]string{"重考成绩:", ""}, [2]string{"重修成绩:", ""})))
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			got := res.Items[0]
			if got.CourseName != tt.wantName || got.CourseCode != tt.wantCode || got.Credit != tt.wantCredit {
				t.Fatalf("got %+v, 期望 name=%q code=%q credit=%v", got, tt.wantName, tt.wantCode, tt.wantCredit)
			}
			if got.Flag != "主修" && got.Flag != "重修" {
				t.Fatalf("flag 应取 span.flag，实际 %q", got.Flag)
			}
			if got.CourseType != "必修课" {
				t.Fatalf("courseType 应取 div.type mark，实际 %q", got.CourseType)
			}
		})
	}
}

func TestParseScoresNullableColumns(t *testing.T) {
	res, err := (&Query{}).parseScores(scorePage(scoreItemBlock(
		"【主修】【1500100101】职业生涯与发展规划(学分:0.5)",
		[2]string{"期末成绩:", "良好"}, [2]string{"重考成绩:", ""}, [2]string{"重修成绩:", "77"})))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	got := res.Items[0]
	if got.FinalScore != "良好" {
		t.Fatalf("等级分应照抄，实际 %q", got.FinalScore)
	}
	if got.RetakeScore != nil {
		t.Fatalf("空 span 应为 null，实际 %q", *got.RetakeScore)
	}
	if got.RepeatScore == nil || *got.RepeatScore != "77" {
		t.Fatalf("重修成绩应为 77，实际 %v", got.RepeatScore)
	}
}

func TestParseScoresAnchorsColumnsByLabel(t *testing.T) {
	// 三列按标签锚定：打乱顺序，含插入未知列前的位置变化，仍取对。
	res, err := (&Query{}).parseScores(scorePage(scoreItemBlock(
		"【主修】【1508100012】高等数学(A)Ⅱ(学分:4.0)",
		[2]string{"重修成绩:", "重修80"}, [2]string{"重考成绩:", "重考85"}, [2]string{"期末成绩:", "79"})))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	got := res.Items[0]
	if got.FinalScore != "79" {
		t.Fatalf("期末成绩错位，实际 %q", got.FinalScore)
	}
	if got.RetakeScore == nil || *got.RetakeScore != "重考85" {
		t.Fatalf("重考成绩错位，实际 %v", got.RetakeScore)
	}
	if got.RepeatScore == nil || *got.RepeatScore != "重修80" {
		t.Fatalf("重修成绩错位，实际 %v", got.RepeatScore)
	}
}

func TestParseScoresRejectsMalformedPages(t *testing.T) {
	tests := []struct {
		name  string
		fix   string
		body  []byte
		subst string
	}{
		{"学分后缀缺失（fixture 改版样本）", "score-missing-credit.html", nil, "credit suffix not found"},
		{"页头缺失（fixture 改版样本）", "score-missing-headinfo.html", nil, "without head info"},
		{"成绩列只有两列", "", scorePage(scoreItemBlock("【主修】【1】课名(学分:1.0)",
			[2]string{"期末成绩:", "86"}, [2]string{"重考成绩:", ""})), "missing grade column repeat"},
		{"重复成绩列", "", scorePage(scoreItemBlock("【主修】【1】课名(学分:1.0)",
			[2]string{"期末成绩:", "86"}, [2]string{"期末成绩:", "87"}, [2]string{"重考成绩:", ""},
			[2]string{"重修成绩:", ""})), "duplicate grade column final"},
		{"无标签的分数列", "", scorePage(scoreItemBlock("【主修】【1】课名(学分:1.0)",
			[2]string{"", "86"}, [2]string{"期末成绩:", "86"}, [2]string{"重考成绩:", ""},
			[2]string{"重修成绩:", ""})), "score column without label"},
		{"课名括号组缺失", "", scorePage(scoreItemBlock("1508100012 高等数学(学分:4.0)",
			[2]string{"期末成绩:", "86"}, [2]string{"重考成绩:", ""}, [2]string{"重修成绩:", ""})),
			"course code brackets not found"},
		{"课程代码非数字", "", scorePage(scoreItemBlock("【主修】【高等数学】课名(学分:4.0)",
			[2]string{"期末成绩:", "86"}, [2]string{"重考成绩:", ""}, [2]string{"重修成绩:", ""})),
			"course code is not numeric"},
		{"学分为空括号", "", scorePage(scoreItemBlock("【主修】【1】课名(学分:)",
			[2]string{"期末成绩:", "86"}, [2]string{"重考成绩:", ""}, [2]string{"重修成绩:", ""})),
			"credit suffix not found"},
		{"课名为空", "", scorePage(scoreItemBlock("【主修】【1】(学分:1.0)",
			[2]string{"期末成绩:", "86"}, [2]string{"重考成绩:", ""}, [2]string{"重修成绩:", ""})),
			"course name is empty"},
	}
	q := &Query{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.body
			if tt.fix != "" {
				body = testsupport.Fixture(t, tt.fix)
			}
			_, err := q.parseScores(body)
			if err == nil {
				t.Fatal("期望报错，实际返回了结果")
			}
			if !ecjtu.IsKind(err, ecjtu.KindParse) {
				t.Fatalf("分类应为 KindParse，实际 %v", err)
			}
			if !strings.Contains(err.Error(), tt.subst) {
				t.Fatalf("错误串应含 %q，实际 %q", tt.subst, err.Error())
			}
			if strings.Contains(err.Error(), "<") || strings.Contains(err.Error(), testOpenID) {
				t.Fatalf("错误串泄露了页面原文或凭证: %q", err.Error())
			}
		})
	}
}

// 页面自带条数与解析条数不一致时不升级为错误，只留给 --verbose 警告。
func TestParseScoresNoRowsIsNotSilentlyEmpty(t *testing.T) {
	res, err := (&Query{}).parseScores(testsupport.Fixture(t, "score-no-rows.html"))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if res.Empty {
		t.Fatal("items 为空不等于空态页：empty 必须为 false")
	}
	if res.Count != 0 || len(res.Items) != 0 {
		t.Fatalf("count 应为解析值 0，实际 %d", res.Count)
	}
	if res.PageCount == nil || *res.PageCount != 15 {
		t.Fatalf("页面自带条数应为 15（供 verbose 比对），实际 %v", res.PageCount)
	}
	if res.Term == nil || *res.Term != "2025.2" {
		t.Fatalf("页面学期回显应为 2025.2，实际 %v", res.Term)
	}
	wantTerms := []string{"2026.2", "2026.1", "2025.2", "2025.1"}
	if strings.Join(res.AvailableTerms, ",") != strings.Join(wantTerms, ",") {
		t.Fatalf("学期列表应保持页面顺序，实际 %v", res.AvailableTerms)
	}
}
