package logs

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

// LogsToolsConfig defines configuration for all tools
type LogsToolsConfig struct {
	SearchLogs  ToolConfig
	PodLogs     ToolConfig
	PathLogs    ToolConfig
	ListIndices ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() LogsToolsConfig {
	return LogsToolsConfig{
		SearchLogs: ToolConfig{
			Enabled:     true,
			Name:        "search-logs",
			Description: "Full-text search across log messages",
		},
		PodLogs: ToolConfig{
			Enabled:     true,
			Name:        "get-pod-logs",
			Description: "Query logs for a specific Kubernetes pod",
		},
		PathLogs: ToolConfig{
			Enabled:     true,
			Name:        "get-path-logs",
			Description: "Query logs for a specific path",
		},
		ListIndices: ToolConfig{
			Enabled:     true,
			Name:        "list-log-indices",
			Description: "List all indices in the Elasticsearch cluster",
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
func (m *Module) BuildTools(toolsConfig LogsToolsConfig) []server.ServerTool {
	var tools []server.ServerTool

	// Search Logs Tool
	if toolsConfig.SearchLogs.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildSearchLogsToolDefinition(toolsConfig.SearchLogs),
			Handler: m.handleSearchLogs,
		})
	}

	// Pod Logs Tool
	if toolsConfig.PodLogs.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildPodLogsToolDefinition(toolsConfig.PodLogs),
			Handler: m.handleGetPodLogs,
		})
	}

	// Path Logs Tool
	if toolsConfig.PathLogs.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildPathLogsToolDefinition(toolsConfig.PathLogs),
			Handler: m.handleGetPathLogs,
		})
	}

	// List Indices Tool
	if toolsConfig.ListIndices.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildListIndicesToolDefinition(toolsConfig.ListIndices),
			Handler: m.handleListIndices,
		})
	}

	return tools
}

// Tool definition builder methods

func (m *Module) buildSearchLogsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("search_term", mcp.Required(), mcp.Description("Text to search for in log messages")),
		mcp.WithString("size", mcp.Description("Maximum number of results to return (default: 50)")),
		mcp.WithString("start_time", mcp.Description("Start time for log filtering (e.g., '2025-11-14T00:00:00Z' or 'now-1h')")),
		mcp.WithString("end_time", mcp.Description("End time for log filtering (e.g., '2025-11-14T23:59:59Z' or 'now')")),
		mcp.WithString("index", mcp.Description("Specific index or index pattern to search (default: * for all indices)")),
	)
}

func (m *Module) buildPodLogsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("pod", mcp.Required(), mcp.Description("Pod name to query logs for (e.g., polity-v5-55899f979f-xt7rx)")),
		mcp.WithString("size", mcp.Description("Maximum number of log entries to return (default: 100)")),
		mcp.WithString("start_time", mcp.Description("Start time for log filtering (ISO format or relative like '1h', '30m', '7d')")),
		mcp.WithString("end_time", mcp.Description("End time for log filtering (ISO format or relative like '1h', '30m', '7d')")),
		mcp.WithString("index", mcp.Description("Specific index or index pattern to search (default: * for all indices)")),
	)
}

func (m *Module) buildListIndicesToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("format", mcp.Description("Output format (table, json) - default: table")),
		mcp.WithString("health", mcp.Description("Filter by health status (green, yellow, red)")),
		mcp.WithString("status", mcp.Description("Filter by status (open, close)")),
	)
}

func (m *Module) buildPathLogsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to query logs for (e.g., /api/v1/users, /health)")),
		mcp.WithString("size", mcp.Description("Maximum number of log entries to return (default: 100)")),
		mcp.WithString("start_time", mcp.Description("Start time for log filtering (ISO format or relative like '1h', '30m', '7d')")),
		mcp.WithString("end_time", mcp.Description("End time for log filtering (ISO format or relative like '1h', '30m', '7d')")),
		mcp.WithString("index", mcp.Description("Specific index or index pattern to search (default: * for all indices)")),
		mcp.WithString("method", mcp.Description("HTTP method filter (e.g., GET, POST, PUT, DELETE)")),
		mcp.WithString("status_code", mcp.Description("HTTP status code filter (e.g., 200, 404, 500)")),
	)
}
