package ecjtu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultUserAgent 模拟微信内置浏览器；实测 UA 不是鉴权条件。
const defaultUserAgent = "Mozilla/5.0 (Linux; Android 16; 24129PN74C Build/BP2A.250605.031.A3; wv) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/116.0.0.0 " +
	"Mobile Safari/537.36 XWEB/1160117 MMWEBSDK/20250904 MMWEBID/1666 " +
	"MicroMessenger/8.0.65.2942(0x28004142) WeChat/arm64 Weixin GPVersion/1 " +
	"NetType/5G Language/zh_CN ABI/arm64"

const (
	// maxAttempts 是首次请求加最多两次退避重试。
	maxAttempts = 3
	// defaultMaxBodyBytes 是响应体上限，超限视为畸形页面按 KindParse 处理。
	defaultMaxBodyBytes = 8 << 20
)

// errNoResponse 标记请求失败或响应体读取失败，可重试。
var errNoResponse = errors.New("no response")

// errOversizedBody 标记响应体超限，不重试。
var errOversizedBody = errors.New("oversized response body")

// Response 是一次抓取的产物：页面字节与排障元数据（--verbose 的来源）。
type Response struct {
	Status   int
	Body     []byte
	Attempts int
	Elapsed  time.Duration
}

// Transport 是无状态抓取器：复用单个 http.Client、不保存 cookie，并对连接或读取失败做有界退避重试。
// 超时预算由调用方 ctx 控制，故不设 client.Timeout，以免把「整命令总预算」切成「单次尝试」。
type Transport struct {
	baseURL string
	openID  string // 领域词是 OpenID；weiXinID 只允许出现在下面的参数映射处

	client       *http.Client
	userAgent    string
	backoff      []time.Duration
	maxBodyBytes int64
	// sleep 是退避等待的注入点：测试不得真实等待。
	sleep func(ctx context.Context, d time.Duration) error
}

// TransportConfig 配置抓取器的上游地址、凭证、UA 与可选传输层。
type TransportConfig struct {
	BaseURL      string
	OpenID       string
	UserAgent    string
	RoundTripper http.RoundTripper
}

// PublicTransportConfig 配置不携带身份信息的上游抓取器。
type PublicTransportConfig struct {
	BaseURL      string
	UserAgent    string
	RoundTripper http.RoundTripper
}

// NewTransport 用上游基地址与 openid 装配抓取器。
func NewTransport(baseURL, openID string) *Transport {
	return NewTransportWithConfig(TransportConfig{BaseURL: baseURL, OpenID: openID})
}

// NewTransportWithUserAgent 创建抓取器；空 UA 使用默认微信 UA。
func NewTransportWithUserAgent(baseURL, openID, userAgent string) *Transport {
	return NewTransportWithConfig(TransportConfig{BaseURL: baseURL, OpenID: openID, UserAgent: userAgent})
}

// NewTransportWithConfig 按配置创建抓取器；空 UA 使用默认微信 UA，空 RoundTripper 使用默认传输层。
func NewTransportWithConfig(cfg TransportConfig) *Transport {
	return newTransport(cfg.BaseURL, cfg.OpenID, cfg.UserAgent, cfg.RoundTripper)
}

// NewPublicTransportWithConfig 创建不自动添加 OpenID 的抓取器。
func NewPublicTransportWithConfig(cfg PublicTransportConfig) *Transport {
	return newTransport(cfg.BaseURL, "", cfg.UserAgent, cfg.RoundTripper)
}

func newTransport(baseURL, openID, configuredUserAgent string, roundTripper http.RoundTripper) *Transport {
	userAgent := configuredUserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	return &Transport{
		userAgent: userAgent,
		baseURL:   strings.TrimRight(baseURL, "/"),
		openID:    openID,
		// 只接受 RoundTripper 而非外部 *http.Client，确保 Jar 为 nil（不回送 JSESSIONID）
		// 且不设 Client.Timeout，避免把整次命令的 ctx 预算切碎成单次尝试超时。
		client:       &http.Client{Transport: roundTripper},
		backoff:      []time.Duration{500 * time.Millisecond, time.Second},
		maxBodyBytes: defaultMaxBodyBytes,
		sleep:        sleepContext,
	}
}

// Get 向 path 发出携带 query 的 GET，并自动带上 openid 参数。op 形如 "course.Day"，
// 只用于错误归因，绝不参与拼 URL。
func (t *Transport) Get(ctx context.Context, op, path string, query url.Values) (*Response, error) {
	return t.get(ctx, op, path, query, true)
}

