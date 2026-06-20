package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type FetchURLInput struct {
	URL string `json:"url" jsonschema:"The web page URL link to read and parse into clean text"`
}

type FetchURLOutput struct {
	Content string `json:"content" jsonschema:"The clean readable plain text from the webpage body"`
}

func HandleFetchURLExecution(ctx context.Context, req *mcp.CallToolRequest, input FetchURLInput) (*mcp.CallToolResult, FetchURLOutput, error) {
	if input.URL == "" {
		return &mcp.CallToolResult{IsError: true}, FetchURLOutput{Content: "Error: empty url"}, fmt.Errorf("missing url")
	}

	fetchCtx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(fetchCtx, "GET", input.URL, nil)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, FetchURLOutput{Content: "Error initializing crawler"}, err
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, FetchURLOutput{Content: "Network connection timeout"}, err
	}
	if resp == nil || resp.Body == nil {
		return &mcp.CallToolResult{IsError: true}, FetchURLOutput{Content: "No payload"}, fmt.Errorf("empty stream")
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, FetchURLOutput{Content: "Fail stream reading"}, err
	}
	htmlContent := string(bodyBytes)

	reStyle := regexp.MustCompile(`(?s)<style.*?>.*?</style>`)
	htmlContent = reStyle.ReplaceAllString(htmlContent, "")

	reScript := regexp.MustCompile(`(?s)<script.*?>.*?</script>`)
	htmlContent = reScript.ReplaceAllString(htmlContent, "")

	reTags := regexp.MustCompile(`<.*?>`)
	plainText := reTags.ReplaceAllString(htmlContent, " ")

	reSpaces := regexp.MustCompile(`\s+`)
	cleanedText := strings.TrimSpace(reSpaces.ReplaceAllString(plainText, " "))

	if len(cleanedText) > 8000 {
		cleanedText = cleanedText[:8000] + "... [Truncated]"
	}

	if cleanedText == "" {
		cleanedText = "Failed to parse text nodes out of target webpage."
	}

	toolResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: cleanedText,
			},
		},
	}
	return toolResult, FetchURLOutput{Content: cleanedText}, nil
}
