package ecjtu

import (
	"fmt"
	"time"
)

// Shanghai 返回教务系统所在的东八区时区。
func Shanghai() *time.Location {
	// Cgo 禁用或极简容器缺少 tzdata 时，退回固定 +08:00，保证缺省日期仍可用。
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}

// ResolveDate 解析 YYYY-MM-DD 格式的课表日期；raw 为空时取 now 在东八区的当天，
// now 为零值时回退当前时间。格式或日历日非法时返回 KindInvalidInput。
func ResolveDate(raw string, now time.Time) (time.Time, error) {
	if raw == "" {
		if now.IsZero() {
			now = time.Now()
		}
		return now.In(Shanghai()), nil
	}
	t, err := time.Parse("2006-01-02", raw)
	// 回环格式化用于拦截格式变体，保证传给上游的一定是标准 YYYY-MM-DD。
	if err != nil || t.Format("2006-01-02") != raw {
		return time.Time{}, &Error{
			Kind:  KindInvalidInput,
			Op:    "ecjtu.ResolveDate",
			Cause: fmt.Errorf("date %q is not YYYY-MM-DD", raw),
		}
	}
	return t, nil
}
