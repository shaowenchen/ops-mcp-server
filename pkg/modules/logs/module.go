package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Config contains logs module configuration
type Config struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
}

// Module represents the logs module
type Module struct {
	config *Config
	logger *zap.Logger
}

// LogEntry represents a single log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// New creates a new logs module
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("logs config is required")
	}

	m := &Module{
		config: config,
		logger: logger.Named("logs"),
	}

	m.logger.Info("Logs module created",
		zap.String("endpoint", config.Endpoint),
	)

	return m, nil
}

// GetTools returns all MCP tools for the logs module
func GetTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Tool:    getLogsStatusToolDefinition(),
			Handler: handleGetLogsStatus,
		},
		{
			Tool:    queryLogsToolDefinition(),
			Handler: handleQueryLogs,
		},
		{
			Tool:    getLogStatsToolDefinition(),
			Handler: handleGetLogStats,
		},
		{
			Tool:    getLogServicesToolDefinition(),
			Handler: handleGetLogServices,
		},
		{
			Tool:    getLogLevelsToolDefinition(),
			Handler: handleGetLogLevels,
		},
		{
			Tool:    searchLogsToolDefinition(),
			Handler: handleSearchLogs,
		},
		{
			Tool:    getRecentErrorsToolDefinition(),
			Handler: handleGetRecentErrors,
		},
		{
			Tool:    getLogsByTraceIDToolDefinition(),
			Handler: handleGetLogsByTraceID,
		},
	}
}

// Tool definitions
func getLogsStatusToolDefinition() mcp.Tool {
	return mcp.NewTool("get_logs_status",
		mcp.WithDescription("Get the current status and health of the logs module"),
	)
}

func queryLogsToolDefinition() mcp.Tool {
	return mcp.NewTool("query_logs",
		mcp.WithDescription("Query logs with advanced filtering options"),
		mcp.WithString("service", mcp.Description("Filter by service name")),
		mcp.WithString("level", mcp.Description("Filter by log level (DEBUG, INFO, WARN, ERROR)")),
		mcp.WithString("start_time", mcp.Description("Start time for query (RFC3339 format)")),
		mcp.WithString("end_time", mcp.Description("End time for query (RFC3339 format)")),
		mcp.WithString("limit", mcp.Description("Maximum number of log entries to return (default: 100)")),
	)
}

func getLogStatsToolDefinition() mcp.Tool {
	return mcp.NewTool("get_log_stats",
		mcp.WithDescription("Get statistics about log entries by level and service"),
		mcp.WithString("time_range", mcp.Description("Time range for statistics (1h, 24h, 7d, 30d)")),
	)
}

func getLogServicesToolDefinition() mcp.Tool {
	return mcp.NewTool("get_log_services",
		mcp.WithDescription("Get list of all services that have logs available"),
	)
}

func getLogLevelsToolDefinition() mcp.Tool {
	return mcp.NewTool("get_log_levels",
		mcp.WithDescription("Get list of all available log levels"),
	)
}

func searchLogsToolDefinition() mcp.Tool {
	return mcp.NewTool("search_logs",
		mcp.WithDescription("Full-text search across log messages"),
		mcp.WithString("search_term", mcp.Required(), mcp.Description("Text to search for in log messages")),
		mcp.WithString("limit", mcp.Description("Maximum number of results to return (default: 50)")),
	)
}

func getRecentErrorsToolDefinition() mcp.Tool {
	return mcp.NewTool("get_recent_errors",
		mcp.WithDescription("Get recent error and warning log entries"),
		mcp.WithString("hours", mcp.Description("Number of hours to look back (default: 24)")),
		mcp.WithString("limit", mcp.Description("Maximum number of errors to return (default: 20)")),
	)
}

func getLogsByTraceIDToolDefinition() mcp.Tool {
	return mcp.NewTool("get_logs_by_trace_id",
		mcp.WithDescription("Get all logs for a specific trace ID"),
		mcp.WithString("trace_id", mcp.Required(), mcp.Description("Trace ID to search for")),
	)
}

