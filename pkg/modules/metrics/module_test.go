package metrics

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func callToolRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{Arguments: args},
	}
}

func TestResolveRangeQueryWindowAbsolute(t *testing.T) {
	start, end, err := resolveRangeQueryWindow("1h", "2026-05-27T13:00:00+08:00", "2026-05-27T14:00:00+08:00")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := start.Format(time.RFC3339), "2026-05-27T13:00:00+08:00"; got != want {
		t.Errorf("start = %s, want %s", got, want)
	}
	if got, want := end.Format(time.RFC3339), "2026-05-27T14:00:00+08:00"; got != want {
		t.Errorf("end = %s, want %s", got, want)
	}
}

func TestResolveRangeQueryWindowRelative(t *testing.T) {
	start, end, err := resolveRangeQueryWindow("1h", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delta := end.Sub(start); delta != time.Hour {
		t.Errorf("window = %s, want 1h", delta)
	}
}

func TestResolveRangeQueryWindowErrors(t *testing.T) {
	tests := []struct {
		name                  string
		timeRange, start, end string
		wantErrSub            string
	}{
		{"only start", "", "2026-05-27T13:00:00+08:00", "", "together"},
		{"only end", "", "", "2026-05-27T14:00:00+08:00", "together"},
		{"end before start", "", "2026-05-27T14:00:00+08:00", "2026-05-27T13:00:00+08:00", "after start"},
		{"end equals start", "", "2026-05-27T14:00:00+08:00", "2026-05-27T14:00:00+08:00", "after start"},
		{"nothing provided", "", "", "", "required"},
		{"bad time_range", "abc", "", "", "invalid time_range"},
		{"bad start", "", "not-a-time", "2026-05-27T14:00:00+08:00", "invalid start time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := resolveRangeQueryWindow(tt.timeRange, tt.start, tt.end)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrSub)
			}
		})
	}
}

func TestParseQueryTime(t *testing.T) {
	// Both 10-digit seconds and 13-digit milliseconds must resolve to the same instant.
	want := time.Unix(1779925200, 0)
	cases := map[string]time.Time{
		"2026-05-27T13:00:00+08:00":           time.Date(2026, 5, 27, 13, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		"2026-05-27T13:00:00.123456789+08:00": time.Date(2026, 5, 27, 13, 0, 0, 123456789, time.FixedZone("CST", 8*3600)),
		"1779925200":                          want,
		"1779925200000":                       want,
	}

	for value, wantTime := range cases {
		got, err := parseQueryTime(value)
		if err != nil {
			t.Fatalf("parseQueryTime(%q) error: %v", value, err)
		}
		if !got.Equal(wantTime) {
			t.Errorf("parseQueryTime(%q) = %s, want %s", value, got, wantTime)
		}
	}

	if _, err := parseQueryTime(""); err == nil {
		t.Error("expected error for empty input")
	}
	if _, err := parseQueryTime("yesterday"); err == nil {
		t.Error("expected error for non-timestamp input")
	}
}

func TestPrometheusTimeoutDefaultsAndOverride(t *testing.T) {
	m, err := New(&Config{Tools: ToolsConfig{}}, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.httpClient.Timeout != 30*time.Second {
		t.Errorf("default timeout = %s, want 30s", m.httpClient.Timeout)
	}

	m, err = New(&Config{
		Tools:      ToolsConfig{},
		Prometheus: &PrometheusConfig{Endpoint: "http://localhost", Timeout: 90},
	}, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.httpClient.Timeout != 90*time.Second {
		t.Errorf("configured timeout = %s, want 90s", m.httpClient.Timeout)
	}
}

func TestHandleExecuteRangeQueryRejectsBadStep(t *testing.T) {
	m, err := New(&Config{
		Tools:      ToolsConfig{},
		Prometheus: &PrometheusConfig{Endpoint: "http://127.0.0.1:0"},
	}, testLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	request := callToolRequest(map[string]any{
		"query":      "up",
		"time_range": "1h",
		"step":       "not-a-step",
	})
	if _, err := m.handleExecuteRangeQuery(context.Background(), request); err == nil || !strings.Contains(err.Error(), "invalid step") {
		t.Fatalf("expected invalid step error, got %v", err)
	}
}
