package logs

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNormalizeCatFormat(t *testing.T) {
	tests := map[string]string{
		"":       "json",
		"json":   "json",
		"JSON":   "json",
		"table":  "text",
		"text":   "text",
		"yaml":   "yaml",
		"table ": "text",
	}

	for input, want := range tests {
		if got := normalizeCatFormat(input); got != want {
			t.Errorf("normalizeCatFormat(%q) = %q, want %q", input, got, want)
		}
	}
}

// The version probe must not break tools on clusters where GET / is blocked;
// the call should still go through with the modern parameters.
func TestListIndicesUsesExpandWildcardsAndLocalStatusFilter(t *testing.T) {
	var gotPath string
	var gotQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			io.WriteString(w, `{"version":{"number":"8.19.0"}}`)
		case r.URL.Path == "/_cat/indices":
			gotPath = r.URL.Path
			gotQuery = r.URL.Query()
			io.WriteString(w, `[
				{"health":"green","status":"open","index":"a"},
				{"health":"green","status":"close","index":"b"},
				{"health":"green","status":"open","index":"c"}
			]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	result, err := m.handleListIndices(context.Background(), callRequest(map[string]any{
		"format": "json",
		"status": "open",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("unexpected error result: %#v", result)
	}

	if gotPath != "/_cat/indices" {
		t.Errorf("path = %q", gotPath)
	}
	if got := gotQuery.Get("format"); got != "json" {
		t.Errorf("format = %q, want json", got)
	}
	if got := gotQuery.Get("expand_wildcards"); got != "open" {
		t.Errorf("expand_wildcards = %q, want open", got)
	}
	if gotQuery.Has("status") {
		t.Errorf("status must not be sent as a query parameter, got %q", gotQuery.Encode())
	}
}

func TestListIndicesTableFormatMapsToText(t *testing.T) {
	var gotFormat string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			io.WriteString(w, `{"version":{"number":"8.19.0"}}`)
		case "/_cat/indices":
			gotFormat = r.URL.Query().Get("format")
			io.WriteString(w, "green open a 1 0 1 0 1kb 1kb\n")
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	if _, err := m.handleListIndices(context.Background(), callRequest(map[string]any{"format": "table"})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFormat != "text" {
		t.Errorf("format = %q, want text", gotFormat)
	}
}

// 6.x has no expand_wildcards on _cat/indices, so the parameter must be
// omitted and the filtering done locally.
func TestListIndicesOn6xOmitsExpandWildcards(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			io.WriteString(w, `{"version":{"number":"6.8.23"}}`)
		case "/_cat/indices":
			gotQuery = r.URL.Query()
			io.WriteString(w, `[{"status":"open","index":"a"},{"status":"close","index":"b"}]`)
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	result, err := m.handleListIndices(context.Background(), callRequest(map[string]any{
		"format": "json",
		"status": "close",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery.Has("expand_wildcards") {
		t.Errorf("expand_wildcards must be omitted on 6.x, got %q", gotQuery.Encode())
	}
	if result == nil || result.IsError {
		t.Fatalf("unexpected error result: %#v", result)
	}
}

func TestListIndicesSupportsIndexPattern(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			io.WriteString(w, `{"version":{"number":"8.19.0"}}`)
		case r.URL.Path == "/_cat/indices/logs-*":
			gotPath = r.URL.Path
			io.WriteString(w, `[{"status":"open","index":"logs-1"}]`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	if _, err := m.handleListIndices(context.Background(), callRequest(map[string]any{
		"format": "json",
		"index":  "logs-*",
	})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/_cat/indices/logs-*" {
		t.Errorf("path = %q, want /_cat/indices/logs-*", gotPath)
	}
}

func TestGetShardsFiltersStateLocally(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/":
			io.WriteString(w, `{"version":{"number":"8.19.0"}}`)
		case r.URL.Path == "/_cat/shards":
			gotQuery = r.URL.Query()
			io.WriteString(w, `[{"index":"a","state":"STARTED"},{"index":"b","state":"UNASSIGNED"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	m := newTestModule(t, server.URL)
	result, err := m.handleGetShards(context.Background(), callRequest(map[string]any{
		"format": "json",
		"state":  "STARTED",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("unexpected error result: %#v", result)
	}
	if got := gotQuery.Get("s"); got != "state" {
		t.Errorf("s = %q, want state (sort column, not a filter)", got)
	}
}
