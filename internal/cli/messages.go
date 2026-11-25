package cli

import (
	"errors"
	"fmt"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
)

// 用户可见文案集中在这里。约定：stdout 只放数据，以下句子一律走 stderr；
// 唯一例外是空结果提示（如「当天无课」），那是答案本身，走 stdout。
const (
	msgUsage                  = "用法：ecjtu [course|score|exam|classroom] [参数]，运行 ecjtu --help 查看全部命令。"
	msgNeedSubCmd             = "缺少子命令：要查课表运行 ecjtu course，成绩 ecjtu score，考试 ecjtu exam，空教室运行 ecjtu classroom，或运行 ecjtu --help 查看全部用法。"
	msgBadFormat              = "--format 只接受 json 或 text。"
	msgBadDate                = "--date 必须是 YYYY-MM-DD 格式，例如 2026-09-14。"
	msgBadTerm                = "--term 必须是 YYYY.N 格式，例如 2025.2。"
	msgBadClassroomWeek       = "--week 必须是正整数，并且必须指定；可先运行 ecjtu classroom options 查看可选周次。"
	msgBadClassroomWeekday    = "--weekday 必须是星期一至星期日之一。"
	msgBadClassroomPeriod     = "--period 必须是 1,2、3,4、5,6、7,8、9,10 或 11,12。"
	msgBadClassroomRoom       = "--room 不能为空白，例如 --room 31；不筛选请省略该参数。"
	msgBadTimeout             = "--timeout 必须是正的时间长度，例如 20s。"
	msgBadBaseURL             = "ECJTU_BASE_URL 必须是完整的 http(s) 地址，例如 https://jwxt.ecjtu.edu.cn/weixin。"
	msgBadTimeoutEn           = "ECJTU_TIMEOUT 必须是正的时间长度，例如 20s；也可用 --timeout 指定。"
	msgConfig                 = "教务凭证未配置：请在环境变量里设置 ECJTU_WEIXIN_TOKEN（教务绑定的微信 openid）。"
	msgNotBound               = "教务系统未绑定该微信账号，请在公众号里重新绑定。"
	msgTransport              = "无法连接教务系统，请检查网络后重试。"
	msgUpstream               = "教务系统暂时不可用（上游返回 %d），请稍后重试。"
	msgParse                  = "教务页面结构与预期不符，疑似上游改版，请稍后重试或更新解析器。"
	msgUnknown                = "命令参数或用法有误。"
	msgInvalidInput           = "参数不合法，请检查命令格式。"
	msgNoCourse               = "当天无课。"
	msgEmptyScore             = "该学期无成绩。"
	msgEmptyExam              = "该学期无考试安排。"
	msgEmptyClassroom         = "该条件下没有空闲教室。"
	msgEmptyClassroomFiltered = "该条件下没有教室名包含「%s」的空闲教室。"
)

// inputError 同时携带中文用户文案与英文原因：前者给用户，后者供 --verbose 排障；退出码 2。
type inputError struct {
	message string
	cause   error
}

func (e *inputError) Error() string {
	if e.cause == nil {
		return e.message
	}
	return e.cause.Error()
}

func (e *inputError) Unwrap() error { return e.cause }

// Message 返回中文用户文案。
func (e *inputError) Message() string { return e.message }

// InvalidInput 构造一个带中文用户文案和英文原因的用法/输入错误（退出码 2）。
func InvalidInput(message string, cause error) error {
	return &inputError{message: message, cause: cause}
}

// ConfigError 把 config.Load 的英文原因包成中文用户错误（退出码 2）。env 是 config.EnvOf 给出的
// 变量名，三种配置问题各有对应文案，不会把所有配置错误都念成「凭证未配置」。
//
// 这里写字面量而非 import internal/config：cli 不依赖配置层，变量名与文案的绑定由
// cmd/ecjtu 的 TestConfigMessagesBindEnvNames 防漂移。
func ConfigError(env string, cause error) error {
	switch env {
	case "ECJTU_BASE_URL":
		return &inputError{message: msgBadBaseURL, cause: cause}
	case "ECJTU_TIMEOUT":
		return &inputError{message: msgBadTimeoutEn, cause: cause}
	case "ECJTU_WEIXIN_TOKEN":
		return &inputError{message: msgConfig, cause: cause}
	default:
		return &inputError{message: msgConfig, cause: cause}
	}
}

// UserMessage 把 error 翻成一句中文用户文案。
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	var ie *inputError
	if errors.As(err, &ie) {
		return ie.Message()
	}
	var e *ecjtu.Error
	if errors.As(err, &e) {
		switch e.Kind {
		case ecjtu.KindNotBound:
			return msgNotBound
		case ecjtu.KindTransport:
			return msgTransport
		case ecjtu.KindUpstream:
			return fmt.Sprintf(msgUpstream, e.Status)
		case ecjtu.KindParse:
			return msgParse
		case ecjtu.KindInvalidInput:
			return msgInvalidInput
		}
	}
	return msgUnknown
}
