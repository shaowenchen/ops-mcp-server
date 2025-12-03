package sops

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shaowenchen/ops-mcp-server/pkg/metrics"
)

// ToolConfig defines configuration for a single tool
type ToolConfig struct {
	Name        string // Tool name
	Description string // Tool description
	Enabled     bool   // Whether the tool is enabled
}

// SOPSToolsConfig defines configuration for all tools
type SOPSToolsConfig struct {
	ExecuteSOPS    ToolConfig
	ListSOPS       ToolConfig
	ListParameters ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() SOPSToolsConfig {
	return SOPSToolsConfig{
		ExecuteSOPS: ToolConfig{
			Name:        "execute-sops",
			Description: "Execute a standard operation procedure (SOPS)",
			Enabled:     true,
		},
		ListSOPS: ToolConfig{
			Name:        "list-sops",
			Description: "List all available SOPS procedures",
			Enabled:     true,
		},
		ListParameters: ToolConfig{
			Name:        "list-sops-parameters",
			Description: "List all required parameters for a specific SOPS procedure",
			Enabled:     true,
		},
	}
}

// BuildToolName builds tool name based on configuration
func (m *Module) BuildToolName(baseName string) string {
	name := baseName
	if m.config.Tools.Prefix != "" {
		name = m.config.Tools.Prefix + name
	}
	if m.config.Tools.Suffix != "" {
		name = name + m.config.Tools.Suffix
	}
	return name
}

// BuildTools builds tool list based on configuration
func (m *Module) BuildTools(toolsConfig SOPSToolsConfig) []server.ServerTool {
	var tools []server.ServerTool

	// Execute SOPS Tool
	if toolsConfig.ExecuteSOPS.Enabled {
		toolName := m.BuildToolName(toolsConfig.ExecuteSOPS.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildExecuteSOPSToolDefinition(toolsConfig.ExecuteSOPS),
			Handler: metrics.WrapToolHandler(m.handleExecuteSOPS, toolName, "sops"),
		})
	}

	// List SOPS Tool
	if toolsConfig.ListSOPS.Enabled {
		toolName := m.BuildToolName(toolsConfig.ListSOPS.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildListSOPSToolDefinition(toolsConfig.ListSOPS),
			Handler: metrics.WrapToolHandler(m.handleListSOPS, toolName, "sops"),
		})
	}

	// List Parameters Tool
	if toolsConfig.ListParameters.Enabled {
		toolName := m.BuildToolName(toolsConfig.ListParameters.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildListParametersToolDefinition(toolsConfig.ListParameters),
			Handler: metrics.WrapToolHandler(m.handleListParameters, toolName, "sops"),
		})
	}

	return tools
}

// Tool definition builder methods
func (m *Module) buildExecuteSOPSToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("sops_id", mcp.Required(), mcp.Description("ID of the SOPS procedure to execute")),
		mcp.WithString("parameters", mcp.Description("JSON string of parameters to pass to the SOPS procedure")),
	)
}

func (m *Module) buildListSOPSToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
	)
}

func (m *Module) buildListParametersToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("sops_id", mcp.Required(), mcp.Description("ID of the SOPS procedure to get parameters for")),
	)
}
