package docs

import (
	"encoding/json"
	"time"

	"github.com/shaowenchen/ops-mcp-server/cmd/version"
	"github.com/shaowenchen/ops-mcp-server/pkg/config"
	eventsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/events"
	logsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/logs"
	metricsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/metrics"
	sopsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/sops"
	tracesModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/traces"
	"go.uber.org/zap"
)

// Collector collects tool information from all enabled modules
type Collector struct {
	config *config.Config
	logger *zap.Logger
}

// NewCollector creates a new docs collector
func NewCollector(cfg *config.Config, logger *zap.Logger) *Collector {
	return &Collector{
		config: cfg,
		logger: logger,
	}
}

// CollectToolsInfo collects tool information from all enabled modules
func (c *Collector) CollectToolsInfo() ToolsInfoResponse {
	var tools []ToolInfo
	var enabledModules []string
	totalTools := 0

	versionInfo := version.Get()

	// Collect tools from SOPS module
	if c.config.Sops.Enabled {
		enabledModules = append(enabledModules, "sops")
		sopsTools := c.collectSOPSTools()
		tools = append(tools, sopsTools...)
		totalTools += len(sopsTools)
	}

	// Collect tools from Events module
	if c.config.Events.Enabled {
		enabledModules = append(enabledModules, "events")
		eventsTools := c.collectEventsTools()
		tools = append(tools, eventsTools...)
		totalTools += len(eventsTools)
	}

	// Collect tools from Metrics module
	if c.config.Metrics.Enabled {
		enabledModules = append(enabledModules, "metrics")
		metricsTools := c.collectMetricsTools()
		tools = append(tools, metricsTools...)
		totalTools += len(metricsTools)
	}

	// Collect tools from Logs module
	if c.config.Logs.Enabled {
		enabledModules = append(enabledModules, "logs")
		logsTools := c.collectLogsTools()
		tools = append(tools, logsTools...)
		totalTools += len(logsTools)
	}

	// Collect tools from Traces module
	if c.config.Traces.Enabled {
		enabledModules = append(enabledModules, "traces")
		tracesTools := c.collectTracesTools()
		tools = append(tools, tracesTools...)
		totalTools += len(tracesTools)
	}

	return ToolsInfoResponse{
		Service:    "ops-mcp-server",
		Version:    versionInfo.Version,
		TotalTools: totalTools,
		Modules:    enabledModules,
		Tools:      tools,
	}
}

// collectSOPSTools collects tools from SOPS module
func (c *Collector) collectSOPSTools() []ToolInfo {
	var tools []ToolInfo
	
	sopsConfig := &sopsModule.Config{
		Tools: sopsModule.ToolsConfig{
			Prefix: c.config.Sops.Tools.Prefix,
			Suffix: c.config.Sops.Tools.Suffix,
		},
	}
	if c.config.Sops.Ops != nil {
		sopsConfig.Endpoint = c.config.Sops.Ops.Endpoint
		sopsConfig.Token = c.config.Sops.Ops.Token
	}
	
	sopsModuleInstance, err := sopsModule.New(sopsConfig, c.logger)
	if err != nil {
		c.logger.Error("Failed to create SOPS module for docs", zap.Error(err))
		return tools
	}
	
	sopsTools := sopsModuleInstance.GetTools()
	for _, serverTool := range sopsTools {
		toolInfo := ToolInfo{
			Name:        serverTool.Tool.Name,
			Description: serverTool.Tool.Description,
			Parameters:  convertToolParameters(serverTool.Tool.InputSchema),
			Module:      "sops",
		}
		tools = append(tools, toolInfo)
	}
	
	return tools
}

// collectEventsTools collects tools from Events module
func (c *Collector) collectEventsTools() []ToolInfo {
	var tools []ToolInfo
	
	eventsConfig := &eventsModule.Config{
		PollInterval: 30 * time.Second,
		Tools: eventsModule.ToolsConfig{
			Prefix: c.config.Events.Tools.Prefix,
			Suffix: c.config.Events.Tools.Suffix,
		},
	}
	if c.config.Events.Ops != nil {
		eventsConfig.Endpoint = c.config.Events.Ops.Endpoint
		eventsConfig.Token = c.config.Events.Ops.Token
	}
	
	eventsModuleInstance, err := eventsModule.New(eventsConfig, c.logger)
	if err != nil {
		c.logger.Error("Failed to create Events module for docs", zap.Error(err))
		return tools
	}
	
	eventsTools := eventsModuleInstance.GetTools()
	for _, serverTool := range eventsTools {
		toolInfo := ToolInfo{
			Name:        serverTool.Tool.Name,
			Description: serverTool.Tool.Description,
			Parameters:  convertToolParameters(serverTool.Tool.InputSchema),
			Module:      "events",
		}
		tools = append(tools, toolInfo)
	}
	
	return tools
}

