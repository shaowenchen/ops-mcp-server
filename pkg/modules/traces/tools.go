package traces

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ToolConfig defines configuration for a single tool
type ToolConfig struct {
	Enabled     bool   // Whether the tool is enabled
	Name        string // Tool name
	Description string // Tool description
}

// JaegerToolsConfig defines configuration for all tools
type JaegerToolsConfig struct {
	GetServices   ToolConfig
	GetOperations ToolConfig
	GetTrace      ToolConfig
	FindTraces    ToolConfig
}

// GetDefaultToolsConfig returns default tool configuration
func GetDefaultToolsConfig() JaegerToolsConfig {
	return JaegerToolsConfig{
		GetServices: ToolConfig{
			Enabled:     true,
			Name:        "get-services",
			Description: "Gets the service names as JSON array of string. No input parameter supported.",
		},
		GetOperations: ToolConfig{
			Enabled:     true,
			Name:        "get-operations",
			Description: "Gets the operations as JSON array of object with name and spanKind properties.",
		},
		GetTrace: ToolConfig{
			Enabled:     true,
			Name:        "get-trace",
			Description: "Gets the spans by the given trace by ID. Returns both original Jaeger format and converted OpenTelemetry format with standardized trace/span IDs and attributes.",
		},
		FindTraces: ToolConfig{
			Enabled:     true,
			Name:        "find-traces",
			Description: "Searches for traces based on criteria. Returns both original Jaeger format and converted OpenTelemetry format with standardized trace/span IDs and attributes.",
		},
	}
}

// convertJaegerTraceToOpenTelemetry converts Jaeger trace data to OpenTelemetry format
func (m *Module) convertJaegerTraceToOpenTelemetry(traceData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Extract trace ID
	if traceID, ok := traceData["traceID"].(string); ok {
		result["trace_id"] = traceID
		// Convert to OpenTelemetry TraceID format
		if tid, err := trace.TraceIDFromHex(traceID); err == nil {
			result["trace_id_hex"] = tid.String()
		}
	}

	// Extract spans
	if spans, ok := traceData["spans"].([]interface{}); ok {
		convertedSpans := make([]map[string]interface{}, 0, len(spans))
		for _, span := range spans {
			if spanMap, ok := span.(map[string]interface{}); ok {
				convertedSpan := m.convertJaegerSpanToOpenTelemetry(spanMap)
				convertedSpans = append(convertedSpans, convertedSpan)
			}
		}
		result["spans"] = convertedSpans
	}

	// Extract processes
	if processes, ok := traceData["processes"].(map[string]interface{}); ok {
		result["processes"] = processes
	}

	// Extract warnings
	if warnings, ok := traceData["warnings"].([]interface{}); ok {
		result["warnings"] = warnings
	}

	return result
}

// convertJaegerSpanToOpenTelemetry converts Jaeger span data to OpenTelemetry format
func (m *Module) convertJaegerSpanToOpenTelemetry(spanData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Extract span ID
	if spanID, ok := spanData["spanID"].(string); ok {
		result["span_id"] = spanID
		// Convert to OpenTelemetry SpanID format
		if sid, err := trace.SpanIDFromHex(spanID); err == nil {
			result["span_id_hex"] = sid.String()
		}
	}

	// Extract trace ID
	if traceID, ok := spanData["traceID"].(string); ok {
		result["trace_id"] = traceID
	}

	// Extract parent span ID
	if parentSpanID, ok := spanData["parentSpanID"].(string); ok {
		result["parent_span_id"] = parentSpanID
	}

	// Extract operation name
	if operationName, ok := spanData["operationName"].(string); ok {
		result["name"] = operationName
		result["operation_name"] = operationName
	}

	// Extract start time and duration
	if startTime, ok := spanData["startTime"].(float64); ok {
		result["start_time"] = int64(startTime)
		result["start_time_ns"] = int64(startTime * 1000) // Convert microseconds to nanoseconds
	}

	if duration, ok := spanData["duration"].(float64); ok {
		result["duration"] = int64(duration)
		result["duration_ns"] = int64(duration * 1000) // Convert microseconds to nanoseconds
	}

	// Extract tags and convert to OpenTelemetry attributes
	if tags, ok := spanData["tags"].([]interface{}); ok {
		attributes := make([]attribute.KeyValue, 0, len(tags))
		tagMap := make(map[string]interface{})

		for _, tag := range tags {
			if tagMap, ok := tag.(map[string]interface{}); ok {
				if key, keyOk := tagMap["key"].(string); keyOk {
					if value, valueOk := tagMap["value"].(string); valueOk {
						attributes = append(attributes, attribute.String(key, value))
						tagMap[key] = value
					}
				}
			}
		}

		result["attributes"] = tagMap
		result["otel_attributes"] = attributes
	}

	// Extract logs
	if logs, ok := spanData["logs"].([]interface{}); ok {
		result["logs"] = logs
	}

	// Extract process
	if process, ok := spanData["process"].(map[string]interface{}); ok {
		result["process"] = process
	}

	// Extract references
	if references, ok := spanData["references"].([]interface{}); ok {
		result["references"] = references
	}

	return result
}

