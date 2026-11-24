# testdata 脱敏说明与 fixture 清单

本目录只存放**脱敏后**的 HTML fixture（规则见 [docs/agent/conventions.md](../docs/agent/conventions.md)）。
未脱敏的原始上游响应包含个人身份信息，禁止放入本目录或提交到版本库。

## 脱敏规则

| 类别 | 原始来源 | fixture 中的替换方式 |
|---|---|---|
| OpenID | query 参数 `weiXinID` | 固定假值 `TESTOPENIDXXXXXXXXXXXXXXXXX` |
| `jsessionid` | URL 内嵌 `;jsessionid=...` | 整串删除（含 `;jsessionid=` 段） |
| 学生姓名 | `div.right` | `张三` |
| 教师姓名 | 课表 `教师：` 行 | `教师A` / `教师B` / `教师C` / `教师D`（一人一映射） |

脱敏时保留原始 CRLF、缩进、标签嵌套和属性顺序，使测试能够覆盖上游页面的实际排版。

## fixture 清单

`pages/` 下分为两类：

**页面判定最小样本**（非整页快照，只服务 `Classify`）：`calendar_normal.html`、`calendar_noclass.html`、`score_empty.html`、`exam_empty.html`、`notbound.html`、`tomcat500.html`、`unknown.html`。

**整页脱敏 fixture**（文件名前缀标记数据域）：

| fixture | 用途 |
|---|---|
| `course-normal-week3-monday.html` | 3 门课、多段周次、分号地点、`status` 剥离 |
| `course-normal-week5-thursday.html` | 2 门课、`5-6 7,8` 两段周次 |
| `course-noclass-sunday.html` | 无课日：装饰 `easy*.png` 的 `li` 必须忽略 |
| `course-far-date-friday.html` | 学期外远日期：周次 18 不设上界 |
| `course-missing-center.html` | 缺失 `div.center` 的改版负例 → `KindParse` |
| `course-missing-teacher.html` | 缺失 `教师：` 行的标签缺失负例 → `KindParse` |
| `course-broken-time.html` | `时间：` 改为单段的变形负例 → `KindParse` |
| `course-unexpected-empty.html` | 课表域出现空态文案的合成负例 → `KindParse` |
| `score-normal-2025-2.html` | 15 门、等级分与数字分混排、空重考/重修列 |
| `score-normal-2025-1.html` | 13 门、`学分:0.0` 与 `(A)Ⅰ` 课名 |
| `score-empty-2026-2.html` | 空态页（Query 层断言，不走 parse） |
| `score-no-rows.html` | 正常页截掉全部 `div.row`：`items:[]` 但不升级为错误 |
| `score-missing-credit.html` | 缺少 `(学分:X)` 后缀 → `KindParse` |
| `score-missing-headinfo.html` | 缺少 `div.right` 页头 → `KindParse` |
| `exam-normal-2025-1.html` | 3 门、红色备注 div（含空白与实义两种） |
| `exam-normal-2025-2.html` | 5 门、验证周次与日期直接取页面值 |
| `exam-empty-current.html` | 空态页 |
| `exam-no-rows.html` | 正常页截掉全部 `div.row`：`items:[]` 但不升级为错误 |
| `exam-missing-label.html` | 缺少 `考试人数:` 及其 `span` → `KindParse` |
| `exam-bad-time.html` | 考试时间括号变为全角 → `KindParse` |
| `classroom-options-synthetic.html` | 合成表单页：选项列表、选中学期、默认值 |
| `classroom-results-synthetic.html` | 合成结果页：空教室结果表、页内条数与两条教室记录 |

结构级微缩负例（列序打乱、重复标签、非法日期、无标签取值等）在各域 `_test.go` 内构造，不入库为 fixture，避免与真实页面形态混淆。
