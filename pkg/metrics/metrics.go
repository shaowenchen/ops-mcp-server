package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// Metrics holds all the Prometheus metrics for the application
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal       *prometheus.CounterVec
	HTTPRequestDuration     *prometheus.HistogramVec
	HTTPRequestSize         *prometheus.HistogramVec
	HTTPResponseSize        *prometheus.HistogramVec
	HTTPRequestsInFlight    *prometheus.GaugeVec

	// SSE metrics
	SSEConnectionsTotal     prometheus.Counter
	SSEActiveConnections    prometheus.Gauge
	SSEConnectionDuration   prometheus.Histogram

	// MCP tool metrics
	MCPToolCallsTotal       *prometheus.CounterVec
	MCPToolCallDuration     *prometheus.HistogramVec
	MCPToolErrorsTotal      *prometheus.CounterVec

	// Module metrics
	ModuleEnabled           *prometheus.GaugeVec
	ModuleRequestsTotal     *prometheus.CounterVec

	// Backend service metrics
	BackendRequestsTotal    *prometheus.CounterVec
	BackendRequestDuration  *prometheus.HistogramVec
	BackendErrorsTotal      *prometheus.CounterVec

	// Auth metrics
	AuthRequestsTotal       *prometheus.CounterVec
	AuthValidationDuration  prometheus.Histogram

	// System metrics
	ProcessGoroutines        prometheus.Gauge
	ProcessMemoryBytes       *prometheus.GaugeVec

	logger *zap.Logger
}

var (
	// Default instance
	defaultMetrics *Metrics
)

// Init initializes the metrics system
func Init(logger *zap.Logger) *Metrics {
	if defaultMetrics != nil {
		return defaultMetrics
	}

	m := &Metrics{
		logger: logger,
	}

	// HTTP metrics
	m.HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code", "mode"},
	)

	m.HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "endpoint"},
	)

	m.HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "endpoint"},
	)

	m.HTTPRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
		[]string{"endpoint"},
	)

	// SSE metrics
	m.SSEConnectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sse_connections_total",
			Help: "Total number of SSE connections",
		},
	)

	m.SSEActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sse_active_connections",
			Help: "Number of active SSE connections",
		},
	)

	m.SSEConnectionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "sse_connection_duration_seconds",
			Help:    "SSE connection duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600}, // 1s to 1h
		},
	)

	// MCP tool metrics
	m.MCPToolCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_tool_calls_total",
			Help: "Total number of MCP tool calls",
		},
		[]string{"tool_name", "module", "status"},
	)

	m.MCPToolCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_tool_call_duration_seconds",
			Help:    "MCP tool call duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tool_name", "module"},
	)

	m.MCPToolErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_tool_errors_total",
			Help: "Total number of MCP tool errors",
		},
		[]string{"tool_name", "module", "error_type"},
	)

	// Module metrics
	m.ModuleEnabled = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "module_enabled",
			Help: "Module enabled status (0=disabled, 1=enabled)",
		},
		[]string{"module_name"},
	)

	m.ModuleRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "module_requests_total",
			Help: "Total number of requests per module",
		},
		[]string{"module_name"},
	)

	// Backend service metrics
	m.BackendRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_requests_total",
			Help: "Total number of backend service requests",
		},
		[]string{"backend", "status"},
	)

	m.BackendRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_request_duration_seconds",
			Help:    "Backend service request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend"},
	)

	m.BackendErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_errors_total",
			Help: "Total number of backend service errors",
		},
		[]string{"backend", "error_type"},
	)

	// Auth metrics
	m.AuthRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_requests_total",
			Help: "Total number of authentication requests",
		},
		[]string{"status"},
	)

	m.AuthValidationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "auth_token_validation_duration_seconds",
			Help:    "Authentication token validation duration in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
	)

	// System metrics
	m.ProcessGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "process_goroutines",
			Help: "Number of goroutines",
		},
	)

	m.ProcessMemoryBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "process_memory_bytes",
			Help: "Process memory usage in bytes",
		},
		[]string{"type"},
	)

	// Register build info
	buildInfo := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "build_info",
			Help: "Build information",
		},
		[]string{"version", "git_commit", "build_date"},
	)
	// This will be set by the application
	_ = buildInfo

	defaultMetrics = m
	logger.Info("Metrics system initialized")
	return m
}

// Get returns the default metrics instance
func Get() *Metrics {
	return defaultMetrics
}

// SetModuleEnabled sets the enabled status for a module
func (m *Metrics) SetModuleEnabled(moduleName string, enabled bool) {
	value := 0.0
	if enabled {
		value = 1.0
	}
	m.ModuleEnabled.WithLabelValues(moduleName).Set(value)
}

// SetBuildInfo sets the build information metric
func SetBuildInfo(version, gitCommit, buildDate string) {
	buildInfo := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "build_info",
			Help: "Build information",
		},
		[]string{"version", "git_commit", "build_date"},
	)
	buildInfo.WithLabelValues(version, gitCommit, buildDate).Set(1)
}

