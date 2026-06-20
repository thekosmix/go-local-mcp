package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CodeCompilerInput struct {
	Language string `json:"language" jsonschema:"The programming language or file extension (e.g. py, cpp, c, java, js, php, rb, go, bash)"`
	Code     string `json:"code" jsonschema:"The source code to compile and run"`
	Input    string `json:"input" jsonschema:"Optional stdin input to provide to the program during execution"`
}

type CodeCompilerOutput struct {
	Output string `json:"output" jsonschema:"The stdout/stderr output or error messages from compilation and execution"`
}

func normalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "python", "python3", "py":
		return "py"
	case "perl", "pl":
		return "pl"
	case "javascript", "js", "node", "nodejs":
		return "js"
	case "c++", "cpp", "cc", "cxx":
		return "cpp"
	case "golang", "go":
		return "go"
	case "ruby", "rb":
		return "rb"
	case "bash", "sh":
		return "bash"
	case "pascal", "pas":
		return "pas"
	case "haskell", "hs":
		return "hs"
	case "fortran", "f":
		return "f"
	case "cobol", "cob":
		return "cob"
	case "brainfuck", "bf":
		return "bf"
	default:
		return lang
	}
}

func HandleCodeCompilerExecution(ctx context.Context, req *mcp.CallToolRequest, input CodeCompilerInput) (*mcp.CallToolResult, CodeCompilerOutput, error) {
	if input.Language == "" {
		return &mcp.CallToolResult{IsError: true}, CodeCompilerOutput{Output: "Error: language field is required"}, fmt.Errorf("missing language argument")
	}
	if input.Code == "" {
		return &mcp.CallToolResult{IsError: true}, CodeCompilerOutput{Output: "Error: code field is required"}, fmt.Errorf("missing code argument")
	}

	input.Language = normalizeLanguage(input.Language)

	switch input.Language {
	case "awk", "bash", "c", "cpp", "php", "py", "pl":
		// Supported
	default:
		errMsg := fmt.Sprintf("Error: Unsupported language '%s'. Supported languages are: awk, bash, c, cpp, php, py, pl.", input.Language)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: errMsg,
				},
			},
		}, CodeCompilerOutput{Output: errMsg}, fmt.Errorf("unsupported language: %s", input.Language)
	}

	apiURL := "https://phantom-disloyal-similarly.ngrok-free.dev/online_compiler/create_file.php"

	formData := url.Values{}
	formData.Set("prog_lang", input.Language)
	formData.Set("example_1", input.Code)
	formData.Set("input", input.Input)
	formData.Set("title", "Main")
	formData.Set("comment", "MCP_Code_Compiler")

	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Use a custom Redirect policy to propagate User-Agent and ngrok-skip-browser-warning headers
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// Sanitize the upcoming request URL (e.g. escaping raw spaces from Location headers)
			if req.URL != nil {
				req.URL.RawQuery = strings.ReplaceAll(req.URL.RawQuery, " ", "%20")
			}
			// Propagate custom headers to redirect requests on the same host
			if len(via) > 0 {
				req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
				req.Header.Set("ngrok-skip-browser-warning", "69420")
			}
			return nil
		},
	}

	httpReq, err := http.NewRequestWithContext(fetchCtx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, CodeCompilerOutput{Output: "Error creating request"}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	httpReq.Header.Set("ngrok-skip-browser-warning", "69420")

	resp, err := client.Do(httpReq)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, CodeCompilerOutput{Output: fmt.Sprintf("Error performing request: %v", err)}, err
	}
	if resp == nil || resp.Body == nil {
		return &mcp.CallToolResult{IsError: true}, CodeCompilerOutput{Output: "Error: empty response payload"}, fmt.Errorf("empty response")
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{IsError: true}, CodeCompilerOutput{Output: fmt.Sprintf("Error reading response body: %v", err)}, err
	}

	htmlContent := string(bodyBytes)

	// Extract content from between <h3>Output</h3>\s*<pre> and </pre>
	reOutput := regexp.MustCompile(`(?s)<h3>Output</h3>\s*<pre>(.*?)</pre>`)
	matches := reOutput.FindStringSubmatch(htmlContent)

	var outputText string
	if len(matches) > 1 {
		outputText = strings.TrimSpace(matches[1])
	} else {
		finalURL := "unknown"
		if resp.Request != nil {
			finalURL = resp.Request.URL.String()
		}
		snippet := htmlContent
		if len(snippet) > 1000 {
			snippet = snippet[:1000] + "... [Truncated]"
		}
		if strings.Contains(htmlContent, "ngrok-browser-warning") {
			outputText = fmt.Sprintf("Error: Bypassing ngrok warning failed.\nFinal URL: %s\nStatus Code: %d\nBody Snippet:\n%s", finalURL, resp.StatusCode, snippet)
		} else {
			outputText = fmt.Sprintf("Error: Unable to find output in the response.\nFinal URL: %s\nStatus Code: %d\nBody Snippet:\n%s", finalURL, resp.StatusCode, snippet)
		}
	}

	toolResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: outputText,
			},
		},
	}

	return toolResult, CodeCompilerOutput{Output: outputText}, nil
}
