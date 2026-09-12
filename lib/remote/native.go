package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sundowndev/phoneinfoga/v2/lib/number"
	"github.com/sundowndev/phoneinfoga/v2/lib/searchlist"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

type SearchResult struct {
	Query    string `json:"query,omitempty" console:"Query,omitempty"`
	Category string `json:"category,omitempty" console:"Category,omitempty"`
	Title    string `json:"title" console:"Title"`
	URL      string `json:"url" console:"URL"`
	Snippet  string `json:"snippet,omitempty" console:"Snippet,omitempty"`
}
type SearchResponse struct {
	Status  string         `json:"status" console:"Status"`
	Note    string         `json:"note,omitempty" console:"Note,omitempty"`
	Query   string         `json:"query,omitempty" console:"Query,omitempty"`
	Results []SearchResult `json:"results" console:"Matches,omitempty"`
}

func (r SearchResponse) ScanStatus() string { return r.Status }

type nativeScanner struct {
	name   string
	client *http.Client
}

func newNativeScanner(name string) *nativeScanner {
	return &nativeScanner{name: name, client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (s *nativeScanner) Name() string { return s.name }
func (s *nativeScanner) Description() string {
	switch s.name {
	case "serpapi":
		return "Google results via SerpAPI (API key required)"
	case "github":
		return "Public GitHub code search (token required)"
	case "reddit":
		return "Reddit posts (approved API access and OAuth token required)"
	default:
		return "DuckDuckGo instant answers only; not a full web search"
	}
}
func (s *nativeScanner) DryRun(_ number.Number, _ ScannerOptions) error { return nil }
func (s *nativeScanner) runOne(n number.Number, opts ScannerOptions) (interface{}, error) {
	result := SearchResponse{Status: "skipped", Results: []SearchResult{}}
	if opts.GetStringEnv("PHONEINFOGA_NATIVE") != "true" {
		result.Note = "Select this source with --sources or the menu to enable it"
		return result, nil
	}
	q := `"` + n.E164 + `"`
	if custom, ok := opts["query"].(string); ok {
		q = custom
	}
	result.Query = q
	params := url.Values{"q": {q}}
	endpoint := ""
	key := ""
	keyName := ""
	switch s.name {
	case "serpapi":
		endpoint = "https://serpapi.com/search.json"
		keyName = "SERPAPI_KEY"
		params.Set("engine", "google")
		params.Set("num", "10")
	case "github":
		endpoint = "https://api.github.com/search/code"
		keyName = "GITHUB_TOKEN"
		params.Set("q", q+" is:public")
		params.Set("per_page", "10")
	case "reddit":
		endpoint = "https://oauth.reddit.com/search"
		keyName = "REDDIT_ACCESS_TOKEN"
		params.Set("limit", "10")
		params.Set("type", "link")
	case "duckduckgo":
		endpoint = "https://api.duckduckgo.com/"
		params.Set("format", "json")
		params.Set("no_html", "1")
		params.Set("no_redirect", "1")
		result.Note = "Instant answers only; empty output does not mean no web matches"
	default:
		return nil, errors.New("unknown provider")
	}
	if keyName != "" {
		key = opts.GetStringEnv(keyName)
		if key == "" {
			result.Note = "Missing " + keyName
			return result, nil
		}
	}
	if s.name == "serpapi" {
		params.Set("api_key", key)
	}
	ctx := context.Background()
	if c, ok := opts["context"].(context.Context); ok {
		ctx = c
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "phoneinfoga-fork/1.0")
	if s.name == "github" {
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	if s.name == "reddit" {
		ua := opts.GetStringEnv("REDDIT_USER_AGENT")
		if ua == "" {
			result.Note = "Missing REDDIT_USER_AGENT"
			return result, nil
		}
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, e := s.client.Do(req)
	if e != nil {
		result.Status = "network_error"
		result.Note = "Request failed or timed out"
		if ctx.Err() != nil {
			result.Status = "cancelled"
			result.Note = "Request cancelled or timed out"
		}
		return result, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		result.Status = "provider_error"
		switch resp.StatusCode {
		case 401:
			result.Status = "unauthorized"
		case 403:
			result.Status = "blocked"
		case 429:
			result.Status = "rate_limited"
		}
		result.Note = fmt.Sprintf("HTTP %d; no automatic retries", resp.StatusCode)
		return result, nil
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if e != nil || len(b) > 2*1024*1024 {
		result.Status = "parse_error"
		result.Note = "Unreadable or oversized response"
		return result, nil
	}
	results, note, e := parseNative(s.name, b)
	if e != nil {
		result.Status = "parse_error"
		result.Note = "Unexpected provider response"
		return result, nil
	}
	if note != "" {
		result.Status = "provider_error"
		result.Note = note
		return result, nil
	}
	result.Results = results
	result.Status = "completed"
	if len(results) == 0 {
		result.Status = "no_results"
	}
	return result, nil
}
func cleanSearchText(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s)
	r := []rune(s)
	if len(r) > 2000 {
		s = string(r[:2000])
	}
	return strings.TrimSpace(s)
}
func parseNative(provider string, b []byte) ([]SearchResult, string, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(b, &root) != nil || root == nil {
		return nil, "", errors.New("not an object")
	}
	if v, ok := root["error"]; ok && string(v) != "null" && string(v) != `""` {
		return nil, "Provider reported an error; check configuration or quota", nil
	}
	out := []SearchResult{}
	seen := map[string]bool{}
	add := func(title, link, snippet string) {
		u, e := url.Parse(link)
		if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return
		}
		u.Fragment = ""
		u.Host = strings.ToLower(u.Host)
		key := u.String()
		if seen[key] || len(out) >= 10 {
			return
		}
		seen[key] = true
		out = append(out, SearchResult{Title: cleanSearchText(title), URL: key, Snippet: cleanSearchText(snippet)})
	}
	switch provider {
	case "serpapi":
		var rows []struct{ Title, Link, Snippet string }
		v, ok := root["organic_results"]
		if !ok {
			var info struct {
				OrganicResultsState string `json:"organic_results_state"`
			}
			json.Unmarshal(root["search_information"], &info)
			if strings.Contains(strings.ToLower(info.OrganicResultsState), "empty") {
				return out, "", nil
			}
			return nil, "", errors.New("missing organic results")
		}
		if json.Unmarshal(v, &rows) != nil {
			return nil, "", errors.New("invalid results")
		}
		for _, r := range rows {
			add(r.Title, r.Link, r.Snippet)
		}
	case "github":
		var rows []struct {
			Name    string
			HTMLURL string `json:"html_url"`
			Path    string
		}
		v, ok := root["items"]
		if !ok || json.Unmarshal(v, &rows) != nil {
			return nil, "", errors.New("missing items")
		}
		for _, r := range rows {
			add(r.Name, r.HTMLURL, r.Path)
		}
	case "reddit":
		var data struct {
			Children []struct {
				Data struct{ Title, Permalink, Selftext string }
			}
		}
		v, ok := root["data"]
		if !ok || json.Unmarshal(v, &data) != nil {
			return nil, "", errors.New("missing listing")
		}
		var listing map[string]json.RawMessage
		json.Unmarshal(v, &listing)
		if _, ok := listing["children"]; !ok {
			return nil, "", errors.New("missing children")
		}
		for _, r := range data.Children {
			if strings.HasPrefix(r.Data.Permalink, "/r/") {
				add(r.Data.Title, "https://www.reddit.com"+r.Data.Permalink, r.Data.Selftext)
			}
		}
	case "duckduckgo":
		var data struct {
			Heading, AbstractText, AbstractURL string
			RelatedTopics                      []struct{ Text, FirstURL string }
		}
		if _, ok := root["AbstractText"]; !ok {
			return nil, "", errors.New("missing instant answer")
		}
		if json.Unmarshal(b, &data) != nil {
			return nil, "", errors.New("invalid instant answer")
		}
		if data.AbstractText != "" {
			add(data.Heading, data.AbstractURL, data.AbstractText)
		}
		for _, r := range data.RelatedTopics {
			add(r.Text, r.FirstURL, "")
		}
	default:
		return nil, "", errors.New("unknown provider")
	}
	return out, "", nil
}

func (s *nativeScanner) Run(n number.Number, opts ScannerOptions) (interface{}, error) {
	path, _ := opts["search_list"].(string)
	if s.name != "serpapi" || path == "" || opts.GetStringEnv("PHONEINFOGA_NATIVE") != "true" {
		return s.runOne(n, opts)
	}
	queries, e := searchlist.Load(path, n.E164, n.RawLocal)
	if e != nil {
		return nil, e
	}
	max, ok := opts["max_queries"].(int)
	if !ok || max < 1 {
		max = 5
	}
	if max > 45 {
		max = 45
	}
	if len(queries) > max {
		queries = queries[:max]
	}
	pace, ok := opts["pace"].(time.Duration)
	if !ok || pace < time.Second {
		pace = time.Second
	}
	ctx := context.Background()
	if c, ok := opts["context"].(context.Context); ok {
		ctx = c
	}
	combined := SearchResponse{Status: "no_results", Note: "Custom Google query list; bounded first-page searches", Results: []SearchResult{}}
	for i, q := range queries {
		if i > 0 {
			timer := time.NewTimer(pace)
			select {
			case <-ctx.Done():
				timer.Stop()
				combined.Status = "cancelled"
				return combined, nil
			case <-timer.C:
			}
		}
		options := ScannerOptions{}
		for k, v := range opts {
			options[k] = v
		}
		options["query"] = q.Text
		raw, e := s.runOne(n, options)
		if e != nil {
			return nil, e
		}
		r := raw.(SearchResponse)
		for _, hit := range r.Results {
			hit.Query = q.Text
			hit.Category = q.Category
			combined.Results = append(combined.Results, hit)
		}
		if r.Status != "completed" && r.Status != "no_results" {
			combined.Status = r.Status
			combined.Note = r.Note
			return combined, nil
		}
	}
	if len(combined.Results) > 0 {
		combined.Status = "completed"
	}
	return combined, nil
}
