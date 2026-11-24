# Go 规范落地

基准是 [Google Go Style Guide](https://google.github.io/styleguide/go/guide)，加上本仓的具体选择。工具链只用 Go 自带：`gofmt`、`go vet`、`go test`。不引入 golangci-lint、gofumpt、staticcheck、testify。

## 四层语言策略

| 层 | 语言 | 例 |
|---|---|---|
| doc comment | 中文，但必须以标识符名开头 | `// Day 返回指定日期的课表。日期按 Asia/Shanghai 判定今天。` |
| 行内注释 | 中文 | `// 上游把 openid 拼成 weiXinID，是它的历史拼写错误。` |
| error string | 英文小写，不加句号分号 | `fmt.Errorf("course: parse day row: unexpected label %q", label)` |
| CLI 用户文案 | 中文，集中在 `internal/cli/messages.go` | `教务系统未绑定该微信账号，请在公众号里重新绑定。` |

类型与变量名一律英文，禁止拼音（如 `kebiao`、`chengji`）。

## 注释

注释应补充代码和命名不能直接表达的信息。

要写：

- 每个包的包注释恰好一处，挂主文件的 `package` 子句上，讲「这个包负责什么、边界在哪」，不复述依赖规则表。
- 导出标识符要有 doc comment：以标识符名开头、完整句子、句末句号（`// Day 返回指定日期的课表。`），内容讲契约——返回什么、何时报错、空值与单位语义。
- 行内注释只写「读者会疑惑的为什么」：上游怪癖、刻意的非直觉选择、踩过的坑。能一行说清就一行。

不要写：

- 不复述代码与命名。反例：`// NewQuery 装配课表查询。`、`// Unwrap 暴露原因，供 errors.Is/As 穿透。`、循环上方的 `// 遍历…`；注释与函数名信息量相同即视为噪声。
- 不复述项目规则、术语文档和任务记录。
- 不逐字段加注：类型与字段名说清了就不写；只有单位、形态、可选性不显然时，才在同一行补一句。
- 不留被注释掉的死代码，不画分隔线，不写「注意」「TODO」式无主语句。

注释应能独立读懂，不写决策编号或一次性任务文档路径。人工操作、安全提醒和需要同步文档的护栏除外。

测试文件：

- `_test.go` 不进 godoc，测试函数上方不写复述测试名的注释（测试名已经描述行为）。只写测试名与表字段名给不出的信息：这个负例为什么存在、期望值怎么算出来的。

## 命名

- 包名：短、单数、小写，禁 `util`/`common`/`misc`/`share`。
- 无 stutter：`course.WeekSpan`，不是 `course.CourseWeekSpan`；`ecjtu.Error`，不是 `ecjtu.EcjtError`。
- 缩写词大小写恒定：`ID`、`URL`、`OpenID`、`HTTPClient`。写 `Id`/`Url` 视为错误。
- 不用 `Get` 前缀：`Query.Day(...)`，不是 `GetDaySchedule(...)`。
- 接收器名短且全包一致：`func (q *Query) Day(...)`。禁 `this`/`self`/`me`。
- 构造函数 `New` / `NewXxx`：`client.New(cfg)`、`course.NewQuery(t)`。
- 域入口类型叫 `Query`，不叫 `Service`/`Reader`/`Client`/`Manager`。
- 上下文第一参数：`func (q *Query) Day(ctx context.Context, date time.Time)`。`ctx` 不存进 struct。

## 结构与禁止项

- 禁止 `init()` 做实际工作；测试里也不靠 `TestMain` 建 fixture。
- 禁止包级可变状态。`var ErrX = errors.New(...)` 只允许出现在确实需要跨包识别的分类里，本仓用 `Kind` 枚举代替导出 sentinel。`cmd/ecjtu` 的 `buildTime` 是链接时注入、运行时只读的构建元数据例外。
- 接口只在有第二个实现或需要 fake 时才定义，不为「可测试性」预先造接口。
- 不内嵌结构体来模拟继承；内嵌只为组合行为。
- 成功的查询结果中，课表 `Courses` 与成绩、考试 `Items` 必须为非 nil 切片，空结果输出 `[]`。
- 用 `any` 不用 `interface{}`；用 `slices`/`maps`/`min`/`max` 而不手写。
- 不 panic；`recover` 不用于流程控制。
- 函数选项模式只在可选参数达到 3 个以上时使用。
- import 分三组：标准库 / 第三方 / 本仓 `internal`，空行分隔，`gofmt` 后仍保持分组。
- 错误一律 `%w` 包装，不丢原因；不返回 `nil` error 配半满结果——结构不符预期就返回 `KindParse`。
- 当前成绩和考试正常页零条目会返回空列表，不因页上计数不一致而报错，见 [项目契约](conventions.md#输出)；是否应将这类页面视为结构异常尚待决定。
- 枚举必须有 `String()` 且含 unknown 分支。

## 测试写法

用原生 `testing`，表驱动为主，`t.Run` 命名子测试，测试名描述行为不描述实现（`TestParseDayKeepsParenthesisedCourseName`，不是 `TestParseDayLineLoop`）。

- 解析器同包白盒测试（`package course`），CLI 端到端用外部包（`package cli_test`）。
- fixture 一律放 `testdata/`（`go build` 会忽略该目录）。
- 断言用 stdlib：结构体比较优先使用直接比较或 `reflect.DeepEqual`，失败时输出实际值与期望值。不引入 `go-cmp`。
- 自动网络测试使用 `httptest.Server`。真实上游测试仅供人工运行，`ECJTU_LIVE` 非空时开启，否则跳过。
- 禁用 `time.Sleep` 等真实等待；超时测试用 `context.WithCancel` 与假 transport。
- 优先测试可观察的输入输出，不测试私有函数的具体分派路径。

## 提交

`type(scope): 中文描述`，无 emoji，scope 是真实包名。例：`feat(course): 支持多段周次解析`、`fix(ecjtu): 未绑定页面不再被判为空课表`。提交前运行 `make gate`。