// Tool handlers
func handleGetLogsStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := map[string]interface{}{
		"module":      "logs",
		"status":      "healthy",
		"uptime":      "168h42m",
		"last_update": time.Now().Format(time.RFC3339),
		"endpoint":    "http://localhost:9200",
		"version":     "8.11.0",
		"index_stats": map[string]interface{}{
			"total_indices":   15,
			"total_documents": 2847362,
			"storage_size_gb": 12.5,
			"ingestion_rate":  1250,
		},
		"cluster_health": "green",
	}

	data, err := json.Marshal(status)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func handleQueryLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse parameters
	var service, level, startTime, endTime string
	var limit int = 100

	if val, ok := args["service"].(string); ok {
		service = val
	}
	if val, ok := args["level"].(string); ok {
		level = val
	}
	if val, ok := args["start_time"].(string); ok {
		startTime = val
	}
	if val, ok := args["end_time"].(string); ok {
		endTime = val
	}
	if val, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			limit = parsed
		}
	}

	// Generate mock log entries
	logs := generateMockLogs()

	// Apply filters
	filtered := make([]LogEntry, 0)
	for _, log := range logs {
		if service != "" && log.Service != service {
			continue
		}
		if level != "" && log.Level != level {
			continue
		}
		filtered = append(filtered, log)
	}

	// Apply limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	response := map[string]interface{}{
		"logs":  filtered,
		"total": len(filtered),
		"limit": limit,
		"filters": map[string]interface{}{
			"service":    service,
			"level":      level,
			"start_time": startTime,
			"end_time":   endTime,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func handleGetLogStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	timeRange := "24h"
	if val, ok := args["time_range"].(string); ok {
		timeRange = val
	}

	stats := map[string]interface{}{
		"time_range": timeRange,
		"total_logs": 145230,
		"by_level": map[string]int{
			"DEBUG": 82150,
			"INFO":  45820,
			"WARN":  15260,
			"ERROR": 2000,
		},
		"by_service": map[string]int{
			"api-gateway":          35000,
			"user-service":         28000,
			"payment-service":      22000,
			"notification-service": 18000,
			"auth-service":         15000,
			"database":             12000,
			"redis-cache":          8000,
			"message-queue":        7230,
		},
		"error_rate_percent":     1.38,
		"ingestion_rate_per_min": 1250,
		"generated_at":           time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func handleGetLogServices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	services := []string{
		"api-gateway",
		"user-service",
		"payment-service",
		"notification-service",
		"auth-service",
		"database",
		"redis-cache",
		"message-queue",
		"file-storage",
		"logging-service",
		"monitoring-service",
	}

	response := map[string]interface{}{
		"services": services,
		"total":    len(services),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func handleGetLogLevels(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	levels := []string{
		"DEBUG",
		"INFO",
		"WARN",
		"ERROR",
		"FATAL",
	}

	response := map[string]interface{}{
		"levels": levels,
		"total":  len(levels),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func handleSearchLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	searchTerm, ok := args["search_term"].(string)
	if !ok {
		return nil, fmt.Errorf("search_term is required")
	}

	var limit int = 50
	if val, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			limit = parsed
		}
	}

	// Mock search results
	results := []LogEntry{
		{
			ID:        "log_search_001",
			Timestamp: time.Now().Add(-5 * time.Minute),
			Level:     "ERROR",
			Service:   "payment-service",
			Message:   fmt.Sprintf("Payment processing failed: %s not found", searchTerm),
			TraceID:   "trace_abc123",
			Fields: map[string]interface{}{
				"user_id":      "user_12345",
				"payment_id":   "pay_67890",
				"error_code":   "PAYMENT_FAILED",
				"search_match": searchTerm,
			},
		},
		{
			ID:        "log_search_002",
			Timestamp: time.Now().Add(-10 * time.Minute),
			Level:     "WARN",
			Service:   "api-gateway",
			Message:   fmt.Sprintf("Rate limit exceeded for %s endpoint", searchTerm),
			TraceID:   "trace_def456",
			Fields: map[string]interface{}{
				"endpoint":     "/api/v1/" + searchTerm,
				"client_ip":    "192.168.1.100",
				"rate_limit":   1000,
				"search_match": searchTerm,
			},
		},
	}

	response := map[string]interface{}{
		"search_term": searchTerm,
		"results":     results,
		"total":       len(results),
		"limit":       limit,
		"searched_at": time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func handleGetRecentErrors(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	var hours int = 24
	var limit int = 20

	if val, ok := args["hours"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			hours = parsed
		}
	}
	if val, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			limit = parsed
		}
	}

	// Generate mock recent errors
	errors := []LogEntry{
		{
			ID:        "error_001",
			Timestamp: time.Now().Add(-2 * time.Hour),
			Level:     "ERROR",
			Service:   "database",
			Message:   "Connection timeout to primary database",
			TraceID:   "trace_error_001",
			Fields: map[string]interface{}{
				"database":    "primary_db",
				"timeout_ms":  5000,
				"retry_count": 3,
				"error_code":  "CONNECTION_TIMEOUT",
			},
		},
		{
			ID:        "error_002",
			Timestamp: time.Now().Add(-4 * time.Hour),
			Level:     "ERROR",
			Service:   "payment-service",
			Message:   "Payment gateway API returned 502 Bad Gateway",
			TraceID:   "trace_error_002",
			Fields: map[string]interface{}{
				"gateway":      "stripe",
				"status_code":  502,
				"payment_id":   "pay_12345",
				"amount_cents": 2000,
			},
		},
		{
			ID:        "error_003",
			Timestamp: time.Now().Add(-6 * time.Hour),
			Level:     "WARN",
			Service:   "auth-service",
			Message:   "Multiple failed login attempts detected",
			TraceID:   "trace_warn_001",
			Fields: map[string]interface{}{
				"user_id":       "user_suspicious",
				"attempt_count": 5,
				"client_ip":     "192.168.1.200",
				"blocked":       true,
			},
		},
	}

	response := map[string]interface{}{
		"errors":       errors,
		"total":        len(errors),
		"limit":        limit,
		"time_range":   fmt.Sprintf("%dh", hours),
		"generated_at": time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func handleGetLogsByTraceID(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	traceID, ok := args["trace_id"].(string)
	if !ok {
		return nil, fmt.Errorf("trace_id is required")
	}

	// Mock trace logs
	logs := []LogEntry{
		{
			ID:        "trace_log_001",
			Timestamp: time.Now().Add(-10 * time.Minute),
			Level:     "INFO",
			Service:   "api-gateway",
			Message:   "Incoming request received",
			TraceID:   traceID,
			Fields: map[string]interface{}{
				"method":    "POST",
				"endpoint":  "/api/v1/payments",
				"client_ip": "192.168.1.100",
			},
		},
		{
			ID:        "trace_log_002",
			Timestamp: time.Now().Add(-10*time.Minute + 50*time.Millisecond),
			Level:     "DEBUG",
			Service:   "auth-service",
			Message:   "Validating authentication token",
			TraceID:   traceID,
			Fields: map[string]interface{}{
				"user_id":    "user_12345",
				"token_type": "bearer",
				"valid":      true,
			},
		},
		{
			ID:        "trace_log_003",
			Timestamp: time.Now().Add(-10*time.Minute + 120*time.Millisecond),
			Level:     "INFO",
			Service:   "payment-service",
			Message:   "Processing payment request",
			TraceID:   traceID,
			Fields: map[string]interface{}{
				"payment_id":   "pay_67890",
				"amount_cents": 2000,
				"currency":     "USD",
			},
		},
		{
			ID:        "trace_log_004",
			Timestamp: time.Now().Add(-10*time.Minute + 250*time.Millisecond),
			Level:     "INFO",
			Service:   "payment-service",
			Message:   "Payment completed successfully",
			TraceID:   traceID,
			Fields: map[string]interface{}{
				"payment_id":    "pay_67890",
				"status":        "completed",
				"processing_ms": 130,
			},
		},
	}

	response := map[string]interface{}{
		"trace_id":    traceID,
		"logs":        logs,
		"total":       len(logs),
		"duration_ms": 250,
		"services":    []string{"api-gateway", "auth-service", "payment-service"},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// Helper function to generate mock logs
func generateMockLogs() []LogEntry {
	return []LogEntry{
		{
			ID:        "log_001",
			Timestamp: time.Now().Add(-5 * time.Minute),
			Level:     "INFO",
			Service:   "api-gateway",
			Message:   "Request processed successfully",
			TraceID:   "trace_123",
			Fields: map[string]interface{}{
				"method":      "GET",
				"endpoint":    "/api/v1/users",
				"status_code": 200,
				"response_ms": 45,
			},
		},
		{
			ID:        "log_002",
			Timestamp: time.Now().Add(-10 * time.Minute),
			Level:     "ERROR",
			Service:   "database",
			Message:   "Query execution failed",
			TraceID:   "trace_124",
			Fields: map[string]interface{}{
				"query":       "SELECT * FROM users",
				"error_code":  "TIMEOUT",
				"duration_ms": 5000,
			},
		},
		{
			ID:        "log_003",
			Timestamp: time.Now().Add(-15 * time.Minute),
			Level:     "WARN",
			Service:   "auth-service",
			Message:   "Rate limit warning",
			TraceID:   "trace_125",
			Fields: map[string]interface{}{
				"client_ip": "192.168.1.100",
				"requests":  950,
				"limit":     1000,
			},
		},
		{
			ID:        "log_004",
			Timestamp: time.Now().Add(-20 * time.Minute),
			Level:     "DEBUG",
			Service:   "payment-service",
			Message:   "Payment validation started",
			TraceID:   "trace_126",
			Fields: map[string]interface{}{
				"payment_id": "pay_12345",
				"amount":     "29.99",
				"currency":   "USD",
			},
		},
	}
}
