package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// PrometheusConfig contains Prometheus configuration
type PrometheusConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
}

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Suffix string `mapstructure:"suffix" json:"suffix" yaml:"suffix"`
}

// Config contains metrics module configuration
type Config struct {
	// Prometheus configuration - required
	Prometheus *PrometheusConfig `mapstructure:"prometheus" json:"prometheus" yaml:"prometheus"`
	Tools      ToolsConfig       `mapstructure:"tools" json:"tools" yaml:"tools"`
}

// Module represents the metrics module
type Module struct {
	config     *Config
	logger     *zap.Logger
	httpClient *http.Client
}

// New creates a new metrics module
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("metrics config is required")
	}

	// Create HTTP client with optimized connection pooling and TIME_WAIT management
	transport := &http.Transport{
		MaxIdleConns:        50,               // Reduce maximum idle connections
		MaxIdleConnsPerHost: 5,                // Reduce idle connections per host
		MaxConnsPerHost:     20,               // Reduce maximum connections per host
		IdleConnTimeout:     30 * time.Second, // Significantly reduce idle connection timeout for faster release
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // Reduce connection timeout
			KeepAlive: 15 * time.Second, // Reduce keep-alive interval
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second, // Reduce TLS handshake timeout
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false, // Enable connection reuse
		ForceAttemptHTTP2:     false, // Force HTTP/1.1 for better connection reuse
		// Add connection cleanup mechanism
		ResponseHeaderTimeout: 10 * time.Second, // Response header timeout
		DisableCompression:    false,            // Enable compression to reduce transmission time
	}

	m := &Module{
		config: config,
		logger: logger.Named("metrics"),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second, // Reduce client timeout for faster connection release
		},
	}

	if config.Prometheus != nil {
		m.logger.Info("Metrics module created with Prometheus",
			zap.String("prometheus_endpoint", config.Prometheus.Endpoint),
		)
	} else {
		m.logger.Info("Metrics module created without Prometheus configuration")
	}

	return m, nil
}

// makePrometheusRequest creates and executes an HTTP request to Prometheus API
func (m *Module) makePrometheusRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	url := m.config.Prometheus.Endpoint + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Log request details
	m.logger.Info("Making Prometheus Request",
		zap.String("method", method),
		zap.String("full_url", url),
		zap.String("path", path),
		zap.String("endpoint", m.config.Prometheus.Endpoint),
		zap.Bool("has_body", body != nil))

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.logger.Error("Prometheus Request Failed",
			zap.String("method", method),
			zap.String("url", url),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// Log response details
	m.logger.Info("Prometheus Response Received",
		zap.String("method", method),
		zap.String("url", url),
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
		zap.Int64("content_length", resp.ContentLength))

	return resp, nil
}

