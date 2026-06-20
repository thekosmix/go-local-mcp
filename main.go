package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
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

type ReadFileInput struct {
	FilePath string `json:"file_path" jsonschema:"The absolute path to the local text or markdown file to read"`
}

type ReadFileOutput struct {
	Content string `json:"content" jsonschema:"The chunked text content extracted from the file"`
}

type FetchURLInput struct {
	URL string `json:"url" jsonschema:"The web page URL link to read and parse into clean text"`
}

type FetchURLOutput struct {
	Content string `json:"content" jsonschema:"The clean readable plain text from the webpage body"`
}

func main() {
	serverInfo := &mcp.Implementation{
		Name:    "go-unified-mcp-suite",
		Version: "1.0.0",
	}
	server := mcp.NewServer(serverInfo, nil)

	// Register Tool 1: Web Search
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search",
		Description: "Queries DuckDuckGo HTML endpoint to get text search results for web lookups.",
	}, HandleSearchExecution)

	// Register Tool 2: Document RAG Reader
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_local_document",
		Description: "Reads and chunks local text or markdown documents safely to preserve model context limits.",
	}, HandleReadFileExecution)

	// Register Tool 3: Web Page Content Fetcher
	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetch_url_content",
		Description: "Downloads a web page and strips out script, style, and HTML elements to extract raw readable text.",
	}, HandleFetchURLExecution)

	httpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})

	globalCORS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, accept, mcp-protocol-version")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Location")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var reqBody string
		if r.Method == "POST" && r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				reqBody = string(bodyBytes)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		startTime := time.Now()
		httpHandler.ServeHTTP(w, r)
		duration := time.Since(startTime)

		if r.Method == "POST" {
			log.Printf("📥 [REQ BODY]: %s | ⏱️ [RESP TIME]: %v\n", strings.TrimSpace(reqBody), duration)
		}
	})

	log.Println("🚀 Low-footprint Unified Go MCP Suite active on port :8080")
	if err := http.ListenAndServe("0.0.0.0:8080", globalCORS); err != nil {
		log.Fatal(err)
	}
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

func HandleReadFileExecution(ctx context.Context, req *mcp.CallToolRequest, input ReadFileInput) (*mcp.CallToolResult, ReadFileOutput, error) {
	if input.FilePath == "" {
		return &mcp.CallToolResult{IsError: true}, ReadFileOutput{Content: "Error: empty path"}, fmt.Errorf("missing path")
	}

	file, err := os.Open(input.FilePath)
	if err != nil {
		errMsg := fmt.Sprintf("Error opening local file: %v", err)
		return &mcp.CallToolResult{IsError: true}, ReadFileOutput{Content: errMsg}, err
	}
	defer file.Close()

	limitReader := io.LimitReader(file, 15000)
	contentBytes, err := io.ReadAll(limitReader)
	if err != nil {
		errMsg := fmt.Sprintf("Error reading stream layout: %v", err)
		return &mcp.CallToolResult{IsError: true}, ReadFileOutput{Content: errMsg}, err
	}

	outputText := string(contentBytes)
	if len(outputText) == 15000 {
		outputText += "\n\n[!!! SYSTEM NOTICE: Text truncated by MCP layer to preserve token limits !!!]"
	}

	toolResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: outputText,
			},
		},
	}
	return toolResult, ReadFileOutput{Content: outputText}, nil
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

