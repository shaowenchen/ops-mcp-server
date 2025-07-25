package events

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

// EventsToolsConfig defines configuration for all tools
type EventsToolsConfig struct {
	PodEvents        ToolConfig
	DeploymentEvents ToolConfig
	NodeEvents       ToolConfig
	RawEvents        ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() EventsToolsConfig {
	return EventsToolsConfig{
		PodEvents: ToolConfig{
			Enabled:     true,
			Name:        "get-pod-events",
			Description: "Get Kubernetes pod events from all pods in specified namespace/cluster. Returns events with pod names in parsed_info.name field. No need to specify individual pod names.",
		},
		DeploymentEvents: ToolConfig{
			Enabled:     true,
			Name:        "get-deployment-events",
			Description: "Get Kubernetes deployment events from all deployments in specified namespace/cluster. Returns events with deployment names in parsed_info.name field. No need to specify individual deployment names.",
		},
		NodeEvents: ToolConfig{
			Enabled:     true,
			Name:        "get-node-events",
			Description: "Get Kubernetes node events from all nodes in specified cluster. Returns events with node names in parsed_info.name field. No need to specify individual node names.",
		},
		RawEvents: ToolConfig{
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

	// Pod Events Tool
	if toolsConfig.PodEvents.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildPodEventsToolDefinition(toolsConfig.PodEvents),
			Handler: m.handleGetPodEvents,
		})
	}

	// Deployment Events Tool
	if toolsConfig.DeploymentEvents.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildDeploymentEventsToolDefinition(toolsConfig.DeploymentEvents),
			Handler: m.handleGetDeploymentEvents,
		})
	}

	// Node Events Tool
	if toolsConfig.NodeEvents.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildNodeEventsToolDefinition(toolsConfig.NodeEvents),
			Handler: m.handleGetNodesEvents,
		})
	}

	// Raw Events Tool
	if toolsConfig.RawEvents.Enabled {
		tools = append(tools, server.ServerTool{
			Tool:    m.buildRawEventsToolDefinition(toolsConfig.RawEvents),
			Handler: m.handleGetRawEvents,
		})
	}

	return tools
}

// Tool definition builder methods

func (m *Module) buildPodEventsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("cluster", mcp.Description("Filter by cluster name (optional)")),
		mcp.WithString("namespace", mcp.Description("Filter by namespace (optional - if not provided, shows all namespaces)")),
		mcp.WithString("pod", mcp.Description("Specific pod name to query (optional - if not provided, shows all pods)")),
		mcp.WithString("limit", mcp.Description("Maximum number of events to return (default: 10)")),
		mcp.WithString("offset", mcp.Description("Number of events to skip (default: 0)")),
		mcp.WithString("start_time", mcp.Description("Start time for filtering events (timestamp, default: 30 minutes ago)")),
	)
}

func (m *Module) buildDeploymentEventsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("cluster", mcp.Description("Filter by cluster name (optional)")),
		mcp.WithString("namespace", mcp.Description("Filter by namespace (optional - if not provided, shows all namespaces)")),
		mcp.WithString("deployment", mcp.Description("Specific deployment name to query (optional - if not provided, shows all deployments)")),
		mcp.WithString("limit", mcp.Description("Maximum number of events to return (default: 10)")),
		mcp.WithString("offset", mcp.Description("Number of events to skip (default: 0)")),
		mcp.WithString("start_time", mcp.Description("Start time for filtering events (timestamp, default: 30 minutes ago)")),
	)
}

func (m *Module) buildNodeEventsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("cluster", mcp.Description("Filter by cluster name (optional - if not provided, shows all clusters)")),
		mcp.WithString("node", mcp.Description("Specific node name to query (optional - if not provided, shows all nodes)")),
		mcp.WithString("limit", mcp.Description("Maximum number of events to return (default: 10)")),
		mcp.WithString("offset", mcp.Description("Number of events to skip (default: 0)")),
		mcp.WithString("start_time", mcp.Description("Start time for filtering events (timestamp, default: 30 minutes ago)")),
	)
}

func (m *Module) buildRawEventsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("subject_pattern", mcp.Required(), mcp.Description("NATS subject pattern for event querying (supports wildcards * and > for flexible matching)")),
		mcp.WithString("limit", mcp.Description("Maximum number of events to return (default: 10)")),
		mcp.WithString("offset", mcp.Description("Number of events to skip (default: 0)")),
		mcp.WithString("start_time", mcp.Description("Start time for filtering events (timestamp, default: 30 minutes ago)")),
	)
}
