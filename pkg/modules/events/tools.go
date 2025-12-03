package events

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shaowenchen/ops-mcp-server/pkg/metrics"
)

// ToolConfig defines configuration for a single tool
type ToolConfig struct {
	Enabled     bool   // Whether the tool is enabled
	Name        string // Tool name
	Description string // Tool description
}

// EventsToolsConfig defines configuration for all tools
type EventsToolsConfig struct {
	ListEvents ToolConfig
	GetEvents  ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() EventsToolsConfig {
	return EventsToolsConfig{
		ListEvents: ToolConfig{
			Enabled:     true,
			Name:        "list-events",
			Description: "List available event types by querying the backend API. Supports search filtering and pagination to discover different event categories and patterns.",
		},
		GetEvents: ToolConfig{
			Enabled:     true,
			Name:        "get-events",
			Description: "Get events using raw NATS subject patterns. Supports three query types: 1) Direct query (exact subject), 2) Wildcard query (using * for single level), 3) Prefix matching (using > for multi-level suffix). Examples: 'ops.clusters.{cluster}.namespaces.{namespace}.pods.{pod-name}.event', 'ops.clusters.*.namespaces.ops-system.webhooks.*', 'ops.clusters.*.namespaces.{namespace}.hosts.>'",
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
func (m *Module) BuildTools(toolsConfig EventsToolsConfig) []server.ServerTool {
	var tools []server.ServerTool

	// List Events Tool
	if toolsConfig.ListEvents.Enabled {
		toolName := m.BuildToolName(toolsConfig.ListEvents.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildListEventsToolDefinition(toolsConfig.ListEvents),
			Handler: metrics.WrapToolHandler(m.handleListEvents, toolName, "events"),
		})
	}

	// Get Events Tool
	if toolsConfig.GetEvents.Enabled {
		toolName := m.BuildToolName(toolsConfig.GetEvents.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildGetEventsToolDefinition(toolsConfig.GetEvents),
			Handler: metrics.WrapToolHandler(m.handleGetEvents, toolName, "events"),
		})
	}

	return tools
}

// Tool definition builder methods

func (m *Module) buildListEventsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("search", mcp.Description("Search term to filter event types (optional)")),
		mcp.WithString("page_size", mcp.Description("Number of event types to return (default: 10)")),
		mcp.WithString("page", mcp.Description("Page number for pagination (default: 1)")),
	)
}

func (m *Module) buildGetEventsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("subject_pattern", mcp.Required(), mcp.Description("NATS subject pattern for event querying (supports wildcards * and > for flexible matching)")),
		mcp.WithString("page_size", mcp.Description("Number of events per page (default: 10)")),
		mcp.WithString("page", mcp.Description("Page number for pagination (default: 1)")),
		mcp.WithString("start_time", mcp.Description("Start time for filtering events (timestamp, eg, 1758928888000)")),
	)
}
