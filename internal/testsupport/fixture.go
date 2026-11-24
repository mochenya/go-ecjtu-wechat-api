// Package testsupport 提供测试页面读取。
package testsupport

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Root 从本文件位置向上找 go.mod，返回仓库根目录。
func Root(tb *testing.T) string {
	tb.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatalf("无法定位 testsupport 文件路径")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatalf("未找到 go.mod")
		}
		dir = parent
	}
}

// Fixture 读取 testdata/pages 下的脱敏 fixture。
func Fixture(tb *testing.T, name string) []byte {
	tb.Helper()
	b, err := os.ReadFile(filepath.Join(Root(tb), "testdata", "pages", name))
	if err != nil {
		tb.Fatalf("读取 fixture %s 失败: %v", name, err)
	}
	return b
}