// collectMetricsTools collects tools from Metrics module
func (c *Collector) collectMetricsTools() []ToolInfo {
	var tools []ToolInfo
	
	metricsConfig := &metricsModule.Config{
		Tools: metricsModule.ToolsConfig{
			Prefix: c.config.Metrics.Tools.Prefix,
			Suffix: c.config.Metrics.Tools.Suffix,
		},
	}
	if c.config.Metrics.Prometheus != nil {
		metricsConfig.Prometheus = &metricsModule.PrometheusConfig{
			Endpoint: c.config.Metrics.Prometheus.Endpoint,
		}
	}
	
	metricsModuleInstance, err := metricsModule.New(metricsConfig, c.logger)
	if err != nil {
		c.logger.Error("Failed to create Metrics module for docs", zap.Error(err))
		return tools
	}
	
	metricsTools := metricsModuleInstance.GetTools()
	for _, serverTool := range metricsTools {
		toolInfo := ToolInfo{
			Name:        serverTool.Tool.Name,
			Description: serverTool.Tool.Description,
			Parameters:  convertToolParameters(serverTool.Tool.InputSchema),
			Module:      "metrics",
		}
		tools = append(tools, toolInfo)
	}
	
	return tools
}

// collectLogsTools collects tools from Logs module
func (c *Collector) collectLogsTools() []ToolInfo {
	var tools []ToolInfo
	
	logsConfig := &logsModule.Config{
		Tools: logsModule.ToolsConfig{
			Prefix: c.config.Logs.Tools.Prefix,
			Suffix: c.config.Logs.Tools.Suffix,
		},
	}
	if c.config.Logs.Elasticsearch != nil {
		logsConfig.Elasticsearch = &logsModule.ElasticsearchConfig{
			Endpoint: c.config.Logs.Elasticsearch.Endpoint,
			Username: c.config.Logs.Elasticsearch.Username,
			Password: c.config.Logs.Elasticsearch.Password,
			APIKey:   c.config.Logs.Elasticsearch.APIKey,
			Timeout:  c.config.Logs.Elasticsearch.Timeout,
		}
	}
	
	logsModuleInstance, err := logsModule.New(logsConfig, c.logger)
	if err != nil {
		c.logger.Error("Failed to create Logs module for docs", zap.Error(err))
		return tools
	}
	
	logsTools := logsModuleInstance.GetTools()
	for _, serverTool := range logsTools {
		toolInfo := ToolInfo{
			Name:        serverTool.Tool.Name,
			Description: serverTool.Tool.Description,
			Parameters:  convertToolParameters(serverTool.Tool.InputSchema),
			Module:      "logs",
		}
		tools = append(tools, toolInfo)
	}
	
	return tools
}

// collectTracesTools collects tools from Traces module
func (c *Collector) collectTracesTools() []ToolInfo {
	var tools []ToolInfo
	
	tracesConfig := &tracesModule.Config{
		Tools: tracesModule.ToolsConfig{
			Prefix: c.config.Traces.Tools.Prefix,
			Suffix: c.config.Traces.Tools.Suffix,
		},
	}
	if c.config.Traces.Jaeger != nil {
		tracesConfig.Endpoint = c.config.Traces.Jaeger.Endpoint
		tracesConfig.Protocol = "HTTP"
		tracesConfig.Port = 16686
		tracesConfig.Timeout = c.config.Traces.Jaeger.Timeout
	}
	
	tracesModuleInstance, err := tracesModule.New(tracesConfig, c.logger)
	if err != nil {
		c.logger.Error("Failed to create Traces module for docs", zap.Error(err))
		return tools
	}
	
	tracesTools := tracesModuleInstance.GetTools()
	for _, serverTool := range tracesTools {
		toolInfo := ToolInfo{
			Name:        serverTool.Tool.Name,
			Description: serverTool.Tool.Description,
			Parameters:  convertToolParameters(serverTool.Tool.InputSchema),
			Module:      "traces",
		}
		tools = append(tools, toolInfo)
	}
	
	return tools
}

// convertToolParameters converts MCP tool input schema to a more readable format
func convertToolParameters(inputSchema interface{}) map[string]interface{} {
	params := make(map[string]interface{})
	
	// Convert the inputSchema to JSON first, then parse it as a map
	schemaBytes, err := json.Marshal(inputSchema)
	if err != nil {
		return params
	}
	
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return params
	}
	
	if properties, exists := schemaMap["properties"]; exists {
		if propsMap, ok := properties.(map[string]interface{}); ok {
			for paramName, paramDef := range propsMap {
				if paramDefMap, ok := paramDef.(map[string]interface{}); ok {
					paramInfo := map[string]interface{}{
						"type": paramDefMap["type"],
					}
					
					if description, exists := paramDefMap["description"]; exists {
						paramInfo["description"] = description
					}
					
					// Check if parameter is required
					if required, exists := schemaMap["required"]; exists {
						if requiredList, ok := required.([]interface{}); ok {
							for _, req := range requiredList {
								if reqStr, ok := req.(string); ok && reqStr == paramName {
									paramInfo["required"] = true
									break
								}
							}
						}
					}
					
					params[paramName] = paramInfo
				}
			}
		}
	}
	
	return params
}
