// Package config 只读系统环境变量装配配置：无配置文件、无自动发现。
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// 环境变量名。ECJTU_WEIXIN_TOKEN 只从系统环境变量读取，不读也不打印 .env。
const (
	EnvOpenID  = "ECJTU_WEIXIN_TOKEN"
	EnvBaseURL = "ECJTU_BASE_URL"
	EnvTimeout = "ECJTU_TIMEOUT"
)

const (
	// DefaultTimeout 是整个命令的 ctx 总预算，不是单次尝试。
	DefaultTimeout = 15 * time.Second
)

// Config 是装配上游客户端所需的最小配置。故意不实现 String 与 MarshalJSON，
// 避免 openid 被无意打印。
type Config struct {
	OpenID  string
	BaseURL string
	Timeout time.Duration
}

// PublicConfig 是公开查询所需的配置，不包含 OpenID。
type PublicConfig struct {
	BaseURL string
	Timeout time.Duration
}

// EnvError 指出哪个环境变量没通过校验：错误串仍是小写英文，但上层可以按 Env 给出
// 针对性的中文文案，而不是把所有配置问题都念成「凭证未配置」。
type EnvError struct {
	Env   string
	Cause error
}

// Error 委托给原因，保证日志与 --verbose 里的英文原因原样可见。
func (e *EnvError) Error() string { return e.Cause.Error() }

func (e *EnvError) Unwrap() error { return e.Cause }

// EnvOf 取出出错的环境变量名；不是 EnvError 时返回空串。
func EnvOf(err error) string {
	var target *EnvError
	if errors.As(err, &target) {
		return target.Env
	}
	return ""
}

func envError(env, format string, args ...any) error {
	return &EnvError{Env: env, Cause: fmt.Errorf(format, args...)}
}

// Load 从环境变量读取并校验配置。
func Load() (Config, error) {
	openID := strings.TrimSpace(os.Getenv(EnvOpenID))
	if openID == "" {
		return Config{}, envError(EnvOpenID, "%s is not set", EnvOpenID)
	}
	public, err := LoadPublic()
	if err != nil {
		return Config{}, err
	}
	return Config{OpenID: openID, BaseURL: public.BaseURL, Timeout: public.Timeout}, nil
}

// LoadPublic 从环境变量读取公开查询配置，不要求 ECJTU_WEIXIN_TOKEN。
func LoadPublic() (PublicConfig, error) {
	cfg := PublicConfig{BaseURL: strings.TrimSpace(os.Getenv(EnvBaseURL))}
	if cfg.BaseURL != "" {
		u, err := url.Parse(cfg.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return PublicConfig{}, envError(EnvBaseURL, "%s must be an absolute http(s) url", EnvBaseURL)
		}
	}

	raw := strings.TrimSpace(os.Getenv(EnvTimeout))
	if raw == "" {
		return PublicConfig{BaseURL: cfg.BaseURL, Timeout: DefaultTimeout}, nil
	} else {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			return PublicConfig{}, envError(EnvTimeout, "%s must be a positive go duration", EnvTimeout)
		}
		cfg.Timeout = d
	}
	return cfg, nil
}
