package metrics

import (
	"time"
)

// RecordMCPToolCall records an MCP tool call
func RecordMCPToolCall(toolName, module string, duration time.Duration, success bool) {
	m := Get()
	if m == nil {
		return
	}

	status := "failure"
	if success {
		status = "success"
	}

	m.MCPToolCallsTotal.WithLabelValues(toolName, module, status).Inc()
	m.MCPToolCallDuration.WithLabelValues(toolName, module).Observe(duration.Seconds())
}

// RecordMCPToolError records an MCP tool error
func RecordMCPToolError(toolName, module, errorType string) {
	m := Get()
	if m != nil {
		m.MCPToolErrorsTotal.WithLabelValues(toolName, module, errorType).Inc()
	}
}

// RecordModuleRequest records a module request
func RecordModuleRequest(moduleName string) {
	m := Get()
	if m != nil {
		m.ModuleRequestsTotal.WithLabelValues(moduleName).Inc()
	}
}

