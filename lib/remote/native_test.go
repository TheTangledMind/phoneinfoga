package remote

import (
	"context"
	"errors"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestNativeProviderStatuses(t *testing.T) {
	n := number.Number{E164: "+12025550123"}
	s := newNativeScanner("github")
	called := 0
	s.client = &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		called++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"items":[{"name":"sample","html_url":"https://example.com/a"},{"name":"duplicate","html_url":"https://example.com/a"},{"name":"bad","html_url":"javascript:alert(1)"}]}`)), Header: make(http.Header)}, nil
	})}
	opts := ScannerOptions{"PHONEINFOGA_NATIVE": "true", "GITHUB_TOKEN": "fixture-only"}
	raw, e := s.Run(n, opts)
	r := raw.(SearchResponse)
	if e != nil || r.Status != "completed" || len(r.Results) != 1 {
		t.Fatalf("%+v %v", r, e)
	}
	delete(opts, "GITHUB_TOKEN")
	raw, _ = s.Run(n, opts)
	if raw.(SearchResponse).Status != "skipped" || called != 1 {
		t.Fatal("missing key sent request")
	}
	s.client.Transport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		called++
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("blocked")), Header: make(http.Header)}, nil
	})
	opts["GITHUB_TOKEN"] = "fixture-only"
	raw, _ = s.Run(n, opts)
	if raw.(SearchResponse).Status != "rate_limited" || called != 2 {
		t.Fatal("block handling")
	}
}

func TestNativeFixtures(t *testing.T) {
	for _, provider := range []string{"serpapi", "github", "reddit", "duckduckgo"} {
		t.Run(provider, func(t *testing.T) {
			b, e := os.ReadFile("testdata/native/" + provider + ".json")
			if e != nil {
				t.Fatal(e)
			}
			r, n, e := parseNative(provider, b)
			if e != nil || n != "" || len(r) != 1 {
				t.Fatalf("%+v %s %v", r, n, e)
			}
			if _, _, e := parseNative(provider, []byte(`{"unexpected":true}`)); e == nil {
				t.Fatal("missing schema accepted")
			}
			if _, _, e := parseNative(provider, []byte("<html>blocked</html>")); e == nil {
				t.Fatal("html accepted")
			}
		})
	}
}
func TestNativeCancellationAndHTTPFailures(t *testing.T) {
	s := newNativeScanner("github")
	n := number.Number{E164: "+12025550123"}
	opts := ScannerOptions{"PHONEINFOGA_NATIVE": "true", "GITHUB_TOKEN": "fixture-only"}
	for code, want := range map[int]string{401: "unauthorized", 403: "blocked", 429: "rate_limited", 500: "provider_error"} {
		calls := 0
		s.client.Transport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("Authorization") != "Bearer fixture-only" || !strings.Contains(r.URL.Query().Get("q"), "is:public") {
				t.Fatal("incorrect authenticated public search")
			}
			return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		})
		raw, e := s.Run(n, opts)
		if e != nil || raw.(SearchResponse).Status != want || calls != 1 {
			t.Fatalf("%d %+v %v", code, raw, e)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts["context"] = ctx
	s.client.Transport = fixtureTransport(func(r *http.Request) (*http.Response, error) { return nil, errors.New("fixture-only secret URL") })
	raw, _ := s.Run(n, opts)
	r := raw.(SearchResponse)
	if r.Status != "cancelled" || strings.Contains(r.Note, "fixture-only") {
		t.Fatal(r)
	}
}

func TestSerpAPIUsesBoundedCustomQueries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queries.json")
	os.WriteFile(path, []byte(`{"version":1,"queries":[{"category":"general","query":"site:example.com {number}","enabled":true},{"category":"reputation","query":"{number} spam","enabled":true}]}`), 0600)
	s := newNativeScanner("serpapi")
	calls := 0
	s.client.Transport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("q") != "site:example.com +12025550123" {
			t.Fatal(r.URL.Query().Get("q"))
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"organic_results":[]}`))}, nil
	})
	r, e := s.Run(number.Number{E164: "+12025550123"}, ScannerOptions{"PHONEINFOGA_NATIVE": "true", "SERPAPI_KEY": "fixture-only", "search_list": path, "max_queries": 1, "pace": time.Millisecond})
	if e != nil || calls != 1 || r.(SearchResponse).Status != "no_results" {
		t.Fatalf("%+v %v %d", r, e, calls)
	}
}

func TestCSEUsesCustomQueryList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queries.json")
	os.WriteFile(path, []byte(`{"version":1,"queries":[{"category":"general","query":"site:example.com {number}","enabled":true}]}`), 0600)
	calls := 0
	client := &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("q") != "site:example.com +12025550123" {
			t.Fatal("custom list ignored")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"searchInformation":{"totalResults":"0"},"items":[]}`))}, nil
	})}
	_, e := NewGoogleCSEScanner(client).Run(number.Number{E164: "+12025550123"}, ScannerOptions{"search_list": path, "GOOGLECSE_CX": "fixture", "GOOGLE_API_KEY": "fixture", "max_queries": 1})
	if e != nil || calls != 1 {
		t.Fatalf("%v %d", e, calls)
	}
}
