package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bytes"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"
)

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Suffix string `mapstructure:"suffix" json:"suffix" yaml:"suffix"`
}

// Config contains logs module configuration
type Config struct {
	// Elasticsearch configuration - required
	Elasticsearch *ElasticsearchConfig `mapstructure:"elasticsearch" json:"elasticsearch" yaml:"elasticsearch"`
	Tools         ToolsConfig          `mapstructure:"tools" json:"tools" yaml:"tools"`
}

// ElasticsearchConfig contains elasticsearch backend configuration
type ElasticsearchConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	APIKey   string `mapstructure:"api_key" json:"api_key" yaml:"api_key"`
	Timeout  int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

// Module represents the logs module
type Module struct {
	config     *Config
	logger     *zap.Logger
	httpClient *http.Client
}

// New creates a new logs module
func New(config *Config, logger *zap.Logger) (*Module, error) {
	if config == nil {
		return nil, fmt.Errorf("logs config is required")
	}

	// Elasticsearch configuration is optional - module can be created without it

	timeout := 120 * time.Second // Increase default timeout to 120 seconds
	if config.Elasticsearch != nil && config.Elasticsearch.Timeout > 0 {
		timeout = time.Duration(config.Elasticsearch.Timeout) * time.Second
	}

	// Create HTTP client with optimized connection pooling and TIME_WAIT management
	transport := &http.Transport{
		MaxIdleConns:        50,               // Reduce maximum idle connections
		MaxIdleConnsPerHost: 5,                // Reduce idle connections per host
		MaxConnsPerHost:     20,               // Reduce maximum connections per host
		IdleConnTimeout:     30 * time.Second, // Significantly reduce idle connection timeout for faster release
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // Increase connection timeout
			KeepAlive: 15 * time.Second, // Reduce keep-alive interval
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second, // Increase TLS handshake timeout
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false, // Enable connection reuse
		ForceAttemptHTTP2:     false, // Force HTTP/1.1 for better connection reuse
		// Add connection cleanup mechanism
		ResponseHeaderTimeout: timeout, // Use configured timeout for response headers
		DisableCompression:    false,   // Enable compression to reduce transmission time
	}

	m := &Module{
		config: config,
		logger: logger.Named("logs"),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout, // Use configured timeout for client
		},
	}

	if config.Elasticsearch != nil && config.Elasticsearch.Endpoint != "" {
		m.logger.Info("Logs module created with Elasticsearch backend",
			zap.String("endpoint", config.Elasticsearch.Endpoint),
			zap.Duration("timeout", timeout),
		)
	} else {
		m.logger.Info("Logs module created without Elasticsearch configuration - tools will return configuration required error")
	}

	return m, nil
}

// GetTools returns all MCP tools for the logs module
func (m *Module) GetTools() []server.ServerTool {
	// Get default tool configuration
	toolsConfig := GetDefaultToolsConfig()

	// Tool configuration can be modified based on config file or other conditions
	// For example: disable certain tools based on m.config
	// toolsConfig.SearchLogs.Enabled = false

	return m.BuildTools(toolsConfig)
}

// Tool handlers

