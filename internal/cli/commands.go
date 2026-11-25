package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	upstream "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/classroom"
	ecjtu "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/client"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/course"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/exam"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/score"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/render"
)

func (a *app) courseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "course",
		Short:   "查某一天的课表",
		Long:    "查询指定日期的课表，默认输出 JSON。需要设置环境变量 ECJTU_WEIXIN_TOKEN。",
		Example: "  ecjtu course\n  ecjtu course --date 2026-09-14 --format text",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// 输入校验先于装配：凭证缺失时也不要把「日期写错」报成「凭证没配」。
			date, err := a.resolveDate(a.fl.date)
			if err != nil {
				return err
			}
			return a.run(cmd.Context(), func(ctx context.Context, c *ecjtu.Query) error {
				schedule, err := c.Course(ctx, date)
				if err != nil {
					return err
				}
				return a.emitCourse(schedule)
			})
		},
	}
	cmd.Flags().StringVar(&a.fl.date, "date", "", "查询日期 YYYY-MM-DD（如 2026-09-14），缺省为今天（Asia/Shanghai）")
	a.addFormatFlag(cmd)
	return cmd
}

func (a *app) scoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "score",
		Short:   "查一个学期的成绩",
		Long:    "查询指定学期的成绩，默认输出 JSON。需要设置环境变量 ECJTU_WEIXIN_TOKEN。",
		Example: "  ecjtu score\n  ecjtu score --term 2025.1 --format text",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			term, err := a.resolveTerm(a.fl.term)
			if err != nil {
				return err
			}
			return a.run(cmd.Context(), func(ctx context.Context, c *ecjtu.Query) error {
				res, err := c.Scores(ctx, term)
				if err != nil {
					return err
				}
				a.warnCountMismatch("score", res.PageCount, res.Count)
				return a.emitScore(res)
			})
		},
	}
	cmd.Flags().StringVar(&a.fl.term, "term", "", "学期 YYYY.N（如 2025.2），缺省时由教务系统选择默认学期")
	a.addFormatFlag(cmd)
	return cmd
}

func (a *app) examCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exam",
		Short:   "查一个学期的考试安排",
		Long:    "查询指定学期的考试安排，默认输出 JSON。需要设置环境变量 ECJTU_WEIXIN_TOKEN。",
		Example: "  ecjtu exam\n  ecjtu exam --term 2025.1 --format text",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			term, err := a.resolveTerm(a.fl.term)
			if err != nil {
				return err
			}
			return a.run(cmd.Context(), func(ctx context.Context, c *ecjtu.Query) error {
				res, err := c.Exams(ctx, term)
				if err != nil {
					return err
				}
				a.warnCountMismatch("exam", res.PageCount, res.Count)
				return a.emitExam(res)
			})
		},
	}
	cmd.Flags().StringVar(&a.fl.term, "term", "", "学期 YYYY.N（如 2025.2），缺省时由教务系统选择默认学期")
	a.addFormatFlag(cmd)
	return cmd
}

