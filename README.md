# go-ecjtu-wechat-api

华东交通大学教务系统（微信公众号版）的 Go 命令行工具 `ecjtu`，用于查询课表、成绩、考试安排与空闲教室。

默认向 `stdout` 输出 JSON 供脚本消费，传入 `--format text` 输出终端对齐表格；诊断信息与错误提示一律写入 `stderr`。

## 功能

- **课表查询（`course`）**：按日期查询当日课表（默认东八区今天）
- **成绩查询（`score`）**：按学期查询课程成绩与学分
- **考试查询（`exam`）**：按学期查询考试时间、地点、人数与备注
- **空闲教室查询（`classroom`）**：匿名探测可用表单选项，并按学期、周次、星期、节次查询空闲教室

## 安装与构建

需要 Go 1.27.1 或更高版本。

```bash
# 直接安装
go install github.com/mochenya/go-ecjtu-wechat-api/cmd/ecjtu@latest

# 或从源码构建
git clone https://github.com/mochenya/go-ecjtu-wechat-api.git
cd go-ecjtu-wechat-api
make build     # 产出 bin/ecjtu
make release   # 无 CGO 单静态二进制
```

## 凭证配置

- `course`、`score`、`exam` 需要通过系统环境变量提供教务系统绑定的微信 OpenID；
- `classroom` 为匿名查询，无需配置凭证；
- 程序只读取系统环境变量，不读取 `.env` 文件。

```bash
export ECJTU_WEIXIN_TOKEN='你的教务绑定 OpenID'
```

## 使用示例

```bash
# 课表
ecjtu course
ecjtu course --date 2026-09-14 --format text

# 成绩与考试安排（省略 --term 时使用教务系统默认学期）
ecjtu score
ecjtu score --term 2025.2 --format text
ecjtu exam --term 2025.1 --format text

# 空闲教室（无需 ECJTU_WEIXIN_TOKEN）
ecjtu classroom                                      # 等同于 classroom options
ecjtu classroom options --format text
ecjtu classroom search --week 3 --format text
ecjtu classroom search --term 2026.1 --week 3 --weekday 星期四 --period 1,2

# 诊断与补全
ecjtu course --verbose                               # 诊断写入 stderr，不污染 stdout
ecjtu completion zsh                                 # 生成 shell 补全脚本
ecjtu --help
ecjtu --version
```

### 输出示例

`--format text`：

```text
2026-09-14 星期一（第3周）
节次   课程                  时间           地点     教师
-----  --------------------  -------------  -------  ------
1-2节  大学英语Ⅲ(上课)      1-8 1,2        31-422   教师B
3-4节  体育Ⅲ(上课)          1-8,10-19 3,4  北区…    教师A
```

默认 JSON：

```json
{
  "date": "2026-09-14",
  "weekday": 1,
  "week": 3,
  "courses": [
    {
      "periods": [1, 2],
      "name": "大学英语Ⅲ",
      "status": "上课",
      "weeks": [{ "start": 1, "end": 8 }],
      "location": "31-422",
      "teacher": "教师B"
    }
  ]
}
```

## 环境变量与参数

### 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `ECJTU_WEIXIN_TOKEN` | `course`/`score`/`exam` 必填；`classroom` 否 | 无 | 教务绑定的微信 OpenID |
| `ECJTU_BASE_URL` | 否 | `https://jwxt.ecjtu.edu.cn/weixin` | 上游服务基地址 |
| `ECJTU_TIMEOUT` | 否 | `15s` | 整个命令的超时预算（如 `20s`，重试共享） |

### 常用 Flag

| 命令 | Flag | 默认值 | 说明 |
|---|---|---|---|
| 全局 | `--timeout <duration>` | `ECJTU_TIMEOUT`（`15s`） | 覆盖超时预算，如 `--timeout 20s` |
| 全局 | `-v`, `--verbose` | `false` | 输出请求耗时、状态码与页面判定到 `stderr` |
| 查询子命令 | `--format <json\|text>` | `json` | 输出格式 |
| `course` | `--date <YYYY-MM-DD>` | 今天（`Asia/Shanghai`） | 查询日期 |
| `score` / `exam` | `--term <YYYY.N>` | 上游默认学期 | 查询学期，如 `2025.2` |
| `classroom search` | `--week <int>` | 必填 | 教学周次（通过 `ecjtu classroom options` 查看范围） |
| `classroom search` | `--term` / `--weekday` / `--period` | 表单默认值 | 学期、星期一至星期日、节次（`1,2` / `3,4` / `5,6` / `7,8` / `9,10` / `11,12`） |

## 退出码

| 退出码 | 含义 |
|---|---|
| `0` | 成功（包含无课、无成绩、无空闲教室等正常空结果） |
| `2` | 参数、用法或环境变量配置错误（不发请求） |
| `3` | OpenID 未绑定或已失效，需在公众号重新绑定 |
| `4` | 上游不可用（网络失败、超时或非 2xx / 异常页） |
| `5` | 页面结构与预期不符（疑似上游改版） |

## 隐私与安全

- 错误信息与 `--verbose` 输出中的 OpenID 自动脱敏为「前 4 位 + `***`」，`jsessionid` 替换为 `***`。
- 不输出完整 URL query 与原始 HTML 正文，上游 500 堆栈直接丢弃。
- `testdata/` 仅包含脱敏后的测试样本，不含真实个人数据。

## KDE Plasma 6 小部件

仓库内置「华交校园」（`widgets/ecjtu-campus`）KDE Plasma 6 液态玻璃课表小部件，通过本地 `ecjtu` CLI 异步查询今明课表，支持面板弹窗与桌面常驻。

```bash
./widgets/ecjtu-campus/scripts/install.sh --reload   # 构建、用户级安装到 ~/.local 并重载面板
./widgets/ecjtu-campus/scripts/uninstall.sh          # 卸载小部件与插件
```

架构说明、脚本参数与离线预览见 [widgets/ecjtu-campus/README.md](widgets/ecjtu-campus/README.md)。

## 开发与测试

```bash
make gate   # gofmt、go vet、go test -race 与依赖检查（离线假上游）
make live   # 人工对真实上游执行冒烟测试（需配置 ECJTU_WEIXIN_TOKEN，勿用于 CI）
```

## 免责声明

本项目仅供个人学习与查询本人教务信息使用，请合理控制请求频率，避免对学校服务器造成压力。
