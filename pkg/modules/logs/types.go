package logs

// Placeholder types for future logs implementation

// LogData represents basic log data (placeholder)
type LogData struct {
	Level   string `json:"level" yaml:"level"`
	Message string `json:"message" yaml:"message"`
	Service string `json:"service" yaml:"service"`
}

// LogsResponse represents a basic logs response (placeholder)
type LogsResponse struct {
	Status string    `json:"status" yaml:"status"`
	Logs   []LogData `json:"logs" yaml:"logs"`
}
