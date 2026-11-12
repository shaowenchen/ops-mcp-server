package logs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

func newTestModule(t *testing.T, endpoint string) *Module {
	t.Helper()
	resetVersionCache()
	m, err := New(&Config{
		Elasticsearch: &ElasticsearchConfig{Endpoint: endpoint},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("failed to create module: %v", err)
	}
	return m
}

func callRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

func TestParseESVersion(t *testing.T) {
	tests := []struct {
		raw          string
		major, minor int
	}{
		{"8.19.0", 8, 19},
		{"7.17.9", 7, 17},
		{"6.8.23", 6, 8},
		{"8.11.0-SNAPSHOT", 8, 11},
		{"9.0.0", 9, 0},
	}

	for _, tt := range tests {
		version, err := parseESVersion(tt.raw)
		if err != nil {
			t.Fatalf("parseESVersion(%q) error: %v", tt.raw, err)
		}
		if version.Major != tt.major || version.Minor != tt.minor {
			t.Errorf("parseESVersion(%q) = %d.%d, want %d.%d", tt.raw, version.Major, version.Minor, tt.major, tt.minor)
		}
		if version.Raw != tt.raw {
			t.Errorf("Raw = %q, want %q", version.Raw, tt.raw)
		}
	}

	for _, raw := range []string{"", "8", "abc.def", "8.x"} {
		if _, err := parseESVersion(raw); err == nil {
			t.Errorf("parseESVersion(%q) expected error", raw)
		}
	}
}

func TestVersionCapabilities(t *testing.T) {
	tests := []struct {
		raw     string
		esql    bool
		catWild bool
	}{
		{"6.8.23", false, false},
		{"7.10.2", false, true},
		{"8.10.4", false, true},
		{"8.11.0", true, true},
		{"8.19.0", true, true},
		{"9.1.0", true, true},
	}

	for _, tt := range tests {
		version, err := parseESVersion(tt.raw)
		if err != nil {
			t.Fatalf("parseESVersion(%q) error: %v", tt.raw, err)
		}
		if got := version.supportsESQL(); got != tt.esql {
			t.Errorf("%s supportsESQL() = %v, want %v", tt.raw, got, tt.esql)
		}
		if got := version.supportsCatExpandWildcards(); got != tt.catWild {
			t.Errorf("%s supportsCatExpandWildcards() = %v, want %v", tt.raw, got, tt.catWild)
		}
	}
}

func TestClusterVersionIsDetectedAndCached(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/" {
			t.Errorf("expected probe on /, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"version":{"number":"8.19.0"},"tagline":"You Know, for Search"}`)
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)

	for i := 0; i < 3; i++ {
		version, known := m.clusterVersion(context.Background())
		if !known {
			t.Fatalf("expected version to be known")
		}
		if version.Raw != "8.19.0" {
			t.Errorf("version = %q, want 8.19.0", version.Raw)
		}
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("probe called %d times, want 1 (cache should be reused)", got)
	}
}

func TestClusterVersionUnknownOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"unauthorized"}`)
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	if _, known := m.clusterVersion(context.Background()); known {
		t.Errorf("expected version to be unknown for an error response")
	}
}

func TestESQLRowsNonColumnar(t *testing.T) {
	resp := ESQLResponse{
		Columns: []ESQLColumn{{Name: "level"}, {Name: "count"}},
		Values: [][]interface{}{
			{"error", 3},
			{"info", 10},
		},
	}

	rows, err := esqlRows(resp, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0]["level"] != "error" || rows[0]["count"] != 3 {
		t.Errorf("unexpected first row: %#v", rows[0])
	}
}

func TestESQLRowsColumnar(t *testing.T) {
	// Columnar responses return one array per column.
	resp := ESQLResponse{
		Columns: []ESQLColumn{{Name: "level"}, {Name: "count"}},
		Values: [][]interface{}{
			{"error", "info"},
			{3, 10},
		},
	}

	rows, err := esqlRows(resp, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0]["level"] != "error" || rows[0]["count"] != 3 {
		t.Errorf("unexpected first row: %#v", rows[0])
	}
	if rows[1]["level"] != "info" || rows[1]["count"] != 10 {
		t.Errorf("unexpected second row: %#v", rows[1])
	}
}

func TestFilterIndicesByStatus(t *testing.T) {
	indices := []ElasticsearchIndex{
		{Index: "a", Status: "open"},
		{Index: "b", Status: "close"},
		{Index: "c", Status: "open"},
	}

	open := filterIndicesByStatus(indices, "open")
	if len(open) != 2 || open[0].Index != "a" || open[1].Index != "c" {
		t.Errorf("unexpected open indices: %#v", open)
	}

	closed := filterIndicesByStatus(indices, "close")
	if len(closed) != 1 || closed[0].Index != "b" {
		t.Errorf("unexpected closed indices: %#v", closed)
	}

	if got := filterIndicesByStatus(indices, ""); len(got) != 3 {
		t.Errorf("empty filter should return all, got %d", len(got))
	}
}

func TestFilterShardsByState(t *testing.T) {
	shards := []ElasticsearchShard{
		{Index: "a", State: "STARTED"},
		{Index: "b", State: "UNASSIGNED"},
	}

	started := filterShardsByState(shards, "started")
	if len(started) != 1 || started[0].Index != "a" {
		t.Errorf("unexpected shards: %#v", started)
	}
}

// handleESQL must use POST /_query, pass format as a URL parameter, and keep
// the columnar flag in the body.
func TestHandleESQLRequestShape(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotFormat, gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			io.WriteString(w, `{"version":{"number":"8.19.0"}}`)
			return
		case r.URL.Path == "/_query":
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotQuery = r.URL.RawQuery
			gotFormat = r.URL.Query().Get("format")
			b, _ := io.ReadAll(r.Body)
			gotBody = string(b)
			io.WriteString(w, `{"columns":[{"name":"level","type":"keyword"}],"values":[["error"]]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	request := callRequest(map[string]any{
		"query":    "FROM logs-* | STATS count() BY level",
		"format":   "csv",
		"columnar": "true",
	})

	result, err := m.handleESQL(context.Background(), request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("unexpected error result: %#v", result)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/_query" {
		t.Errorf("path = %s, want /_query", gotPath)
	}
	if gotFormat != "csv" {
		t.Errorf("format query param = %q, want csv", gotFormat)
	}
	if strings.Contains(gotQuery, "columnar") {
		t.Errorf("columnar must stay in the body, got query %q", gotQuery)
	}
	if !strings.Contains(gotBody, `"columnar":true`) {
		t.Errorf("body missing columnar flag: %s", gotBody)
	}
	if strings.Contains(gotBody, `"format"`) {
		t.Errorf("format must not be sent in the body: %s", gotBody)
	}
}

func TestHandleESQLReportsUnsupportedCluster(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			io.WriteString(w, `{"version":{"number":"7.17.9"}}`)
		case "/_query":
			t.Errorf("_query must not be called on a 7.x cluster")
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	result, err := m.handleESQL(context.Background(), callRequest(map[string]any{"query": "FROM logs-*"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected an error result, got %#v", result)
	}
	body, _ := json.Marshal(result)
	if !strings.Contains(string(body), "requires 8.11 or later") {
		t.Errorf("expected friendly message, got %s", body)
	}
}

func TestHandleESQLNoHandlerFoundMapsToFriendlyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// Version probe fails so the handler falls back to trying _query.
			w.WriteHeader(http.StatusInternalServerError)
		case "/_query":
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":"no handler found for uri [/_query] and method [POST]"}`)
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	result, err := m.handleESQL(context.Background(), callRequest(map[string]any{"query": "FROM logs-*"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected an error result, got %#v", result)
	}
	body, _ := json.Marshal(result)
	if !strings.Contains(string(body), "requires 8.11 or later") {
		t.Errorf("expected friendly message, got %s", body)
	}
}