func (m *Module) handleQueryLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if Elasticsearch is configured
	if m.config.Elasticsearch == nil || m.config.Elasticsearch.Endpoint == "" {
		return nil, fmt.Errorf("Elasticsearch configuration not found - please set logs.elasticsearch.endpoint in config")
	}

	args := request.GetArguments()

	// Parse parameters
	var service, level, startTime, endTime string
	var size int = 100

	if val, ok := args["service"].(string); ok {
		service = val
	}
	if val, ok := args["level"].(string); ok {
		level = val
	}
	if val, ok := args["start_time"].(string); ok {
		startTime = val
	}
	if val, ok := args["end_time"].(string); ok {
		endTime = val
	}
	if val, ok := args["size"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			size = parsed
		}
	}

	// Build Elasticsearch query
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []map[string]interface{}{},
		},
	}

	mustClauses := query["bool"].(map[string]interface{})["must"].([]map[string]interface{})

	// Add filters
	if service != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"service.keyword": service,
			},
		})
	}
	if level != "" {
		mustClauses = append(mustClauses, map[string]interface{}{
			"match": map[string]interface{}{
				"level.keyword": level,
			},
		})
	}
	if startTime != "" || endTime != "" {
		timeRange := map[string]interface{}{}
		if startTime != "" {
			// Parse start time to handle relative formats like "1h", "30m", etc.
			parsedStartTime, err := parseTimeInput(startTime)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Invalid start_time format: %v", err),
						},
					},
				}, nil
			}
			timeRange["gte"] = parsedStartTime
		}
		if endTime != "" {
			// Parse end time to handle relative formats
			parsedEndTime, err := parseTimeInput(endTime)
			if err != nil {
				return &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Invalid end_time format: %v", err),
						},
					},
				}, nil
			}
			timeRange["lte"] = parsedEndTime
		}
		mustClauses = append(mustClauses, map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": timeRange,
			},
		})
	}

	query["bool"].(map[string]interface{})["must"] = mustClauses

	// Execute search
	searchQuery := map[string]interface{}{
		"query": query,
		"size":  size,
		"sort": []map[string]interface{}{
			{"@timestamp": map[string]interface{}{"order": "desc"}},
		},
	}

	resp, err := m.makeElasticsearchRequest(ctx, "POST", "*/_search", searchQuery)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to query Elasticsearch: %v", err),
				},
			},
		}, nil
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to read response: %v", err),
				},
			},
		}, nil
	}

	if resp.StatusCode != 200 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Elasticsearch returned status %d: %s", resp.StatusCode, string(body)),
				},
			},
		}, nil
	}

	var searchResult ElasticsearchSearchResponse
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to parse response: %v", err),
				},
			},
		}, nil
	}

	response := map[string]interface{}{
		"logs":  searchResult.Hits.Hits,
		"total": searchResult.Hits.Total.Value,
		"size":  size,
		"filters": map[string]interface{}{
			"service":    service,
			"level":      level,
			"start_time": startTime,
			"end_time":   endTime,
		},
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

func (m *Module) handleGetLogStats(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	timeRange := "24h"
	if val, ok := args["time_range"].(string); ok {
		timeRange = val
	}

	// Build Elasticsearch aggregation query
	aggQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"gte": "now-" + timeRange,
				},
			},
		},
		"size": 0,
		"aggs": map[string]interface{}{
			"by_level": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "level.keyword",
					"size":  10,
				},
			},
			"by_service": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "service.keyword",
					"size":  20,
				},
			},
		},
	}

	resp, err := m.makeElasticsearchRequest(ctx, "POST", "*/_search", aggQuery)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to query Elasticsearch: %v", err),
				},
			},
		}, nil
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to read response: %v", err),
				},
			},
		}, nil
	}

	if resp.StatusCode != 200 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Elasticsearch returned status %d: %s", resp.StatusCode, string(body)),
				},
			},
		}, nil
	}

	var aggResult map[string]interface{}
	if err := json.Unmarshal(body, &aggResult); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to parse response: %v", err),
				},
			},
		}, nil
	}

	// Extract aggregation results
	byLevel := make(map[string]int)
	byService := make(map[string]int)
	totalLogs := 0

	if aggs, ok := aggResult["aggregations"].(map[string]interface{}); ok {
		if levelAgg, ok := aggs["by_level"].(map[string]interface{}); ok {
			if buckets, ok := levelAgg["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						key := b["key"].(string)
						count := int(b["doc_count"].(float64))
						byLevel[key] = count
						totalLogs += count
					}
				}
			}
		}
		if serviceAgg, ok := aggs["by_service"].(map[string]interface{}); ok {
			if buckets, ok := serviceAgg["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						key := b["key"].(string)
						count := int(b["doc_count"].(float64))
						byService[key] = count
					}
				}
			}
		}
	}

	// Calculate error rate
	errorCount := 0
	if count, ok := byLevel["ERROR"]; ok {
		errorCount = count
	}
	if count, ok := byLevel["FATAL"]; ok {
		errorCount += count
	}

	errorRate := 0.0
	if totalLogs > 0 {
		errorRate = float64(errorCount) / float64(totalLogs) * 100
	}

	stats := map[string]interface{}{
		"time_range":         timeRange,
		"total_logs":         totalLogs,
		"by_level":           byLevel,
		"by_service":         byService,
		"error_rate_percent": errorRate,
		"generated_at":       time.Now().Format(time.RFC3339),
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

func (m *Module) handleGetLogServices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Query Elasticsearch for unique services
	aggQuery := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"services": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "service.keyword",
					"size":  100,
				},
			},
		},
	}

	resp, err := m.makeElasticsearchRequest(ctx, "POST", "*/_search", aggQuery)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to query Elasticsearch: %v", err),
				},
			},
		}, nil
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to read response: %v", err),
				},
			},
		}, nil
	}

	if resp.StatusCode != 200 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Elasticsearch returned status %d: %s", resp.StatusCode, string(body)),
				},
			},
		}, nil
	}

	var aggResult map[string]interface{}
	if err := json.Unmarshal(body, &aggResult); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to parse response: %v", err),
				},
			},
		}, nil
	}

	var services []string
	if aggs, ok := aggResult["aggregations"].(map[string]interface{}); ok {
		if serviceAgg, ok := aggs["services"].(map[string]interface{}); ok {
			if buckets, ok := serviceAgg["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						key := b["key"].(string)
						services = append(services, key)
					}
				}
			}
		}
	}

	response := map[string]interface{}{
		"services": services,
		"total":    len(services),
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

func (m *Module) handleGetLogLevels(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Query Elasticsearch for unique log levels actually used
	aggQuery := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"levels": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "level.keyword",
					"size":  50, // Should be enough for all possible log levels
				},
			},
		},
	}

	resp, err := m.makeElasticsearchRequest(ctx, "POST", "*/_search", aggQuery)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to query Elasticsearch: %v", err),
				},
			},
		}, nil
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to read response: %v", err),
				},
			},
		}, nil
	}

	if resp.StatusCode != 200 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Elasticsearch returned status %d: %s", resp.StatusCode, string(body)),
				},
			},
		}, nil
	}

	var aggResult map[string]interface{}
	if err := json.Unmarshal(body, &aggResult); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to parse response: %v", err),
				},
			},
		}, nil
	}

	var levels []map[string]interface{}
	var levelNames []string
	totalDocuments := int64(0)

	if aggs, ok := aggResult["aggregations"].(map[string]interface{}); ok {
		if levelAgg, ok := aggs["levels"].(map[string]interface{}); ok {
			if buckets, ok := levelAgg["buckets"].([]interface{}); ok {
				for _, bucket := range buckets {
					if b, ok := bucket.(map[string]interface{}); ok {
						levelName := b["key"].(string)
						count := int64(b["doc_count"].(float64))

						levelNames = append(levelNames, levelName)
						levels = append(levels, map[string]interface{}{
							"level": levelName,
							"count": count,
						})
						totalDocuments += count
					}
				}
			}
		}
	}

	// Calculate percentages
	for i := range levels {
		count := levels[i]["count"].(int64)
		percentage := float64(0)
		if totalDocuments > 0 {
			percentage = float64(count) / float64(totalDocuments) * 100
		}
		levels[i]["percentage"] = percentage
	}

	response := map[string]interface{}{
		"levels":          levelNames,
		"levels_detailed": levels,
		"total":           len(levels),
		"total_documents": totalDocuments,
		"queried_at":      time.Now().Format(time.RFC3339),
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

func (m *Module) handleGetRecentErrors(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	var hours int = 24
	var size int = 20

	if val, ok := args["hours"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			hours = parsed
		}
	}
	if val, ok := args["size"].(string); ok {
		if parsed, err := strconv.Atoi(val); err == nil {
			size = parsed
		}
	}

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	// Build Elasticsearch query for error and warning logs
	errorQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							"level.keyword": []string{"ERROR", "WARN", "FATAL"},
						},
					},
					{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"gte": startTime.Format(time.RFC3339),
								"lte": endTime.Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
		"size": size,
		"sort": []map[string]interface{}{
			{"@timestamp": map[string]interface{}{"order": "desc"}},
		},
		"_source": true,
	}

	// Execute search against logs indices
	resp, err := m.makeElasticsearchRequest(ctx, "POST", "*/_search", errorQuery)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to query Elasticsearch: %v", err),
				},
			},
		}, nil
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to read response: %v", err),
				},
			},
		}, nil
	}

	if resp.StatusCode != 200 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Elasticsearch returned status %d: %s", resp.StatusCode, string(body)),
				},
			},
		}, nil
	}

	var searchResult ElasticsearchSearchResponse
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to parse response: %v", err),
				},
			},
		}, nil
	}

	// Convert Elasticsearch hits to structured format with all fields
	var errors []map[string]interface{}
	for _, hit := range searchResult.Hits.Hits {
		logEntry := map[string]interface{}{
			"id": hit.ID,
		}

		// Include all fields from _source
		if source := hit.Source; source != nil {
			// Add all source fields to the log entry
			for key, value := range source {
				logEntry[key] = value
			}
		}

		errors = append(errors, logEntry)
	}

	response := map[string]interface{}{
		"errors":       errors,
		"total":        searchResult.Hits.Total.Value,
		"size":         size,
		"time_range":   fmt.Sprintf("%dh", hours),
		"generated_at": time.Now().Format(time.RFC3339),
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

// makeElasticsearchRequest creates and executes an HTTP request to Elasticsearch
func (m *Module) makeElasticsearchRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	if m.config.Elasticsearch == nil {
		return nil, fmt.Errorf("elasticsearch configuration is not available")
	}

	fullURL := strings.TrimRight(m.config.Elasticsearch.Endpoint, "/") + "/" + strings.TrimLeft(path, "/")

	var reqBody io.Reader
	var bodyStr string
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = strings.NewReader(v)
			bodyStr = v
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewBuffer(jsonData)
			bodyStr = string(jsonData)
		}
	}

	// Log request details
	m.logger.Info("🌐 Making Elasticsearch Request",
		zap.String("method", method),
		zap.String("full_url", fullURL),
		zap.String("path", path),
		zap.String("endpoint", m.config.Elasticsearch.Endpoint),
		zap.Bool("has_body", body != nil),
		zap.Int("timeout_seconds", m.config.Elasticsearch.Timeout))

	// Also print to console for visibility
	fmt.Printf("🔍 Elasticsearch API Call: %s %s\n", method, fullURL)
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

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set authentication
	authMethod := "none"
	if m.config.Elasticsearch.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+m.config.Elasticsearch.APIKey)
		authMethod = "api_key"
	} else if m.config.Elasticsearch.Username != "" && m.config.Elasticsearch.Password != "" {
		req.SetBasicAuth(m.config.Elasticsearch.Username, m.config.Elasticsearch.Password)
		authMethod = "basic_auth"
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.logger.Error("❌ Elasticsearch Request Failed",
			zap.String("method", method),
			zap.String("url", fullURL),
			zap.Error(err))
		fmt.Printf("❌ Elasticsearch Request Failed: %s %s - %v\n", method, fullURL, err)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	// Log response details
	m.logger.Info("✅ Elasticsearch Response Received",
		zap.String("method", method),
		zap.String("url", fullURL),
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status),
		zap.String("auth_method", authMethod),
		zap.Int64("content_length", resp.ContentLength))

	fmt.Printf("✅ Elasticsearch Response: %d %s\n", resp.StatusCode, resp.Status)

	return resp, nil
}

