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
	Search      ToolConfig
	ListIndices ToolConfig
	ESQL        ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() LogsToolsConfig {
	return LogsToolsConfig{
		Search: ToolConfig{
			Enabled:     true,
			Name:        "search-logs",
			Description: "Full-text search across log messages",
		},
		ListIndices: ToolConfig{
			Enabled:     true,
			Name:        "list-log-indices",
			Description: "List all available log indices in Elasticsearch",
		},
		ESQL: ToolConfig{
			Enabled:     true,
			Name:        "query-logs",
			Description: "Query logs using ES|QL (Elasticsearch Query Language)",
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

	// Elasticsearch Search Tool
	if toolsConfig.Search.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildSearchToolDefinition(toolsConfig.Search),
			Handler: m.handleElasticsearchSearch,
		})
	}

	// List Indices Tool
	if toolsConfig.ListIndices.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildListIndicesToolDefinition(toolsConfig.ListIndices),
			Handler: m.handleListIndices,
		})
	}

	// ES|QL Query Tool
	if toolsConfig.ESQL.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildESQLToolDefinition(toolsConfig.ESQL),
			Handler: m.handleESQL,
		})
	}

	return tools
}

// Tool definition builder methods

func (m *Module) buildSearchToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("index", mcp.Required(), mcp.Description("Index name or pattern to search (e.g., 'logs-*', 'filebeat-*')")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Complete Elasticsearch query body as JSON string. Supports all ES Query DSL features: query, aggs, size, from, sort, _source, etc. Example: '{\"size\":0,\"query\":{\"query_string\":{\"query\":\"error\"}},\"aggs\":{\"by_level\":{\"terms\":{\"field\":\"level.keyword\"}}}}'")),
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

func (m *Module) buildESQLToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("query", mcp.Required(), mcp.Description("ES|QL query string. Example: 'FROM logs-* | WHERE @timestamp > NOW() - 1 hour | STATS count() BY level'")),
		mcp.WithString("format", mcp.Description("Response format (json, csv, tsv, txt) - default: json")),
		mcp.WithString("columnar", mcp.Description("Return results in columnar format (true or false) - default: false")),
	)
}