// Tool definition builder methods
func (m *Module) buildGetServicesToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
	)
}

func (m *Module) buildGetOperationsToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("service", mcp.Required(), mcp.Description("Filters operations by service name")),
		mcp.WithString("spanKind", mcp.Description("Filters operations by OpenTelemetry span kind (server, client, producer, consumer, internal)")),
	)
}

func (m *Module) buildGetTraceToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("traceId", mcp.Required(), mcp.Description("Filters spans by OpenTelemetry compatible trace id in 32-character hexadecimal string format")),
		mcp.WithString("startTime", mcp.Description("The start time to filter spans in the RFC 3339, section 5.6 format, (e.g., 2017-07-21T17:32:28Z)")),
		mcp.WithString("endTime", mcp.Description("The end time to filter spans in the RFC 3339, section 5.6 format, (e.g., 2017-07-21T17:32:28Z)")),
	)
}

func (m *Module) buildFindTracesToolDefinition(config ToolConfig) mcp.Tool {
	return mcp.NewTool(m.BuildToolName(config.Name),
		mcp.WithDescription(config.Description),
		mcp.WithString("serviceName", mcp.Required(), mcp.Description("Filters spans by service name")),
		mcp.WithString("operationName", mcp.Description("The operation name to filter spans")),
		mcp.WithString("startTimeMin", mcp.Required(), mcp.Description("Start of the time interval (inclusive) in the RFC 3339, section 5.6 format")),
		mcp.WithString("startTimeMax", mcp.Required(), mcp.Description("End of the time interval (exclusive) in the RFC 3339, section 5.6 format")),
		mcp.WithString("durationMin", mcp.Description("Minimum duration of a span in milliseconds")),
		mcp.WithString("durationMax", mcp.Description("Maximum duration of a span in milliseconds")),
		mcp.WithString("searchDepth", mcp.Description("Defines the maximum search depth")),
	)
}