// Elasticsearch tool handlers

func (m *Module) handleListIndices(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Build query parameters
	params := url.Values{}
	if format, ok := args["format"].(string); ok && format != "" {
		params.Add("format", format)
	} else {
		params.Add("format", "json")
	}

	if health, ok := args["health"].(string); ok && health != "" {
		params.Add("health", health)
	}

	if status, ok := args["status"].(string); ok && status != "" {
		params.Add("status", status)
	}

	// Add standard parameters
	params.Add("h", "health,status,index,uuid,pri,rep,docs.count,docs.deleted,store.size,pri.store.size")
	params.Add("s", "index")

	path := "_cat/indices"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := m.makeElasticsearchRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch error (%d): %s", resp.StatusCode, string(responseData))
	}

	// Parse response based on format
	var result interface{}
	if params.Get("format") == "json" {
		var indices []ElasticsearchIndex
		if err := json.Unmarshal(responseData, &indices); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		result = map[string]interface{}{
			"indices": indices,
			"total":   len(indices),
		}
	} else {
		result = string(responseData)
	}

	data, err := json.Marshal(result)
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

func (m *Module) handleGetMappings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	indexName, ok := args["index"].(string)
	if !ok || indexName == "" {
		return nil, fmt.Errorf("index parameter is required")
	}

	includeSettings := false
	if val, ok := args["include_settings"].(bool); ok {
		includeSettings = val
	}

	// Get mappings
	mappingsPath := fmt.Sprintf("%s/_mapping", indexName)
	mappingsResp, err := m.makeElasticsearchRequest(ctx, "GET", mappingsPath, nil)
	if err != nil {
		return nil, err
	}
	defer mappingsResp.Body.Close()

	mappingsData, err := io.ReadAll(mappingsResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read mappings response: %w", err)
	}

	if mappingsResp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch error (%d): %s", mappingsResp.StatusCode, string(mappingsData))
	}

	var mappings map[string]interface{}
	if err := json.Unmarshal(mappingsData, &mappings); err != nil {
		return nil, fmt.Errorf("failed to parse mappings response: %w", err)
	}

	result := map[string]interface{}{
		"index":    indexName,
		"mappings": mappings,
	}

	// Get settings if requested
	if includeSettings {
		settingsPath := fmt.Sprintf("%s/_settings", indexName)
		settingsResp, err := m.makeElasticsearchRequest(ctx, "GET", settingsPath, nil)
		if err != nil {
			m.logger.Warn("Failed to get settings", zap.Error(err))
		} else {
			defer settingsResp.Body.Close()
			settingsData, err := io.ReadAll(settingsResp.Body)
			if err == nil && settingsResp.StatusCode < 400 {
				var settings map[string]interface{}
				if err := json.Unmarshal(settingsData, &settings); err == nil {
					result["settings"] = settings
				}
			}
		}
	}

	data, err := json.Marshal(result)
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