// queryPrometheus executes a Prometheus query directly
func (m *Module) queryPrometheus(ctx context.Context, query string, queryType string, params map[string]string) (*PrometheusResponse, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	// Format: {endpoint}/api/v1/{queryType}
	path := fmt.Sprintf("/api/v1/%s", queryType)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("query", query)

	for key, value := range params {
		queryParams.Set(key, value)
	}

	fullURL := m.config.Prometheus.Endpoint + path + "?" + queryParams.Encode()

	m.logger.Info("Executing Prometheus Query",
		zap.String("url", fullURL),
		zap.String("query", query),
		zap.String("query_type", queryType),
		zap.Any("params", params))

	resp, err := m.makePrometheusRequest(ctx, "GET", path+"?"+queryParams.Encode(), nil)
	if err != nil {
		m.logger.Error("Prometheus query failed",
			zap.String("query", query),
			zap.Error(err))
		return nil, fmt.Errorf("failed to query Prometheus: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Prometheus API returned non-200 status",
			zap.String("query", query),
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("Prometheus API returned status %d", resp.StatusCode)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logger.Error("Failed to read response body",
			zap.String("query", query),
			zap.Error(err))
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(respBody, &promResp); err != nil {
		m.logger.Error("Failed to decode Prometheus response",
			zap.String("query", query),
			zap.Error(err),
			zap.String("response_body", string(respBody)))
		return nil, fmt.Errorf("failed to decode Prometheus response: %w", err)
	}

	// Log query results
	resultCount := 0
	if promResp.Data.ResultType == "vector" {
		resultCount = len(promResp.Data.Result)
	} else if promResp.Data.ResultType == "matrix" {
		resultCount = len(promResp.Data.Result)
	}

	if promResp.Status == "success" {
		m.logger.Info("Prometheus Query Successful",
			zap.String("query", query),
			zap.String("status", promResp.Status),
			zap.String("result_type", promResp.Data.ResultType),
			zap.Int("result_count", resultCount))

		// Log first few results for debugging
		if resultCount > 0 && len(promResp.Data.Result) > 0 {
			firstResult := promResp.Data.Result[0]
			if promResp.Data.ResultType == "vector" {
				m.logger.Debug("📊 Sample Result (Vector)",
					zap.String("query", query),
					zap.Any("labels", firstResult.Labels),
					zap.String("value", firstResult.Value.Value),
					zap.Float64("timestamp", firstResult.Value.Timestamp))
			} else if promResp.Data.ResultType == "matrix" {
				valueCount := len(firstResult.Values)
				m.logger.Debug("📊 Sample Result (Matrix)",
					zap.String("query", query),
					zap.Any("labels", firstResult.Labels),
					zap.Int("value_count", valueCount))
				if valueCount > 0 {
					m.logger.Debug("📊 First Matrix Value",
						zap.String("value", firstResult.Values[0].Value),
						zap.Float64("timestamp", firstResult.Values[0].Timestamp))
				}
			}
		}
	} else {
		m.logger.Warn("Prometheus Query Warning",
			zap.String("query", query),
			zap.String("status", promResp.Status),
			zap.String("error", promResp.Error),
			zap.Strings("warnings", promResp.Warnings))
	}

	return &promResp, nil
}

// GetTools returns all MCP tools for the metrics module
func (m *Module) GetTools() []server.ServerTool {
	// Get default tool configuration
	toolsConfig := GetDefaultToolsConfig()

	// Tool configuration can be modified based on config file or other conditions
	// For example: disable certain tools based on m.config
	// toolsConfig.ListMetrics.Enabled = false

	return m.BuildTools(toolsConfig)
}

func (m *Module) handleListMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	args := request.GetArguments()

	// Get search filter if provided
	searchFilter := ""
	if search, ok := args["search"].(string); ok {
		searchFilter = search
	}

	// Get limit if provided
	limit := 100
	if limitStr, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	m.logger.Info("Listing available metrics",
		zap.String("search_filter", searchFilter),
		zap.Int("limit", limit))

	// Query Prometheus metadata API to get all metrics
	resp, err := m.makePrometheusRequest(ctx, "GET", "/api/v1/label/__name__/values", nil)
	if err != nil {
		m.logger.Error("Failed to query metrics list", zap.Error(err))
		return nil, fmt.Errorf("failed to query metrics list: %w", err)
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Prometheus API returned non-200 status",
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("Prometheus API returned status %d", resp.StatusCode)
	}

	// Read and parse response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logger.Error("Failed to read response body", zap.Error(err))
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		m.logger.Error("Failed to decode response",
			zap.Error(err),
			zap.String("response_body", string(respBody)))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Status != "success" {
		m.logger.Error("API request failed",
			zap.String("status", apiResp.Status))
		return nil, fmt.Errorf("API request failed with status: %s", apiResp.Status)
	}

	// Filter metrics if search pattern provided
	filteredMetrics := make([]string, 0)
	for _, metric := range apiResp.Data {
		if searchFilter == "" || strings.Contains(metric, searchFilter) {
			filteredMetrics = append(filteredMetrics, metric)
		}
	}

	// Apply limit
	if len(filteredMetrics) > limit {
		filteredMetrics = filteredMetrics[:limit]
	}

	result := map[string]interface{}{
		"metrics":       filteredMetrics,
		"total_count":   len(filteredMetrics),
		"search_filter": searchFilter,
		"limit":         limit,
		"timestamp":     time.Now().Format(time.RFC3339),
		"status":        "success",
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	m.logger.Info("Metrics list completed successfully",
		zap.Int("returned_count", len(filteredMetrics)),
		zap.Int("total_available", len(apiResp.Data)))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (m *Module) handleExecuteQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	m.logger.Info("Executing PromQL instant query",
		zap.String("query", query))

	// Execute instant query
	params := make(map[string]string)
	params["time"] = fmt.Sprintf("%d", time.Now().Unix())

	promResp, err := m.queryPrometheus(ctx, query, "query", params)
	if err != nil {
		m.logger.Error("Failed to execute PromQL query",
			zap.String("query", query),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// Convert to our response format
	response := MetricsQueryResponse{
		Status:   promResp.Status,
		Data:     promResp.Data,
		Error:    promResp.Error,
		Warnings: promResp.Warnings,
		Metadata: map[string]string{
			"query":     query,
			"type":      "instant",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	m.logger.Info("PromQL instant query completed successfully",
		zap.String("query", query),
		zap.String("status", promResp.Status))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (m *Module) handleExecuteRangeQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	timeRange, ok := args["time_range"].(string)
	if !ok {
		return nil, fmt.Errorf("time_range parameter is required")
	}

	// Get step parameter or use default
	step := "60s"
	if stepArg, ok := args["step"].(string); ok && stepArg != "" {
		step = stepArg
	}

	m.logger.Info("Executing PromQL range query",
		zap.String("query", query),
		zap.String("time_range", timeRange),
		zap.String("step", step))

	// Parse time range
	var duration time.Duration
	switch timeRange {
	case "1h":
		duration = time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		return nil, fmt.Errorf("unsupported time range: %s (supported: 1h, 24h, 7d, 30d)", timeRange)
	}

	now := time.Now()
	start := now.Add(-duration)

	// Execute range query
	params := make(map[string]string)
	params["start"] = fmt.Sprintf("%d", start.Unix())
	params["end"] = fmt.Sprintf("%d", now.Unix())
	params["step"] = step

	promResp, err := m.queryPrometheus(ctx, query, "query_range", params)
	if err != nil {
		m.logger.Error("Failed to execute PromQL range query",
			zap.String("query", query),
			zap.String("time_range", timeRange),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute range query: %w", err)
	}

	// Convert to our response format
	response := MetricsQueryResponse{
		Status:   promResp.Status,
		Data:     promResp.Data,
		Error:    promResp.Error,
		Warnings: promResp.Warnings,
		Metadata: map[string]string{
			"query":      query,
			"type":       "range",
			"time_range": timeRange,
			"start_time": start.Format(time.RFC3339),
			"end_time":   now.Format(time.RFC3339),
			"step":       step,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	m.logger.Info("PromQL range query completed successfully",
		zap.String("query", query),
		zap.String("time_range", timeRange),
		zap.String("status", promResp.Status))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}
