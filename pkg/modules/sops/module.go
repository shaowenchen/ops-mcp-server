package sops

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shaowenchen/ops-copilot/pkg/copilot"
	opsv1 "github.com/shaowenchen/ops/api/v1"
	"github.com/shaowenchen/ops/pkg/log"
	"go.uber.org/zap"
)

// parseSOPSParameters decodes the optional nested "parameters" argument (string, object, or RawMessage).
func parseSOPSParameters(raw any) (map[string]interface{}, error) {
	if raw == nil {
		return make(map[string]interface{}), nil
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return make(map[string]interface{}), nil
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, fmt.Errorf("failed to parse parameters JSON: %w", err)
		}
		return m, nil
	case map[string]interface{}:
		return v, nil
	case json.RawMessage:
		if len(v) == 0 {
			return make(map[string]interface{}), nil
		}
		var m map[string]interface{}
		if err := json.Unmarshal(v, &m); err != nil {
			return nil, fmt.Errorf("failed to parse parameters JSON: %w", err)
		}
		return m, nil
	default:
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("parameters must be a JSON string or object, got %T", raw)
		}
		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parameters must be a JSON object, got %T: %w", raw, err)
		}
		return m, nil
	}
}

// collectExecuteSOPSVariables merges nested "parameters" (if present) with top-level keys.
// Reserved: sops_id, parameters. Top-level values override the same key from nested parameters.
func collectExecuteSOPSVariables(args map[string]any) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	if raw, ok := args["parameters"]; ok && raw != nil {
		parsed, err := parseSOPSParameters(raw)
		if err != nil {
			return nil, err
		}
		for k, v := range parsed {
			out[k] = v
		}
	}
	for k, v := range args {
		if k == "sops_id" || k == "parameters" {
			continue
		}
		out[k] = v
	}
	return out, nil
}

// Module represents the sops module
type Module struct {
	config     *Config
	logger     *zap.Logger
	httpClient *http.Client
	sops       map[string]*SOPSConfig
}

// New creates a new sops module instance
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Create HTTP client - each request uses a new connection, closes after request
	transport := &http.Transport{
		DisableKeepAlives: true, // Disable connection reuse - close after each request
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	module := &Module{
		config: config,
		logger: logger,
		sops:   make(map[string]*SOPSConfig),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second, // SOPS operations may take longer
		},
	}

	// Load SOPS configurations from API only if endpoint is configured
	if config.Endpoint != "" {
		if err := module.loadSOPSConfigsFromAPI(); err != nil {
			return nil, fmt.Errorf("failed to load SOPS configs from API: %w", err)
		}
	} else {
		module.logger.Info("SOPS module created without API configuration - tools will return configuration required error")
	}

	return module, nil
}

// GetTools returns the list of available tools
func (m *Module) GetTools() []server.ServerTool {
	// Get default tool configuration
	toolsConfig := GetDefaultToolsConfig()

	// Tool configuration can be modified based on config file or other conditions
	// For example: disable certain tools based on m.config

	return m.BuildTools(toolsConfig)
}

// loadSOPSConfigsFromAPI loads SOPS configurations from the API endpoint
func (m *Module) loadSOPSConfigsFromAPI() error {
	// Try to load SOPS configurations from API
	pipelinerunsManager, err := copilot.NewPipelineRunsManager(m.config.Endpoint, m.config.Token, "ops-system")
	if err != nil {
		return fmt.Errorf("failed to create pipeline runs manager: %w", err)
	}

	pipelines, err := pipelinerunsManager.GetPipelines()
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}
	for _, pipeline := range pipelines {
		m.sops[pipeline.Name] = &SOPSConfig{
			Desc:      pipeline.Spec.Desc,
			Variables: pipeline.Spec.Variables,
		}
	}

	return nil
}

