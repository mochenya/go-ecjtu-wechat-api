# Makefile 只做 docs/agent/workflow.md 里那串命令的别名，不藏额外逻辑。
# 逐条对应关系写在每行右边：`make gate` 与 CI 跑的是同一串，本地不存在第二套真相。

GO      ?= go
BIN     := bin/ecjtu
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LD      = -ldflags "-s -w -X main.buildTime=$(BUILD_TIME)"

.PHONY: help build release fmt vet test gate live clean

help: ## 列出全部目标
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | sed -E 's/^([a-z]+):.*## (.*)$$/  make \1\t\2/'

build: ## 本机构建并注入 UTC 编译时间：bin/ecjtu
	$(GO) build -trimpath -ldflags "-X main.buildTime=$(BUILD_TIME)" -o $(BIN) ./cmd/ecjtu

release: ## 发布构建并注入 UTC 编译时间：单个无 cgo 二进制
	CGO_ENABLED=0 $(GO) build -trimpath $(LD) -o $(BIN) ./cmd/ecjtu

fmt: ## gofmt -l .（必须无输出；-l 不改变退出码，故用 test -z 兜住）
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

vet: ## go vet ./...
	$(GO) vet ./...

test: ## go test ./... -race
	$(GO) test ./... -race

gate: fmt vet test ## 提交前门禁：再加依赖整理与空白检查
	$(GO) mod tidy && git diff --exit-code go.mod go.sum
	git diff --check

live: ## 人工：向学校服务器发真实请求（需先注入 ECJTU_WEIXIN_TOKEN），永不进 CI
	ECJTU_LIVE=1 $(GO) test ./test/live -run Live -v

clean: ## 删除构建产物
	rm -rf bin coverage.out
