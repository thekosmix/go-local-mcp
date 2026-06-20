package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	// Register Tool 4: Online Code Compiler and Executor
	mcp.AddTool(server, &mcp.Tool{
		Name:        "code_compiler",
		Description: "Sends programming code to a hosted PHP compiler server to execute and return the stdout/stderr or compilation errors.",
	}, HandleCodeCompilerExecution)

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
