# 项目契约

## 环境变量

配置只从系统环境变量读取，不读取配置文件。

| 变量 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `ECJTU_WEIXIN_TOKEN` | `course`/`score`/`exam` 是；`classroom` 否 | 无 | 教务绑定的微信 OpenID。个人数据命令缺失即退出码 2，不发请求；空教室走独立公开装配 |
| `ECJTU_BASE_URL` | 否 | `https://jwxt.ecjtu.edu.cn/weixin` | 测试注入点，`httptest` 靠它；非法即退出码 2 |
| `ECJTU_TIMEOUT` | 否 | `15s` | 整个命令的 ctx 总预算（重试共享），接受 Go duration 字面量 |

三种配置失败各有对应中文文案（缺 openid、`ECJTU_BASE_URL` 非法、`ECJTU_TIMEOUT` 非法），不再一律念成「凭证未配置」。新增环境变量必须同时出现在本表、`internal/config` 的测试与 `cli.ConfigError` 的分支里。配置文件尚未定案。

## 命令与 flag

| 命令 | flag | 默认 | 备注 |
|---|---|---|---|
| `ecjtu course` | `--date YYYY-MM-DD` | 今天（Asia/Shanghai） | 格式非法即退出码 2 |
| `ecjtu score` | `--term YYYY.N` | 上游默认学期 | 年份为四位数字，期数为数字；非法时退出码 2 |
| `ecjtu exam` | `--term YYYY.N` | 上游默认学期 | 同上 |
| `ecjtu classroom` / `classroom options` | `--format json\|text` | `json` | 一次匿名 GET 探测上游选项；`classroom` 等同 `classroom options` |
| `ecjtu classroom search` | `--term YYYY.N`、`--week`、`--weekday`、`--period`、`--room` | 学期/星期/节次使用表单默认值，周次无默认，`--room` 默认不筛选 | `--week` 必须显式指定；可用值以同次 Options 为准；`--room` 在本地按教室名子串筛选（忽略大小写），显式传空或纯空白退出码 2 |
| 查询子命令（`course`/`score`/`exam`/`classroom`） | `--format json\|text` | `json` | 未知值退出码 2 |
| 全局 | `--timeout` | 未显式给值时用 `ECJTU_TIMEOUT`（默认 `15s`） | 正值覆盖 env；0 使用配置值；负值退出码 2；预算覆盖整个命令 |
| 全局 | `-v`, `--verbose` | false | 只加诊断到 stderr，不改 stdout |
| 全局 | `--version` | — | 输出版本、UTC 编译时间、Go 版本、目标平台和编译器；发布构建注入 Git 标签，普通构建默认为 `dev`，直接 `go build` 时编译时间为 `unknown` |
| 子命令 | `help`、`completion <shell>` | — | cobra 自带；与 `--version`、`--help` 都不需要凭证 |

## 帮助与提示文案

CLI 帮助与错误文案须让用户在终端里不翻 README 也能完成调用与排障：

- **命令帮助分层**：每个命令同时提供 `Short`（一行概要）与 `Long`（鉴权前提、默认语义与行为边界），所有可执行子命令提供 1–2 条可直接复制的 `Example`。根命令 `Long` 列出环境变量、凭证适用范围（`classroom` 无需 token）与退出码。
- **Flag 说明**：写清格式示例（如 `2026-09-14`、`2025.2`、`20s`）、合法枚举值、缺省行为与必填标记。
- **错误文案三要素**：集中在 `internal/cli/messages.go`，说明错在哪里、合法格式是什么，以及下一步可执行动作（如补配环境变量、去公众号重新绑定、先跑 `ecjtu classroom options`）。
- **Cobra 框架语言**：保留 Cobra 默认英文结构标签（`Usage:`、`Flags:` 等）与内置 `help`/`completion` 骨架，所有业务命令描述、示例和报错文案一律使用简体中文。

## 退出码

| 码 | 含义 | 对应 `Kind` | 用户动作 |
|---|---|---|---|
| 0 | 成功，含空结果（`empty: true`） | — | 无 |
| 2 | 用法或输入错误 | `KindInvalidInput` | 改命令 |
| 3 | 未绑定 / 凭证失效 | `KindNotBound` | 去公众号重新绑定 |
| 4 | 上游不可用（非 2xx、超时、网络） | `KindTransport` `KindUpstream` | 稍后重试 |
| 5 | 解析失败，疑上游改版 | `KindParse` + 无法归类页面 | 报障或改解析器 |

空结果不是错误：教务查询返回空是正常状态，无法归类的页面按解析失败处理。

## 输出

数据查询的 stdout 只放 JSON 或 text 结果，诊断与错误写入 stderr。JSON 键用 lowerCamelCase，各域的顶层形状为：

| 命令 | 顶层键 | 空结果 |
|---|---|---|
| `course` | `date` `weekday` `week` `courses[]` | 无课日 `courses: []`，`empty` 概念不存在 |
| `score` | `term` `studentName` `availableTerms[]` `empty` `count` `items[]` | 空态页 `empty: true`、`count: 0`、`items: []`、页头三字段 `null` |
| `exam` | 同 `score`（`items` 元素不同） | 同上 |
| `classroom options` | `selectableTerms[]` `selectedTerm` `weeks[]` `weekdays[]` `periods[]` `defaults` | 不适用 |
| `classroom search` | `term` `week` `weekday` `period` `count` `rooms[]` | `count: 0`、`rooms: []` |

