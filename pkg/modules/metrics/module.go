package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// Config contains metrics module configuration
type Config struct {
	Endpoint   string            `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Prometheus *PrometheusConfig `mapstructure:"prometheus" json:"prometheus" yaml:"prometheus"`
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

	m := &Module{
		config: config,
		logger: logger.Named("metrics"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
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

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// queryPrometheus executes a Prometheus query directly
func (m *Module) queryPrometheus(ctx context.Context, query string, queryType string, params map[string]string) (*PrometheusResponse, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	// Build the Prometheus API path (endpoint already includes /api/v1)
	path := fmt.Sprintf("/%s", queryType)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("query", query)

	for key, value := range params {
		queryParams.Set(key, value)
	}

	fullURL := m.config.Prometheus.Endpoint + path + "?" + queryParams.Encode()

	m.logger.Info("🔍 Executing Prometheus Query",
		zap.String("url", fullURL),
		zap.String("query", query),
		zap.String("query_type", queryType),
		zap.Any("params", params))

	resp, err := m.makePrometheusRequest(ctx, "GET", path+"?"+queryParams.Encode(), nil)
	if err != nil {
		m.logger.Error("❌ Prometheus query failed",
			zap.String("query", query),
			zap.Error(err))
		return nil, fmt.Errorf("failed to query Prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("❌ Prometheus API returned non-200 status",
			zap.String("query", query),
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("Prometheus API returned status %d", resp.StatusCode)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logger.Error("❌ Failed to read response body",
			zap.String("query", query),
			zap.Error(err))
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(respBody, &promResp); err != nil {
		m.logger.Error("❌ Failed to decode Prometheus response",
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
		m.logger.Info("✅ Prometheus Query Successful",
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
		m.logger.Warn("⚠️ Prometheus Query Warning",
			zap.String("query", query),
			zap.String("status", promResp.Status),
			zap.String("error", promResp.Error),
			zap.Strings("warnings", promResp.Warnings))
	}

	return &promResp, nil
}

// GetTools returns all MCP tools for the metrics module
func (m *Module) GetTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Tool:    getMetricsStatusToolDefinition(),
			Handler: m.handleGetMetricsStatus,
		},
		{
			Tool:    getSystemOverviewToolDefinition(),
			Handler: m.handleGetSystemOverview,
		},
		{
			Tool:    getServiceMetricsToolDefinition(),
			Handler: m.handleGetServiceMetrics,
		},
		{
			Tool:    getMetricsServicesToolDefinition(),
			Handler: m.handleGetMetricsServices,
		},
		{
			Tool:    getMetricHistoryToolDefinition(),
			Handler: m.handleGetMetricHistory,
		},
		{
			Tool:    getMetricsAlertsToolDefinition(),
			Handler: m.handleGetMetricsAlerts,
		},
		{
			Tool:    queryMetricsToolDefinition(),
			Handler: m.handleQueryMetrics,
		},
		{
			Tool:    queryMetricsRangeToolDefinition(),
			Handler: m.handleQueryMetricsRange,
		},
		// Kubernetes resource listing tools
		{
			Tool:    getClustersToolDefinition(),
			Handler: m.handleGetClusters,
		},
		{
			Tool:    getNamespacesToolDefinition(),
			Handler: m.handleGetNamespaces,
		},
		{
			Tool:    getPodsToolDefinition(),
			Handler: m.handleGetPods,
		},
		// Kubernetes resource usage tools
		{
			Tool:    getPodResourceUsageToolDefinition(),
			Handler: m.handleGetPodResourceUsage,
		},
		{
			Tool:    getNodeResourceUsageToolDefinition(),
			Handler: m.handleGetNodeResourceUsage,
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

func queryMetricsRangeToolDefinition() mcp.Tool {
	return mcp.NewTool("query_metrics_range",
		mcp.WithDescription("Execute a custom metrics query with a time range (PromQL-style)"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Metrics query expression (PromQL syntax)")),
		mcp.WithString("time_range", mcp.Required(), mcp.Description("Time range for query (1h, 24h, 7d, 30d)")),
		mcp.WithString("step", mcp.Description("Query resolution step (default: 60s, examples: 30s, 5m, 1h)")),
	)
}

// Kubernetes resource listing tool definitions
func getClustersToolDefinition() mcp.Tool {
	return mcp.NewTool("get_clusters",
		mcp.WithDescription("Get list of all available Kubernetes clusters from metrics"),
	)
}

func getNamespacesToolDefinition() mcp.Tool {
	return mcp.NewTool("get_namespaces",
		mcp.WithDescription("Get list of all Kubernetes namespaces"),
		mcp.WithString("cluster", mcp.Description("Filter by cluster name (optional)")),
	)
}

func getPodsToolDefinition() mcp.Tool {
	return mcp.NewTool("get_pods",
		mcp.WithDescription("Get list of all Kubernetes pods"),
		mcp.WithString("cluster", mcp.Description("Filter by cluster name (optional)")),
		mcp.WithString("namespace", mcp.Description("Filter by namespace (optional)")),
		mcp.WithString("limit", mcp.Description("Maximum number of pods to return (default: 50)")),
	)
}

// Kubernetes resource usage tool definitions
func getPodResourceUsageToolDefinition() mcp.Tool {
	return mcp.NewTool("get_pod_resource_usage",
		mcp.WithDescription("Get CPU and memory usage for Kubernetes pods"),
		mcp.WithString("cluster", mcp.Description("Filter by cluster name (optional)")),
		mcp.WithString("namespace", mcp.Description("Filter by namespace (optional)")),
		mcp.WithString("pod", mcp.Description("Filter by pod name (optional)")),
		mcp.WithString("limit", mcp.Description("Maximum number of pods to return (default: 20)")),
	)
}

func getNodeResourceUsageToolDefinition() mcp.Tool {
	return mcp.NewTool("get_node_resource_usage",
		mcp.WithDescription("Get CPU and memory usage for Kubernetes nodes"),
		mcp.WithString("cluster", mcp.Description("Filter by cluster name (optional)")),
		mcp.WithString("node", mcp.Description("Filter by node name (optional)")),
		mcp.WithString("limit", mcp.Description("Maximum number of nodes to return (default: 20)")),
	)
}

// Tool handlers
func (m *Module) handleGetMetricsStatus(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	m.logger.Info("🏥 Starting metrics status check")

	// Query Prometheus build info
	buildInfoResp, err := m.queryPrometheus(ctx, "prometheus_build_info", "query", nil)
	if err != nil {
		m.logger.Error("Failed to query Prometheus build info", zap.Error(err))
		return nil, fmt.Errorf("failed to query Prometheus build info: %w", err)
	}

	// Query total scrape targets
	totalTargetsResp, err := m.queryPrometheus(ctx, "count(up)", "query", nil)
	if err != nil {
		m.logger.Error("Failed to query total targets", zap.Error(err))
		return nil, fmt.Errorf("failed to query total targets: %w", err)
	}

	// Query up targets
	upTargetsResp, err := m.queryPrometheus(ctx, "count(up == 1)", "query", nil)
	if err != nil {
		m.logger.Error("Failed to query up targets", zap.Error(err))
		return nil, fmt.Errorf("failed to query up targets: %w", err)
	}

	// Query Prometheus uptime
	uptimeResp, err := m.queryPrometheus(ctx, "time() - prometheus_tsdb_lowest_timestamp", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query Prometheus uptime", zap.Error(err))
	}

	status := map[string]interface{}{
		"module":      "metrics",
		"status":      "healthy",
		"last_update": time.Now().Format(time.RFC3339),
		"endpoint":    m.config.Prometheus.Endpoint,
	}

	// Extract version from build info
	if buildInfoResp != nil && buildInfoResp.Data.ResultType == "vector" && len(buildInfoResp.Data.Result) > 0 {
		buildInfo := buildInfoResp.Data.Result[0]
		if version, exists := buildInfo.Labels["version"]; exists {
			status["version"] = version
		}
	}

	// Extract uptime
	if uptimeResp != nil && uptimeResp.Data.ResultType == "vector" && len(uptimeResp.Data.Result) > 0 {
		if uptimeSeconds, ok := strconv.ParseFloat(uptimeResp.Data.Result[0].Value.Value, 64); ok == nil {
			uptime := time.Duration(uptimeSeconds) * time.Second
			status["uptime"] = uptime.String()
		}
	}

	// Extract target counts
	var totalTargets, upTargets float64
	if totalTargetsResp != nil && totalTargetsResp.Data.ResultType == "vector" && len(totalTargetsResp.Data.Result) > 0 {
		totalTargets, _ = strconv.ParseFloat(totalTargetsResp.Data.Result[0].Value.Value, 64)
	}
	if upTargetsResp != nil && upTargetsResp.Data.ResultType == "vector" && len(upTargetsResp.Data.Result) > 0 {
		upTargets, _ = strconv.ParseFloat(upTargetsResp.Data.Result[0].Value.Value, 64)
	}

	downTargets := totalTargets - upTargets
	healthPercent := 0.0
	if totalTargets > 0 {
		healthPercent = (upTargets / totalTargets) * 100
	}

	status["scrape_targets"] = map[string]interface{}{
		"total":  int(totalTargets),
		"up":     int(upTargets),
		"down":   int(downTargets),
		"health": fmt.Sprintf("%.1f%%", healthPercent),
	}

	data, err := json.Marshal(status)
	if err != nil {
		return nil, err
	}

	m.logger.Info("🏥 Metrics status collection completed",
		zap.Any("scrape_targets", status["scrape_targets"]))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (m *Module) handleGetSystemOverview(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	m.logger.Info("🖥️ Starting system overview collection")

	overview := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Query CPU metrics
	cpuUsageResp, err := m.queryPrometheus(ctx, "100 - (avg(irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query CPU usage", zap.Error(err))
	}

	cpuCoresResp, err := m.queryPrometheus(ctx, "count(count by (cpu) (node_cpu_seconds_total{mode=\"idle\"}))", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query CPU cores", zap.Error(err))
	}

	load1Resp, err := m.queryPrometheus(ctx, "avg(node_load1)", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query load1", zap.Error(err))
	}

	load5Resp, err := m.queryPrometheus(ctx, "avg(node_load5)", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query load5", zap.Error(err))
	}

	load15Resp, err := m.queryPrometheus(ctx, "avg(node_load15)", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query load15", zap.Error(err))
	}

	// CPU metrics
	cpuData := make(map[string]interface{})
	if cpuUsageResp != nil && cpuUsageResp.Data.ResultType == "vector" && len(cpuUsageResp.Data.Result) > 0 {
		if usage, err := strconv.ParseFloat(cpuUsageResp.Data.Result[0].Value.Value, 64); err == nil {
			cpuData["usage_percent"] = usage
		}
	}
	if cpuCoresResp != nil && cpuCoresResp.Data.ResultType == "vector" && len(cpuCoresResp.Data.Result) > 0 {
		if cores, err := strconv.ParseFloat(cpuCoresResp.Data.Result[0].Value.Value, 64); err == nil {
			cpuData["cores"] = int(cores)
		}
	}
	if load1Resp != nil && load1Resp.Data.ResultType == "vector" && len(load1Resp.Data.Result) > 0 {
		if load1, err := strconv.ParseFloat(load1Resp.Data.Result[0].Value.Value, 64); err == nil {
			cpuData["load_1m"] = load1
		}
	}
	if load5Resp != nil && load5Resp.Data.ResultType == "vector" && len(load5Resp.Data.Result) > 0 {
		if load5, err := strconv.ParseFloat(load5Resp.Data.Result[0].Value.Value, 64); err == nil {
			cpuData["load_5m"] = load5
		}
	}
	if load15Resp != nil && load15Resp.Data.ResultType == "vector" && len(load15Resp.Data.Result) > 0 {
		if load15, err := strconv.ParseFloat(load15Resp.Data.Result[0].Value.Value, 64); err == nil {
			cpuData["load_15m"] = load15
		}
	}
	overview["cpu"] = cpuData

	// Query Memory metrics
	memTotalResp, err := m.queryPrometheus(ctx, "avg(node_memory_MemTotal_bytes)", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query total memory", zap.Error(err))
	}

	memAvailableResp, err := m.queryPrometheus(ctx, "avg(node_memory_MemAvailable_bytes)", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query available memory", zap.Error(err))
	}

	memCachedResp, err := m.queryPrometheus(ctx, "avg(node_memory_Cached_bytes)", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query cached memory", zap.Error(err))
	}

	// Memory metrics
	memData := make(map[string]interface{})
	var totalGB, availableGB, cachedGB float64
	if memTotalResp != nil && memTotalResp.Data.ResultType == "vector" && len(memTotalResp.Data.Result) > 0 {
		if total, err := strconv.ParseFloat(memTotalResp.Data.Result[0].Value.Value, 64); err == nil {
			totalGB = total / (1024 * 1024 * 1024)
			memData["total_gb"] = totalGB
		}
	}
	if memAvailableResp != nil && memAvailableResp.Data.ResultType == "vector" && len(memAvailableResp.Data.Result) > 0 {
		if available, err := strconv.ParseFloat(memAvailableResp.Data.Result[0].Value.Value, 64); err == nil {
			availableGB = available / (1024 * 1024 * 1024)
			memData["free_gb"] = availableGB
		}
	}
	if memCachedResp != nil && memCachedResp.Data.ResultType == "vector" && len(memCachedResp.Data.Result) > 0 {
		if cached, err := strconv.ParseFloat(memCachedResp.Data.Result[0].Value.Value, 64); err == nil {
			cachedGB = cached / (1024 * 1024 * 1024)
			memData["cached_gb"] = cachedGB
		}
	}
	if totalGB > 0 && availableGB > 0 {
		usedGB := totalGB - availableGB
		memData["used_gb"] = usedGB
		memData["usage_percent"] = (usedGB / totalGB) * 100
	}
	overview["memory"] = memData

	// Query Disk metrics
	diskTotalResp, err := m.queryPrometheus(ctx, "avg(node_filesystem_size_bytes{fstype!=\"tmpfs\"})", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query total disk", zap.Error(err))
	}

	diskAvailResp, err := m.queryPrometheus(ctx, "avg(node_filesystem_avail_bytes{fstype!=\"tmpfs\"})", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query available disk", zap.Error(err))
	}

	diskReadIOPSResp, err := m.queryPrometheus(ctx, "sum(rate(node_disk_reads_completed_total[5m]))", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query disk read IOPS", zap.Error(err))
	}

	diskWriteIOPSResp, err := m.queryPrometheus(ctx, "sum(rate(node_disk_writes_completed_total[5m]))", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query disk write IOPS", zap.Error(err))
	}

	// Disk metrics
	diskData := make(map[string]interface{})
	var diskTotalGB, diskAvailGB float64
	if diskTotalResp != nil && diskTotalResp.Data.ResultType == "vector" && len(diskTotalResp.Data.Result) > 0 {
		if total, err := strconv.ParseFloat(diskTotalResp.Data.Result[0].Value.Value, 64); err == nil {
			diskTotalGB = total / (1024 * 1024 * 1024)
			diskData["total_gb"] = diskTotalGB
		}
	}
	if diskAvailResp != nil && diskAvailResp.Data.ResultType == "vector" && len(diskAvailResp.Data.Result) > 0 {
		if avail, err := strconv.ParseFloat(diskAvailResp.Data.Result[0].Value.Value, 64); err == nil {
			diskAvailGB = avail / (1024 * 1024 * 1024)
			diskData["free_gb"] = diskAvailGB
		}
	}
	if diskTotalGB > 0 && diskAvailGB > 0 {
		usedGB := diskTotalGB - diskAvailGB
		diskData["used_gb"] = usedGB
		diskData["usage_percent"] = (usedGB / diskTotalGB) * 100
	}
	if diskReadIOPSResp != nil && diskReadIOPSResp.Data.ResultType == "vector" && len(diskReadIOPSResp.Data.Result) > 0 {
		if readIOPS, err := strconv.ParseFloat(diskReadIOPSResp.Data.Result[0].Value.Value, 64); err == nil {
			diskData["iops_read"] = int(readIOPS)
		}
	}
	if diskWriteIOPSResp != nil && diskWriteIOPSResp.Data.ResultType == "vector" && len(diskWriteIOPSResp.Data.Result) > 0 {
		if writeIOPS, err := strconv.ParseFloat(diskWriteIOPSResp.Data.Result[0].Value.Value, 64); err == nil {
			diskData["iops_write"] = int(writeIOPS)
		}
	}
	overview["disk"] = diskData

	// Query Network metrics
	netRxResp, err := m.queryPrometheus(ctx, "sum(rate(node_network_receive_bytes_total[5m]))", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query network RX", zap.Error(err))
	}

	netTxResp, err := m.queryPrometheus(ctx, "sum(rate(node_network_transmit_bytes_total[5m]))", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query network TX", zap.Error(err))
	}

	netConnsResp, err := m.queryPrometheus(ctx, "node_netstat_Tcp_CurrEstab", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query network connections", zap.Error(err))
	}

	netDroppedResp, err := m.queryPrometheus(ctx, "sum(rate(node_network_receive_drop_total[5m]))", "query", nil)
	if err != nil {
		m.logger.Warn("Failed to query network dropped packets", zap.Error(err))
	}

	// Network metrics
	netData := make(map[string]interface{})
	if netRxResp != nil && netRxResp.Data.ResultType == "vector" && len(netRxResp.Data.Result) > 0 {
		if rx, err := strconv.ParseFloat(netRxResp.Data.Result[0].Value.Value, 64); err == nil {
			netData["rx_bytes_per_sec"] = int(rx)
		}
	}
	if netTxResp != nil && netTxResp.Data.ResultType == "vector" && len(netTxResp.Data.Result) > 0 {
		if tx, err := strconv.ParseFloat(netTxResp.Data.Result[0].Value.Value, 64); err == nil {
			netData["tx_bytes_per_sec"] = int(tx)
		}
	}
	if netConnsResp != nil && netConnsResp.Data.ResultType == "vector" && len(netConnsResp.Data.Result) > 0 {
		if conns, err := strconv.ParseFloat(netConnsResp.Data.Result[0].Value.Value, 64); err == nil {
			netData["connections"] = int(conns)
		}
	}
	if netDroppedResp != nil && netDroppedResp.Data.ResultType == "vector" && len(netDroppedResp.Data.Result) > 0 {
		if dropped, err := strconv.ParseFloat(netDroppedResp.Data.Result[0].Value.Value, 64); err == nil {
			netData["packets_dropped"] = int(dropped)
		}
	}
	overview["network"] = netData

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

func (m *Module) handleGetServiceMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	serviceName, ok := args["service_name"].(string)
	if !ok {
		return nil, fmt.Errorf("service_name is required")
	}

	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	m.logger.Info("🔧 Collecting service metrics", zap.String("service", serviceName))

	metrics := map[string]interface{}{
		"service":   serviceName,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Query service metrics using common labels like service, job, instance
	// Try different label patterns for flexibility
	serviceFilter := fmt.Sprintf("{service=\"%s\"}", serviceName)
	jobFilter := fmt.Sprintf("{job=\"%s\"}", serviceName)

	// Request rate (QPS)
	qpsQueries := []string{
		fmt.Sprintf("sum(rate(http_requests_total%s[5m]))", serviceFilter),
		fmt.Sprintf("sum(rate(http_requests_total%s[5m]))", jobFilter),
		fmt.Sprintf("sum(rate(requests_total%s[5m]))", serviceFilter),
		fmt.Sprintf("sum(rate(requests_total%s[5m]))", jobFilter),
	}

	var qpsValue float64
	for _, query := range qpsQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				qpsValue = val
				m.logger.Debug("✅ Found QPS data", zap.String("service", serviceName), zap.Float64("qps", qpsValue))
				break
			}
		}
	}

	// Response time
	responseTimeQueries := []string{
		fmt.Sprintf("histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket%s[5m])) by (le)) * 1000", serviceFilter),
		fmt.Sprintf("histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket%s[5m])) by (le)) * 1000", jobFilter),
		fmt.Sprintf("avg(http_request_duration_seconds%s) * 1000", serviceFilter),
		fmt.Sprintf("avg(http_request_duration_seconds%s) * 1000", jobFilter),
	}

	var responseTimeMs float64
	for _, query := range responseTimeQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				responseTimeMs = val
				break
			}
		}
	}

	// Error rate
	errorRateQueries := []string{
		fmt.Sprintf("(sum(rate(http_requests_total%s{status=~\"5..\"}[5m])) / sum(rate(http_requests_total%s[5m]))) * 100", serviceFilter, serviceFilter),
		fmt.Sprintf("(sum(rate(http_requests_total%s{status=~\"5..\"}[5m])) / sum(rate(http_requests_total%s[5m]))) * 100", jobFilter, jobFilter),
	}

	var errorRate float64
	for _, query := range errorRateQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				errorRate = val
				break
			}
		}
	}

	// CPU usage
	cpuQueries := []string{
		fmt.Sprintf("avg(rate(container_cpu_usage_seconds_total%s[5m])) * 100", serviceFilter),
		fmt.Sprintf("avg(rate(container_cpu_usage_seconds_total%s[5m])) * 100", jobFilter),
		fmt.Sprintf("avg(process_cpu_seconds_total%s) * 100", serviceFilter),
	}

	var cpuUsage float64
	for _, query := range cpuQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				cpuUsage = val
				break
			}
		}
	}

	// Memory usage
	memoryQueries := []string{
		fmt.Sprintf("avg(container_memory_working_set_bytes%s) / 1024 / 1024", serviceFilter),
		fmt.Sprintf("avg(container_memory_working_set_bytes%s) / 1024 / 1024", jobFilter),
		fmt.Sprintf("avg(process_resident_memory_bytes%s) / 1024 / 1024", serviceFilter),
	}

	var memoryMB float64
	for _, query := range memoryQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				memoryMB = val
				break
			}
		}
	}

	// Active connections (if available)
	connQueries := []string{
		fmt.Sprintf("sum(http_connections_active%s)", serviceFilter),
		fmt.Sprintf("sum(nginx_connections_active%s)", serviceFilter),
	}

	var activeConnections float64
	for _, query := range connQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				activeConnections = val
				break
			}
		}
	}

	// Network throughput
	throughputQueries := []string{
		fmt.Sprintf("(sum(rate(container_network_receive_bytes_total%s[5m])) + sum(rate(container_network_transmit_bytes_total%s[5m]))) / 1024 / 1024", serviceFilter, serviceFilter),
	}

	var throughputMbps float64
	for _, query := range throughputQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				throughputMbps = val
				break
			}
		}
	}

	// Determine health status based on metrics
	health := "healthy"
	if errorRate > 5.0 || cpuUsage > 80.0 || responseTimeMs > 1000 {
		health = "degraded"
	}
	if errorRate > 10.0 || cpuUsage > 90.0 || responseTimeMs > 5000 {
		health = "unhealthy"
	}

	metrics["health"] = health
	metrics["metrics"] = map[string]interface{}{
		"requests_per_second": qpsValue,
		"response_time_ms":    responseTimeMs,
		"error_rate_percent":  errorRate,
		"cpu_usage_percent":   cpuUsage,
		"memory_usage_mb":     memoryMB,
		"active_connections":  int(activeConnections),
		"throughput_mbps":     throughputMbps,
	}

	// Query status code distribution
	statusCodeData := make(map[string]int)
	statusCodeQueries := map[string]string{
		"2xx": fmt.Sprintf("sum(rate(http_requests_total%s{status=~\"2..\"}[5m])) * 300", serviceFilter), // multiply by 5min to get count
		"3xx": fmt.Sprintf("sum(rate(http_requests_total%s{status=~\"3..\"}[5m])) * 300", serviceFilter),
		"4xx": fmt.Sprintf("sum(rate(http_requests_total%s{status=~\"4..\"}[5m])) * 300", serviceFilter),
		"5xx": fmt.Sprintf("sum(rate(http_requests_total%s{status=~\"5..\"}[5m])) * 300", serviceFilter),
	}

	for statusRange, query := range statusCodeQueries {
		if resp, err := m.queryPrometheus(ctx, query, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				statusCodeData[statusRange] = int(val)
			}
		}
	}

	// If no status code data found with service filter, try job filter
	if len(statusCodeData) == 0 {
		for statusRange, query := range statusCodeQueries {
			jobQuery := strings.Replace(query, serviceFilter, jobFilter, -1)
			if resp, err := m.queryPrometheus(ctx, jobQuery, "query", nil); err == nil && resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
				if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
					statusCodeData[statusRange] = int(val)
				}
			}
		}
	}

	if len(statusCodeData) > 0 {
		metrics["status_codes"] = statusCodeData
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return nil, err
	}

	m.logger.Info("🔧 Service metrics collection completed",
		zap.String("service", serviceName),
		zap.String("health", metrics["health"].(string)),
		zap.Float64("qps", qpsValue),
		zap.Float64("error_rate", errorRate))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (m *Module) handleGetMetricsServices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	m.logger.Info("🔍 Discovering services from Prometheus")

	// Query for unique services using different common label patterns
	queries := []string{
		"group by (service) (up{service!=\"\"})",
		"group by (job) (up{job!=\"\"})",
		"group by (app) (up{app!=\"\"})",
		"group by (application) (up{application!=\"\"})",
	}

	servicesSet := make(map[string]bool)

	for _, query := range queries {
		resp, err := m.queryPrometheus(ctx, query, "query", nil)
		if err != nil {
			m.logger.Warn("Failed to query services", zap.String("query", query), zap.Error(err))
			continue
		}

		if resp.Data.ResultType == "vector" {
			for _, result := range resp.Data.Result {
				// Check each possible label
				for _, labelKey := range []string{"service", "job", "app", "application"} {
					if serviceName, exists := result.Labels[labelKey]; exists && serviceName != "" {
						servicesSet[serviceName] = true
					}
				}
			}
		}
	}

	// Convert set to slice
	services := make([]string, 0, len(servicesSet))
	for service := range servicesSet {
		services = append(services, service)
	}

	// If no services found from labels, try to extract from metric names
	if len(services) == 0 {
		m.logger.Info("No services found from labels, trying to extract from metric names")

		// Query all metrics and try to extract service names from metric names
		metricQuery := "group by (__name__) ({__name__=~\".*\"})"
		resp, err := m.queryPrometheus(ctx, metricQuery, "query", nil)
		if err == nil && resp.Data.ResultType == "vector" {
			for _, result := range resp.Data.Result {
				if metricName, exists := result.Labels["__name__"]; exists {
					// Try to extract service name from metric name patterns like:
					// service_name_metric, http_requests_total{service="name"}, etc.
					parts := strings.Split(metricName, "_")
					if len(parts) >= 2 {
						// Use first part as potential service name
						serviceName := parts[0]
						if serviceName != "up" && serviceName != "node" && serviceName != "prometheus" {
							servicesSet[serviceName] = true
						}
					}
				}
			}

			// Convert updated set to slice
			services = make([]string, 0, len(servicesSet))
			for service := range servicesSet {
				services = append(services, service)
			}
		}
	}

	response := map[string]interface{}{
		"services": services,
		"total":    len(services),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	m.logger.Info("🔍 Service discovery completed",
		zap.Int("service_count", len(services)),
		zap.Strings("services", services))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (m *Module) handleGetMetricHistory(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	metricName, ok := args["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name is required")
	}

	timeRange := "1h"
	if val, ok := args["time_range"].(string); ok {
		timeRange = val
	}

	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

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
		return nil, fmt.Errorf("unsupported time range: %s", timeRange)
	}

	now := time.Now()
	start := now.Add(-duration)

	// Determine appropriate step based on time range
	step := "60s"
	if duration > 24*time.Hour {
		step = "5m"
	}
	if duration > 7*24*time.Hour {
		step = "1h"
	}

	// Execute range query
	params := make(map[string]string)
	params["start"] = fmt.Sprintf("%d", start.Unix())
	params["end"] = fmt.Sprintf("%d", now.Unix())
	params["step"] = step

	resp, err := m.queryPrometheus(ctx, metricName, "query_range", params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute range query: %w", err)
	}

	// Process response data
	points := make([]map[string]interface{}, 0)
	var min, max, sum float64
	var count int
	var current float64

	if resp.Data.ResultType == "matrix" {
		for _, result := range resp.Data.Result {
			// For simplicity, use the first result series
			for _, value := range result.Values {
				timestamp := time.Unix(int64(value.Timestamp), 0)
				val, err := strconv.ParseFloat(value.Value, 64)
				if err != nil {
					continue
				}

				points = append(points, map[string]interface{}{
					"timestamp": timestamp.Format(time.RFC3339),
					"value":     val,
				})

				// Calculate statistics
				if count == 0 {
					min = val
					max = val
				} else {
					if val < min {
						min = val
					}
					if val > max {
						max = val
					}
				}
				sum += val
				count++
				current = val // last value becomes current
			}
			// Only process the first series for now
			break
		}
	} else if resp.Data.ResultType == "vector" {
		// Handle instant query result as a single point
		for _, result := range resp.Data.Result {
			val, err := strconv.ParseFloat(result.Value.Value, 64)
			if err != nil {
				continue
			}

			timestamp := time.Unix(int64(result.Value.Timestamp), 0)
			points = append(points, map[string]interface{}{
				"timestamp": timestamp.Format(time.RFC3339),
				"value":     val,
			})

			min = val
			max = val
			sum = val
			count = 1
			current = val
			break
		}
	}

	// Calculate average
	avg := 0.0
	if count > 0 {
		avg = sum / float64(count)
	}

	// Determine unit based on metric name
	unit := ""
	if strings.Contains(metricName, "percent") || strings.Contains(metricName, "ratio") {
		unit = "percent"
	} else if strings.Contains(metricName, "bytes") {
		unit = "bytes"
	} else if strings.Contains(metricName, "seconds") {
		unit = "seconds"
	} else if strings.Contains(metricName, "requests") {
		unit = "requests/sec"
	}

	history := map[string]interface{}{
		"metric":     metricName,
		"time_range": timeRange,
		"unit":       unit,
		"points":     points,
		"summary": map[string]interface{}{
			"min":     min,
			"max":     max,
			"avg":     avg,
			"current": current,
			"count":   count,
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

func (m *Module) handleGetMetricsAlerts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if m.config.Prometheus == nil {
		return nil, fmt.Errorf("Prometheus configuration is not available")
	}

	m.logger.Info("🚨 Checking metrics alerts and thresholds")

	// Define alert rules to check
	alertRules := []map[string]interface{}{
		{
			"name":        "High CPU Usage",
			"query":       "100 - (avg(irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
			"threshold":   80.0,
			"severity":    "warning",
			"description": "CPU usage is above threshold",
		},
		{
			"name":        "Memory Usage",
			"query":       "(1 - (avg(node_memory_MemAvailable_bytes) / avg(node_memory_MemTotal_bytes))) * 100",
			"threshold":   85.0,
			"severity":    "critical",
			"description": "Memory usage is above threshold",
		},
		{
			"name":        "Disk Space",
			"query":       "(1 - (avg(node_filesystem_avail_bytes{fstype!=\"tmpfs\"}) / avg(node_filesystem_size_bytes{fstype!=\"tmpfs\"}))) * 100",
			"threshold":   90.0,
			"severity":    "warning",
			"description": "Disk space is above threshold",
		},
		{
			"name":        "High Error Rate",
			"query":       "(sum(rate(http_requests_total{status=~\"5..\"}[5m])) / sum(rate(http_requests_total[5m]))) * 100",
			"threshold":   5.0,
			"severity":    "critical",
			"description": "HTTP error rate is above threshold",
		},
		{
			"name":        "High Response Time",
			"query":       "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) * 1000",
			"threshold":   1000.0,
			"severity":    "warning",
			"description": "95th percentile response time is above threshold",
		},
		{
			"name":        "Low Available Memory",
			"query":       "avg(node_memory_MemAvailable_bytes) / 1024 / 1024 / 1024",
			"threshold":   2.0,
			"comparison":  "less_than",
			"severity":    "critical",
			"description": "Available memory is below threshold",
		},
	}

	alerts := make([]map[string]interface{}, 0)
	activeAlerts := 0

	for _, rule := range alertRules {
		alertName := rule["name"].(string)
		query := rule["query"].(string)
		threshold := rule["threshold"].(float64)
		severity := rule["severity"].(string)
		description := rule["description"].(string)

		comparison := "greater_than"
		if comp, exists := rule["comparison"]; exists {
			comparison = comp.(string)
		}

		// Execute the query
		resp, err := m.queryPrometheus(ctx, query, "query", nil)
		if err != nil {
			m.logger.Warn("Failed to execute alert query",
				zap.String("alert", alertName),
				zap.String("query", query),
				zap.Error(err))

			// Add alert with unknown status
			alerts = append(alerts, map[string]interface{}{
				"name":        alertName,
				"metric":      query,
				"threshold":   threshold,
				"current":     nil,
				"status":      "unknown",
				"severity":    severity,
				"description": description,
				"error":       err.Error(),
			})
			continue
		}

		var currentValue float64
		var hasValue bool

		if resp.Data.ResultType == "vector" && len(resp.Data.Result) > 0 {
			if val, parseErr := strconv.ParseFloat(resp.Data.Result[0].Value.Value, 64); parseErr == nil {
				currentValue = val
				hasValue = true
			}
		}

		// Determine alert status
		status := "ok"
		if hasValue {
			var isTriggered bool
			if comparison == "less_than" {
				isTriggered = currentValue < threshold
			} else {
				isTriggered = currentValue > threshold
			}

			if isTriggered {
				status = "firing"
				activeAlerts++
			}
		} else {
			status = "no_data"
		}

		alert := map[string]interface{}{
			"name":        alertName,
			"metric":      query,
			"threshold":   threshold,
			"status":      status,
			"severity":    severity,
			"description": description,
			"comparison":  comparison,
		}

		if hasValue {
			alert["current"] = currentValue
		}

		alerts = append(alerts, alert)
	}

	// Query Alertmanager alerts if available
	alertmanagerQuery := "ALERTS"
	if alertResp, err := m.queryPrometheus(ctx, alertmanagerQuery, "query", nil); err == nil {
		if alertResp.Data.ResultType == "vector" {
			for _, result := range alertResp.Data.Result {
				alertName := "Unknown"
				if name, exists := result.Labels["alertname"]; exists {
					alertName = name
				}

				severity := "info"
				if sev, exists := result.Labels["severity"]; exists {
					severity = sev
				}

				status := "firing"
				if val, parseErr := strconv.ParseFloat(result.Value.Value, 64); parseErr == nil && val == 0 {
					status = "resolved"
				} else if val == 1 {
					activeAlerts++
				}

				alert := map[string]interface{}{
					"name":        fmt.Sprintf("Alertmanager: %s", alertName),
					"metric":      alertmanagerQuery,
					"status":      status,
					"severity":    severity,
					"description": fmt.Sprintf("Alertmanager alert: %s", alertName),
					"labels":      result.Labels,
				}

				alerts = append(alerts, alert)
			}
		}
	}

	response := map[string]interface{}{
		"alerts":        alerts,
		"total":         len(alerts),
		"active_alerts": activeAlerts,
		"last_updated":  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	m.logger.Info("🚨 Alerts check completed",
		zap.Int("total_alerts", len(alerts)),
		zap.Int("active_alerts", activeAlerts))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

func (m *Module) handleQueryMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required")
	}

	// Execute instant query
	params := make(map[string]string)
	params["time"] = fmt.Sprintf("%d", time.Now().Unix())

	promResp, err := m.queryPrometheus(ctx, query, "query", params)
	if err != nil {
		m.logger.Error("Failed to execute Prometheus query", zap.Error(err))
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
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
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

func (m *Module) handleQueryMetricsRange(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required")
	}

	timeRange, ok := args["time_range"].(string)
	if !ok {
		return nil, fmt.Errorf("time_range is required")
	}

	// Get step parameter or use default
	step := "60s"
	if stepArg, ok := args["step"].(string); ok && stepArg != "" {
		step = stepArg
	}

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
		return nil, fmt.Errorf("unsupported time range: %s", timeRange)
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
		m.logger.Error("Failed to execute Prometheus range query", zap.Error(err))
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// Kubernetes resource listing handlers
func (m *Module) handleGetClusters(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Query to get all clusters from metrics
	query := "group by (cluster) (up{cluster!=\"\"})"

	response, err := m.queryPrometheus(ctx, query, "query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query clusters: %w", err)
	}

	// Extract unique cluster names
	clusters := make([]string, 0)
	if response.Data.ResultType == "vector" {
		for _, result := range response.Data.Result {
			if cluster, exists := result.Labels["cluster"]; exists {
				clusters = append(clusters, cluster)
			}
		}
	}

	result := map[string]interface{}{
		"clusters": clusters,
		"count":    len(clusters),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
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

func (m *Module) handleGetNamespaces(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	var query string
	if cluster, ok := args["cluster"].(string); ok && cluster != "" {
		query = fmt.Sprintf("group by (namespace) (kube_namespace_created{cluster=\"%s\"})", cluster)
	} else {
		query = "group by (namespace, cluster) (kube_namespace_created)"
	}

	response, err := m.queryPrometheus(ctx, query, "query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query namespaces: %w", err)
	}

	// Extract namespaces
	namespaces := make([]map[string]string, 0)
	if response.Data.ResultType == "vector" {
		for _, result := range response.Data.Result {
			ns := make(map[string]string)
			if namespace, exists := result.Labels["namespace"]; exists {
				ns["namespace"] = namespace
			}
			if cluster, exists := result.Labels["cluster"]; exists {
				ns["cluster"] = cluster
			}
			namespaces = append(namespaces, ns)
		}
	}

	result := map[string]interface{}{
		"namespaces": namespaces,
		"count":      len(namespaces),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
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

func (m *Module) handleGetPods(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Build query with filters
	var queryFilters []string
	if cluster, ok := args["cluster"].(string); ok && cluster != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("cluster=\"%s\"", cluster))
	}
	if namespace, ok := args["namespace"].(string); ok && namespace != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("namespace=\"%s\"", namespace))
	}

	var filterStr string
	if len(queryFilters) > 0 {
		filterStr = "{" + strings.Join(queryFilters, ",") + "}"
	}

	query := fmt.Sprintf("group by (pod, namespace, cluster) (kube_pod_info%s)", filterStr)

	response, err := m.queryPrometheus(ctx, query, "query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pods: %w", err)
	}

	// Extract pods
	pods := make([]map[string]string, 0)
	if response.Data.ResultType == "vector" {
		for _, result := range response.Data.Result {
			pod := make(map[string]string)
			if podName, exists := result.Labels["pod"]; exists {
				pod["pod"] = podName
			}
			if namespace, exists := result.Labels["namespace"]; exists {
				pod["namespace"] = namespace
			}
			if cluster, exists := result.Labels["cluster"]; exists {
				pod["cluster"] = cluster
			}
			pods = append(pods, pod)
		}
	}

	// Apply limit
	limit := 50
	if limitStr, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if len(pods) > limit {
		pods = pods[:limit]
	}

	result := map[string]interface{}{
		"pods":  pods,
		"count": len(pods),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
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

func (m *Module) handleGetPodResourceUsage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Build filter string for queries
	var queryFilters []string
	if cluster, ok := args["cluster"].(string); ok && cluster != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("cluster=\"%s\"", cluster))
	}
	if namespace, ok := args["namespace"].(string); ok && namespace != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("namespace=\"%s\"", namespace))
	}
	if pod, ok := args["pod"].(string); ok && pod != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("pod=\"%s\"", pod))
	}

	var filterStr string
	if len(queryFilters) > 0 {
		filterStr = "{" + strings.Join(queryFilters, ",") + "}"
	}

	// Query CPU usage (rate over 5 minutes)
	cpuQuery := fmt.Sprintf("sum by (pod, namespace, cluster) (rate(container_cpu_usage_seconds_total%s[5m]))", filterStr)

	// Query Memory usage (working set)
	memQuery := fmt.Sprintf("sum by (pod, namespace, cluster) (container_memory_working_set_bytes%s)", filterStr)

	// Execute both queries
	cpuResponse, err := m.queryPrometheus(ctx, cpuQuery, "query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pod CPU usage: %w", err)
	}

	memResponse, err := m.queryPrometheus(ctx, memQuery, "query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query pod memory usage: %w", err)
	}

	// Process results
	podUsage := make(map[string]map[string]interface{})

	// Process CPU data
	if cpuResponse.Data.ResultType == "vector" {
		for _, result := range cpuResponse.Data.Result {
			key := fmt.Sprintf("%s/%s/%s",
				result.Labels["cluster"], result.Labels["namespace"], result.Labels["pod"])
			if podUsage[key] == nil {
				podUsage[key] = make(map[string]interface{})
				podUsage[key]["pod"] = result.Labels["pod"]
				podUsage[key]["namespace"] = result.Labels["namespace"]
				podUsage[key]["cluster"] = result.Labels["cluster"]
			}
			podUsage[key]["cpu_cores"] = result.Value.Value
		}
	}

	// Process Memory data
	if memResponse.Data.ResultType == "vector" {
		for _, result := range memResponse.Data.Result {
			key := fmt.Sprintf("%s/%s/%s",
				result.Labels["cluster"], result.Labels["namespace"], result.Labels["pod"])
			if podUsage[key] == nil {
				podUsage[key] = make(map[string]interface{})
				podUsage[key]["pod"] = result.Labels["pod"]
				podUsage[key]["namespace"] = result.Labels["namespace"]
				podUsage[key]["cluster"] = result.Labels["cluster"]
			}
			podUsage[key]["memory_bytes"] = result.Value.Value
		}
	}

	// Convert to slice and apply limit
	pods := make([]map[string]interface{}, 0, len(podUsage))
	for _, usage := range podUsage {
		pods = append(pods, usage)
	}

	limit := 20
	if limitStr, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if len(pods) > limit {
		pods = pods[:limit]
	}

	result := map[string]interface{}{
		"pod_usage": pods,
		"count":     len(pods),
		"queries": map[string]string{
			"cpu_query": cpuQuery,
			"mem_query": memQuery,
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
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

func (m *Module) handleGetNodeResourceUsage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Build filter string for queries
	var queryFilters []string
	if cluster, ok := args["cluster"].(string); ok && cluster != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("cluster=\"%s\"", cluster))
	}
	if node, ok := args["node"].(string); ok && node != "" {
		queryFilters = append(queryFilters, fmt.Sprintf("instance=\"%s\"", node))
	}

	var filterStr string
	if len(queryFilters) > 0 {
		filterStr = "{" + strings.Join(queryFilters, ",") + "}"
	}

	// Query CPU usage percentage (100 - idle)
	cpuQuery := fmt.Sprintf("100 - (avg by (instance, cluster) (irate(node_cpu_seconds_total%s{mode=\"idle\"}[5m])) * 100)", filterStr)

	// Query Memory usage percentage
	memQuery := fmt.Sprintf("(1 - (node_memory_MemAvailable_bytes%s / node_memory_MemTotal_bytes%s)) * 100", filterStr, filterStr)

	// Execute both queries
	cpuResponse, err := m.queryPrometheus(ctx, cpuQuery, "query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query node CPU usage: %w", err)
	}

	memResponse, err := m.queryPrometheus(ctx, memQuery, "query", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query node memory usage: %w", err)
	}

	// Process results
	nodeUsage := make(map[string]map[string]interface{})

	// Process CPU data
	if cpuResponse.Data.ResultType == "vector" {
		for _, result := range cpuResponse.Data.Result {
			key := fmt.Sprintf("%s/%s", result.Labels["cluster"], result.Labels["instance"])
			if nodeUsage[key] == nil {
				nodeUsage[key] = make(map[string]interface{})
				nodeUsage[key]["node"] = result.Labels["instance"]
				nodeUsage[key]["cluster"] = result.Labels["cluster"]
			}
			nodeUsage[key]["cpu_percentage"] = result.Value.Value
		}
	}

	// Process Memory data
	if memResponse.Data.ResultType == "vector" {
		for _, result := range memResponse.Data.Result {
			key := fmt.Sprintf("%s/%s", result.Labels["cluster"], result.Labels["instance"])
			if nodeUsage[key] == nil {
				nodeUsage[key] = make(map[string]interface{})
				nodeUsage[key]["node"] = result.Labels["instance"]
				nodeUsage[key]["cluster"] = result.Labels["cluster"]
			}
			nodeUsage[key]["memory_percentage"] = result.Value.Value
		}
	}

	// Convert to slice and apply limit
	nodes := make([]map[string]interface{}, 0, len(nodeUsage))
	for _, usage := range nodeUsage {
		nodes = append(nodes, usage)
	}

	limit := 20
	if limitStr, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if len(nodes) > limit {
		nodes = nodes[:limit]
	}

	result := map[string]interface{}{
		"node_usage": nodes,
		"count":      len(nodes),
		"queries": map[string]string{
			"cpu_query": cpuQuery,
			"mem_query": memQuery,
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
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
