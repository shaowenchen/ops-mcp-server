package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// WrapToolHandler wraps a tool handler with metrics collection
func WrapToolHandler(handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), toolName, moduleName string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		// Record module request
		RecordModuleRequest(moduleName)

		// Call the actual handler
		result, err := handler(ctx, request)

		duration := time.Since(start)
		// Success is determined by: no error AND result is not marked as error
		success := err == nil && (result == nil || !result.IsError)

		// Record tool call metrics
		RecordMCPToolCall(toolName, moduleName, duration, success)

		// Record error if any (either handler returned error or result has IsError=true)
		if err != nil || (result != nil && result.IsError) {
			// Prefer the returned error; fall back to the text content of an
			// IsError result, which is how some modules report tool failures.
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg = errorTextFromResult(result)
			}
			RecordMCPToolError(toolName, moduleName, classifyError(errMsg))
		}

		return result, err
	}
}

// errorTextFromResult extracts the text content of a tool result, which is how
// handlers that return IsError=true with a nil error report what went wrong.
func errorTextFromResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	var sb strings.Builder
	for _, content := range result.Content {
		switch c := content.(type) {
		case mcp.TextContent:
			sb.WriteString(c.Text)
			sb.WriteString(" ")
		case *mcp.TextContent:
			sb.WriteString(c.Text)
			sb.WriteString(" ")
		}
	}
	return sb.String()
}

// classifyError maps an error message to a coarse error_type label.
func classifyError(errMsg string) string {
	errorType := "unknown"
	errStr := strings.ToLower(errMsg)
	switch {
	case strings.Contains(errStr, "not found"):
		errorType = "not_found"
	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline"):
		errorType = "timeout"
	case strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "forbidden"):
		errorType = "auth_error"
	case strings.Contains(errStr, "invalid"):
		errorType = "invalid_input"
	case strings.Contains(errStr, "connection") || strings.Contains(errStr, "network"):
		errorType = "network_error"
	}
	return errorType
}