// handleExecuteSOPS handles the execution of a SOPS procedure
func (m *Module) handleExecuteSOPS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if SOPS API is configured
	if m.config.Endpoint == "" {
		return nil, fmt.Errorf("SOPS API endpoint not configured - please set sops.ops.endpoint in config")
	}

	args := request.GetArguments()
	if args == nil && request.Params.Arguments != nil {
		return nil, fmt.Errorf("invalid arguments format")
	}
	if args == nil {
		args = make(map[string]any)
	}

	sopsID, ok := args["sops_id"].(string)
	if !ok {
		return nil, fmt.Errorf("sops_id is required")
	}

	// Get SOPS configuration
	sops, exists := m.sops[sopsID]
	if !exists {
		// Return available SOPS IDs
		availableIDs := make([]string, 0, len(m.sops))
		for id := range m.sops {
			availableIDs = append(availableIDs, id)
		}
		return nil, fmt.Errorf("SOPS with ID '%s' not found. Available SOPS IDs: %v", sopsID, availableIDs)
	}

	parameters, err := collectExecuteSOPSVariables(args)
	if err != nil {
		return nil, err
	}

	// Execute SOPS
	executionJSON, err := m.executeSOPS(ctx, sopsID, sops, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SOPS: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(executionJSON),
		},
	}, nil
}

// executeSOPS executes a SOPS procedure via API
func (m *Module) executeSOPS(ctx context.Context, sopsID string, sops *SOPSConfig, parameters map[string]interface{}) (string, error) {
	pipelinerunsManager, err := copilot.NewPipelineRunsManager(m.config.Endpoint, m.config.Token, "ops-system")
	if err != nil {
		return "", fmt.Errorf("failed to create pipeline runs manager: %w", err)
	}
	variables := make(map[string]string)
	for k, v := range parameters {
		variables[k] = fmt.Sprintf("%v", v)
	}
	logger := log.NewLogger()
	pr := &opsv1.PipelineRun{
		Spec: opsv1.PipelineRunSpec{
			PipelineRef: sopsID,
			Variables:   variables,
		},
	}
	err = pipelinerunsManager.Run(logger, pr)
	if err != nil {
		return "", fmt.Errorf("failed to run pipeline: %w", err)
	}
	return fmt.Sprintf("%s", pipelinerunsManager.PrintMarkdownPipelineRuns(pr)), nil
}

// handleListSOPS handles listing all available SOPS procedures
func (m *Module) handleListSOPS(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if SOPS API is configured
	if m.config.Endpoint == "" {
		return nil, fmt.Errorf("SOPS API endpoint not configured - please set sops.ops.endpoint in config")
	}

	// Get all available SOPS IDs and their descriptions
	sopsList := make([]map[string]interface{}, 0, len(m.sops))
	for id, config := range m.sops {
		sopsList = append(sopsList, map[string]interface{}{
			"id":          id,
			"description": config.Desc,
			"variables":   config.Variables,
		})
	}

	// Convert to JSON
	sopsJSON, err := json.MarshalIndent(map[string]interface{}{
		"available_sops": sopsList,
		"count":          len(sopsList),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOPS list: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(sopsJSON)),
		},
	}, nil
}

// handleListParameters handles listing all required parameters for a specific SOPS
func (m *Module) handleListParameters(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	sopsID, ok := args["sops_id"].(string)
	if !ok {
		return nil, fmt.Errorf("sops_id is required")
	}

	// Get SOPS configuration
	sops, exists := m.sops[sopsID]
	if !exists {
		// Return available SOPS IDs
		availableIDs := make([]string, 0, len(m.sops))
		for id := range m.sops {
			availableIDs = append(availableIDs, id)
		}
		return nil, fmt.Errorf("SOPS with ID '%s' not found. Available SOPS IDs: %v", sopsID, availableIDs)
	}

	// Extract parameters from variables
	parameters := make([]map[string]interface{}, 0)
	for name, variable := range sops.Variables {
		param := map[string]interface{}{
			"name":        name,
			"description": variable.Desc,
			"required":    variable.Required,
			"display":     variable.Display,
		}
		if variable.Value != "" {
			param["value"] = variable.Value
		}
		if variable.Default != "" {
			param["default"] = variable.Default
		}
		if variable.Regex != "" {
			param["regex"] = variable.Regex
		}
		if len(variable.Enums) > 0 {
			param["enums"] = variable.Enums
		}
		if len(variable.Examples) > 0 {
			param["examples"] = variable.Examples
		}
		parameters = append(parameters, param)
	}

	// Convert to JSON
	paramsJSON, err := json.MarshalIndent(map[string]interface{}{
		"sops_id":    sopsID,
		"parameters": parameters,
		"count":      len(parameters),
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(string(paramsJSON)),
		},
	}, nil
}