func (m *Module) handleElasticsearchSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Check if Elasticsearch is configured
	if m.config.Elasticsearch == nil || m.config.Elasticsearch.Endpoint == "" {
		return nil, fmt.Errorf("Elasticsearch configuration not found - please set logs.elasticsearch.endpoint in config")
	}

	args := request.GetArguments()

	indexName, ok := args["index"].(string)
	if !ok || indexName == "" {
		return nil, fmt.Errorf("index parameter is required")
	}

	bodyStr, ok := args["body"].(string)
	if !ok || bodyStr == "" {
		return nil, fmt.Errorf("body parameter is required")
	}

	// Parse the complete ES query body
	var searchRequest map[string]interface{}
	if err := json.Unmarshal([]byte(bodyStr), &searchRequest); err != nil {
		return nil, fmt.Errorf("invalid query body JSON: %w", err)
	}

	// Log the query for debugging
	if queryJSON, err := json.MarshalIndent(searchRequest, "", "  "); err == nil {
		m.logger.Info("Elasticsearch native query",
			zap.String("index", indexName),
			zap.String("query", string(queryJSON)))
	}

	// Execute the native ES search request
	path := fmt.Sprintf("%s/_search", indexName)
	resp, err := m.makeElasticsearchRequest(ctx, "POST", path, searchRequest)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to execute Elasticsearch query: %v", err),
				},
			},
		}, nil
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to read response: %v", err),
				},
			},
		}, nil
	}

	if resp.StatusCode >= 400 {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Elasticsearch returned status %d: %s", resp.StatusCode, string(responseData)),
				},
			},
		}, nil
	}

	// Return the raw ES response
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(responseData),
			},
		},
	}, nil
}

