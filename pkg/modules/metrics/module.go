package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Config contains metrics module configuration
type Config struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
}

// Module represents the metrics module
type Module struct {
	config *Config
	logger *zap.Logger
}

// New creates a new metrics module
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("metrics config is required")
	}

	m := &Module{
		config: config,
		logger: logger.Named("metrics"),
	}

	m.logger.Info("Metrics module created",
		zap.String("endpoint", config.Endpoint),
	)

	return m, nil
}

// GetTools returns all MCP tools for the metrics module
func GetTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Tool:    getMetricsStatusToolDefinition(),
			Handler: handleGetMetricsStatus,
		},
		{
			Tool:    getSystemOverviewToolDefinition(),
			Handler: handleGetSystemOverview,
		},
		{
			Tool:    getServiceMetricsToolDefinition(),
			Handler: handleGetServiceMetrics,
		},
		{
			Tool:    getMetricsServicesToolDefinition(),
			Handler: handleGetMetricsServices,
		},
		{
			Tool:    getMetricHistoryToolDefinition(),
			Handler: handleGetMetricHistory,
		},
		{
			Tool:    getMetricsAlertsToolDefinition(),
			Handler: handleGetMetricsAlerts,
		},
		{
			Tool:    queryMetricsToolDefinition(),
			Handler: handleQueryMetrics,
		},
	}
}

// Tool definitions
func getMetricsStatusToolDefinition() mcp.Tool {
	return mcp.NewTool("get_metrics_status",
		mcp.WithDescription("Get the current status and health of the metrics module"),
	)
}

func getSystemOverviewToolDefinition() mcp.Tool {
	return mcp.NewTool("get_system_overview",
		mcp.WithDescription("Get overall system metrics including CPU, memory, disk, and network"),
	)
}

func getServiceMetricsToolDefinition() mcp.Tool {
	return mcp.NewTool("get_service_metrics",
		mcp.WithDescription("Get metrics for a specific service"),
		mcp.WithString("service_name", mcp.Required(), mcp.Description("Name of the service to get metrics for")),
	)
}

func getMetricsServicesToolDefinition() mcp.Tool {
	return mcp.NewTool("get_metrics_services",
		mcp.WithDescription("Get list of all services that have metrics available"),
	)
}

func getMetricHistoryToolDefinition() mcp.Tool {
	return mcp.NewTool("get_metric_history",
		mcp.WithDescription("Get historical data for a specific metric"),
		mcp.WithString("metric_name", mcp.Required(), mcp.Description("Name of the metric")),
		mcp.WithString("time_range", mcp.Description("Time range for history (1h, 24h, 7d, 30d)")),
	)
}

func getMetricsAlertsToolDefinition() mcp.Tool {
	return mcp.NewTool("get_metrics_alerts",
		mcp.WithDescription("Get current metrics alerts and thresholds"),
	)
}

