// Package ecjtu 是教务微信接口的上游共同层：抓取、页面判定、错误分类与脱敏。
package ecjtu

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Kind 是查询失败的分类。
type Kind int

const (
	// KindTransport 指请求构造、连接、超时或读取响应体等抓取错误。
	KindTransport Kind = iota
	// KindUpstream 指上游返回非 2xx（含 Tomcat 异常页），异常正文一律丢弃。
	KindUpstream
	// KindNotBound 指凭证失效：上游返回绑定表单页，需要人工重新绑定。
	KindNotBound
	// KindParse 指页面能归类但结构不符预期，疑似上游改版。
	KindParse
	// KindInvalidInput 指本地校验未通过，不发请求。
	KindInvalidInput
)

// String 返回失败分类的名称。
func (k Kind) String() string {
	switch k {
	case KindTransport:
		return "transport"
	case KindUpstream:
		return "upstream"
	case KindNotBound:
		return "notbound"
	case KindParse:
		return "parse"
	case KindInvalidInput:
		return "invalidinput"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

// Error 聚合失败分类与操作名。Op 只允许「包名.方法名」，绝不含 URL 与 query；
// Cause 在进入本类型前必须已经过 Sanitize。
type Error struct {
	Kind   Kind
	Op     string
	Status int   // 仅 KindUpstream 有意义
	Cause  error // 可为 nil
}

// Error 返回操作名与错误原因。
func (e *Error) Error() string {
	switch {
	case e.Kind == KindUpstream && e.Status != 0:
		return fmt.Sprintf("%s: upstream returned %d", e.Op, e.Status)
	case e.Cause != nil:
		return e.Op + ": " + e.Cause.Error()
	default:
		return e.Op + ": " + e.Kind.String()
	}
}

func (e *Error) Unwrap() error { return e.Cause }

// KindOf 提取错误链上第一个 ecjtu.Error 的分类；无分类时 ok 为 false。
func KindOf(err error) (Kind, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target.Kind, true
	}
	return 0, false
}

// MaskOpenID 把 s 中出现的 openid 替换为前 4 位加 ***；openid 不足 5 位时整体替换，
// 避免反而暴露更多字符。
func MaskOpenID(s, openid string) string {
	if openid == "" {
		return s
	}
	mask := "***"
	if len(openid) >= 5 {
		mask = openid[:4] + "***"
	}
	return strings.ReplaceAll(s, openid, mask)
}

// sessionIDRe 覆盖 jsessionid 的两种实测形态：URL 内嵌（;jsessionid=...）与响应头（JSESSIONID=...）。
var sessionIDRe = regexp.MustCompile(`(?i)jsessionid[=:][0-9a-zA-Z\-]+`)

// DropSessionIDs 把文本中的 jsessionid 值替换为 ***。
func DropSessionIDs(s string) string {
	return sessionIDRe.ReplaceAllString(s, "jsessionid=***")
}

// Sanitize 组合 openid 与 jsessionid 两类脱敏，是脱敏的统一出口。
func Sanitize(s, openid string) string {
	return DropSessionIDs(MaskOpenID(s, openid))
}

// IsKind 报告错误链中的分类是否为 want。
func IsKind(err error, want Kind) bool {
	k, ok := KindOf(err)
	return ok && k == want
}
