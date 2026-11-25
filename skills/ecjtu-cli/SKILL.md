---
name: ecjtu-cli
description: 使用 ecjtu CLI 查询华东交通大学课表、成绩、考试和空闲教室，或安装该 CLI。
---

# ECJTU CLI

项目与下载：[go-ecjtu-wechat-api](https://github.com/mochenya/go-ecjtu-wechat-api) · [Releases](https://github.com/mochenya/go-ecjtu-wechat-api/releases)。

先用 `command -v ecjtu` 检查安装。Linux 和 macOS 使用安装脚本下载预编译程序，重复运行可更新：

```sh
curl -fsSLo install-ecjtu.sh https://raw.githubusercontent.com/mochenya/go-ecjtu-wechat-api/main/scripts/install.sh
sh install-ecjtu.sh
```

Windows 从 Releases 下载对应平台的预编译程序。安装后的可执行文件需在 PATH 中；`ecjtu --version` 显示版本，`-v` 是详细日志。

## 常用查询

```sh
ecjtu course                                      # 今天的课表
ecjtu course --date 2026-09-14                    # 指定日期
ecjtu score                                       # 默认学期成绩
ecjtu score --term 2025.2                         # 指定学期成绩
ecjtu exam                                        # 默认学期考试
ecjtu exam --term 2025.2                          # 指定学期考试
ecjtu classroom options                           # 空教室查询可选条件
ecjtu classroom search --week 3                   # 指定教学周
ecjtu classroom search --week 3 --weekday 星期四 --period 1,2
```

默认输出 JSON；给查询命令添加 `--format text` 可显示表格。更多参数使用对应命令的 `--help`。成绩、考试的默认学期和空教室搜索省略的学期、星期、节次由上游表单决定；空教室搜索必须指定 `--week`，不要把表单默认周次当作当前教学周。

课表、成绩、考试需要通过环境变量 `ECJTU_WEIXIN_TOKEN` 提供绑定教务系统的微信 OpenID；空教室无需凭证。凭证缺失时请用户按 [README 凭证配置](https://github.com/mochenya/go-ecjtu-wechat-api#1-配置凭证) 在自己的环境中获取并配置，不索取、打印或写入凭证，不读取 `.env`；已有 Agent 进程需要重新启动才能继承新环境变量。

查询会访问学校服务器。按用户要求的条件查询，不批量循环；在项目工作区遵守该仓库 AGENTS.md 的真实请求约束。失败时根据 CLI 错误说明处理，不把失败当作空结果。
