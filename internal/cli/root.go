// Package cli 提供教务查询命令、用户文案和诊断输出。
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	upstream "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	ecjtu "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/client"
)

// Assembly 是一次运行装配出来的客户端与其时间预算。装配由 cmd 侧提供（读 config、建 transport
// 是 main 的职责），CLI 只在真要发请求时调用，这样 --help 与 --version 在缺凭证时也能工作。
type Assembly struct {
	Client       *ecjtu.Query
	PublicClient *ecjtu.PublicQuery
	Timeout      time.Duration // 整个命令的 ctx 总预算；<=0 表示不设预算
}

// Assembler 惰性装配客户端。返回的错误必须已带中文文案分类（见 ConfigError）。
type Assembler func() (Assembly, error)

// PublicAssembler 惰性装配不需要 OpenID 的公开查询入口。
type PublicAssembler func() (Assembly, error)

// Deps 是一次运行的注入面；不持有 context。
type Deps struct {
	Assemble       Assembler
	AssemblePublic PublicAssembler
	Stdout         io.Writer
	Stderr         io.Writer
	BuildInfo      BuildInfo
	Now            func() time.Time // 「今天」的来源；测试注入固定时钟，不做真实等待
}

type flags struct {
	format           string
	timeout          time.Duration
	verbose          bool
	date             string
	term             string
	classroomTerm    string
	classroomWeek    int
	classroomWeekday string
	classroomPeriod  string
}

type app struct {
	deps Deps
	root *cobra.Command
	fl   flags
}

func (a *app) now() time.Time {
	if a.deps.Now != nil {
		return a.deps.Now()
	}
	return time.Now()
}

func newApp(d Deps) *app {
	a := &app{deps: d, fl: flags{format: "json"}}

	root := &cobra.Command{
		Use:   "ecjtu",
		Short: "抓取华东交通大学教务系统（微信公众号版）数据",
		Long: strings.TrimSpace(`
抓取华东交通大学教务系统（微信公众号版）数据，默认输出 JSON（stdout），诊断与错误写入 stderr。

环境变量：
  ECJTU_WEIXIN_TOKEN  教务绑定的微信 OpenID（course/score/exam 必填；classroom 无需配置）
  ECJTU_BASE_URL      上游服务基地址（可选，默认学校教务地址）
  ECJTU_TIMEOUT       整个命令的时间预算，如 15s（可选，默认 15s）

退出码：
  0 成功（含空结果）  2 参数或配置错误  3 未绑定  4 上游不可用  5 页面解析失败
`),
		// 打印全部由本包接管：SilenceUsage/SilenceErrors 让 cobra 不再自己输出。
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       d.BuildInfo.Version,
		Args:          cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			return &inputError{message: msgNeedSubCmd}
		},
	}
	pf := root.PersistentFlags()
	pf.DurationVar(&a.fl.timeout, "timeout", 0, "整个命令的时间预算，如 20s；覆盖 ECJTU_TIMEOUT（默认 15s），重试共享该预算")
	pf.BoolVarP(&a.fl.verbose, "verbose", "v", false, "把诊断写到 stderr，不影响 stdout 的数据")

	root.AddCommand(a.courseCommand(), a.scoreCommand(), a.examCommand(), a.classroomCommand())
	root.SetOut(d.Stdout)
	root.SetErr(d.Stderr)
	root.SetVersionTemplate(d.BuildInfo.String())
	a.root = root
	return a
}

// New 暴露根命令，供测试与未来的补全子命令复用。
func New(d Deps) *cobra.Command { return newApp(d).root }

// Execute 解析 args 并执行命令，返回进程退出码。
func Execute(ctx context.Context, d Deps, args []string) int {
	a := newApp(d)
	a.root.SetArgs(args)
	if err := a.root.ExecuteContext(ctx); err != nil {
		return a.report(err)
	}
	return ExitOK
}

