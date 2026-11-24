package classroom

import (
	"reflect"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

func TestParseOptionsUsesSelectedAndBrowserDefaults(t *testing.T) {
	got, err := parseOptions(testsupport.Fixture(t, "classroom-options-synthetic.html"))
	if err != nil {
		t.Fatal(err)
	}
	want := Options{
		SelectableTerms: []string{"2026.2", "2026.1", "2025.2"},
		SelectedTerm:    "2026.1",
		Weeks:           []int{1, 2, 3},
		Weekdays:        []Weekday{Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday},
		Periods:         []Period{Period1To2, Period3To4, Period5To6},
		Defaults:        Defaults{Week: 1, Weekday: Thursday, Period: Period1To2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("options = %#v, want %#v", got, want)
	}
}

func TestParseResultKeepsInputAndPageCount(t *testing.T) {
	input := Input{Term: "2026.1", Week: 3, Weekday: Thursday, Period: Period1To2}
	got, err := parseResult(testsupport.Fixture(t, "classroom-results-synthetic.html"), input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Term != input.Term || got.Week != input.Week || got.Weekday != input.Weekday || got.Period != input.Period {
		t.Fatalf("result input = %#v, want %#v", got, input)
	}
	if got.Count != 2 || len(got.Rooms) != 2 || got.Rooms[0].Name != "教一-101" || got.Rooms[1].SeatCount != 80 {
		t.Fatalf("rooms = %#v", got.Rooms)
	}
	if got.PageCount == nil || *got.PageCount != 2 {
		t.Fatalf("page count = %v", got.PageCount)
	}
}

func TestParseOptionsRejectsInvalidForm(t *testing.T) {
	_, err := parseOptions([]byte(`<html><head><title>空闲教室查询</title></head><body></body></html>`))
	if !ecjtu.IsKind(err, ecjtu.KindParse) {
		t.Fatalf("error = %v, want parse", err)
	}
}

func TestParseResultAllowsMissingPageCount(t *testing.T) {
	body := []byte(`<html><head><title>空闲教室使用情况</title></head><body><div class="tb"><table><tr><th>教室</th><th>教室类型</th><th>座位数</th></tr></table></div></body></html>`)
	got, err := parseResult(body, Input{Term: "2026.1", Week: 1, Weekday: Monday, Period: Period1To2})
	if err != nil {
		t.Fatal(err)
	}
	if got.Rooms == nil || got.Count != 0 || got.PageCount != nil {
		t.Fatalf("result = %#v", got)
	}
}
