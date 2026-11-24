# 架构与依赖方向

## 包职责

| 包 | 职责 | 允许 import |
|---|---|---|
| `cmd/ecjtu` | 唯一 `main`：按命令装配凭证查询入口或公开查询入口，注册命令，把 error 转退出码 | `internal/cli`, `internal/config`, `internal/ecjtu`（含 `client`） |
| `internal/cli` | cobra 命令、flag 校验、中文用户文案（`messages.go`）、Kind→退出码（`exit.go`）、`--verbose` 诊断 | `internal/render`, `internal/ecjtu/client`, `internal/ecjtu`；**不** import `internal/config`（配置错误文案由 `cmd` 传入变量名） |
| `internal/ecjtu/client` | 统一教务查询入口：提供绑定 OpenID 的课表、成绩、考试查询与学期发现，以及不绑定身份的空教室查询；调用方可指定 UA | `internal/ecjtu` 及其域子包 |
| `internal/ecjtu` | 上游共同层：`transport.go`（抓取与诊断）`page.go`（页面判定与内容提取）`errors.go` `result.go`（泛型容器）`term.go`（学期校验、发现协议与排序）`date.go`（课表日期解析与默认东八区） | 标准库、`golang.org/x/net/html`、`PuerkitoBio/goquery` |
| `internal/ecjtu/course` | 课表域：查询流程 + 纯函数 `parse` + 模型 + `WeekSpan` | `internal/ecjtu`、标准库、`goquery` 与 `x/net/html` |
| `internal/ecjtu/score` | 成绩域：同构，容器用 `ecjtu.Result[ScoreItem]` 别名 | `internal/ecjtu`、标准库、`goquery` 与 `x/net/html` |
| `internal/ecjtu/exam` | 考试域：同构，容器用 `ecjtu.Result[ExamItem]` 别名 | `internal/ecjtu`、标准库、`goquery` 与 `x/net/html` |
| `internal/ecjtu/classroom` | 空教室域：匿名表单选项发现、条件校验、表单提交与教室列表解析 | `internal/ecjtu`、标准库、`goquery` 与 `x/net/html` |
| `internal/config` | 只读系统环境变量 → `Config` / `PublicConfig`，含校验 | 标准库 |
| `internal/render` | 结构化数据 → text/json 呈现，不产生数据 | 域包、标准库 |
| `test/live` | 人工真实上游冒烟（`ECJTU_LIVE` 非空才跑，默认 `t.Skip`），只有测试文件 | `internal/config`, `internal/ecjtu`, `internal/ecjtu/client` |
| `internal/testsupport` | 测试专用：脱敏 fixture 读取 | 标准库 |

## 依赖方向规则

单向，改动时人工检查：

1. `internal/ecjtu` 不得 import 任何域子包（否则与 façade 成环）。
2. 域包之间不得互相 import。
3. `internal/render` 不得 import `internal/cli`；反向可以。
4. `internal/cli`、`cmd` 之外不得出现 `os.Exit`。
5. `internal/config` 不依赖任何本仓内部包。
6. 域包不得 import `cobra`、`os`、`fmt.Println` 级别的呈现依赖。
7. 生产包（含 `cmd`）不得 import `internal/testsupport`；它只能出现在 `_test.go` 里。

改包结构必须同步更新本文的表格，并检查依赖方向。

## 共享层归属判据

拆包依据是「能否独立理解、独立复用」，不是文件数量。每个新包都会增加依赖和跳转成本。

- 多域共用 → 上提到 `internal/ecjtu`：`Term` 校验、`headinfo`（`div.right` 姓名学期 + `ul.dropdown-menu` 学期列表）、`PageKind`、错误分类。
- 单域生产 → 留在域包：`WeekSpan` 只在课表出现，留在 `course`，不进共享层。
- 多个域确有重复且语义一致时，才考虑抽取共享函数。

## 统一查询入口

`internal/ecjtu/client` 是调用方使用的统一教务查询入口；`New` 创建绑定 OpenID 的 `Query`，`NewPublic` 创建不绑定身份的 `PublicQuery`。前者提供 `Course`、`Scores`、`Exams` 与学期发现，后者提供空教室 Options/Search。两类入口都可提供上游地址、UA 与可选的 `HTTPTransport`（`http.RoundTripper`），省略地址和 UA 时分别使用学校地址和既有微信 UA；仍不直接接受 `*http.Client`。域入口与 transport 的装配由统一入口负责。

统一入口放在 `client` 包是为了避免 Go 循环依赖：域包依赖 `internal/ecjtu` 共享上游代码，而统一入口需要依赖域包。各域的路径、参数、抓取和页面判定流程集中在各自 `query.go`；HTML 解析保留在 `parse.go`。统一查询入口不继承 CLI 输出规则。

### 学期发现

`Query.Terms(ctx, client.ScoreTerms, term)` 和 `Query.Terms(ctx, client.ExamTerms, term)` 分别通过成绩、考试域的 `ecjtu.TermProvider` 实现发现，沿用各域查询流程。单次调用只抓取一个页面，不缓存、不批量请求；课表域不实现该协议，仍按日期查询。`term` 为空时由上游选择默认值；非空非法学期在域查询处返回 `KindInvalidInput` 且不发请求。

`ecjtu.TermDiscovery.Candidates` 是本页下拉中发现的非空候选，按年份和期数数值倒序，不保证完整；页面没有下拉时为 nil。`Available` 是本次已确认有数据的学期，恒非 nil，最多一项；空列表不表示所有学期均无数据。`CheckedTerm` 优先取页面回显；空态页没有回显时取显式请求参数，均不存在时为 nil。页面学期或非空候选形态非法时，学期发现返回 `KindParse`。现有成绩与考试结果的 `availableTerms` 仍保留页面原始顺序与既有 JSON 语义。

## 语言分层与错误归属

`Kind` 分类在 `internal/ecjtu/errors.go`，退出码映射在 `internal/cli/exit.go`，中文文案在 `internal/cli/messages.go`。理由：分类是领域事实（跨消费者复用），退出码与文案是 CLI 表现（不该污染领域层）。
