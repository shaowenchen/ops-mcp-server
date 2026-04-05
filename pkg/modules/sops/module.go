package sops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shaowenchen/ops-copilot/pkg/copilot"
	opsv1 "github.com/shaowenchen/ops/api/v1"
	"github.com/shaowenchen/ops/pkg/log"
	"go.uber.org/zap"
)

// collectExecuteSOPSVariables builds pipeline variables from flat tool arguments only.
// Reserved key: sops_id (not sent as a pipeline variable).
func collectExecuteSOPSVariables(args map[string]any) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range args {
		if k == "sops_id" {
			continue
		}
		out[k] = v
	}
	return out
}

// decodePipelineListResponse accepts several Ops / Kubernetes-style list payloads.
// ops-copilot only unmarshals data.list; many servers use data.items or a bare PipelineList.
func decodePipelineListResponse(body []byte, log *zap.Logger) ([]opsv1.Pipeline, error) {
	if len(body) < 2 {
		return nil, fmt.Errorf("empty pipelines response body")
	}

	type nested struct {
		List  []opsv1.Pipeline `json:"list"`
		Items []opsv1.Pipeline `json:"items"`
	}
	var envelope struct {
		Data   nested           `json:"data"`
		Result nested           `json:"result"`
		Items  []opsv1.Pipeline `json:"items"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decode pipelines envelope: %w", err)
	}

	tryNested := func(n nested) []opsv1.Pipeline {
		if len(n.List) > 0 {
			return n.List
		}
		return n.Items
	}

	if p := tryNested(envelope.Data); len(p) > 0 {
		return p, nil
	}
	if p := tryNested(envelope.Result); len(p) > 0 {
		return p, nil
	}
	if len(envelope.Items) > 0 {
		return envelope.Items, nil
	}

	// Root Kubernetes PipelineList { "items": [...] }
	var pl opsv1.PipelineList
	if err := json.Unmarshal(body, &pl); err == nil && len(pl.Items) > 0 {
		return pl.Items, nil
	}

	// { "data": [ {...}, ... ] }
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err == nil {
		if raw, ok := root["data"]; ok {
			var asArr []opsv1.Pipeline
			if err := json.Unmarshal(raw, &asArr); err == nil && len(asArr) > 0 {
				return asArr, nil
			}
			var inner nested
			if err := json.Unmarshal(raw, &inner); err == nil {
				if p := tryNested(inner); len(p) > 0 {
					return p, nil
				}
			}
		}
	}

	if log != nil {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		log.Debug("pipelines list parsed zero entries; check API response shape",
			zap.Int("body_len", len(body)),
			zap.String("body_snippet", snippet))
	}
	return nil, nil
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

// fetchPipelinesFromOpsAPI lists pipelines using the same path as ops-copilot but tolerates
// multiple JSON shapes (data.list vs data.items vs Kubernetes PipelineList).
func (m *Module) fetchPipelinesFromOpsAPI(ctx context.Context) ([]opsv1.Pipeline, error) {
	const uri = "/api/v1/namespaces/ops-system/pipelines?labels_selector=ops/copilot=enabled&page_size=999"
	base := strings.TrimRight(m.config.Endpoint, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.config.Token)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pipelines request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 512 {
			snippet = snippet[:512] + "..."
		}
		return nil, fmt.Errorf("pipelines API returned HTTP %d: %s", resp.StatusCode, snippet)
	}

	pl, err := decodePipelineListResponse(body, m.logger)
	if err != nil {
		return nil, err
	}
	m.logger.Info("listed pipelines from Ops API", zap.Int("count", len(pl)))
	return pl, nil
}

func (m *Module) rebuildSopsFromPipelines(pipelines []opsv1.Pipeline) {
	next := make(map[string]*SOPSConfig, len(pipelines))
	for _, pipeline := range pipelines {
		if pipeline.Name == "" {
			continue
		}
		next[pipeline.Name] = &SOPSConfig{
			Desc:      pipeline.Spec.Desc,
			Variables: pipeline.Spec.Variables,
		}
	}
	m.sops = next
}

// loadSOPSConfigsFromAPI loads SOPS configurations from the API endpoint
func (m *Module) loadSOPSConfigsFromAPI() error {
	pipelines, err := m.fetchPipelinesFromOpsAPI(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}
	m.rebuildSopsFromPipelines(pipelines)
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

	parameters := collectExecuteSOPSVariables(args)

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

	// Refresh from API so the list matches the server (and uses the same decoding as load)
	pipelines, err := m.fetchPipelinesFromOpsAPI(ctx)
	if err != nil {
		return nil, err
	}
	m.rebuildSopsFromPipelines(pipelines)

	sopsList := make([]map[string]interface{}, 0, len(pipelines))
	for _, pipeline := range pipelines {
		if pipeline.Name == "" {
			continue
		}
		sopsList = append(sopsList, map[string]interface{}{
			"id":          pipeline.Name,
			"description": pipeline.Spec.Desc,
			"variables":   pipeline.Spec.Variables,
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

// handleGetSOPSParameters returns the parameter schema for a specific SOPS procedure.
func (m *Module) handleGetSOPSParameters(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
