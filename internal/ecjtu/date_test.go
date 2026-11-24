package ecjtu

import (
	"testing"
	"time"
)

func TestResolveDate(t *testing.T) {
	// UTC 20:00 跨进东八区次日凌晨 04:00，用于验证默认「今天」按东八区换日。
	utcEvening := time.Date(2026, 9, 14, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		raw      string
		now      time.Time
		wantDate string
		wantErr  bool
	}{
		{"空串按东八区取今天", "", utcEvening, "2026-09-15", false},
		{"标准YYYY-MM-DD", "2026-09-14", utcEvening, "2026-09-14", false},
		{"缺少前导零", "2026-9-14", utcEvening, "", true},
		{"不存在的日历日", "2026-02-30", utcEvening, "", true},
		{"带尾随空白", "2026-09-14 ", utcEvening, "", true},
		{"非日期文本", "today", utcEvening, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveDate(tt.raw, tt.now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际返回 %v", got)
				}
				if kind, ok := KindOf(err); !ok || kind != KindInvalidInput {
					t.Fatalf("错误分类 = %v (ok=%v)，期望 invalidinput", kind, ok)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外报错: %v", err)
			}
			if got.Format("2006-01-02") != tt.wantDate {
				t.Fatalf("日期 = %s，期望 %s", got.Format("2006-01-02"), tt.wantDate)
			}
		})
	}
}

func TestResolveDateZeroNowUsesShanghaiFallback(t *testing.T) {
	got, err := ResolveDate("", time.Time{})
	if err != nil {
		t.Fatalf("零值 now 回退失败: %v", err)
	}
	if got.IsZero() {
		t.Fatal("返回了零值时间")
	}
	_, offset := got.Zone()
	if offset != 8*3600 {
		t.Fatalf("时区偏移 = %d，期望 +28800", offset)
	}
}
