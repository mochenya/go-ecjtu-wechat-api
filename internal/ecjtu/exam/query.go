package exam

import (
	"context"
	"errors"
	"net/url"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

// opExams 是错误归因用的操作名，形如「包名.方法名」，绝不参与拼 URL。
const opExams = "exam.Exams"

// Query 是考试域的查询入口，已装配好 transport。
type Query struct {
	t *ecjtu.Transport
}

// NewQuery 创建使用指定抓取器的考试查询入口。
func NewQuery(t *ecjtu.Transport) *Query { return &Query{t: t} }

// Terms 从一次考试查询中发现学期；空学期由上游选择默认值。
// 非空学期无效时返回输入错误，页面或学期形态异常时返回对应分类的错误。
func (q *Query) Terms(ctx context.Context, term string) (ecjtu.TermDiscovery, error) {
	result, err := q.Exams(ctx, term)
	if err != nil {
		return ecjtu.TermDiscovery{}, err
	}
	return ecjtu.DiscoverTerms(opExams, term, result)
}

// Exams 返回指定学期的考试安排。term 为空串时不发送该参数，由上游落到它的默认学期。
func (q *Query) Exams(ctx context.Context, term string) (Result, error) {
	if term != "" && !ecjtu.ValidTerm(term) {
		return Result{}, &ecjtu.Error{Kind: ecjtu.KindInvalidInput, Op: opExams, Cause: errors.New("invalid term")}
	}
	params := url.Values{}
	if term != "" {
		params.Set("term", term)
	}
	resp, err := q.t.Get(ctx, opExams, "ExamArrangeCl", params)
	if err != nil {
		return Result{}, err
	}
	kind, err := ecjtu.CheckPage(opExams, resp, ecjtu.TitleExam)
	if err != nil {
		return Result{}, err
	}
	ecjtu.Observe(ctx, ecjtu.Observation{Op: opExams, Status: resp.Status, Kind: kind,
		Attempts: resp.Attempts, Elapsed: resp.Elapsed})
	if kind == ecjtu.PageEmpty {
		return Result{Empty: true, Count: 0, Items: []ExamItem{}}, nil
	}
	return q.parseExams(resp.Body)
}