// run 是子命令的公共前置：校验 --format、装配客户端、套时间预算、按需挂 --verbose 观察者。
// 命令自己的 --date/--term 校验在调用本函数之前完成。
func (a *app) run(ctx context.Context, run func(ctx context.Context, c *ecjtu.Query) error) error {
	if err := a.checkFormat(); err != nil {
		return err
	}
	assembly, err := a.assemble()
	if err != nil {
		return err
	}
	ctx, cancel, err := a.budget(ctx, assembly.Timeout)
	if err != nil {
		return err
	}
	defer cancel()
	if a.fl.verbose {
		ctx = upstream.WithObserver(ctx, a.observe)
	}
	return run(ctx, assembly.Client)
}

func (a *app) runPublic(ctx context.Context, run func(ctx context.Context, c *ecjtu.PublicQuery) error) error {
	if err := a.checkFormat(); err != nil {
		return err
	}
	if a.deps.AssemblePublic == nil {
		return &inputError{message: msgConfig, cause: errors.New("no public assembler configured")}
	}
	assembly, err := a.deps.AssemblePublic()
	if err != nil {
		return err
	}
	ctx, cancel, err := a.budget(ctx, assembly.Timeout)
	if err != nil {
		return err
	}
	defer cancel()
	if a.fl.verbose {
		ctx = upstream.WithObserver(ctx, a.observe)
	}
	return run(ctx, assembly.PublicClient)
}

func (a *app) assemble() (Assembly, error) {
	if a.deps.Assemble == nil {
		return Assembly{}, &inputError{message: msgConfig, cause: errors.New("no assembler configured")}
	}
	return a.deps.Assemble()
}

// checkFormat 校验 --format，未知值退出码 2。
func (a *app) checkFormat() error {
	switch a.fl.format {
	case "json", "text":
		return nil
	default:
		return &inputError{message: msgBadFormat, cause: fmt.Errorf("format %q is not json or text", a.fl.format)}
	}
}

// budget 给出整个命令的 ctx 总预算：--timeout 覆盖配置值，重试共享同一预算。
func (a *app) budget(ctx context.Context, configured time.Duration) (context.Context, context.CancelFunc, error) {
	timeout := configured
	if a.fl.timeout > 0 {
		timeout = a.fl.timeout
	} else if a.fl.timeout < 0 {
		return nil, func() {}, &inputError{message: msgBadTimeout, cause: errors.New("timeout must be positive")}
	}
	if timeout <= 0 {
		return ctx, func() {}, nil
	}
	cancelled, cancel := context.WithTimeout(ctx, timeout)
	return cancelled, cancel, nil
}

// observe 把一次抓取的诊断写成一行 k=v：只进 stderr，stdout 永远是数据。
// domain 取 op 的包名前缀，便于按域聚合日志。
func (a *app) observe(o upstream.Observation) {
	domain, _, _ := strings.Cut(o.Op, ".")
	fmt.Fprintf(a.deps.Stderr, "phase=fetch domain=%s op=%s status=%d kind=%s attempts=%d ms=%d\n",
		domain, o.Op, o.Status, o.Kind, o.Attempts, o.Elapsed.Milliseconds())
}

// report 打印中文文案并给出退出码；--verbose 时追加一行可机读的英文原因。
func (a *app) report(err error) int {
	fmt.Fprintln(a.deps.Stderr, UserMessage(err))
	var ie *inputError
	_, classified := upstream.KindOf(err)
	if !classified && !errors.As(err, &ie) {
		// 只可能是 cobra/pflag 的用法错误：英文原因对用户有用（"unknown flag: --nope"），
		// 一并给出，再补一句用法提示。
		fmt.Fprintln(a.deps.Stderr, err.Error())
		fmt.Fprintln(a.deps.Stderr, msgUsage)
	}
	if a.fl.verbose {
		fmt.Fprintf(a.deps.Stderr, "phase=error kind=%s err=%q\n", kindLabel(err), err.Error())
	}
	return ExitCode(err)
}
