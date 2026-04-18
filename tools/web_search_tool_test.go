package tools

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebSearchTool_Metadata(t *testing.T) {
	tool := NewWebSearchTool()

	if got := tool.Name(); got != "web_search" {
		t.Errorf("Name() = %q, want %q", got, "web_search")
	}
	if !tool.IsReadOnly() {
		t.Error("IsReadOnly() = false, want true")
	}
	if tool.httpClient == nil {
		t.Error("httpClient is nil, want non-nil default client")
	}
}

func TestWebSearchTool_Validate(t *testing.T) {
	tool := NewWebSearchTool()

	tests := []struct {
		name           string
		input          map[string]any
		wantErr        bool
		wantQuery      string
		wantMaxResults int
		wantSearchURL  string
	}{
		{
			name:           "valid with defaults",
			input:          map[string]any{"query": "golang testing"},
			wantQuery:      "golang testing",
			wantMaxResults: defaultMaxResults,
		},
		{
			name:           "valid with max_results",
			input:          map[string]any{"query": "q", "max_results": 3},
			wantQuery:      "q",
			wantMaxResults: 3,
		},
		{
			name:           "valid with max_results as float",
			input:          map[string]any{"query": "q", "max_results": float64(7)},
			wantQuery:      "q",
			wantMaxResults: 7,
		},
		{
			name:           "valid with search_url override",
			input:          map[string]any{"query": "q", "search_url": "https://example.test/search"},
			wantQuery:      "q",
			wantMaxResults: defaultMaxResults,
			wantSearchURL:  "https://example.test/search",
		},
		{
			name:    "missing query",
			input:   map[string]any{"max_results": 5},
			wantErr: true,
		},
		{
			name:    "empty query",
			input:   map[string]any{"query": ""},
			wantErr: true,
		},
		{
			name:    "max_results below minimum",
			input:   map[string]any{"query": "q", "max_results": 0},
			wantErr: true,
		},
		{
			name:    "max_results above maximum",
			input:   map[string]any{"query": "q", "max_results": 11},
			wantErr: true,
		},
		{
			name:    "invalid query type",
			input:   map[string]any{"query": 123},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tool.Validate(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (args=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			args, ok := got.(*WebSearchToolInput)
			if !ok {
				t.Fatalf("Validate returned %T, want *WebSearchToolInput", got)
			}
			if args.Query != tc.wantQuery {
				t.Errorf("Query = %q, want %q", args.Query, tc.wantQuery)
			}
			if args.MaxResults != tc.wantMaxResults {
				t.Errorf("MaxResults = %d, want %d", args.MaxResults, tc.wantMaxResults)
			}
			if args.SearchURL != tc.wantSearchURL {
				t.Errorf("SearchURL = %q, want %q", args.SearchURL, tc.wantSearchURL)
			}
		})
	}
}

// ddgFixture mimics a simplified DuckDuckGo HTML result page with two results.
const ddgFixture = `
<html><body>
<div class="results">
  <div class="result">
    <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Ffoo&amp;rut=abc">Example &amp; Foo</a>
    <div class="result__snippet">Some <b>foo</b>   snippet text.</div>
  </div>
  <div class="result">
    <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fbar">Example Bar</a>
    <div class="result__snippet">Bar snippet.</div>
  </div>
  <div class="result">
    <a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fbaz">Example Baz</a>
    <div class="result__snippet">Baz snippet.</div>
  </div>
</div>
</body></html>`

func TestWebSearchTool_Execute_Success(t *testing.T) {
	var gotQuery, gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, ddgFixture)
	}))
	defer server.Close()

	tool := NewWebSearchTool()
	result := tool.Execute(&WebSearchToolInput{
		Query:      "openharness",
		MaxResults: 5,
		SearchURL:  server.URL,
	}, &ToolExecutionContext{})

	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	if gotQuery != "openharness" {
		t.Errorf("server got q=%q, want %q", gotQuery, "openharness")
	}
	if gotUA != webSearchUserAgent {
		t.Errorf("server got User-Agent=%q, want %q", gotUA, webSearchUserAgent)
	}

	want := strings.Join([]string{
		"Search results for: openharness",
		"1. Example & Foo",
		"   URL: https://example.com/foo",
		"   Some foo snippet text.",
		"2. Example Bar",
		"   URL: https://example.com/bar",
		"   Bar snippet.",
		"3. Example Baz",
		"   URL: https://example.com/baz",
		"   Baz snippet.",
	}, "\n")
	if result.Output != want {
		t.Errorf("Output mismatch.\n got: %q\nwant: %q", result.Output, want)
	}
}

