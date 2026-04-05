package sops

import (
	"encoding/json"

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
	ListSOPS        ToolConfig
	GetParameters   ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() SOPSToolsConfig {
	return SOPSToolsConfig{
		ExecuteSOPS: ToolConfig{
			Name:        "execute-sop",
			Description: "Execute a standard operation procedure (SOPS)",
			Enabled:     true,
		},
		ListSOPS: ToolConfig{
			Name:        "list-sops",
			Description: "List all available SOPS procedures",
			Enabled:     true,
		},
		GetParameters: ToolConfig{
			Name:        "get-sop-parameters",
			Description: "Get parameter schema for a specific SOPS procedure",
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

	// Get SOPS parameters tool
	if toolsConfig.GetParameters.Enabled {
		toolName := m.BuildToolName(toolsConfig.GetParameters.Name)
		tools = append(tools, server.ServerTool{
			Tool:    m.buildGetSOPSParametersToolDefinition(toolsConfig.GetParameters),
			Handler: metrics.WrapToolHandler(m.handleGetSOPSParameters, toolName, "sops"),
		})
	}

	return tools
}

// executeSOPSToolInputSchema: required sops_id; other keys are pipeline variables (additionalProperties).
var executeSOPSToolInputSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"sops_id": {
			"type": "string",
			"description": "ID of the SOPS procedure to execute"
		}
	},
	"required": ["sops_id"],
	"additionalProperties": true
}`)

// Tool definition builder methods
func (m *Module) buildExecuteSOPSToolDefinition(config ToolConfig) mcp.Tool {
	// NewTool always seeds InputSchema with type "object"; WithRawInputSchema does not
	// clear it, which triggers MarshalJSON conflict (InputSchema + RawInputSchema).
	tool := mcp.NewToolWithRawSchema(
		m.BuildToolName(config.Name),
		config.Description+". Pass pipeline variables as top-level arguments (same names as get-sop-parameters). Only sops_id is reserved.",
		executeSOPSToolInputSchema,
	)
	// Match defaults from mcp.NewTool for consistent client hints.
	tool.Annotations = mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(false),
		DestructiveHint: mcp.ToBoolPtr(true),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(true),
	}
	return tool
}

func (m *Module) buildListSOPSToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
	)
}

func (m *Module) buildGetSOPSParametersToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("sops_id", mcp.Required(), mcp.Description("ID of the SOPS procedure to get parameters for")),
	)
}
