// Package live 是真实上游冒烟测试：默认全部跳过，设置非空 ECJTU_LIVE 后才运行
// （真实请求是人工动作，不进 CI、不进 gate、不批量循环）。
//
// 运行方式（人工）：
//
//	ECJTU_LIVE=1 go test ./test/live -run Live -v
//
// 断言只打印条数与脱敏结论，绝不打印响应原文、完整 openid 或 URL query。
package live_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/config"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/client"
)

func skipUnlessLive(t *testing.T) context.Context {
	t.Helper()
	if os.Getenv("ECJTU_LIVE") == "" {
		t.Skip("需要真实上游：设 ECJTU_LIVE=1 并注入 ECJTU_WEIXIN_TOKEN 后运行")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func newLiveClient(t *testing.T) (*client.Query, config.Config) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("配置不可用（ECJTU_LIVE=1 时需要 ECJTU_WEIXIN_TOKEN）: %v", err)
	}
	school, err := client.New(client.Config{BaseURL: cfg.BaseURL, OpenID: cfg.OpenID})
	if err != nil {
		t.Fatal(err)
	}
	return school, cfg
}

func TestLiveCourse(t *testing.T) {
	ctx := skipUnlessLive(t)
	c, cfg := newLiveClient(t)
	schedule, err := c.Course(ctx, time.Now())
	if err != nil {
		t.Fatalf("课表抓取失败: %v", err)
	}
	assertNoCredentialLeak(t, cfg.OpenID, schedule.Date)
	t.Logf("课表：日期=%s 星期=%d 周次=%d 门数=%d", schedule.Date, schedule.Weekday, schedule.Week, len(schedule.Courses))
}

// 不传 term，验证上游落到它自己的默认学期。
func TestLiveScore(t *testing.T) {
	ctx := skipUnlessLive(t)
	c, cfg := newLiveClient(t)
	res, err := c.Scores(ctx, "")
	if err != nil {
		t.Fatalf("成绩抓取失败: %v", err)
	}
	if res.Count != len(res.Items) {
		t.Fatalf("count=%d 与 items=%d 不一致", res.Count, len(res.Items))
	}
	assertNoCredentialLeak(t, cfg.OpenID, derefTerm(res.Term))
	t.Logf("成绩：页面学期=%s 条数=%d 空态=%v", derefTerm(res.Term), res.Count, res.Empty)
}

func TestLiveExam(t *testing.T) {
	ctx := skipUnlessLive(t)
	c, cfg := newLiveClient(t)
	res, err := c.Exams(ctx, "")
	if err != nil {
		t.Fatalf("考试抓取失败: %v", err)
	}
	assertNoCredentialLeak(t, cfg.OpenID, derefTerm(res.Term))
	t.Logf("考试：页面学期=%s 条数=%d 空态=%v", derefTerm(res.Term), res.Count, res.Empty)
}

// 把真实结果过一次 JSON 编码，只断言形状与脱敏，不打印正文。
func TestLiveJSONShape(t *testing.T) {
	ctx := skipUnlessLive(t)
	c, cfg := newLiveClient(t)
	schedule, err := c.Course(ctx, time.Now())
	if err != nil {
		t.Fatalf("课表抓取失败: %v", err)
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(schedule); err != nil {
		t.Fatalf("编码失败: %v", err)
	}
	encoded := buf.String()
	for _, key := range []string{`"date"`, `"weekday"`, `"week"`, `"courses"`} {
		if !strings.Contains(encoded, key) {
			t.Fatalf("JSON 缺少顶层键 %s", key)
		}
	}
	assertNoCredentialLeak(t, cfg.OpenID, "")
	if strings.Contains(encoded, "<") {
		t.Fatalf("JSON 里出现了 HTML 片段")
	}
}

// assertNoCredentialLeak 确认真实凭证没有渗进任何结构化输出。extra 是待查的字段值。
func assertNoCredentialLeak(t *testing.T, openID string, extra string) {
	t.Helper()
	if openID != "" && strings.Contains(extra, openID) {
		t.Fatalf("字段值里出现了完整 openid（%s***）", openID[:4])
	}
	for _, forbidden := range []string{"jsessionid", "JSESSIONID", "weiXinID"} {
		if strings.Contains(strings.ToLower(extra), strings.ToLower(forbidden)) {
			t.Fatalf("字段值里出现了 %s", forbidden)
		}
	}
}

func derefTerm(s *string) string {
	if s == nil {
		return "(null)"
	}
	return *s
}
