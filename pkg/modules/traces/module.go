package traces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
	appMetrics "github.com/shaowenchen/ops-mcp-server/pkg/metrics"
	"go.uber.org/zap"
)

// Config contains Jaeger module configuration
type Config struct {
	Endpoint string      `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Protocol string      `mapstructure:"protocol" json:"protocol" yaml:"protocol"`
	Port     int         `mapstructure:"port" json:"port" yaml:"port"`
	Auth     string      `mapstructure:"auth" json:"auth" yaml:"auth"`
	Timeout  int         `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	Tools    ToolsConfig `mapstructure:"tools" json:"tools" yaml:"tools"`
}

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Suffix string `mapstructure:"suffix" json:"suffix" yaml:"suffix"`
}

// Module represents the Jaeger module
type Module struct {
	config     *Config
	logger     *zap.Logger
	httpClient *http.Client
	baseURL    string
}

// New creates a new Jaeger module
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("jaeger config is required")
	}

	// Set defaults
	if config.Protocol == "" {
		config.Protocol = "HTTP"
	}
	if config.Port == 0 {
		if config.Protocol == "GRPC" {
			config.Port = 16685
		} else {
			config.Port = 16686
		}
	}

	// Build base URL only if endpoint is configured
	var baseURL string
	if config.Endpoint != "" {
		baseURL = config.Endpoint
		if !strings.HasPrefix(baseURL, "http") {
			if config.Protocol == "GRPC" {
				baseURL = "http://" + baseURL
			} else {
				baseURL = "http://" + baseURL
			}
		}
		if !strings.Contains(baseURL, ":") {
			baseURL = fmt.Sprintf("%s:%d", baseURL, config.Port)
		}
	}

	// Set default timeout if not specified
	timeout := 15 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
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
		logger: logger.Named("jaeger"),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout, // Use configured timeout for faster connection release
		},
		baseURL: baseURL,
	}

	if config.Endpoint != "" {
		m.logger.Info("Jaeger module created with Jaeger backend",
			zap.String("endpoint", config.Endpoint),
			zap.String("protocol", config.Protocol),
			zap.Int("port", config.Port),
			zap.Int("timeout", config.Timeout),
			zap.String("base_url", baseURL))
	} else {
		m.logger.Info("Jaeger module created without Jaeger configuration - tools will return configuration required error")
	}

	return m, nil
}

// makeJaegerRequest creates and executes an HTTP request to Jaeger API
func (m *Module) makeJaegerRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := m.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	m.logger.Info("Making Jaeger request",
		zap.String("method", method),
		zap.String("url", url))

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add authorization header if provided
	if m.config.Auth != "" {
		req.Header.Set("Authorization", m.config.Auth)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.logger.Error("Jaeger request failed",
			zap.String("method", method),
			zap.String("url", url),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	m.logger.Info("Jaeger response received",
		zap.String("method", method),
		zap.String("url", url),
		zap.Int("status_code", resp.StatusCode))
	return resp, nil
}

// GetTools returns all MCP tools for the Jaeger module
func (m *Module) GetTools() []server.ServerTool {
	toolsConfig := GetDefaultToolsConfig()
	return m.BuildTools(toolsConfig)
}

// BuildToolName builds tool name based on configuration
func (m *Module) BuildToolName(baseName string) string {
	toolName := baseName
	if m.config.Tools.Prefix != "" {
		toolName = m.config.Tools.Prefix + toolName
	}
	if m.config.Tools.Suffix != "" {
		toolName = toolName + m.config.Tools.Suffix
	}
	return toolName
}

// BuildTools builds tool list based on configuration
func (m *Module) BuildTools(toolsConfig JaegerToolsConfig) []server.ServerTool {
	var tools []server.ServerTool

	if toolsConfig.GetServices.Enabled {
		toolName := m.BuildToolName(toolsConfig.GetServices.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildGetServicesToolDefinition(toolsConfig.GetServices),
			Handler: appMetrics.WrapToolHandler(m.handleGetServices, toolName, "traces"),
		})
	}

	if toolsConfig.GetOperations.Enabled {
		toolName := m.BuildToolName(toolsConfig.GetOperations.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildGetOperationsToolDefinition(toolsConfig.GetOperations),
			Handler: appMetrics.WrapToolHandler(m.handleGetOperations, toolName, "traces"),
		})
	}

	if toolsConfig.GetTrace.Enabled {
		toolName := m.BuildToolName(toolsConfig.GetTrace.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildGetTraceToolDefinition(toolsConfig.GetTrace),
			Handler: appMetrics.WrapToolHandler(m.handleGetTrace, toolName, "traces"),
		})
	}

	if toolsConfig.FindTraces.Enabled {
		toolName := m.BuildToolName(toolsConfig.FindTraces.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildFindTracesToolDefinition(toolsConfig.FindTraces),
			Handler: appMetrics.WrapToolHandler(m.handleFindTraces, toolName, "traces"),
		})
	}

	return tools
}