func (a *app) classroomCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "classroom",
		Short:   "探测空教室选项，或查询空闲教室",
		Long:    "匿名查询空闲教室或探测可用表单选项，无需配置 ECJTU_WEIXIN_TOKEN。\n直接运行 ecjtu classroom 等同于 ecjtu classroom options。",
		Example: "  ecjtu classroom\n  ecjtu classroom search --week 3 --weekday 星期四 --period 1,2",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.runClassroomOptions(cmd.Context())
		},
	}
	cmd.PersistentFlags().StringVar(&a.fl.format, "format", "json", "输出格式：json 或 text")
	cmd.AddCommand(&cobra.Command{
		Use:     "options",
		Short:   "探测空教室查询选项",
		Long:    "探测上游空教室表单当前提供的学期、周次、星期、节次及默认值，无需配置 ECJTU_WEIXIN_TOKEN。",
		Example: "  ecjtu classroom options\n  ecjtu classroom options --format text",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.runClassroomOptions(cmd.Context())
		},
	})
	search := &cobra.Command{
		Use:     "search",
		Short:   "按学期、周次、星期和节次查询空闲教室",
		Long:    "按条件查询空闲教室，无需配置 ECJTU_WEIXIN_TOKEN。\n必须显式指定 --week，未指定的其余条件沿用本次表单默认值；可用 --room 按教室名子串筛选结果。",
		Example: "  ecjtu classroom search --week 3 --room 31 --format text\n  ecjtu classroom search --term 2026.1 --week 3 --weekday 星期四 --period 1,2",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, err := a.resolveClassroomInput()
			if err != nil {
				return err
			}
			roomFilter, err := a.resolveClassroomRoom(cmd.Flags().Changed("room"))
			if err != nil {
				return err
			}
			return a.runPublic(cmd.Context(), func(ctx context.Context, c *ecjtu.PublicQuery) error {
				result, err := c.Classrooms(ctx, input)
				if err != nil {
					return err
				}
				// 条数一致性告警比对上游页面声明数与解析条数，必须在本地筛选之前执行。
				a.warnCountMismatch("classroom", result.PageCount, result.Count)
				if roomFilter != "" {
					result = filterClassroomByName(result, roomFilter)
				}
				return a.emitClassroom(result, roomFilter)
			})
		},
	}
	search.Flags().StringVar(&a.fl.classroomTerm, "term", "", "学期 YYYY.N（如 2026.1），缺省使用表单选中的学期")
	search.Flags().IntVar(&a.fl.classroomWeek, "week", 0, "周次（必填，正整数），可先运行 ecjtu classroom options 查看可选范围")
	search.Flags().StringVar(&a.fl.classroomWeekday, "weekday", "", "星期一至星期日，缺省使用表单默认值")
	search.Flags().StringVar(&a.fl.classroomPeriod, "period", "", "连续两节（可选 1,2 / 3,4 / 5,6 / 7,8 / 9,10 / 11,12），缺省使用表单默认值")
	search.Flags().StringVar(&a.fl.classroomRoom, "room", "", "按教室名子串筛选（忽略大小写，如 31），缺省返回全部空闲教室")
	cmd.AddCommand(search)
	return cmd
}

func (a *app) runClassroomOptions(ctx context.Context) error {
	return a.runPublic(ctx, func(ctx context.Context, c *ecjtu.PublicQuery) error {
		options, err := c.ClassroomOptions(ctx)
		if err != nil {
			return err
		}
		if a.fl.format == "json" {
			return render.JSON(a.deps.Stdout, options)
		}
		return a.println(render.ClassroomOptions(options))
	})
}

func (a *app) resolveClassroomInput() (classroom.Input, error) {
	if a.fl.classroomWeek <= 0 {
		return classroom.Input{}, &inputError{message: msgBadClassroomWeek, cause: errors.New("week must be a positive integer")}
	}
	term, err := a.resolveTerm(a.fl.classroomTerm)
	if err != nil {
		return classroom.Input{}, err
	}
	weekday := classroom.Weekday(a.fl.classroomWeekday)
	if weekday != "" && !isClassroomWeekday(weekday) {
		return classroom.Input{}, &inputError{message: msgBadClassroomWeekday, cause: fmt.Errorf("weekday %q is invalid", weekday)}
	}
	period := classroom.Period(a.fl.classroomPeriod)
	if period != "" && !isClassroomPeriod(period) {
		return classroom.Input{}, &inputError{message: msgBadClassroomPeriod, cause: fmt.Errorf("period %q is invalid", period)}
	}
	return classroom.Input{Term: term, Week: a.fl.classroomWeek, Weekday: weekday, Period: period}, nil
}

func (a *app) resolveClassroomRoom(changed bool) (string, error) {
	if !changed {
		return "", nil
	}
	keyword := strings.TrimSpace(a.fl.classroomRoom)
	if keyword == "" {
		return "", &inputError{message: msgBadClassroomRoom, cause: errors.New("room filter must not be blank")}
	}
	return keyword, nil
}

