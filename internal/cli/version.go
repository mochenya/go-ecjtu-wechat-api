package cli

import "fmt"

// BuildInfo 描述 CLI 的发布版本和构建环境。
type BuildInfo struct {
	Version   string
	BuildTime string
	GoVersion string
	OS        string
	Arch      string
	Compiler  string
}

// String 返回 --version 使用的人类可读构建信息。
func (b BuildInfo) String() string {
	return fmt.Sprintf(
		"ecjtu %s\n编译时间 (UTC): %s\nGo 版本: %s\n平台: %s/%s\n编译器: %s\n",
		valueOrUnknown(b.Version),
		valueOrUnknown(b.BuildTime),
		valueOrUnknown(b.GoVersion),
		valueOrUnknown(b.OS),
		valueOrUnknown(b.Arch),
		valueOrUnknown(b.Compiler),
	)
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
