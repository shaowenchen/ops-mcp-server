package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestWrapToolHandlerIsErrorResultDoesNotPanic(t *testing.T) {
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "index parameter is required"}},
		}, nil
	}

	result, err := WrapToolHandler(handler, "search-logs", "logs")(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected IsError result to be passed through, got %+v", result)
	}
}

func TestWrapToolHandlerPassesThroughError(t *testing.T) {
	wantErr := errors.New("backend timeout")
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return nil, wantErr
	}

	result, err := WrapToolHandler(handler, "query-metrics", "metrics")(context.Background(), mcp.CallToolRequest{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		errMsg string
		want   string
	}{
		{"tool not found", "not_found"},
		{"context deadline exceeded", "timeout"},
		{"request timeout", "timeout"},
		{"forbidden", "auth_error"},
		{"invalid start_time format", "invalid_input"},
		{"connection refused", "network_error"},
		{"something else", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		if got := classifyError(tt.errMsg); got != tt.want {
			t.Errorf("classifyError(%q) = %q, want %q", tt.errMsg, got, tt.want)
		}
	}
}

func TestErrorTextFromResult(t *testing.T) {
	result := &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: "boom"},
			&mcp.TextContent{Type: "text", Text: "timeout"},
		},
	}
	if got := errorTextFromResult(result); got != "boom timeout " {
		t.Errorf("unexpected text %q", got)
	}
	if got := errorTextFromResult(nil); got != "" {
		t.Errorf("expected empty text for nil result, got %q", got)
	}
}

// ensure the wrapper measures a duration without needing the metrics system
func TestWrapToolHandlerWithoutMetricsInit(t *testing.T) {
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		time.Sleep(time.Millisecond)
		return &mcp.CallToolResult{Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "ok"}}}, nil
	}
	if _, err := WrapToolHandler(handler, "list-metrics", "metrics")(context.Background(), mcp.CallToolRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
