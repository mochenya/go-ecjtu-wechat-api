# 开发流程与命令

## 流程

1. 从根目录 [AGENTS.md](../../AGENTS.md) 按任务进入相关规范，检查工作区已有改动、相关代码、测试和契约；尊重未提交的工作。
2. 明确预期行为、影响范围和验证方式；涉及上游时区分现有样本、实现和未确认的假设。需要决策或跨边界变更时记录方案及未决问题。
3. 保持已有契约，优先验证页面判定、解析、抓取与查询参数等受影响的行为。测试覆盖有意义的输入输出与失败边界，不重复实现细节。
4. 按下方映射检查文档和入口，运行适当的局部检查及 `make gate`；报告结果、未验证部分和仍有争议的事实。提交信息采用 `type(scope): 中文描述`，scope 为真实包名。

## 常用命令

```sh
make build          # 构建 bin/ecjtu，并注入 UTC 编译时间
make release        # CGO_ENABLED=0 发布构建，并注入 UTC 编译时间
make fmt            # 检查 gofmt
make vet            # go vet ./...
make test           # go test -race ./...
make gate           # fmt、vet、test、依赖整理与空白检查
go test ./internal/ecjtu/course -run TestParseDay -v

# KDE Plasma 6 小部件（离线构建与 Qt Test，详参 widgets/ecjtu-campus/README.md）
cmake -S widgets/ecjtu-campus -B widgets/ecjtu-campus/build -G Ninja && cmake --build widgets/ecjtu-campus/build
ctest --test-dir widgets/ecjtu-campus/build --output-on-failure
```

`make gate` 中的 `git diff` 不能检查尚未被 Git 跟踪的文件；遇到新文件需另行核对。
`make gate` 会执行 `go mod tidy` 并可能修改 `go.mod`、`go.sum`；运行前先检查已有改动，事后核对依赖差异。当前仓库若文件尚未跟踪，`git diff` 也无法证明依赖文件未变化，应自行比较。
默认测试使用本地假上游。`ECJTU_LIVE` 非空时人工上游测试会运行，因此自动检查前应确保该变量未设置。

## 手动上游测试

先通过环境变量设置 `ECJTU_WEIXIN_TOKEN`，再由人工执行 `make live`。它会向学校服务器发起真实请求，不得放进 CI 或批量运行。

## 文档同步

- 包职责、依赖方向或跨域查询协议变化：更新 [architecture.md](architecture.md)，检查调用方与依赖方向。
- CLI 命令、配置、输出、错误、脱敏或 fixture 契约变化：更新 [conventions.md](conventions.md)，检查 [README](../../README.md)、相关测试及 [fixture 登记表](../../testdata/README.md)。
- 上游路径、参数、页面判定或结构出现新事实：更新 [upstream.md](upstream.md)，区分样本观察和推断；检查查询、解析、页面判定和脱敏 fixture。原始证据含个人信息，不写入版本库或文档。
- 领域概念或含义变化：更新 [CONTEXT.md](../../CONTEXT.md)，检查模型、相关契约和引用。
- 验证命令、开发规则或文档路径变化：更新本文、[Go 规范](go-style.md)或相应入口；移动文档时检查旧链接和新链接。

不影响以上知识时无需改文档，但交付时说明检查结果。