// GetPublic 向 path 发出不带身份参数的 GET；网络失败时沿用 GET 的有界退避重试。
func (t *Transport) GetPublic(ctx context.Context, op, path string, query url.Values) (*Response, error) {
	return t.get(ctx, op, path, query, false)
}

func (t *Transport) get(ctx context.Context, op, path string, query url.Values, includeOpenID bool) (*Response, error) {
	q := url.Values{}
	for k, vs := range query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	if includeOpenID {
		// 唯一的上游拼写映射点：领域 OpenID → 参数 weiXinID。
		q.Set("weiXinID", t.openID)
	} else {
		q.Del("weiXinID")
	}
	full := t.baseURL + "/" + strings.TrimLeft(path, "/")
	if encoded := q.Encode(); encoded != "" {
		full += "?" + encoded
	}

	started := time.Now()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		body, status, err := t.attempt(ctx, http.MethodGet, full, nil, "")
		if err == nil {
			return &Response{Status: status, Body: body, Attempts: attempt, Elapsed: time.Since(started)}, nil
		}
		lastErr = err
		if !errors.Is(err, errNoResponse) {
			break // 响应体超限或本地错误不重试
		}
		if attempt == maxAttempts || ctx.Err() != nil {
			break
		}
		if serr := t.sleep(ctx, t.backoff[attempt-1]); serr != nil {
			lastErr = serr
			break
		}
	}
	return nil, t.transportError(op, lastErr)
}

// PostPublic 提交不带身份信息的表单；POST 是非幂等操作，网络失败时不重试。
func (t *Transport) PostPublic(ctx context.Context, op, path string, form url.Values) (*Response, error) {
	full := t.baseURL + "/" + strings.TrimLeft(path, "/")
	started := time.Now()
	body, status, err := t.attempt(ctx, http.MethodPost, full, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, t.transportError(op, err)
	}
	return &Response{Status: status, Body: body, Attempts: 1, Elapsed: time.Since(started)}, nil
}

// attempt 返回完整的页面；HTTP 错误状态交给调用方判定。
func (t *Transport) attempt(ctx context.Context, method, full string, requestBody io.Reader, contentType string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, full, requestBody)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		// url.Error 的文本含完整 URL（即含 openid），只保留内层原因。
		cause := err
		var ue *url.Error
		if errors.As(err, &ue) && ue.Err != nil {
			cause = ue.Err
		}
		return nil, 0, fmt.Errorf("%w: %w", errNoResponse, cause)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, t.maxBodyBytes+1))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%w: read body: %w", errNoResponse, err)
	}
	if int64(len(body)) > t.maxBodyBytes {
		return nil, resp.StatusCode, fmt.Errorf("%w: %d bytes exceeds limit %d",
			errOversizedBody, len(body), t.maxBodyBytes)
	}
	return body, resp.StatusCode, nil
}

func (t *Transport) transportError(op string, cause error) error {
	if cause == nil {
		cause = errors.New("request failed")
	}
	kind := KindTransport
	if errors.Is(cause, errOversizedBody) {
		kind = KindParse
	}
	msg := Sanitize(cause.Error(), t.openID)
	return &Error{Kind: kind, Op: op, Cause: errors.New(msg)}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Observation 是一次抓取的诊断事实（--verbose 的来源）。只在拿到页面后发出：
// 抓取失败时诊断由错误链本身携带。
type Observation struct {
	Op       string        // "course.Day"，域名即前缀
	Status   int           // 上游状态码
	Kind     PageKind      // 页面判定结论
	Attempts int           // 含首次在内的请求次数
	Elapsed  time.Duration // 整个抓取耗时（重试共享总预算）
}

// Observer 接收一次抓取的诊断。
type Observer func(Observation)

type observerKey struct{}

// WithObserver 把观察者挂到 ctx 上。诊断通道必须是请求级的，
// 包级可变状态或 Transport 上的 setter 会破坏并发安全。
func WithObserver(ctx context.Context, obs Observer) context.Context {
	if obs == nil {
		return ctx
	}
	return context.WithValue(ctx, observerKey{}, obs)
}

// Observe 在 ctx 里存在观察者时上报一次诊断；没有则什么都不做。
func Observe(ctx context.Context, o Observation) {
	if obs, ok := ctx.Value(observerKey{}).(Observer); ok {
		obs(o)
	}
}
