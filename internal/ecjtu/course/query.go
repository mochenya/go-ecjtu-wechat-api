// Package course 是课表域：抓取「我的日历」页面、判定页面身份、解析成当日课表。
package course

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

// opDay 是错误归因用的操作名，形如「包名.方法名」，绝不参与拼 URL。
const opDay = "course.Day"

// Query 是课表域的查询入口，已装配好 transport。
type Query struct {
	t *ecjtu.Transport
}

// NewQuery 创建使用指定抓取器的课表查询入口。
func NewQuery(t *ecjtu.Transport) *Query { return &Query{t: t} }

// Day 返回指定日期的课表。日期格式由调用方本地校验（上游会因非法日期 500）；
// 星期与周次一律取页面回显值，客户端不做日历推算。
func (q *Query) Day(ctx context.Context, date time.Time) (Schedule, error) {
	if date.IsZero() {
		// 零值会格式化成 0001-01-01 发往上游，必须在本地拦截。
		return Schedule{}, &ecjtu.Error{Kind: ecjtu.KindInvalidInput, Op: opDay, Cause: errors.New("date is required")}
	}
	params := url.Values{"date": {date.Format("2006-01-02")}}
	resp, err := q.t.Get(ctx, opDay, "CalendarServlet", params)
	if err != nil {
		return Schedule{}, err
	}
	kind, err := ecjtu.CheckPage(opDay, resp, ecjtu.TitleCourse)
	if err != nil {
		return Schedule{}, err
	}
	ecjtu.Observe(ctx, ecjtu.Observation{Op: opDay, Status: resp.Status, Kind: kind,
		Attempts: resp.Attempts, Elapsed: resp.Elapsed})
	if kind == ecjtu.PageEmpty {
		// 课表域没有空态页：无课日应是 PageOK + 0 节课。真出现「暂无」文案说明
		// 页面结构与其余业务页都不同，按改版处理，不返回半满的 Schedule。
		return Schedule{}, &ecjtu.Error{Kind: ecjtu.KindParse, Op: opDay,
			Cause: errors.New("empty-state page returned by course domain")}
	}
	return q.parseDay(resp.Body)
}
