package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchInput struct {
	Query string `json:"query" jsonschema:"The search keywords or terms requested"`
}

type SearchOutput struct {
	Results string `json:"results" jsonschema:"The raw text snippets extracted from the web"`
}

func HandleSearchExecution(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
	if input.Query == "" {
		return &mcp.CallToolResult{IsError: true}, SearchOutput{Results: "Error: empty query"}, fmt.Errorf("missing query argument")
	}

	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(input.Query)

	fetchCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(fetchCtx, "GET", searchURL, nil)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, SearchOutput{Results: "Error setting up request"}, err
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, SearchOutput{Results: "Network timeout or connection dropped"}, err
	}
	if resp == nil || resp.Body == nil {
		return &mcp.CallToolResult{IsError: true}, SearchOutput{Results: "Received an empty response payload"}, fmt.Errorf("upstream response structure is empty")
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, SearchOutput{Results: "Failed parsing upstream data"}, err
	}
	rawHTML := string(bodyBytes)

	var snippets []string
	idx := 0
	maxResults := 5
	for i := 0; i < maxResults; i++ {
		startIdx := strings.Index(rawHTML[idx:], "class=\"result__snippet\"")
		if startIdx == -1 {
			break
		}
		absoluteStart := idx + startIdx
		textStart := strings.Index(rawHTML[absoluteStart:], ">") + absoluteStart + 1
		textEnd := strings.Index(rawHTML[textStart:], "</a>") + textStart

		if textEnd > textStart && textEnd < len(rawHTML) {
			snippet := rawHTML[textStart:textEnd]
			snippet = strings.ReplaceAll(snippet, "<b>", "")
			snippet = strings.ReplaceAll(snippet, "</b>", "")
			snippets = append(snippets, strings.TrimSpace(snippet))
		}
		idx = textEnd
	}

	resultText := strings.Join(snippets, "\n\n")
	if resultText == "" {
		resultText = "No direct web summaries found for this term."
	}

	toolResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: resultText,
			},
		},
	}

	return toolResult, SearchOutput{Results: resultText}, nil
}
