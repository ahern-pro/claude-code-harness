package tools

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/li-zeyuan/claude-code-harness/utils"
)

const (
	defaultMaxResults     = 5
	defaultSearchEndpoint = "https://html.duckduckgo.com/html/"
	defaultSearchTimeout  = 20 * time.Second
	webSearchUserAgent    = "OpenHarness-Go/0.1"
)

type WebSearchToolInput struct {
	Query      string `json:"query" validate:"required"`
	MaxResults int    `json:"max_results" validate:"min=1,max=10"`
	SearchURL  string `json:"search_url,omitempty"`
}

type WebSearchTool struct {
	BaseTool
	httpClient *http.Client
}

func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		BaseTool: BaseTool{
			Name:        "web_search",
			Description: "Search the web and return compact top results with titles, URLs, and snippets.",
			InputModel:  map[string]interface{}{},
		},
		httpClient: &http.Client{Timeout: defaultSearchTimeout},
	}
}

func (wst *WebSearchTool) Name() string {
	return wst.BaseTool.Name
}

func (wst *WebSearchTool) IsReadOnly() bool {
	return true
}

func (wst *WebSearchTool) Validate(input map[string]any) (any, error) {
	args := &WebSearchToolInput{MaxResults: defaultMaxResults}

	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}
	if err := json.Unmarshal(data, args); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	if err := utils.Validator.Struct(args); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return args, nil
}

func (wst *WebSearchTool) Execute(input any, ctx *ToolExecutionContext) *ToolResult {
	var args *WebSearchToolInput
	switch v := input.(type) {
	case *WebSearchToolInput:
		if v == nil {
			return &ToolResult{Output: "Invalid input: nil", IsError: true}
		}
		args = v
	case WebSearchToolInput:
		args = &v
	default:
		return &ToolResult{Output: fmt.Sprintf("Invalid input: %T", input), IsError: true}
	}

	endpoint := args.SearchURL
	if endpoint == "" {
		endpoint = defaultSearchEndpoint
	}
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}

	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("web_search failed: %s", err), IsError: true}
	}
	q := reqURL.Query()
	q.Set("q", args.Query)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("web_search failed: %s", err), IsError: true}
	}
	req.Header.Set("User-Agent", webSearchUserAgent)

	client := wst.httpClient
	if client == nil {
		client = &http.Client{Timeout: defaultSearchTimeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("web_search failed: %s", err), IsError: true}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &ToolResult{
			Output:  fmt.Sprintf("web_search failed: HTTP %d", resp.StatusCode),
			IsError: true,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{Output: fmt.Sprintf("web_search failed: %s", err), IsError: true}
	}

	results := parseSearchResults(string(body), maxResults)
	if len(results) == 0 {
		return &ToolResult{Output: "No search results found.", IsError: true}
	}

	lines := []string{fmt.Sprintf("Search results for: %s", args.Query)}
	for i, r := range results {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, r.Title))
		lines = append(lines, fmt.Sprintf("   URL: %s", r.URL))
		if r.Snippet != "" {
			lines = append(lines, fmt.Sprintf("   %s", r.Snippet))
		}
	}
	return &ToolResult{Output: strings.Join(lines, "\n")}
}

type webSearchResult struct {
	Title   string
	URL     string
	Snippet string
}

var (
	snippetRE = regexp.MustCompile(
		`(?is)<(?:a|div|span)[^>]+class="[^"]*(?:result__snippet|result-snippet)[^"]*"[^>]*>(.*?)</(?:a|div|span)>`,
	)
	anchorRE = regexp.MustCompile(`(?is)<a([^>]+)>(.*?)</a>`)
	classRE  = regexp.MustCompile(`(?i)class="([^"]+)"`)
	hrefRE   = regexp.MustCompile(`(?i)href="([^"]+)"`)
	tagRE    = regexp.MustCompile(`(?s)<[^>]+>`)
	wsRE     = regexp.MustCompile(`\s+`)
)

func parseSearchResults(body string, limit int) []webSearchResult {
	snippetMatches := snippetRE.FindAllStringSubmatch(body, -1)
	snippets := make([]string, len(snippetMatches))
	for i, m := range snippetMatches {
		snippets[i] = cleanHTMLFragment(m[1])
	}

	var results []webSearchResult
	anchorMatches := anchorRE.FindAllStringSubmatch(body, -1)
	for i, m := range anchorMatches {
		attrs := m[1]
		cm := classRE.FindStringSubmatch(attrs)
		if cm == nil {
			continue
		}
		classes := cm[1]
		if !strings.Contains(classes, "result__a") && !strings.Contains(classes, "result-link") {
			continue
		}
		hm := hrefRE.FindStringSubmatch(attrs)
		if hm == nil {
			continue
		}
		title := cleanHTMLFragment(m[2])
		u := normalizeResultURL(hm[1])
		snippet := ""
		if i < len(snippets) {
			snippet = snippets[i]
		}
		if title != "" && u != "" {
			results = append(results, webSearchResult{Title: title, URL: u, Snippet: snippet})
		}
		if len(results) >= limit {
			break
		}
	}
	return results
}

func normalizeResultURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if strings.HasSuffix(parsed.Host, "duckduckgo.com") && strings.HasPrefix(parsed.Path, "/l/") {
		target := parsed.Query().Get("uddg")
		if target == "" {
			return rawURL
		}
		decoded, err := url.QueryUnescape(target)
		if err != nil {
			return target
		}
		return decoded
	}
	return rawURL
}

func cleanHTMLFragment(fragment string) string {
	text := tagRE.ReplaceAllString(fragment, " ")
	text = html.UnescapeString(text)
	text = wsRE.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
