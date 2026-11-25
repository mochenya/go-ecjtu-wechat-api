package render

import (
	"strings"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/classroom"
)

func TestClassroomOptionsText(t *testing.T) {
	got := ClassroomOptions(classroom.Options{
		SelectableTerms: []string{"2026.2", "2026.1"},
		SelectedTerm:    "2026.1",
		Weeks:           []int{1, 2, 3},
		Weekdays:        []classroom.Weekday{classroom.Monday, classroom.Thursday},
		Periods:         []classroom.Period{classroom.Period1To2, classroom.Period3To4},
		Defaults:        classroom.Defaults{Week: 1, Weekday: classroom.Thursday, Period: classroom.Period1To2},
	})
	for _, want := range []string{"可选学期：2026.2, 2026.1", "可选周次：1, 2, 3", "表单默认值：第1周，星期四，1,2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output 缺少 %q:\n%s", want, got)
		}
	}
}

func TestClassroomsText(t *testing.T) {
	got := Classrooms(classroom.Result{
		Term:    "2026.1",
		Week:    3,
		Weekday: classroom.Thursday,
		Period:  classroom.Period1To2,
		Count:   2,
		Rooms: []classroom.Room{
			{Name: "教一-101", Type: "普通教室", SeatCount: 120},
			{Name: "教二-203", Type: "多媒体教室", SeatCount: 80},
		},
	}, "")
	for _, want := range []string{"2026.1 第3周 星期四 1,2（共 2 间）", "教室", "教一-101", "多媒体教室", "120"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output 缺少 %q:\n%s", want, got)
		}
	}
}

func TestClassroomsTextWithRoomFilter(t *testing.T) {
	got := Classrooms(classroom.Result{
		Term:    "2026.1",
		Week:    3,
		Weekday: classroom.Thursday,
		Period:  classroom.Period1To2,
		Count:   1,
		Rooms: []classroom.Room{
			{Name: "31-422", Type: "多媒体教室", SeatCount: 90},
		},
	}, "31")
	for _, want := range []string{"2026.1 第3周 星期四 1,2（教室含 31，共 1 间）", "31-422", "多媒体教室", "90"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output 缺少 %q:\n%s", want, got)
		}
	}
}
