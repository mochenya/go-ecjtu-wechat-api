package score

import (
	"context"
	"errors"
	"net/url"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

// opScores 是错误归因用的操作名，形如「包名.方法名」，绝不参与拼 URL。
const opScores = "score.Scores"

// Query 是成绩域的查询入口，已装配好 transport。
type Query struct {
	t *ecjtu.Transport
}

// NewQuery 创建使用指定抓取器的成绩查询入口。
func NewQuery(t *ecjtu.Transport) *Query { return &Query{t: t} }

// Terms 从一次成绩查询中发现学期；空学期由上游选择默认值。
// 非空学期无效时返回输入错误，页面或学期形态异常时返回对应分类的错误。
func (q *Query) Terms(ctx context.Context, term string) (ecjtu.TermDiscovery, error) {
	result, err := q.Scores(ctx, term)
	if err != nil {
		return ecjtu.TermDiscovery{}, err
	}
	return ecjtu.DiscoverTerms(opScores, term, result)
}

// Scores 返回指定学期的成绩。term 为空串时不发送该参数，由上游落到它的默认学期
// （实测该默认学期滞后于校历学期）；本地不做学期猜测，也不为补 headinfo 加发请求。
func (q *Query) Scores(ctx context.Context, term string) (Result, error) {
	if term != "" && !ecjtu.ValidTerm(term) {
		return Result{}, &ecjtu.Error{Kind: ecjtu.KindInvalidInput, Op: opScores, Cause: errors.New("invalid term")}
	}
	params := url.Values{}
	if term != "" {
		params.Set("term", term)
	}
	resp, err := q.t.Get(ctx, opScores, "ScoreQuery", params)
	if err != nil {
		return Result{}, err
	}
	kind, err := ecjtu.CheckPage(opScores, resp, ecjtu.TitleScore)
	if err != nil {
		return Result{}, err
	}
	ecjtu.Observe(ctx, ecjtu.Observation{Op: opScores, Status: resp.Status, Kind: kind,
		Attempts: resp.Attempts, Elapsed: resp.Elapsed})
	if kind == ecjtu.PageEmpty {
		return emptyResult(), nil
	}
	return q.parseScores(resp.Body)
}

// emptyResult 是空态页的正常结果：empty:true、count:0、items:[]，页头三字段为 null，
// 退出码 0，不是错误。
func emptyResult() Result {
	return Result{
		Empty: true,
		Count: 0,
		Items: []ScoreItem{},
		// AvailableTerms 留 nil：空态页没有学期下拉，JSON 里是 null。
	}
}
