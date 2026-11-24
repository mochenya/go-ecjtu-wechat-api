package classroom

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

const (
	opOptions = "classroom.Options"
	opSearch  = "classroom.Search"
)

// Query 是空教室域的查询入口，已装配不携带身份信息的抓取器。
type Query struct {
	t *ecjtu.Transport
}

// NewQuery 创建使用指定抓取器的空教室查询入口。
func NewQuery(t *ecjtu.Transport) *Query { return &Query{t: t} }

// Options 返回上游当前提供的学期、周次、星期、节次和表单默认值。
func (q *Query) Options(ctx context.Context) (Options, error) {
	resp, err := q.t.GetPublic(ctx, opOptions, "OtherQueryServlet", url.Values{"type": {"1"}})
	if err != nil {
		return Options{}, err
	}
	kind, err := ecjtu.CheckPage(opOptions, resp, ecjtu.TitleClassroomOption)
	if err != nil {
		return Options{}, err
	}
	ecjtu.Observe(ctx, ecjtu.Observation{Op: opOptions, Status: resp.Status, Kind: kind,
		Attempts: resp.Attempts, Elapsed: resp.Elapsed})
	return parseOptions(resp.Body)
}

// Search 查询指定周次的空闲教室；其他空条件取本次表单的有效默认值。
func (q *Query) Search(ctx context.Context, input Input) (Result, error) {
	if input.Week <= 0 {
		return Result{}, invalidInput("week is required")
	}
	if input.Term != "" && !ecjtu.ValidTerm(input.Term) {
		return Result{}, invalidInput("invalid term")
	}
	if input.Weekday != "" && !validWeekday(input.Weekday) {
		return Result{}, invalidInput("invalid weekday")
	}
	if input.Period != "" && !validPeriod(input.Period) {
		return Result{}, invalidInput("invalid period")
	}

	options, err := q.Options(ctx)
	if err != nil {
		return Result{}, err
	}
	resolved, err := resolveInput(input, options)
	if err != nil {
		return Result{}, err
	}
	form := url.Values{
		"term_xiala":      {resolved.Term},
		"weekSpan_xiala":  {strconv.Itoa(resolved.Week)},
		"week_xiala":      {string(resolved.Weekday)},
		"classSpan_xiala": {string(resolved.Period)},
	}
	resp, err := q.t.PostPublic(ctx, opSearch, "KongClassRoomQuery", form)
	if err != nil {
		return Result{}, err
	}
	kind, err := ecjtu.CheckPage(opSearch, resp, ecjtu.TitleClassroomResult)
	if err != nil {
		return Result{}, err
	}
	ecjtu.Observe(ctx, ecjtu.Observation{Op: opSearch, Status: resp.Status, Kind: kind,
		Attempts: resp.Attempts, Elapsed: resp.Elapsed})
	if kind == ecjtu.PageEmpty {
		return Result{Term: resolved.Term, Week: resolved.Week, Weekday: resolved.Weekday,
			Period: resolved.Period, Rooms: []Room{}}, nil
	}
	return parseResult(resp.Body, resolved)
}

func resolveInput(input Input, options Options) (Input, error) {
	resolved := input
	if resolved.Term == "" {
		resolved.Term = options.SelectedTerm
	}
	if resolved.Weekday == "" {
		resolved.Weekday = options.Defaults.Weekday
	}
	if resolved.Period == "" {
		resolved.Period = options.Defaults.Period
	}
	if !containsString(options.SelectableTerms, resolved.Term) {
		return Input{}, invalidInput("term is not selectable")
	}
	if !containsInt(options.Weeks, resolved.Week) {
		return Input{}, invalidInput("week is not selectable")
	}
	if !containsWeekday(options.Weekdays, resolved.Weekday) {
		return Input{}, invalidInput("weekday is not selectable")
	}
	if !containsPeriod(options.Periods, resolved.Period) {
		return Input{}, invalidInput("period is not selectable")
	}
	return resolved, nil
}

func invalidInput(message string) error {
	return &ecjtu.Error{Kind: ecjtu.KindInvalidInput, Op: opSearch, Cause: errors.New(message)}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsWeekday(values []Weekday, target Weekday) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsPeriod(values []Period, target Period) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
