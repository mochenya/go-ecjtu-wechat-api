// Package client 提供统一的教务查询入口。
package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/classroom"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/course"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/exam"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu/score"
)

// Config 配置一个绑定 OpenID 的查询入口。BaseURL 为空时使用学校地址，
// UserAgent 为空时使用默认微信 UA，HTTPTransport 为空时使用默认传输层。
type Config struct {
	OpenID        string
	BaseURL       string
	UserAgent     string
	HTTPTransport http.RoundTripper
}

// PublicConfig 配置不需要 OpenID 的公开查询入口。
type PublicConfig struct {
	BaseURL       string
	UserAgent     string
	HTTPTransport http.RoundTripper
}

// Query 是绑定一个 OpenID 的教务查询入口。
type Query struct {
	course *course.Query
	score  *score.Query
	exam   *exam.Query
}

// PublicQuery 是不绑定 OpenID 的公开教务查询入口。
type PublicQuery struct {
	classroom *classroom.Query
}

// TermSource 指定从哪个域发现学期。
type TermSource uint8

const (
	// ScoreTerms 从成绩页面发现学期。
	ScoreTerms TermSource = iota + 1
	// ExamTerms 从考试页面发现学期。
	ExamTerms
)

// String 返回来源名称；未知值以 TermSource(数字) 表示。
func (s TermSource) String() string {
	switch s {
	case ScoreTerms:
		return "score"
	case ExamTerms:
		return "exam"
	default:
		return fmt.Sprintf("TermSource(%d)", s)
	}
}

// Terms 从指定域的一次查询中发现学期；空学期由上游选择默认值，不批量验证候选。
// 来源或学期无效时返回输入错误；上游页面或学期形态异常时返回对应分类的错误。
func (q *Query) Terms(ctx context.Context, source TermSource, term string) (ecjtu.TermDiscovery, error) {
	var provider ecjtu.TermProvider
	switch source {
	case ScoreTerms:
		provider = q.score
	case ExamTerms:
		provider = q.exam
	default:
		return ecjtu.TermDiscovery{}, &ecjtu.Error{Kind: ecjtu.KindInvalidInput, Op: "client.Terms", Cause: errors.New("invalid term source")}
	}
	return provider.Terms(ctx, term)
}

// New 创建统一查询入口，OpenID 为空时返回错误；所有查询共享同一个 transport。
func New(cfg Config) (*Query, error) {
	cfg.OpenID = strings.TrimSpace(cfg.OpenID)
	if cfg.OpenID == "" {
		return nil, errors.New("openid is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://jwxt.ecjtu.edu.cn/weixin"
	}
	t := ecjtu.NewTransportWithConfig(ecjtu.TransportConfig{
		BaseURL:      cfg.BaseURL,
		OpenID:       cfg.OpenID,
		UserAgent:    cfg.UserAgent,
		RoundTripper: cfg.HTTPTransport,
	})
	return &Query{course: course.NewQuery(t), score: score.NewQuery(t), exam: exam.NewQuery(t)}, nil
}

// NewPublic 创建不携带身份信息的公开查询入口。
func NewPublic(cfg PublicConfig) *PublicQuery {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://jwxt.ecjtu.edu.cn/weixin"
	}
	t := ecjtu.NewPublicTransportWithConfig(ecjtu.PublicTransportConfig{
		BaseURL:      cfg.BaseURL,
		UserAgent:    cfg.UserAgent,
		RoundTripper: cfg.HTTPTransport,
	})
	return &PublicQuery{classroom: classroom.NewQuery(t)}
}

// Course 查询指定日期的课表；date 为零值时返回 KindInvalidInput。
func (q *Query) Course(ctx context.Context, date time.Time) (course.Schedule, error) {
	return q.course.Day(ctx, date)
}

// Scores 查询指定学期的成绩；term 为空时由上游选择默认学期，格式非法时返回 KindInvalidInput。
func (q *Query) Scores(ctx context.Context, term string) (score.Result, error) {
	return q.score.Scores(ctx, term)
}

// Exams 查询指定学期的考试安排；term 为空时由上游选择默认学期，格式非法时返回 KindInvalidInput。
func (q *Query) Exams(ctx context.Context, term string) (exam.Result, error) {
	return q.exam.Exams(ctx, term)
}

// ClassroomOptions 返回公开空教室表单当前提供的选项与默认值。
func (q *PublicQuery) ClassroomOptions(ctx context.Context) (classroom.Options, error) {
	return q.classroom.Options(ctx)
}

// Classrooms 按条件查询公开空教室；input.Week<=0 或条件非法、不在可选范围内时返回 KindInvalidInput。
func (q *PublicQuery) Classrooms(ctx context.Context, input classroom.Input) (classroom.Result, error) {
	return q.classroom.Search(ctx, input)
}
