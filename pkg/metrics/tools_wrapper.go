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
		success := err == nil
		
		// Record tool call metrics
		RecordMCPToolCall(toolName, moduleName, duration, success)
		
		// Record error if any
		if err != nil {
			errorType := "unknown"
			if err.Error() != "" {
				// Try to categorize error
				errStr := strings.ToLower(err.Error())
				if strings.Contains(errStr, "not found") {
					errorType = "not_found"
				} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
					errorType = "timeout"
				} else if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "forbidden") {
					errorType = "auth_error"
				} else if strings.Contains(errStr, "invalid") {
					errorType = "invalid_input"
				} else if strings.Contains(errStr, "connection") || strings.Contains(errStr, "network") {
					errorType = "network_error"
				}
			}
			RecordMCPToolError(toolName, moduleName, errorType)
		}
		
		return result, err
	}
}

