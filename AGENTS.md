# go-ecjtu-wechat-api

华东交通大学教务系统（微信公众号版）数据抓取的 Go CLI。

## 强制约束

1. **语言分层**：交流、doc comment、行内注释、CLI 用户文案用简体中文；error string 用小写英文且不加分号句号；标识符一律英文，禁止拼音命名。
2. **凭证**：只从系统环境变量读取 `ECJTU_WEIXIN_TOKEN`。禁止读取或打印 `.env`。任何日志、错误信息、测试快照里的 openid 必须脱敏为前 4 位加 `***`。
3. **项目文档统一维护**：架构、契约、上游事实与规范写在 `docs/` 及根目录说明文件；未脱敏的原始上游响应不得写入版本库。
4. **上游是学校服务器**：真实请求是人工动作，禁止进 CI，禁止批量循环，重试最多 2 次退避且只对幂等 GET。
5. **工具链只用 Go 自带**：`gofmt`、`go vet`、`go test`（含 `-race`）。不引入 golangci-lint、gofumpt、staticcheck、testify。
6. **提交信息**：`type(scope): 中文描述`，无 emoji，`scope` 必须是真实包名。
7. **架构方向不可违反**：`cmd → client → 域包 → ecjtu`，域包之间禁止互相引用。改包结构必须同步更新 [docs/agent/architecture.md](docs/agent/architecture.md)。

## 按任务阅读

- 新代码放哪、依赖方向 → [docs/agent/architecture.md](docs/agent/architecture.md)
- 命名、注释、错误、测试写法 → [docs/agent/go-style.md](docs/agent/go-style.md)
- 开发流程与常用命令 → [docs/agent/workflow.md](docs/agent/workflow.md)
- 本项目专属契约（CLI 输出、退出码、env、fixture）→ [docs/agent/conventions.md](docs/agent/conventions.md)
- 上游接口事实与实测证据 → [docs/agent/upstream.md](docs/agent/upstream.md)
- 术语的唯一说法 → [CONTEXT.md](CONTEXT.md)

## 变更时维护文档

完成有意义的代码或设计变更前，按 [文档同步](docs/agent/workflow.md#文档同步) 检查受影响的契约、上游证据、术语和入口；在同一次变更中更新对应文档与验证，或说明无需更新的原因。新提出的规范与未决设计须标明状态，不能冒充现有约定。

## 未决事项

`week` 聚合命令和配置文件尚未定案，不得自行实现。