func filterClassroomByName(r classroom.Result, keyword string) classroom.Result {
	needle := strings.ToLower(keyword)
	matched := make([]classroom.Room, 0, len(r.Rooms))
	for _, room := range r.Rooms {
		if strings.Contains(strings.ToLower(room.Name), needle) {
			matched = append(matched, room)
		}
	}
	r.Rooms = matched
	r.Count = len(matched)
	return r
}

func isClassroomWeekday(weekday classroom.Weekday) bool {
	switch weekday {
	case classroom.Monday, classroom.Tuesday, classroom.Wednesday, classroom.Thursday,
		classroom.Friday, classroom.Saturday, classroom.Sunday:
		return true
	default:
		return false
	}
}

func isClassroomPeriod(period classroom.Period) bool {
	switch period {
	case classroom.Period1To2, classroom.Period3To4, classroom.Period5To6,
		classroom.Period7To8, classroom.Period9To10, classroom.Period11To12:
		return true
	default:
		return false
	}
}

// addFormatFlag 只挂在查询子命令上，避免未来单二进制新增非查询子命令时误继承 --format。
func (a *app) addFormatFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&a.fl.format, "format", "json", "输出格式：json 或 text")
}

// resolveDate 校验并解析 --date：缺省值按 Asia/Shanghai 判定今天，校验不过就不发请求。
func (a *app) resolveDate(raw string) (time.Time, error) {
	t, err := upstream.ResolveDate(raw, a.now())
	if err != nil {
		// 剥出内层原因而非保留 *ecjtu.Error，使 --verbose 继续归为 kind=usage。
		cause := errors.Unwrap(err)
		if cause == nil {
			cause = err
		}
		return time.Time{}, &inputError{message: msgBadDate, cause: cause}
	}
	return t, nil
}

// resolveTerm 校验 --term；空串表示不发送该参数。
func (a *app) resolveTerm(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if !upstream.ValidTerm(raw) {
		return "", &inputError{message: msgBadTerm, cause: fmt.Errorf("term %q is not YYYY.N", raw)}
	}
	return raw, nil
}

// emitCourse / emitScore / emitExam / emitClassroom 决定 stdout 的形状：json 走规范编码，
// text 走固定列集表格；空结果的句子在这里出，属答案，所以走 stdout。
func (a *app) emitCourse(s course.Schedule) error {
	if a.fl.format == "json" {
		return render.JSON(a.deps.Stdout, s)
	}
	if len(s.Courses) == 0 {
		return a.println(msgNoCourse)
	}
	return a.println(render.Course(s))
}

func (a *app) emitScore(r score.Result) error {
	if a.fl.format == "json" {
		return render.JSON(a.deps.Stdout, r)
	}
	if r.Empty || len(r.Items) == 0 {
		return a.println(msgEmptyScore)
	}
	return a.println(render.Score(r))
}

func (a *app) emitExam(r exam.Result) error {
	if a.fl.format == "json" {
		return render.JSON(a.deps.Stdout, r)
	}
	if r.Empty || len(r.Items) == 0 {
		return a.println(msgEmptyExam)
	}
	return a.println(render.Exam(r))
}

func (a *app) emitClassroom(r classroom.Result, roomFilter string) error {
	if a.fl.format == "json" {
		return render.JSON(a.deps.Stdout, r)
	}
	if len(r.Rooms) == 0 {
		if roomFilter != "" {
			return a.println(fmt.Sprintf(msgEmptyClassroomFiltered, roomFilter))
		}
		return a.println(msgEmptyClassroom)
	}
	return a.println(render.Classrooms(r, roomFilter))
}

// warnCountMismatch 在 --verbose 下比对页面自带条数与解析条数；不一致只警告，不升级为错误。
func (a *app) warnCountMismatch(domain string, declared *int, parsed int) {
	if !a.fl.verbose || declared == nil || *declared == parsed {
		return
	}
	fmt.Fprintf(a.deps.Stderr, "warn=count-mismatch domain=%s declared=%d parsed=%d\n",
		domain, *declared, parsed)
}

func (a *app) println(text string) error {
	_, err := fmt.Fprintln(a.deps.Stdout, text)
	return err
}