func (m *Module) handleESQL(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	format := "json"
	if f, ok := args["format"].(string); ok && f != "" {
		format = f
	}

	columnar := false
	if c, ok := args["columnar"].(string); ok && c == "true" {
		columnar = true
	}

	// Build ES|QL request
	esqlRequest := map[string]interface{}{
		"query": query,
	}

	if format != "json" {
		esqlRequest["format"] = format
	}

	if columnar {
		esqlRequest["columnar"] = true
	}

	resp, err := m.makeElasticsearchRequest(ctx, "GET", "_query", esqlRequest)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch ES|QL error (%d): %s", resp.StatusCode, string(responseData))
	}

	// Return response based on format
	var result interface{}
	if format == "json" {
		var esqlResponse ESQLResponse
		if err := json.Unmarshal(responseData, &esqlResponse); err != nil {
			return nil, fmt.Errorf("failed to parse ES|QL response: %w", err)
		}

		// Transform response into an array of objects (like Rust implementation)
		// This makes the response much more user-friendly
		objects := make([]map[string]interface{}, 0, len(esqlResponse.Values))
		for _, row := range esqlResponse.Values {
			obj := make(map[string]interface{})
			for i, value := range row {
				if i < len(esqlResponse.Columns) {
					obj[esqlResponse.Columns[i].Name] = value
				}
			}
			objects = append(objects, obj)
		}

		result = map[string]interface{}{
			"columns": esqlResponse.Columns,
			"data":    objects,
			"meta":    esqlResponse.Meta,
		}
	} else {
		result = string(responseData)
	}

	data, err := json.Marshal(result)
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

