package traces

import (
	"encoding/json"
	"time"
)

// JaegerOperation represents a Jaeger operation
type JaegerOperation struct {
	Name     string `json:"name"`
	SpanKind string `json:"spanKind"`
}

// JaegerService represents a Jaeger service
type JaegerService struct {
	Name string `json:"name"`
}

// JaegerSpan represents a Jaeger span
type JaegerSpan struct {
	TraceID       string                 `json:"traceID"`
	SpanID        string                 `json:"spanID"`
	ParentSpanID  string                 `json:"parentSpanID,omitempty"`
	OperationName string                 `json:"operationName"`
	StartTime     int64                  `json:"startTime"`
	Duration      int64                  `json:"duration"`
	Tags          map[string]interface{} `json:"tags"`
	Logs          []JaegerLog            `json:"logs,omitempty"`
	Process       JaegerProcess          `json:"process"`
	References    []JaegerReference      `json:"references,omitempty"`
}

// JaegerLog represents a Jaeger log entry
type JaegerLog struct {
	Timestamp int64                  `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// JaegerProcess represents a Jaeger process
type JaegerProcess struct {
	ServiceName string                 `json:"serviceName"`
	Tags        map[string]interface{} `json:"tags"`
}

// JaegerReference represents a Jaeger reference
type JaegerReference struct {
	RefType string `json:"refType"`
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}

// JaegerTrace represents a complete Jaeger trace
type JaegerTrace struct {
	TraceID   string                   `json:"traceID"`
	Spans     []JaegerSpan             `json:"spans"`
	Processes map[string]JaegerProcess `json:"processes"`
	Warnings  []string                 `json:"warnings,omitempty"`
}

// JaegerFindTracesRequest represents a request to find traces
type JaegerFindTracesRequest struct {
	ServiceName   string                 `json:"service"`
	OperationName string                 `json:"operation,omitempty"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	StartTimeMin  string                 `json:"start"`
	StartTimeMax  string                 `json:"end"`
	DurationMin   string                 `json:"minDuration,omitempty"`
	DurationMax   string                 `json:"maxDuration,omitempty"`
	SearchDepth   int                    `json:"limit,omitempty"`
}

// JaegerGetTraceRequest represents a request to get a specific trace
type JaegerGetTraceRequest struct {
	TraceID   string `json:"traceID"`
	StartTime string `json:"startTime,omitempty"`
	EndTime   string `json:"endTime,omitempty"`
}

// JaegerGetOperationsRequest represents a request to get operations
type JaegerGetOperationsRequest struct {
	Service  string `json:"service"`
	SpanKind string `json:"spanKind,omitempty"`
}

// JaegerAPIResponse represents a generic Jaeger API response
type JaegerAPIResponse struct {
	Data   interface{} `json:"data"`
	Total  int         `json:"total,omitempty"`
	Limit  int         `json:"limit,omitempty"`
	Offset int         `json:"offset,omitempty"`
	Errors []string    `json:"errors,omitempty"`
}

// JaegerOperationsResponse represents the response for operations
type JaegerOperationsResponse struct {
	Data []JaegerOperation `json:"data"`
}

// JaegerServicesResponse represents the response for services
type JaegerServicesResponse struct {
	Data []string `json:"data"`
}

// JaegerTracesResponse represents the response for traces
type 
JaegerTracesResponse struct {
	Data []JaegerTrace `json:"data"`
}

// JaegerTraceResponse represents the response for a single trace
type JaegerTraceResponse struct {
	Data []JaegerTrace `json:"data"`
}

// JaegerTime provides custom JSON unmarshaling for time fields
type JaegerTime struct {
	time.Time
}

// UnmarshalJSON unmarshals Jaeger timestamp from JSON
func (jt *JaegerTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}
	jt.Time = time.Unix(0, timestamp*1000) // Jaeger timestamps are in microseconds
	return nil
}

// MarshalJSON marshals Jaeger timestamp to JSON
func (jt JaegerTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(jt.Time.UnixNano() / 1000) // Convert to microseconds
}
