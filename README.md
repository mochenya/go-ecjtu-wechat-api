# go-ecjtu-wechat-api

[![Release](https://img.shields.io/github/v/release/mochenya/go-ecjtu-wechat-api?style=flat-square)](https://github.com/mochenya/go-ecjtu-wechat-api/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/mochenya/go-ecjtu-wechat-api/ci.yml?style=flat-square&label=CI)](https://github.com/mochenya/go-ecjtu-wechat-api/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.27.1%2B-00ADD8?style=flat-square)

华东交通大学教务系统（微信公众号版）的 Go 命令行工具 `ecjtu`，用于查询课表、成绩、考试安排与空闲教室。

默认向 `stdout` 输出结构化 JSON，传入 `--format text` 输出终端对齐表格；诊断信息与错误提示一律写入 `stderr`。

## 命令概览

| 命令 | 说明 | 需要 OpenID |
|---|---|---|
| `ecjtu course` | 按日期查询当日课表（默认 `Asia/Shanghai` 今天） | 是 |
| `ecjtu score` | 按学期查询课程成绩与学分 | 是 |
| `ecjtu exam` | 按学期查询考试时间、地点、人数与备注 | 是 |
| `ecjtu classroom` | 匿名探测表单选项，或按学期、周次、星期、节次查询空闲教室 | 否 |

## 安装

### 预编译二进制

从 [GitHub Releases](https://github.com/mochenya/go-ecjtu-wechat-api/releases) 下载对应平台与架构的压缩包，解压后运行 `ecjtu`（Windows 为 `ecjtu.exe`）。各压缩包的 SHA-256 校验值见 `checksums.txt`。

Linux 和 macOS 安装或更新：

```sh
curl -fsSLo install-ecjtu.sh https://raw.githubusercontent.com/mochenya/go-ecjtu-wechat-api/main/scripts/install.sh
sh install-ecjtu.sh
```

### Go Install

需要 Go 1.27.1 及以上版本：

```bash
go install github.com/mochenya/go-ecjtu-wechat-api/cmd/ecjtu@latest
```

### 源码构建

普通源码构建版本显示为 `dev`，发布构建显示对应的 Git 标签：

```bash
git clone https://github.com/mochenya/go-ecjtu-wechat-api.git
cd go-ecjtu-wechat-api
make build     # 产出 bin/ecjtu
make release   # 无 CGO 单静态二进制
```

### AI Skill

安装 [ecjtu-cli Skill](skills/ecjtu-cli/SKILL.md)：

```sh
npx skills add mochenya/go-ecjtu-wechat-api --skill ecjtu-cli
```

将以下提示词发送给 AI Agent，安装 Skill 和 CLI：

```text
请阅读 https://github.com/mochenya/go-ecjtu-wechat-api/blob/main/README.md，安装 ecjtu-cli Skill 和 ecjtu CLI，并引导我获取和配置 OpenID。
```

## 快速开始

### 1. 配置凭证

`course`、`score`、`exam` 通过系统环境变量读取教务系统绑定的微信 OpenID；`classroom` 为匿名查询，无需配置。程序不读取 `.env` 文件。

**获取 OpenID：**

1. 打开微信中的「华交教务」公众号，确保已绑定教务账号。
2. 在菜单中选择「师生服务」→「我的课表」。
3. 在课表页面选择「用默认浏览器打开」，查看浏览器地址栏。
4. 找到地址中的 `weiXinID=`，复制等号后面的值；若后面还有 `&` 和其他参数，只复制到 `&` 之前。这个值就是 `ECJTU_WEIXIN_TOKEN`，不包含 `weiXinID=`。

**持久配置：**

Linux 和 macOS 用户，将下面一行加入自己使用的 shell 配置文件：zsh 使用 `~/.zshrc`，bash 通常使用 `~/.bashrc`（macOS 的登录 bash 可使用 `~/.bash_profile`）。把示例中的文字替换为自己的 OpenID，然后重新打开终端。

```sh
export ECJTU_WEIXIN_TOKEN='你的 OpenID'
```

Windows PowerShell 用户，在终端运行以下命令并按提示输入 OpenID；关闭终端重新打开后仍然有效：

```powershell
$token = Read-Host '请输入 OpenID'
[Environment]::SetEnvironmentVariable('ECJTU_WEIXIN_TOKEN', $token, 'User')
$env:ECJTU_WEIXIN_TOKEN = $token
Remove-Variable token
```

请在运行 `ecjtu` 或启动 AI Agent 的环境中配置；已运行的终端和 Agent 不会自动获得新设置。OpenID 和含有它的完整链接相当于个人凭证，请勿公开分享。

### 2. 查询示例

```bash
# 课表、成绩与考试（省略 --term 时使用上游默认学期）
ecjtu course
ecjtu course --date 2026-09-14 --format text
ecjtu score --term 2025.2 --format text
ecjtu exam --term 2025.1 --format text

# 空闲教室（无需 ECJTU_WEIXIN_TOKEN）
ecjtu classroom options --format text                 # 等同于 ecjtu classroom
ecjtu classroom search --week 3 --format text
ecjtu classroom search --term 2026.1 --week 3 --weekday 星期四 --period 1,2

# 诊断与辅助
ecjtu course --verbose                                # 诊断信息写入 stderr
ecjtu completion zsh                                  # 生成 shell 补全脚本
ecjtu --version
```

### 3. 输出示例

终端表格（`--format text`）：

```text
2026-09-14 星期一（第3周）
节次   课程                  时间           地点     教师
-----  --------------------  -------------  -------  ------
1-2节  大学英语Ⅲ(上课)      1-8 1,2        31-422   教师B
3-4节  体育Ⅲ(上课)          1-8,10-19 3,4  北区…    教师A
```

<details>
<summary>默认 JSON 输出示例</summary>

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

</details>

## 参考

### 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| `ECJTU_WEIXIN_TOKEN` | `course` / `score` / `exam` 必填；`classroom` 否 | 无 | 教务绑定的微信 OpenID |
| `ECJTU_BASE_URL` | 否 | `https://jwxt.ecjtu.edu.cn/weixin` | 上游服务基地址 |
| `ECJTU_TIMEOUT` | 否 | `15s` | 命令总超时预算（如 `20s`，重试共享） |

### 命令行参数

| 适用范围 | 参数 | 默认值 | 说明 |
|---|---|---|---|
| 全局 | `--timeout <duration>` | `ECJTU_TIMEOUT`（`15s`） | 覆盖超时预算，如 `--timeout 20s` |
| 全局 | `-v`, `--verbose` | `false` | 向 `stderr` 输出请求耗时、状态码与页面判定 |
| 查询子命令 | `--format <json\|text>` | `json` | 输出格式 |
| `course` | `--date <YYYY-MM-DD>` | 今天（`Asia/Shanghai`） | 查询日期 |
| `score` / `exam` | `--term <YYYY.N>` | 上游默认学期 | 查询学期，如 `2025.2` |
| `classroom search` | `--week <int>` | 必填 | 教学周次（通过 `ecjtu classroom options` 查看范围） |
| `classroom search` | `--term` / `--weekday` / `--period` | 表单默认值 | 学期、星期一至星期日、节次（`1,2` / `3,4` / `5,6` / `7,8` / `9,10` / `11,12`） |

### 退出码

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
- `testdata/` 仅保留脱敏测试样本，不含真实个人数据。

## KDE Plasma 6 小部件

仓库内置「华交校园」（`widgets/ecjtu-campus`）桌面小部件，通过本地 `ecjtu` CLI 异步查询今明课表，支持面板弹窗与桌面常驻。

```bash
./widgets/ecjtu-campus/scripts/install.sh --reload   # 构建、用户级安装至 ~/.local 并重载面板
./widgets/ecjtu-campus/scripts/uninstall.sh          # 卸载小部件与插件
```

架构说明、脚本参数与离线预览见 [widgets/ecjtu-campus/README.md](widgets/ecjtu-campus/README.md)。

## 开发与测试

```bash
make gate   # 运行 gofmt、go vet、go test -race 与依赖检查（离线假上游）
make live   # 人工对真实上游执行冒烟测试（需配置 ECJTU_WEIXIN_TOKEN，勿用于 CI）
```

---

> 本项目仅供个人学习与查询本人教务信息使用，请合理控制请求频率，避免对学校服务器造成压力。
