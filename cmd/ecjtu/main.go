// Command ecjtu 是唯一的 main：按需读取环境变量装配客户端并委托 internal/cli 执行命令。
// 单静态二进制的构建参数见 Makefile。
package main

import (
	"context"
	"io"
	"os"
	"runtime"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/cli"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/config"
	ecjtu "github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/client"
)

// version 由发布构建在链接时注入；普通构建标记为开发版。
var version = "dev"

// buildTime 由 Makefile 在链接时注入，直接执行 go build 时为 unknown。
var buildTime = "unknown"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

// run 走完装配并返回退出码。凭证只在真正要发请求时才读取，
// 因此 --help 与 --version 不依赖环境变量。
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	deps := cli.Deps{
		Assemble:       assemble,
		AssemblePublic: assemblePublic,
		Stdout:         stdout,
		Stderr:         stderr,
		BuildInfo: cli.BuildInfo{
			Version:   version,
			BuildTime: buildTime,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			Compiler:  runtime.Compiler,
		},
	}
	return cli.Execute(ctx, deps, args)
}

func assemblePublic() (cli.Assembly, error) {
	cfg, err := config.LoadPublic()
	if err != nil {
		return cli.Assembly{}, cli.ConfigError(config.EnvOf(err), err)
	}
	return cli.Assembly{PublicClient: ecjtu.NewPublic(ecjtu.PublicConfig{BaseURL: cfg.BaseURL}), Timeout: cfg.Timeout}, nil
}

// assemble 读系统环境变量并装配客户端：不读任何文件。错误一律带中文用户文案，
// 退出码 2 且不发请求。
func assemble() (cli.Assembly, error) {
	cfg, err := config.Load()
	if err != nil {
		return cli.Assembly{}, cli.ConfigError(config.EnvOf(err), err)
	}
	school, err := ecjtu.New(ecjtu.Config{BaseURL: cfg.BaseURL, OpenID: cfg.OpenID})
	if err != nil {
		return cli.Assembly{}, cli.ConfigError(config.EnvOpenID, err)
	}
	return cli.Assembly{Client: school, Timeout: cfg.Timeout}, nil
}
