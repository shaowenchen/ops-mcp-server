package metrics

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolConfig defines configuration for a single tool
type ToolConfig struct {
	Enabled     bool   // Whether the tool is enabled
	Name        string // Tool name
	Description string // Tool description
}

// MetricsToolsConfig defines configuration for all tools
type MetricsToolsConfig struct {
	ListMetrics  ToolConfig
	QueryMetrics ToolConfig
	QueryRange   ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() MetricsToolsConfig {
	return MetricsToolsConfig{
		ListMetrics: ToolConfig{
			Enabled:     true,
			Name:        "list-metrics",
			Description: "List all available metrics from Prometheus. Returns metric names, types, and basic information.",
		},
		QueryMetrics: ToolConfig{
			Enabled:     true,
			Name:        "query-metrics",
			Description: "Execute a custom PromQL instant query. Examples: 'up', 'cpu_usage_percent', 'sum(rate(http_requests_total[5m]))'",
		},
		QueryRange: ToolConfig{
			Enabled:     true,
			Name:        "query-metrics-range",
			Description: "Execute a custom PromQL range query over a time period. Examples: 'rate(cpu_usage[5m])', 'sum(memory_usage_bytes) by (pod)'",
		},
	}
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
func (m *Module) BuildTools(toolsConfig MetricsToolsConfig) []server.ServerTool {
	var tools []server.ServerTool

	// List Metrics Tool
	if toolsConfig.ListMetrics.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildListMetricsToolDefinition(toolsConfig.ListMetrics),
			Handler: m.handleListMetrics,
		})
	}

	// Query Metrics Tool
	if toolsConfig.QueryMetrics.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildQueryMetricsToolDefinition(toolsConfig.QueryMetrics),
			Handler: m.handleExecuteQuery,
		})
	}

	// Query Range Tool
	if toolsConfig.QueryRange.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildQueryRangeToolDefinition(toolsConfig.QueryRange),
			Handler: m.handleExecuteRangeQuery,
		})
	}

	return tools
}

// Tool definition builder methods

func (m *Module) buildListMetricsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("search", mcp.Description("Filter metrics by name pattern (optional)")),
		mcp.WithString("limit", mcp.Description("Maximum number of metrics to return (default: 100)")),
	)
}

func (m *Module) buildQueryMetricsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("query", mcp.Required(), mcp.Description("PromQL query expression to execute")),
	)
}

func (m *Module) buildQueryRangeToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("query", mcp.Required(), mcp.Description("PromQL query expression to execute")),
		mcp.WithString("time_range", mcp.Required(), mcp.Description("Time range for query (1h, 24h, 7d, 30d)")),
		mcp.WithString("step", mcp.Description("Query resolution step (default: 60s, examples: 30s, 5m, 1h)")),
	)
}