func (m *Module) handleGetShards(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Build query parameters
	params := url.Values{}
	if format, ok := args["format"].(string); ok && format != "" {
		params.Add("format", format)
	} else {
		params.Add("format", "json")
	}

	if state, ok := args["state"].(string); ok && state != "" {
		params.Add("s", state)
	}

	// Add standard parameters
	params.Add("h", "index,shard,prirep,state,docs,store,ip,node,unassigned.reason")

	path := "_cat/shards"
	if indexName, ok := args["index"].(string); ok && indexName != "" {
		path += "/" + indexName
	}

	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := m.makeElasticsearchRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch error (%d): %s", resp.StatusCode, string(responseData))
	}

	// Parse response based on format
	var result interface{}
	if params.Get("format") == "json" {
		var shards []ElasticsearchShard
		if err := json.Unmarshal(responseData, &shards); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		result = map[string]interface{}{
			"shards": shards,
			"total":  len(shards),
		}
	} else {
		result = string(responseData)
	}

	data, err := json.Marshal(result)
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

// parseTimeInput converts relative time format (like "1h", "30m", "7d") to absolute ISO timestamp
// or returns the input unchanged if it's already in absolute format
func parseTimeInput(timeInput string) (string, error) {
	if timeInput == "" {
		return "", nil
	}

	// Check if it's already an absolute time (ISO format, epoch, etc.)
	// If it contains 'T' or ':' or starts with digits and contains '-', it's likely absolute
	if strings.Contains(timeInput, "T") || strings.Contains(timeInput, ":") ||
		(len(timeInput) > 4 && strings.Contains(timeInput, "-") && timeInput[0] >= '0' && timeInput[0] <= '9') {
		return timeInput, nil
	}

	// Parse relative time format like "1h", "30m", "7d"
	re := regexp.MustCompile(`^(\d+)([smhdw])$`)
	matches := re.FindStringSubmatch(timeInput)
	if len(matches) != 3 {
		// If it doesn't match relative format, return as-is and let Elasticsearch handle it
		return timeInput, nil
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("invalid time value: %s", matches[1])
	}

	unit := matches[2]
	var duration time.Duration

	switch unit {
	case "s":
		duration = time.Duration(value) * time.Second
	case "m":
		duration = time.Duration(value) * time.Minute
	case "h":
		duration = time.Duration(value) * time.Hour
	case "d":
		duration = time.Duration(value) * 24 * time.Hour
	case "w":
		duration = time.Duration(value) * 7 * 24 * time.Hour
	default:
		return "", fmt.Errorf("unsupported time unit: %s", unit)
	}

	// Calculate the absolute time (current time minus the duration for start_time)
	absoluteTime := time.Now().Add(-duration)
	return absoluteTime.Format(time.RFC3339), nil
}