func queryMetricsToolDefinition() mcp.Tool {
	return mcp.NewTool("query_metrics",
		mcp.WithDescription("Execute a custom metrics query (PromQL-style)"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Metrics query expression")),
	)
}

// Tool handlers
func handleGetMetricsStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := map[string]interface{}{
		"module":      "metrics",
		"status":      "healthy",
		"uptime":      "72h15m",
		"last_update": time.Now().Format(time.RFC3339),
		"endpoint":    "http://localhost:9090",
		"version":     "2.45.0",
		"scrape_targets": map[string]interface{}{
			"total":  25,
			"up":     23,
			"down":   2,
			"health": "92%",
		},
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

func handleGetSystemOverview(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	overview := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"cpu": map[string]interface{}{
			"usage_percent": 45.2,
			"cores":         8,
			"load_1m":       2.1,
			"load_5m":       1.8,
			"load_15m":      1.5,
		},
		"memory": map[string]interface{}{
			"total_gb":      32.0,
			"used_gb":       18.4,
			"free_gb":       13.6,
			"usage_percent": 57.5,
			"cached_gb":     4.2,
		},
		"disk": map[string]interface{}{
			"total_gb":      500.0,
			"used_gb":       287.5,
			"free_gb":       212.5,
			"usage_percent": 57.5,
			"iops_read":     1250,
			"iops_write":    850,
		},
		"network": map[string]interface{}{
			"rx_bytes_per_sec": 1048576,
			"tx_bytes_per_sec": 524288,
			"connections":      142,
			"packets_dropped":  0,
		},
	}

	data, err := json.Marshal(overview)
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

func handleGetServiceMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	serviceName, ok := args["service_name"].(string)
	if !ok {
		return nil, fmt.Errorf("service_name is required")
	}

	metrics := map[string]interface{}{
		"service":   serviceName,
		"timestamp": time.Now().Format(time.RFC3339),
		"health":    "healthy",
		"metrics": map[string]interface{}{
			"requests_per_second": 125.5,
			"response_time_ms":    45.2,
			"error_rate_percent":  0.8,
			"cpu_usage_percent":   32.1,
			"memory_usage_mb":     512.8,
			"active_connections":  78,
			"throughput_mbps":     15.2,
		},
		"status_codes": map[string]int{
			"2xx": 8450,
			"3xx": 120,
			"4xx": 45,
			"5xx": 12,
		},
	}

	data, err := json.Marshal(metrics)
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

func handleGetMetricsServices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	services := []string{
		"api-gateway",
		"user-service",
		"payment-service",
		"notification-service",
		"database",
		"redis-cache",
		"message-queue",
		"file-storage",
		"auth-service",
		"logging-service",
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

func handleGetMetricHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	metricName, ok := args["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name is required")
	}

	timeRange := "1h"
	if val, ok := args["time_range"].(string); ok {
		timeRange = val
	}

	// Generate mock historical data
	now := time.Now()
	points := make([]map[string]interface{}, 10)
	for i := 0; i < 10; i++ {
		timestamp := now.Add(time.Duration(-i*6) * time.Minute)
		value := 50.0 + float64(i)*2.5 + float64(i%3)*5.0
		points[9-i] = map[string]interface{}{
			"timestamp": timestamp.Format(time.RFC3339),
			"value":     value,
		}
	}

	history := map[string]interface{}{
		"metric":     metricName,
		"time_range": timeRange,
		"unit":       "percent",
		"points":     points,
		"summary": map[string]interface{}{
			"min":     45.2,
			"max":     72.5,
			"avg":     58.3,
			"current": points[9]["value"],
		},
	}

	data, err := json.Marshal(history)
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

func handleGetMetricsAlerts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	alerts := []map[string]interface{}{
		{
			"name":        "High CPU Usage",
			"metric":      "cpu_usage_percent",
			"threshold":   80.0,
			"current":     45.2,
			"status":      "ok",
			"severity":    "warning",
			"description": "CPU usage is above threshold",
		},
		{
			"name":        "Memory Usage",
			"metric":      "memory_usage_percent",
			"threshold":   85.0,
			"current":     57.5,
			"status":      "ok",
			"severity":    "critical",
			"description": "Memory usage is above threshold",
		},
		{
			"name":        "Disk Space",
			"metric":      "disk_usage_percent",
			"threshold":   90.0,
			"current":     57.5,
			"status":      "ok",
			"severity":    "warning",
			"description": "Disk space is above threshold",
		},
		{
			"name":        "Error Rate",
			"metric":      "error_rate_percent",
			"threshold":   5.0,
			"current":     0.8,
			"status":      "ok",
			"severity":    "critical",
			"description": "Error rate is above threshold",
		},
	}

	response := map[string]interface{}{
		"alerts":        alerts,
		"total":         len(alerts),
		"active_alerts": 0,
		"last_updated":  time.Now().Format(time.RFC3339),
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

func handleQueryMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required")
	}

	// Mock query results
	results := []map[string]interface{}{
		{
			"metric": map[string]string{
				"__name__": "cpu_usage",
				"instance": "localhost:9090",
				"job":      "prometheus",
			},
			"value": []interface{}{
				time.Now().Unix(),
				"45.2",
			},
		},
		{
			"metric": map[string]string{
				"__name__": "memory_usage",
				"instance": "localhost:9090",
				"job":      "prometheus",
			},
			"value": []interface{}{
				time.Now().Unix(),
				"57.5",
			},
		},
	}

	response := map[string]interface{}{
		"query":       query,
		"results":     results,
		"result_type": "vector",
		"timestamp":   time.Now().Format(time.RFC3339),
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
