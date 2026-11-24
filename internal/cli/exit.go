package cli

import (
	"errors"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

const (
	ExitOK       = 0 // 成功，含空结果
	ExitUsage    = 2 // 用法或输入错误
	ExitNotBound = 3 // 未绑定 / 凭证失效
	ExitUpstream = 4 // 上游不可用（非 2xx、超时、网络）
	ExitParse    = 5 // 解析失败，疑上游改版
)

// ExitCode 把错误映射到退出码；未分类的错误按用法错误处理。
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ie *inputError
	if errors.As(err, &ie) {
		return ExitUsage
	}
	var e *ecjtu.Error
	if errors.As(err, &e) {
		switch e.Kind {
		case ecjtu.KindInvalidInput:
			return ExitUsage
		case ecjtu.KindNotBound:
			return ExitNotBound
		case ecjtu.KindTransport, ecjtu.KindUpstream:
			return ExitUpstream
		case ecjtu.KindParse:
			return ExitParse
		}
	}
	return ExitUsage
}

// kindLabel 给出错误分类的英文标签，供 --verbose 的诊断行使用；未分类的统一报 "usage"。
func kindLabel(err error) string {
	if k, ok := ecjtu.KindOf(err); ok {
		return k.String()
	}
	return "usage"
}
