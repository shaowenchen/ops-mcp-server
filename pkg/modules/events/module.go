package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// Config contains events module configuration
type Config struct {
	Endpoint     string        `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	PollInterval time.Duration `mapstructure:"pollInterval" json:"pollInterval" yaml:"pollInterval"`
}

// Module represents the events module
type Module struct {
	config *Config
	logger *zap.Logger
}

// New creates a new events module
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("events config is required")
	}

	m := &Module{
		config: config,
		logger: logger.Named("events"),
	}

	m.logger.Info("Events module created",
		zap.String("endpoint", config.Endpoint),
		zap.Duration("pollInterval", config.PollInterval),
	)

	return m, nil
}

// GetTools returns all MCP tools for the events module
func GetTools() []server.ServerTool {
	return []server.ServerTool{
		{
			Tool:    listEventsToolDefinition(),
			Handler: handleListEvents,
		},
		{
			Tool:    getEventToolDefinition(),
			Handler: handleGetEvent,
		},
		{
			Tool:    getEventTypesToolDefinition(),
			Handler: handleGetEventTypes,
		},
		{
			Tool:    getEventServicesToolDefinition(),
			Handler: handleGetEventServices,
		},
		{
			Tool:    getEventStatsToolDefinition(),
			Handler: handleGetEventStats,
		},
		{
			Tool:    createEventToolDefinition(),
			Handler: handleCreateEvent,
		},
	}
}

// Tool definitions
func listEventsToolDefinition() mcp.Tool {
	return mcp.NewTool("list_events",
		mcp.WithDescription("List operational events with optional filtering and pagination"),
		mcp.WithString("event_type", mcp.Description("Filter by event type (deployment, alert, scaling, etc.)")),
		mcp.WithString("service", mcp.Description("Filter by service name")),
		mcp.WithString("status", mcp.Description("Filter by status (success, warning, error, in_progress)")),
		mcp.WithString("limit", mcp.Description("Maximum number of events to return (default: 10)")),
		mcp.WithString("offset", mcp.Description("Number of events to skip (default: 0)")),
	)
}

func getEventToolDefinition() mcp.Tool {
	return mcp.NewTool("get_event",
		mcp.WithDescription("Get detailed information about a specific event"),
		mcp.WithString("event_id", mcp.Required(), mcp.Description("Unique identifier of the event")),
	)
}

func getEventTypesToolDefinition() mcp.Tool {
	return mcp.NewTool("get_event_types",
		mcp.WithDescription("Get all available event types"),
	)
}

func getEventServicesToolDefinition() mcp.Tool {
	return mcp.NewTool("get_event_services",
		mcp.WithDescription("Get all services that have events"),
	)
}

func getEventStatsToolDefinition() mcp.Tool {
	return mcp.NewTool("get_event_stats",
		mcp.WithDescription("Get event statistics and summaries"),
		mcp.WithString("time_range", mcp.Description("Time range for statistics (1h, 24h, 7d, 30d)")),
	)
}

func createEventToolDefinition() mcp.Tool {
	return mcp.NewTool("create_event",
		mcp.WithDescription("Create a new event for testing purposes"),
		mcp.WithString("type", mcp.Required(), mcp.Description("Event type")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Service name")),
		mcp.WithString("message", mcp.Required(), mcp.Description("Event message")),
	)
}

// Tool handlers
func handleListEvents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse parameters
	var eventType, service, status string
	var limit, offset int = 10, 0

	if val, ok := args["event_type"].(string); ok {
		eventType = val
	}
	if val, ok := args["service"].(string); ok {
		service = val
	}
	if val, ok := args["status"].(string); ok {
		status = val
	}
	if val, ok := args["limit"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			limit = parsed
		}
	}
	if val, ok := args["offset"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			offset = parsed
		}
	}

	// Generate mock events
	events := generateMockEvents()

	// Apply filters
	filtered := make([]Event, 0)
	for _, event := range events {
		if eventType != "" && event.Type != eventType {
			continue
		}
		if service != "" && event.Service != service {
			continue
		}
		if status != "" && event.Status != status {
			continue
		}
		filtered = append(filtered, event)
	}

	// Apply pagination
	total := len(filtered)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	result := filtered[start:end]

	response := map[string]interface{}{
		"events": result,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
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

func handleGetEvent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	eventID, ok := args["event_id"].(string)
	if !ok {
		return nil, fmt.Errorf("event_id is required")
	}

	// Generate mock event
	event := Event{
		ID:        eventID,
		Type:      "deployment",
		Service:   "api-gateway",
		Timestamp: time.Now(),
		Status:    "success",
		Message:   "Deployment completed successfully",
		Details: map[string]interface{}{
			"version":     "v1.2.3",
			"duration":    "45s",
			"user":        "admin",
			"environment": "production",
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
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

func handleGetEventTypes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	types := []string{
		"deployment",
		"alert",
		"scaling",
		"maintenance",
		"backup",
		"security",
		"configuration",
	}

	response := map[string]interface{}{
		"event_types": types,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
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

func handleGetEventServices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	services := []string{
		"api-gateway",
		"database",
		"web-frontend",
		"auth-service",
		"payment-service",
		"notification-service",
	}

	response := map[string]interface{}{
		"services": services,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
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

func handleGetEventStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	
	timeRange := "24h"
	if val, ok := args["time_range"].(string); ok {
		timeRange = val
	}

	stats := map[string]interface{}{
		"total_events": 150,
		"by_type": map[string]int{
			"deployment":    45,
			"alert":         32,
			"scaling":       28,
			"maintenance":   20,
			"backup":        15,
			"security":      6,
			"configuration": 4,
		},
		"by_status": map[string]int{
			"success":     120,
			"warning":     20,
			"error":       8,
			"in_progress": 2,
		},
		"time_range":   timeRange,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return nil, err
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

func handleCreateEvent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	eventType, ok := args["type"].(string)
	if !ok {
		return nil, fmt.Errorf("type is required")
	}

	service, ok := args["service"].(string)
	if !ok {
		return nil, fmt.Errorf("service is required")
	}

	message, ok := args["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message is required")
	}

	event := Event{
		ID:        fmt.Sprintf("evt_%d", time.Now().Unix()),
		Type:      eventType,
		Service:   service,
		Timestamp: time.Now(),
		Status:    "success",
		Message:   message,
		Details: map[string]interface{}{
			"created_by": "mcp_tool",
			"manual":     true,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		return nil, err
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

// Helper function to generate mock events
func generateMockEvents() []Event {
	return []Event{
		{
			ID:        "evt_001",
			Type:      "deployment",
			Service:   "api-gateway",
			Timestamp: time.Now().Add(-10 * time.Minute),
			Status:    "success",
			Message:   "Deployment completed successfully",
			Details: map[string]interface{}{
				"version":  "v1.2.3",
				"duration": "45s",
				"replicas": 3,
				"image":    "api-gateway:v1.2.3",
			},
		},
		{
			ID:        "evt_002",
			Type:      "alert",
			Service:   "database",
			Timestamp: time.Now().Add(-5 * time.Minute),
			Status:    "warning",
			Message:   "High CPU usage detected",
			Details: map[string]interface{}{
				"cpu_usage": "85%",
				"threshold": "80%",
				"duration":  "2m",
			},
		},
		{
			ID:        "evt_003",
			Type:      "scaling",
			Service:   "web-frontend",
			Timestamp: time.Now().Add(-2 * time.Minute),
			Status:    "in_progress",
			Message:   "Auto-scaling triggered",
			Details: map[string]interface{}{
				"from_replicas": 2,
				"to_replicas":   4,
				"trigger":       "cpu_threshold",
			},
		},
		{
			ID:        "evt_004",
			Type:      "maintenance",
			Service:   "database",
			Timestamp: time.Now().Add(-30 * time.Minute),
			Status:    "success",
			Message:   "Scheduled maintenance completed",
			Details: map[string]interface{}{
				"duration":    "25m",
				"maintenance": "index_rebuild",
				"downtime":    "0s",
			},
		},
		{
			ID:        "evt_005",
			Type:      "backup",
			Service:   "database",
			Timestamp: time.Now().Add(-60 * time.Minute),
			Status:    "success",
			Message:   "Database backup completed",
			Details: map[string]interface{}{
				"size":     "2.5GB",
				"duration": "8m",
				"location": "s3://backups/db",
			},
		},
	}
}
