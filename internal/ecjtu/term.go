package ecjtu

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
)

// termRe 匹配学期参数 `YYYY.N` 的形态。上游对两段都做 Integer.parseInt，
// 非法形态直接 500（实测 term=bogus），因此必须在本地拦截。
var termRe = regexp.MustCompile(`^[0-9]{4}\.[0-9]+$`)

// ValidTerm 报告 s 能否作为学期参数发出；空串表示不发送该参数，由调用方处理。
func ValidTerm(s string) bool { return termRe.MatchString(s) }

// TermProvider 从一个数据域的单次查询中发现学期；空学期由上游选择默认值。
type TermProvider interface {
	Terms(ctx context.Context, term string) (TermDiscovery, error)
}

// TermDiscovery 区分本次页面列出的候选与本次确认有数据的学期，候选不保证完整。
// Candidates 为 nil 表示页面未提供学期列表；Available 是本次已确认有数据的学期，恒非 nil。
// CheckedTerm 优先取页面回显；空态页没有回显时取显式请求参数，均不存在时为 nil。
type TermDiscovery struct {
	Candidates  []string `json:"candidates"`
	Available   []string `json:"available"`
	CheckedTerm *string  `json:"checkedTerm"`
}

// DiscoverTerms 根据一次成功查询的结果整理学期，不发请求；页面中的学期形态不符时返回解析错误。
// op 是错误归因的操作名，term 必须与产生 result 的查询参数一致。
func DiscoverTerms[T any](op, term string, result Result[T]) (TermDiscovery, error) {
	for _, candidate := range result.AvailableTerms {
		if !ValidTerm(candidate) {
			return TermDiscovery{}, &Error{Kind: KindParse, Op: op, Cause: errors.New("invalid term in page")}
		}
	}
	if result.Term != nil && !ValidTerm(*result.Term) {
		return TermDiscovery{}, &Error{Kind: KindParse, Op: op, Cause: errors.New("invalid term in page")}
	}
	discovery := TermDiscovery{
		Candidates: slices.Clone(result.AvailableTerms),
		Available:  []string{},
	}
	slices.SortStableFunc(discovery.Candidates, compareTerms)
	if result.Term != nil {
		discovery.CheckedTerm = result.Term
	} else if result.Empty && term != "" {
		discovery.CheckedTerm = &term
	}
	if discovery.CheckedTerm != nil && len(result.Items) > 0 {
		discovery.Available = append(discovery.Available, *discovery.CheckedTerm)
	}
	return discovery, nil
}

func compareTerms(a, b string) int {
	ay, ap, _ := strings.Cut(a, ".")
	by, bp, _ := strings.Cut(b, ".")
	if ay != by {
		return strings.Compare(by, ay)
	}
	ap = strings.TrimLeft(ap, "0")
	bp = strings.TrimLeft(bp, "0")
	if len(ap) != len(bp) {
		return len(bp) - len(ap)
	}
	return strings.Compare(bp, ap)
}
