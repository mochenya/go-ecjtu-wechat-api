package classroom

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

func parseOptions(body []byte) (Options, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Options{}, parseError("options html is malformed")
	}
	form := doc.Find("form").First()
	method, _ := form.Attr("method")
	action, _ := form.Attr("action")
	if form.Length() == 0 || !strings.EqualFold(method, "post") || action != "/weixin/KongClassRoomQuery" {
		return Options{}, parseError("options form does not match")
	}

	terms, selectedTerm, err := parseTerms(form)
	if err != nil {
		return Options{}, err
	}
	weeks, defaultWeek, err := parseWeeks(form)
	if err != nil {
		return Options{}, err
	}
	weekdays, defaultWeekday, err := parseWeekdays(form)
	if err != nil {
		return Options{}, err
	}
	periods, defaultPeriod, err := parsePeriods(form)
	if err != nil {
		return Options{}, err
	}
	return Options{
		SelectableTerms: terms,
		SelectedTerm:    selectedTerm,
		Weeks:           weeks,
		Weekdays:        weekdays,
		Periods:         periods,
		Defaults:        Defaults{Week: defaultWeek, Weekday: defaultWeekday, Period: defaultPeriod},
	}, nil
}

func parseTerms(form *goquery.Selection) ([]string, string, error) {
	options, err := selectOptions(form, "term_xiala")
	if err != nil {
		return nil, "", err
	}
	terms := make([]string, 0, len(options))
	selected := ""
	for _, option := range options {
		if !ecjtu.ValidTerm(option.value) {
			return nil, "", parseError("term option is invalid")
		}
		terms = append(terms, option.value)
		if selected == "" && option.selected {
			selected = option.value
		}
	}
	if selected == "" {
		selected = terms[0]
	}
	return terms, selected, nil
}

func parseWeeks(form *goquery.Selection) ([]int, int, error) {
	options, err := selectOptions(form, "weekSpan_xiala")
	if err != nil {
		return nil, 0, err
	}
	weeks := make([]int, 0, len(options))
	defaultWeek := 0
	for _, option := range options {
		week, err := strconv.Atoi(option.value)
		if err != nil || week <= 0 {
			return nil, 0, parseError("week option is invalid")
		}
		weeks = append(weeks, week)
		if defaultWeek == 0 && option.selected {
			defaultWeek = week
		}
	}
	if defaultWeek == 0 {
		defaultWeek = weeks[0]
	}
	return weeks, defaultWeek, nil
}

func parseWeekdays(form *goquery.Selection) ([]Weekday, Weekday, error) {
	options, err := selectOptions(form, "week_xiala")
	if err != nil {
		return nil, "", err
	}
	weekdays := make([]Weekday, 0, len(options))
	defaultWeekday := Weekday("")
	for _, option := range options {
		weekday := Weekday(option.value)
		if !validWeekday(weekday) {
			return nil, "", parseError("weekday option is invalid")
		}
		weekdays = append(weekdays, weekday)
		if defaultWeekday == "" && option.selected {
			defaultWeekday = weekday
		}
	}
	if defaultWeekday == "" {
		defaultWeekday = weekdays[0]
	}
	return weekdays, defaultWeekday, nil
}

func parsePeriods(form *goquery.Selection) ([]Period, Period, error) {
	options, err := selectOptions(form, "classSpan_xiala")
	if err != nil {
		return nil, "", err
	}
	periods := make([]Period, 0, len(options))
	defaultPeriod := Period("")
	for _, option := range options {
		period := Period(option.value)
		if !validPeriod(period) {
			return nil, "", parseError("period option is invalid")
		}
		periods = append(periods, period)
		if defaultPeriod == "" && option.selected {
			defaultPeriod = period
		}
	}
	if defaultPeriod == "" {
		defaultPeriod = periods[0]
	}
	return periods, defaultPeriod, nil
}

type optionValue struct {
	value    string
	selected bool
}

func selectOptions(form *goquery.Selection, name string) ([]optionValue, error) {
	selectElement := form.Find("select[name='" + name + "']")
	if selectElement.Length() != 1 {
		return nil, parseError("select %s is missing or duplicated", name)
	}
	var options []optionValue
	selectElement.Find("option").Each(func(_ int, option *goquery.Selection) {
		text := ecjtu.Collapse(option.Text())
		value, hasValue := option.Attr("value")
		if !hasValue {
			value = text
		}
		value = ecjtu.Collapse(value)
		if value == "" {
			return
		}
		_, selected := option.Attr("selected")
		options = append(options, optionValue{value: value, selected: selected})
	})
	if len(options) == 0 {
		return nil, parseError("select %s has no options", name)
	}
	return options, nil
}

func parseResult(body []byte, input Input) (Result, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return Result{}, resultParseError("result html is malformed")
	}
	table := doc.Find("div.tb table")
	if table.Length() != 1 {
		return Result{}, resultParseError("result table is missing or duplicated")
	}
	rows := table.Find("tr")
	if rows.Length() == 0 {
		return Result{}, resultParseError("result table has no header")
	}
	var headers []string
	rows.First().Find("th").Each(func(_ int, cell *goquery.Selection) {
		headers = append(headers, ecjtu.Collapse(cell.Text()))
	})
	if len(headers) != 3 || headers[0] != "教室" || headers[1] != "教室类型" || headers[2] != "座位数" {
		return Result{}, resultParseError("result table headers do not match")
	}

	rooms := make([]Room, 0, rows.Length()-1)
	var rowErr error
	rows.Slice(1, rows.Length()).Each(func(index int, row *goquery.Selection) {
		if rowErr != nil {
			return
		}
		cells := row.Find("td")
		if cells.Length() == 0 {
			return
		}
		if cells.Length() != 3 {
			rowErr = resultParseError("result row %d has an unexpected column count", index+1)
			return
		}
		name := ecjtu.Collapse(cells.Eq(0).Text())
		roomType := ecjtu.Collapse(cells.Eq(1).Text())
		seatCount, err := strconv.Atoi(ecjtu.Collapse(cells.Eq(2).Text()))
		if name == "" || roomType == "" || err != nil || seatCount < 0 {
			rowErr = resultParseError("result row %d contains an invalid field", index+1)
			return
		}
		rooms = append(rooms, Room{Name: name, Type: roomType, SeatCount: seatCount})
	})
	if rowErr != nil {
		return Result{}, rowErr
	}

	result := Result{
		Term: input.Term, Week: input.Week, Weekday: input.Weekday, Period: input.Period,
		Count: len(rooms), Rooms: rooms,
	}
	mark := doc.Find("div.text mark").First()
	if mark.Length() > 0 {
		count, err := strconv.Atoi(ecjtu.Collapse(mark.Text()))
		if err != nil || count < 0 {
			return Result{}, resultParseError("result page count is invalid")
		}
		result.PageCount = &count
	}
	return result, nil
}

func validWeekday(weekday Weekday) bool {
	switch weekday {
	case Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday:
		return true
	default:
		return false
	}
}

func validPeriod(period Period) bool {
	switch period {
	case Period1To2, Period3To4, Period5To6, Period7To8, Period9To10, Period11To12:
		return true
	default:
		return false
	}
}

func parseError(format string, args ...any) error {
	return &ecjtu.Error{Kind: ecjtu.KindParse, Op: opOptions, Cause: fmt.Errorf(format, args...)}
}

func resultParseError(format string, args ...any) error {
	return &ecjtu.Error{Kind: ecjtu.KindParse, Op: opSearch, Cause: fmt.Errorf(format, args...)}
}
