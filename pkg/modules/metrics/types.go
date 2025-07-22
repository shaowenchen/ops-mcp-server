package metrics

import (
	"encoding/json"
	"fmt"
	"time"
)

// MetricData represents basic metric data
type MetricData struct {
	Name  string      `json:"name" yaml:"name"`
	Value interface{} `json:"value" yaml:"value"`
}

// MetricsResponse represents a basic metrics response
type MetricsResponse struct {
	Status string       `json:"status" yaml:"status"`
	Data   []MetricData `json:"data" yaml:"data"`
}

// PrometheusQueryRequest represents a Prometheus query request
type PrometheusQueryRequest struct {
	Query   string    `json:"query"`
	Start   time.Time `json:"start,omitempty"`
	End     time.Time `json:"end,omitempty"`
	Step    string    `json:"step,omitempty"`
	Timeout string    `json:"timeout,omitempty"`
}

// PrometheusValue represents a single metric value
type PrometheusValue struct {
	Timestamp float64 `json:"timestamp"`
	Value     string  `json:"value"`
}

// UnmarshalJSON implements custom JSON unmarshaling for PrometheusValue
// Prometheus API returns values as arrays: [timestamp, "value"]
func (pv *PrometheusValue) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as array first (Prometheus format)
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		if len(arr) == 2 {
			// First element should be timestamp (number)
			if timestamp, ok := arr[0].(float64); ok {
				pv.Timestamp = timestamp
			} else {
				return fmt.Errorf("invalid timestamp format in PrometheusValue array")
			}

			// Second element should be value (string)
			if value, ok := arr[1].(string); ok {
				pv.Value = value
			} else {
				return fmt.Errorf("invalid value format in PrometheusValue array")
			}
			return nil
		}
		return fmt.Errorf("PrometheusValue array must have exactly 2 elements, got %d", len(arr))
	}

	// Fallback: try to unmarshal as object (for backwards compatibility)
	var obj struct {
		Timestamp float64 `json:"timestamp"`
		Value     string  `json:"value"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("failed to unmarshal PrometheusValue as array or object: %w", err)
	}

	pv.Timestamp = obj.Timestamp
	pv.Value = obj.Value
	return nil
}

// MarshalJSON implements custom JSON marshaling for PrometheusValue
func (pv PrometheusValue) MarshalJSON() ([]byte, error) {
	// Marshal as array to match Prometheus format
	arr := []interface{}{pv.Timestamp, pv.Value}
	return json.Marshal(arr)
}

// PrometheusMetric represents a metric with labels
type PrometheusMetric struct {
	Labels map[string]string `json:"metric"`
	Value  PrometheusValue   `json:"value,omitempty"`
	Values []PrometheusValue `json:"values,omitempty"`
}

// PrometheusQueryResult represents the result of a Prometheus query
type PrometheusQueryResult struct {
	ResultType string             `json:"resultType"`
	Result     []PrometheusMetric `json:"result"`
}

// PrometheusResponse represents a Prometheus API response
type PrometheusResponse struct {
	Status    string                `json:"status"`
	Data      PrometheusQueryResult `json:"data"`
	ErrorType string                `json:"errorType,omitempty"`
	Error     string                `json:"error,omitempty"`
	Warnings  []string              `json:"warnings,omitempty"`
}

// MetricsQueryRequest represents a metrics query request
type MetricsQueryRequest struct {
	Query     string `json:"query"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Step      string `json:"step,omitempty"`
	Timeout   string `json:"timeout,omitempty"`
}

// MetricsQueryResponse represents a metrics query response
type MetricsQueryResponse struct {
	Status   string            `json:"status"`
	Data     interface{}       `json:"data"`
	Error    string            `json:"error,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