`courses`/`items` 恒为非 nil 切片（输出 `[]` 而非 `null`）；未知标量用 `*string` → `null`；
`status`/`note` 为空时整个键省略，其余字段不省略。页面自带的条数 `div.words` 不进 JSON，
只作为 `--verbose` 的一致性告警（`internal/ecjtu` 的 `PageCount` 是 `json:"-"` 诊断字段）。
JSON 由 `render.JSON` 输出：`json.MarshalIndent` 两空格缩进 + 结尾换行。

成绩或考试正常页即使没有解析到条目，当前也返回 `empty: false`、`count: 0` 和 `items: []`；页面计数不一致只在 `--verbose` 告警。它与明确空态页的 `empty: true` 不同，不代表已确认上游正常页必然允许零条目。

## text 视图

`internal/render` 自算列宽对齐（`text/tabwriter` 不计 CJK 宽度）：

| 命令 | 首行 | 列集 |
|---|---|---|
| `course` | `2026-09-14 星期一（第3周）`（日期/星期/周次全取页面回显，越界星期省略） | `节次 课程 时间 地点 教师` |
| `score` | `张三 2025.2（共 15 门）`，页头缺失时退化为 `共 15 门` | `课程 代码 学分 期末 重考 重修 性质` |
| `exam` | 同 `score`，量词换成「场」 | `日期 时间 周次 地点 课程 性质 人数` |
| `classroom options` | 可选学期、表单选中学期、周次/星期/节次选项及默认值 | 多行选项摘要 |
| `classroom search` | 学期、周次、星期、节次和结果数量（带 `--room` 时附教室名筛选词） | `教室 教室类型 座位数` |

- 列间固定两个空格，行尾不补齐；表头下有 `-` 组成的分隔线。
- 显示宽度：W/F 与 U+2000 到 U+2FFF 的 Ambiguous（`Ⅰ Ⅱ Ⅲ`）计 2，`¤§¨°±×÷` 计 2，组合记号（Mn/Me/Cf）与控制字符计 0，其余 1。
- `null` 标量列渲染成 `-`；整数学分补 `.0`（对齐上游 `学分:4.0`）；课名列合并成 `大学英语Ⅲ(上课)`；节次序列合并成 `1-2,5,7-8节`。
- 考试 `备注` 非空时在该行下方续行，缩进到「课程」列起点；`className` 不进表，只在 JSON 里。
- 空结果的句子（`当天无课。`/`该学期无成绩。`/`该学期无考试安排。`/`该条件下没有空闲教室。`/带 `--room` 时为 `该条件下没有教室名包含「<关键词>」的空闲教室。`）走 stdout，属答案本身；其余诊断与错误一律 stderr。

空教室 Options 中 `selectedTerm` 表示该次表单选中的学期，不等价于校历当前学期。表单默认周次只用于描述浏览器默认行为；CLI 搜索仍要求显式 `--week`，不把表单首项推断为今天的教学周。`weekday` 与 `period` 在 JSON 中直接使用上游枚举值（星期中文文本与 `1,2` 等节次串）。

## `--verbose` 诊断

一次抓取一行，页面判定成功才发出（失败原因由错误链携带）：

```
phase=fetch domain=course op=course.Day status=200 kind=ok attempts=1 ms=812
warn=count-mismatch domain=score declared=15 parsed=0
phase=error kind=notbound err="course.Day: notbound"
```

- `domain` 是 `op` 的包名前缀，`kind` 取 `PageKind`/`Kind` 的小写英文标签；`attempts` 含首次在内。
- 条数一致性告警只在 `--verbose` 下出，不一致**不**升级为错误；`classroom search` 的一致性校验在本地 `--room` 筛选之前执行。
- `phase=error` 行仅 `--verbose` 时追加；`err=` 是英文错误串，openid 已按前 4 位 + `***` 脱敏。
- 未被 `Kind` 覆盖的错误（配置错误与 cobra/pflag 用法错误）在 `phase=error` 里一律报 `kind=usage`。
- 命令树在 `RunE` 之前就能工作：`--help`/`--version`/`completion` 不读环境变量。
- `--version` 输出多行人类可读构建信息；查询命令的 stdout 数据契约不受影响。
- 先校验本命令的 `--date`/`--term`，再校验 `--format`、读取配置并发请求；日期或学期错误优先于配置错误。
- cobra/pflag 的用法错误在中文文案后多打一行英文原因（如 `unknown flag: --nope`），再给用法提示。

## 脱敏

- openid 在任何输出中最多显示前 4 位加 `***`。`Config` 不得实现会打印 openid 的 `String()`/`MarshalJSON`。
- 错误信息、日志、测试快照里禁止出现完整 URL query、`jsessionid`、上游 HTML 原文。
- `Error.Op` 禁止由 URL 拼接得到，只允许 `包名.方法名`。
- 上游 500 的 Tomcat 栈只保留状态码，正文一律丢弃。
- 不提供 `--raw`/落盘导出，除非另行设计脱敏。

## fixture

- `testdata/` 只放**脱敏后**的 HTML：openid 替换为固定假值 `TESTOPENIDXXXXXXXXXXXXXXXXX`，学生姓名替换为 `张三`，教师姓名按 `testdata/README.md` 规则替换（`教师A/B/C/D`，一人一映射），`jsessionid` 连同 `;jsessionid=` 段整串删除，其余结构逐字节保留。未脱敏的原始上游响应禁止提交到版本库。
- fixture 命名为 `<域>-<场景>.html`，按需添加以覆盖页面解析与查询行为。
