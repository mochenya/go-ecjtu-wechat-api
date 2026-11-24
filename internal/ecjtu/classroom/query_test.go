package classroom

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/mochenya/go-ecjtu-wechat-api/internal/ecjtu"
	"github.com/mochenya/go-ecjtu-wechat-api/internal/testsupport"
)

func TestQueryOptionsAndSearchAreAnonymous(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("Cookie") != "" || r.URL.Query().Get("weiXinID") != "" {
			t.Errorf("request carries identity: cookie=%q query=%v", r.Header.Get("Cookie"), r.URL.Query())
		}
		switch r.URL.Path {
		case "/OtherQueryServlet":
			if r.URL.Query().Get("type") != "1" {
				t.Errorf("type = %q", r.URL.Query().Get("type"))
			}
			_, _ = w.Write(testsupport.Fixture(t, "classroom-options-synthetic.html"))
		case "/KongClassRoomQuery":
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("method/content type = %s/%q", r.Method, r.Header.Get("Content-Type"))
			}
			body, _ := io.ReadAll(r.Body)
			form, _ := url.ParseQuery(string(body))
			want := url.Values{"term_xiala": {"2026.1"}, "weekSpan_xiala": {"3"}, "week_xiala": {"星期四"}, "classSpan_xiala": {"1,2"}}
			if !reflect.DeepEqual(form, want) {
				t.Errorf("form = %v, want %v", form, want)
			}
			_, _ = w.Write(testsupport.Fixture(t, "classroom-results-synthetic.html"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	query := NewQuery(ecjtu.NewPublicTransportWithConfig(ecjtu.PublicTransportConfig{BaseURL: server.URL}))
	result, err := query.Search(context.Background(), Input{Week: 3})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(paths, ",") != "/OtherQueryServlet,/KongClassRoomQuery" {
		t.Fatalf("paths = %v", paths)
	}
	if result.Count != 2 || result.Weekday != Thursday || result.Period != Period1To2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestSearchRejectsMissingWeekWithoutRequest(t *testing.T) {
	called := false
	rt := roundTripFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, nil
	})
	query := NewQuery(ecjtu.NewPublicTransportWithConfig(ecjtu.PublicTransportConfig{BaseURL: "https://mock.invalid", RoundTripper: rt}))
	_, err := query.Search(context.Background(), Input{})
	if !ecjtu.IsKind(err, ecjtu.KindInvalidInput) || called {
		t.Fatalf("error = %v, called = %v", err, called)
	}
}

func TestSearchReturnsEmptyRoomsForEmptyPage(t *testing.T) {
	pages := [][]byte{
		testsupport.Fixture(t, "classroom-options-synthetic.html"),
		[]byte(`<html><head><title>空闲教室使用情况</title></head><body><div>暂无空闲教室</div></body></html>`),
	}
	index := 0
	transport := ecjtu.NewPublicTransportWithConfig(ecjtu.PublicTransportConfig{
		BaseURL: "https://mock.invalid/weixin",
		RoundTripper: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			page := pages[index]
			index++
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(page)))}, nil
		}),
	})
	query := NewQuery(transport)
	got, err := query.Search(context.Background(), Input{Week: 3})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 0 || got.Rooms == nil || len(got.Rooms) != 0 || got.Term != "2026.1" || got.Weekday != Thursday {
		t.Fatalf("empty result = %#v", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
