package metrics

// Placeholder types for future metrics implementation

// MetricData represents basic metric data (placeholder)
type MetricData struct {
	Name  string      `json:"name" yaml:"name"`
	Value interface{} `json:"value" yaml:"value"`
}

// MetricsResponse represents a basic metrics response (placeholder)
type MetricsResponse struct {
	Status string       `json:"status" yaml:"status"`
	Data   []MetricData `json:"data" yaml:"data"`
}