func TestWebSearchTool_Execute_MaxResultsLimitsOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, ddgFixture)
	}))
	defer server.Close()

	tool := NewWebSearchTool()
	result := tool.Execute(&WebSearchToolInput{
		Query:      "q",
		MaxResults: 2,
		SearchURL:  server.URL,
	}, &ToolExecutionContext{})

	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	if strings.Contains(result.Output, "Baz") {
		t.Errorf("expected only 2 results, but output contains third: %q", result.Output)
	}
	if !strings.Contains(result.Output, "1. Example & Foo") {
		t.Errorf("expected first result in output, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "2. Example Bar") {
		t.Errorf("expected second result in output, got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_NoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html><body>no results here</body></html>")
	}))
	defer server.Close()

	tool := NewWebSearchTool()
	result := tool.Execute(&WebSearchToolInput{
		Query:     "nothing",
		SearchURL: server.URL,
	}, &ToolExecutionContext{})

	if !result.IsError {
		t.Fatal("expected IsError=true when no results are parsed")
	}
	if !strings.Contains(result.Output, "No search results found") {
		t.Errorf("expected 'No search results found' message, got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_HTTPErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	tool := NewWebSearchTool()
	result := tool.Execute(&WebSearchToolInput{
		Query:     "q",
		SearchURL: server.URL,
	}, &ToolExecutionContext{})

	if !result.IsError {
		t.Fatal("expected IsError=true on HTTP 500")
	}
	if !strings.Contains(result.Output, "web_search failed") {
		t.Errorf("expected 'web_search failed' prefix, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "500") {
		t.Errorf("expected status code in output, got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_RequestError(t *testing.T) {
	tool := NewWebSearchTool()
	tool.httpClient = &http.Client{Timeout: 50 * time.Millisecond}

	result := tool.Execute(&WebSearchToolInput{
		Query:     "q",
		SearchURL: "http://127.0.0.1:1/unreachable",
	}, &ToolExecutionContext{})

	if !result.IsError {
		t.Fatal("expected IsError=true when request fails")
	}
	if !strings.Contains(result.Output, "web_search failed") {
		t.Errorf("expected 'web_search failed' prefix, got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_InvalidSearchURL(t *testing.T) {
	tool := NewWebSearchTool()

	result := tool.Execute(&WebSearchToolInput{
		Query:     "q",
		SearchURL: "://not a url",
	}, &ToolExecutionContext{})

	if !result.IsError {
		t.Fatal("expected IsError=true for invalid URL")
	}
	if !strings.Contains(result.Output, "web_search failed") {
		t.Errorf("expected 'web_search failed' prefix, got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_DefaultMaxResultsOnZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, ddgFixture)
	}))
	defer server.Close()

	tool := NewWebSearchTool()
	result := tool.Execute(&WebSearchToolInput{
		Query:     "q",
		SearchURL: server.URL,
	}, &ToolExecutionContext{})

	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	if !strings.Contains(result.Output, "1. Example & Foo") {
		t.Errorf("expected first result, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "3. Example Baz") {
		t.Errorf("expected third result with default max_results (>=5), got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_AcceptsValueInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, ddgFixture)
	}))
	defer server.Close()

	tool := NewWebSearchTool()
	result := tool.Execute(WebSearchToolInput{
		Query:      "q",
		MaxResults: 1,
		SearchURL:  server.URL,
	}, &ToolExecutionContext{})

	if result.IsError {
		t.Fatalf("unexpected error result: %s", result.Output)
	}
	if !strings.Contains(result.Output, "1. Example & Foo") {
		t.Errorf("expected first result in output, got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_InvalidInputType(t *testing.T) {
	tool := NewWebSearchTool()

	result := tool.Execute("not-a-struct", &ToolExecutionContext{})
	if !result.IsError {
		t.Fatal("expected IsError=true for invalid input type")
	}
	if !strings.Contains(result.Output, "Invalid input") {
		t.Errorf("expected 'Invalid input' message, got %q", result.Output)
	}
}

func TestWebSearchTool_Execute_NilPointerInput(t *testing.T) {
	tool := NewWebSearchTool()

	var nilInput *WebSearchToolInput
	result := tool.Execute(nilInput, &ToolExecutionContext{})
	if !result.IsError {
		t.Fatal("expected IsError=true for nil *WebSearchToolInput")
	}
}

func TestParseSearchResults(t *testing.T) {
	results := parseSearchResults(ddgFixture, 10)
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	want := []webSearchResult{
		{Title: "Example & Foo", URL: "https://example.com/foo", Snippet: "Some foo snippet text."},
		{Title: "Example Bar", URL: "https://example.com/bar", Snippet: "Bar snippet."},
		{Title: "Example Baz", URL: "https://example.com/baz", Snippet: "Baz snippet."},
	}
	for i, w := range want {
		if results[i] != w {
			t.Errorf("results[%d] = %+v, want %+v", i, results[i], w)
		}
	}
}

func TestParseSearchResults_LimitRespected(t *testing.T) {
	results := parseSearchResults(ddgFixture, 1)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Title != "Example & Foo" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Example & Foo")
	}
}

func TestParseSearchResults_ResultLinkClassAlias(t *testing.T) {
	body := `<a class="result-link" href="https://direct.example.com/x">Direct Link</a>` +
		`<div class="result-snippet">Direct snippet.</div>`
	results := parseSearchResults(body, 10)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	got := results[0]
	want := webSearchResult{Title: "Direct Link", URL: "https://direct.example.com/x", Snippet: "Direct snippet."}
	if got != want {
		t.Errorf("results[0] = %+v, want %+v", got, want)
	}
}

func TestParseSearchResults_SkipsAnchorsWithoutClass(t *testing.T) {
	body := `<a href="https://ignored.example.com">Ignored</a>` +
		`<a class="result__a" href="https://kept.example.com">Kept</a>`
	results := parseSearchResults(body, 10)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].URL != "https://kept.example.com" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://kept.example.com")
	}
}

func TestParseSearchResults_NoResults(t *testing.T) {
	results := parseSearchResults("<html><body>nothing here</body></html>", 5)
	if len(results) != 0 {
		t.Errorf("expected no results, got %d: %+v", len(results), results)
	}
}

func TestNormalizeResultURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "ddg protocol-relative redirect with uddg",
			in:   "//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Ffoo",
			want: "https://example.com/foo",
		},
		{
			name: "ddg https redirect with uddg and extra params",
			in:   "https://duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com%2Fbar&rut=abc",
			want: "https://example.com/bar",
		},
		{
			name: "ddg redirect without uddg falls back to raw",
			in:   "https://duckduckgo.com/l/?foo=bar",
			want: "https://duckduckgo.com/l/?foo=bar",
		},
		{
			name: "non-ddg url passes through",
			in:   "https://example.com/page",
			want: "https://example.com/page",
		},
		{
			name: "duckduckgo host but non /l/ path passes through",
			in:   "https://duckduckgo.com/about",
			want: "https://duckduckgo.com/about",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeResultURL(tc.in); got != tc.want {
				t.Errorf("normalizeResultURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestCleanHTMLFragment(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "strips tags and collapses whitespace",
			in:   "Some <b>bold</b>   text\nwith\t<i>italic</i>",
			want: "Some bold text with italic",
		},
		{
			name: "unescapes html entities",
			in:   "A &amp; B &lt;x&gt;",
			want: "A & B <x>",
		},
		{
			name: "trims leading and trailing whitespace",
			in:   "   hello   ",
			want: "hello",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanHTMLFragment(tc.in); got != tc.want {
				t.Errorf("cleanHTMLFragment(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
