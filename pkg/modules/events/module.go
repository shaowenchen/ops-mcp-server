package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Suffix string `mapstructure:"suffix" json:"suffix" yaml:"suffix"`
}

// Config contains events module configuration
type Config struct {
	Endpoint     string        `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Token        string        `mapstructure:"token" json:"token" yaml:"token"`
	PollInterval time.Duration `mapstructure:"poll_interval" json:"poll_interval" yaml:"poll_interval"`
	Tools        ToolsConfig   `mapstructure:"tools" json:"tools" yaml:"tools"`
}

// Module represents the events module
type Module struct {
	config     *Config
	logger     *zap.Logger
	httpClient *http.Client
}

// New creates a new events module
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("events config is required")
	}

	m := &Module{
		config: config,
		logger: logger.Named("events"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	m.logger.Info("Events module created",
		zap.String("endpoint", config.Endpoint),
		zap.Duration("pollInterval", config.PollInterval),
		zap.Bool("token_configured", config.Token != ""),
		zap.Bool("ops_configured", config.Endpoint != ""),
	)

	return m, nil
}

// makeRequest creates and executes an HTTP request with authentication (legacy method with path)
func (m *Module) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	// Check if endpoint is configured
	if m.config.Endpoint == "" {
		return nil, fmt.Errorf("events endpoint not configured - please set events.ops.endpoint in config")
	}
	url := m.config.Endpoint + path
	return m.makeRequestWithFullURL(ctx, method, url, body)
}

// makeRequestWithFullURL creates and executes an HTTP request with authentication using full URL
func (m *Module) makeRequestWithFullURL(ctx context.Context, method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	var bodyStr string
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
		bodyStr = string(jsonData)
	}

	// Log request details
	m.logger.Info("🎪 Making Events API Request",
		zap.String("method", method),
		zap.String("full_url", url),
		zap.String("endpoint", m.config.Endpoint),
		zap.Bool("has_body", body != nil),
		zap.Bool("has_token", m.config.Token != ""))

	// Also print to console for visibility
	fmt.Printf("🔍 Events API Call: %s %s\n", method, url)
	if bodyStr != "" {
		// Pretty print JSON body if it's not too long
		if len(bodyStr) < 500 {
			var prettyBody interface{}
			if err := json.Unmarshal([]byte(bodyStr), &prettyBody); err == nil {
				if prettyJSON, err := json.MarshalIndent(prettyBody, "", "  "); err == nil {
					fmt.Printf("📋 Request Body:\n%s\n", string(prettyJSON))
				} else {
					fmt.Printf("📋 Request Body: %s\n", bodyStr)
				}
			} else {
				fmt.Printf("📋 Request Body: %s\n", bodyStr)
			}
		} else {
			fmt.Printf("📋 Request Body Length: %d bytes\n", len(bodyStr))
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Add Authorization header if token is configured
	authMethod := "none"
	if m.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+m.config.Token)
		authMethod = "bearer_token"
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.logger.Error("❌ Events API Request Failed",
			zap.String("method", method),
			zap.String("url", url),
			zap.Error(err))
		fmt.Printf("❌ Events API Request Failed: %s %s - %v\n", method, url, err)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// Log response details
	m.logger.Info("✅ Events API Response Received",
		zap.String("method", method),
		zap.String("url", url),
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
		zap.String("auth_method", authMethod),
		zap.Int64("content_length", resp.ContentLength))

	fmt.Printf("✅ Events API Response: %d %s\n", resp.StatusCode, resp.Status)

	return resp, nil
}

// enhanceEvent adds parsed information to an event
func enhanceEvent(wrapper EventWrapper) EnhancedEvent {
	enhanced := EnhancedEvent{
		EventWrapper: wrapper,
		ParsedInfo:   ParseSubject(wrapper.Subject),
	}

	// If cluster is not set in parsed info, try to get it from the event
	if enhanced.ParsedInfo.Cluster == "" && wrapper.Event.Cluster != "" {
		enhanced.ParsedInfo.Cluster = wrapper.Event.Cluster
	}

	return enhanced
}

// buildSubjectPattern builds the subject pattern for the API path
func (m *Module) buildSubjectPattern(req EventsListRequest) string {
	// Build subject pattern based on resource type and filters
	// Format: ops.clusters.{cluster}.{resource_path}.event
	// For pods/deployments: ops.clusters.{cluster}.namespaces.{namespace}.{resource}.{name}.event
	// For nodes: ops.clusters.{cluster}.nodes.{name}.event
	// Use * as wildcard when specific names are not provided

	var subjectPattern string

	if req.Resource == "nodes" {
		// Nodes pattern: ops.clusters.{cluster}.nodes.{name}.event
		clusterPart := "*"
		if req.Cluster != "" {
			clusterPart = req.Cluster
		}

		nodePart := "*"
		if req.ResourceName != "" {
			nodePart = req.ResourceName
		}

		subjectPattern = fmt.Sprintf("ops.clusters.%s.nodes.%s.event", clusterPart, nodePart)
	} else {
		// Namespaced resources pattern: ops.clusters.{cluster}.namespaces.{namespace}.{resource}.{name}.event
		clusterPart := "*"
		if req.Cluster != "" {
			clusterPart = req.Cluster
		}

		namespacePart := "*"
		if req.Namespace != "" {
			namespacePart = req.Namespace
		}

		resourcePart := "*"
		if req.Resource != "" {
			resourcePart = req.Resource
		}

		resourceNamePart := "*"
		if req.ResourceName != "" {
			resourceNamePart = req.ResourceName
		}

		subjectPattern = fmt.Sprintf("ops.clusters.%s.namespaces.%s.%s.%s.event",
			clusterPart, namespacePart, resourcePart, resourceNamePart)
	}

	return subjectPattern
}

// fetchEventsFromAPI fetches events from the configured endpoint
func (m *Module) fetchEventsFromAPI(ctx context.Context, req EventsListRequest) (*EventsListResponse, error) {
	if m.config.Endpoint == "" {
		return nil, fmt.Errorf("events endpoint not configured")
	}

	// Use raw subject pattern if provided, otherwise build from structured fields
	var subjectPattern string
	if req.SubjectPattern != "" {
		subjectPattern = req.SubjectPattern
		m.logger.Info("Using raw subject pattern", zap.String("pattern", subjectPattern))
	} else {
		// Build subject pattern for the API path
		subjectPattern = m.buildSubjectPattern(req)
	}

	// Build query parameters
	queryParams := make(map[string]string)
	if req.Limit > 0 {
		queryParams["page_size"] = strconv.Itoa(req.Limit)
	} else {
		queryParams["page_size"] = "10"
	}

	page := 1
	if req.Offset > 0 && req.Limit > 0 {
		page = (req.Offset / req.Limit) + 1
	}
	queryParams["page"] = strconv.Itoa(page)

	if req.StartTime != "" {
		queryParams["start_time"] = req.StartTime
	}

	// Build full URL with path and query parameters
	// Format: {endpoint}/api/v1/events/{subject_pattern}?query_params
	url := m.config.Endpoint + "/api/v1/events/" + subjectPattern
	if len(queryParams) > 0 {
		url += "?"
		first := true
		for key, value := range queryParams {
			if !first {
				url += "&"
			}
			url += key + "=" + value
			first = false
		}
	}

	m.logger.Info("🌐 Making API Request",
		zap.String("full_url", url),
		zap.String("base_endpoint", m.config.Endpoint),
		zap.String("subject_pattern", subjectPattern),
		zap.Any("query_params", queryParams),
		zap.String("resource_type", req.Resource),
		zap.String("cluster", req.Cluster),
		zap.String("namespace", req.Namespace),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset),
		zap.String("start_time", req.StartTime))

	// Also print to console for visibility
	fmt.Printf("🎯 Subject Pattern: %s\n", subjectPattern)
	fmt.Printf("🔍 Full API Query URL: %s\n", url)

	resp, err := m.makeRequestWithFullURL(ctx, "GET", url, nil)
	if err != nil {
		m.logger.Error("Failed to fetch events from API", zap.Error(err))
		return nil, fmt.Errorf("failed to call events API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("API returned non-OK status",
			zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var eventsResp EventsListResponse
	if err := json.Unmarshal(body, &eventsResp); err != nil {
		m.logger.Error("Failed to decode API response",
			zap.Error(err),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	// Enhance all events with parsed information
	for i := range eventsResp.Data.List {
		eventsResp.Data.List[i] = enhanceEvent(eventsResp.Data.List[i].EventWrapper)
	}

	m.logger.Info("Successfully fetched events",
		zap.Int("count", len(eventsResp.Data.List)),
		zap.Int("total", eventsResp.Data.Total))

	return &eventsResp, nil
}

// GetTools returns MCP tools for events (pods, deployments, nodes, etc.)
func (m *Module) GetTools() []server.ServerTool {
	// Get default tool configuration
	toolsConfig := GetDefaultToolsConfig()

	// Tool configuration can be modified based on config file or other conditions
	// For example: disable certain tools based on m.config
	// toolsConfig.PodEvents.Enabled = false

	return m.BuildTools(toolsConfig)
}

// Tool handlers
func (m *Module) handleGetPodEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return m.handleEvents(ctx, request, "pods")
}

func (m *Module) handleGetDeploymentEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return m.handleEvents(ctx, request, "deployments")
}

func (m *Module) handleGetNodesEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return m.handleEvents(ctx, request, "nodes")
}

func (m *Module) handleGetRawEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Log incoming request
	fmt.Printf("📥 Received raw events request with args: %+v\n", args)
	m.logger.Info("Processing raw events request",
		zap.Any("arguments", args))

	// Parse parameters for raw events query
	var subjectPattern, startTime string
	var limit, offset int = 10, 0

	if val, ok := args["subject_pattern"].(string); ok {
		subjectPattern = val
	} else {
		return nil, fmt.Errorf("subject_pattern is required for raw events query")
	}

	if val, ok := args["start_time"].(string); ok {
		startTime = val
	} else {
		// Default: 30 minutes ago
		defaultTime := time.Now().Add(-30 * time.Minute)
		startTime = strconv.FormatInt(defaultTime.UnixMilli(), 10)
	}
	if val, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if val, ok := args["offset"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Create request for events API using raw subject pattern
	req := EventsListRequest{
		Limit:          limit,
		Offset:         offset,
		SubjectPattern: subjectPattern,
		StartTime:      startTime,
	}

	// Fetch events
	response, err := m.fetchEventsFromAPI(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw events: %w", err)
	}

	// Log response summary
	fmt.Printf("✅ Successfully fetched %d events using pattern '%s' (total available: %d)\n",
		len(response.Data.List), subjectPattern, response.Data.Total)

	if len(response.Data.List) > 0 {
		sample := response.Data.List[0]
		fmt.Printf("📋 Sample event - Subject: %s\n", sample.Subject)
		fmt.Printf("🔍 Parsed info - Cluster: %s, Namespace: %s, Resource: %s, Name: %s\n",
			sample.ParsedInfo.Cluster, sample.ParsedInfo.Namespace,
			sample.ParsedInfo.Resource, sample.ParsedInfo.Name)
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}

// Common handler for events (Kubernetes and other types)
func (m *Module) handleEvents(ctx context.Context, request mcp.CallToolRequest, resourceType string) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Log incoming request
	fmt.Printf("📥 Received %s events request with args: %+v\n", resourceType, args)
	m.logger.Info("Processing events request",
		zap.String("resource_type", resourceType),
		zap.Any("arguments", args))

	// Parse parameters for events
	var cluster, namespace, resourceName, startTime string
	var limit, offset int = 10, 0

	if val, ok := args["cluster"].(string); ok {
		cluster = val
	}
	// Only parse namespace for resources that support it (not for nodes)
	if resourceType != "nodes" {
		if val, ok := args["namespace"].(string); ok {
			namespace = val
		}
		// Parse pod for pods
		if resourceType == "pods" {
			if val, ok := args["pod"].(string); ok {
				resourceName = val
			}
		}
		// Parse deployment for deployments
		if resourceType == "deployments" {
			if val, ok := args["deployment"].(string); ok {
				resourceName = val
			}
		}
	} else {
		// Parse node for nodes
		if val, ok := args["node"].(string); ok {
			resourceName = val
		}
	}
	if val, ok := args["start_time"].(string); ok {
		startTime = val
	} else {
		// Default: 30 minutes ago
		defaultTime := time.Now().Add(-30 * time.Minute)
		startTime = strconv.FormatInt(defaultTime.UnixMilli(), 10)
	}
	if val, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if val, ok := args["offset"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Create request for events API
	req := EventsListRequest{
		Limit:        limit,
		Offset:       offset,
		Cluster:      cluster,
		Namespace:    namespace, // Will be empty for nodes, which is correct
		Resource:     resourceType,
		ResourceName: resourceName, // Specific resource name if provided
		StartTime:    startTime,
	}

	// Fetch events
	response, err := m.fetchEventsFromAPI(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s events: %w", resourceType, err)
	}

	// Log response summary and show sample parsing
	fmt.Printf("✅ Successfully fetched %d %s events (total available: %d)\n",
		len(response.Data.List), resourceType, response.Data.Total)

	if len(response.Data.List) > 0 {
		sample := response.Data.List[0]
		fmt.Printf("📋 Sample event - Subject: %s\n", sample.Subject)
		fmt.Printf("🔍 Parsed info - Cluster: %s, Namespace: %s, Resource: %s, Name: %s\n",
			sample.ParsedInfo.Cluster, sample.ParsedInfo.Namespace,
			sample.ParsedInfo.Resource, sample.ParsedInfo.Name)
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(data),
			},
		},
	}, nil
}
