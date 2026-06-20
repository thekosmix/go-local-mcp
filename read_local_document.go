package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ReadFileInput struct {
	FilePath string `json:"file_path" jsonschema:"The absolute path to the local text or markdown file to read"`
}

type ReadFileOutput struct {
	Content string `json:"content" jsonschema:"The chunked text content extracted from the file"`
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