// Tool handlers
func (m *Module) handleGetServices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.logger.Info("Getting services")

	resp, err := m.makeJaegerRequest(ctx, "GET", "/api/services", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Jaeger API returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("Jaeger API returned status %d, body: %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		m.logger.Error("Failed to unmarshal services response",
			zap.Error(err),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("failed to unmarshal services response: %w, body: %s", err, string(body))
	}

	// Extract services from the response
	var services []interface{}
	if data, ok := response["data"]; ok {
		if servicesArray, ok := data.([]interface{}); ok {
			services = servicesArray
		}
	}

	result := map[string]interface{}{
		"services":  services,
		"count":     len(services),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response error: %w, body: %s", err, string(body))
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

func (m *Module) handleGetOperations(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	service, ok := args["service"].(string)
	if !ok {
		return nil, fmt.Errorf("service parameter is required")
	}

	spanKind := ""
	if sk, ok := args["spanKind"].(string); ok {
		spanKind = sk
	}

	m.logger.Info("Getting operations",
		zap.String("service", service),
		zap.String("spanKind", spanKind))

	// Build query parameters
	params := url.Values{}
	params.Set("service", service)
	if spanKind != "" {
		params.Set("spanKind", spanKind)
	}

	path := "/api/operations?" + params.Encode()
	resp, err := m.makeJaegerRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Jaeger API returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("Jaeger API returned status %d, body: %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		m.logger.Error("Failed to unmarshal operations response",
			zap.Error(err),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("failed to unmarshal operations response: %w, body: %s", err, string(body))
	}

	// Extract operations from the response
	var operations []interface{}
	if data, ok := response["data"]; ok {
		if operationsArray, ok := data.([]interface{}); ok {
			operations = operationsArray
		}
	}

	result := map[string]interface{}{
		"operations": operations,
		"count":      len(operations),
		"service":    service,
		"spanKind":   spanKind,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(result)
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

func (m *Module) handleGetTrace(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	traceID, ok := args["traceId"].(string)
	if !ok {
		return nil, fmt.Errorf("traceId parameter is required")
	}

	startTime := ""
	if st, ok := args["startTime"].(string); ok {
		startTime = st
	}

	endTime := ""
	if et, ok := args["endTime"].(string); ok {
		endTime = et
	}

	m.logger.Info("Getting trace",
		zap.String("traceId", traceID),
		zap.String("startTime", startTime),
		zap.String("endTime", endTime))

	// Build query parameters
	params := url.Values{}
	if startTime != "" {
		params.Set("start", startTime)
	}
	if endTime != "" {
		params.Set("end", endTime)
	}

	path := fmt.Sprintf("/api/traces/%s", traceID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := m.makeJaegerRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Jaeger API returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("Jaeger API returned status %d, body: %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		m.logger.Error("Failed to unmarshal trace response",
			zap.Error(err),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("failed to unmarshal trace response: %w, body: %s", err, string(body))
	}

	// Extract traces from the response and convert to OpenTelemetry format
	var traces []interface{}
	var otelTraces []interface{}
	if data, ok := response["data"]; ok {
		if tracesArray, ok := data.([]interface{}); ok {
			traces = tracesArray
			// Convert each trace to OpenTelemetry format
			for _, traceData := range tracesArray {
				if traceMap, ok := traceData.(map[string]interface{}); ok {
					otelTrace := m.convertJaegerTraceToOpenTelemetry(traceMap)
					otelTraces = append(otelTraces, otelTrace)
				}
			}
		}
	}

	result := map[string]interface{}{
		"traces":      traces,     // Original Jaeger format
		"otel_traces": otelTraces, // OpenTelemetry format
		"count":       len(traces),
		"traceId":     traceID,
		"format":      "opentelemetry",
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(result)
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

func (m *Module) handleFindTraces(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	serviceName, ok := args["serviceName"].(string)
	if !ok {
		return nil, fmt.Errorf("serviceName parameter is required")
	}

	startTimeMin, ok := args["startTimeMin"].(string)
	if !ok {
		return nil, fmt.Errorf("startTimeMin parameter is required")
	}

	startTimeMax, ok := args["startTimeMax"].(string)
	if !ok {
		return nil, fmt.Errorf("startTimeMax parameter is required")
	}

	operationName := ""
	if on, ok := args["operationName"].(string); ok {
		operationName = on
	}

	durationMin := ""
	if dm, ok := args["durationMin"].(string); ok {
		durationMin = dm
	}

	durationMax := ""
	if dM, ok := args["durationMax"].(string); ok {
		durationMax = dM
	}

	searchDepth := 20
	if sd, ok := args["searchDepth"].(string); ok {
		if parsed, err := strconv.Atoi(sd); err == nil {
			searchDepth = parsed
		}
	}

	m.logger.Info("Finding traces",
		zap.String("serviceName", serviceName),
		zap.String("operationName", operationName),
		zap.String("startTimeMin", startTimeMin),
		zap.String("startTimeMax", startTimeMax),
		zap.String("durationMin", durationMin),
		zap.String("durationMax", durationMax),
		zap.Int("searchDepth", searchDepth))

	// Build request body
	reqBody := map[string]interface{}{
		"service": serviceName,
		"start":   startTimeMin,
		"end":     startTimeMax,
		"limit":   searchDepth,
	}

	if operationName != "" {
		reqBody["operation"] = operationName
	}
	if durationMin != "" {
		reqBody["minDuration"] = durationMin
	}
	if durationMax != "" {
		reqBody["maxDuration"] = durationMax
	}

	resp, err := m.makeJaegerRequest(ctx, "POST", "/api/traces", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to find traces: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		m.logger.Error("Jaeger API returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("Jaeger API returned status %d, body: %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		m.logger.Error("Failed to unmarshal traces response",
			zap.Error(err),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("failed to unmarshal traces response: %w, body: %s", err, string(body))
	}

	// Extract traces from the response and convert to OpenTelemetry format
	var traces []interface{}
	var otelTraces []interface{}
	if data, ok := response["data"]; ok {
		if tracesArray, ok := data.([]interface{}); ok {
			traces = tracesArray
			// Convert each trace to OpenTelemetry format
			for _, traceData := range tracesArray {
				if traceMap, ok := traceData.(map[string]interface{}); ok {
					otelTrace := m.convertJaegerTraceToOpenTelemetry(traceMap)
					otelTraces = append(otelTraces, otelTrace)
				}
			}
		}
	}

	result := map[string]interface{}{
		"traces":        traces,     // Original Jaeger format
		"otel_traces":   otelTraces, // OpenTelemetry format
		"count":         len(traces),
		"serviceName":   serviceName,
		"operationName": operationName,
		"startTimeMin":  startTimeMin,
		"startTimeMax":  startTimeMax,
		"durationMin":   durationMin,
		"durationMax":   durationMax,
		"searchDepth":   searchDepth,
		"format":        "opentelemetry",
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(result)
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
