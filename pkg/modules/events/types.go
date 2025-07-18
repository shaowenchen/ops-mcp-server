package events

import "time"

// Event represents an operational event
type Event struct {
	ID        string                 `json:"id" yaml:"id"`
	Type      string                 `json:"type" yaml:"type"`
	Service   string                 `json:"service" yaml:"service"`
	Timestamp time.Time              `json:"timestamp" yaml:"timestamp"`
	Status    string                 `json:"status" yaml:"status"`
	Message   string                 `json:"message" yaml:"message"`
	Details   map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
}

// EventsListRequest represents a request to list events
type EventsListRequest struct {
	StartTime  string   `json:"startTime,omitempty" yaml:"startTime,omitempty"`
	EndTime    string   `json:"endTime,omitempty" yaml:"endTime,omitempty"`
	EventType  string   `json:"eventType,omitempty" yaml:"eventType,omitempty"`
	EventTypes []string `json:"eventTypes,omitempty" yaml:"eventTypes,omitempty"`
	Service    string   `json:"service,omitempty" yaml:"service,omitempty"`
	Services   []string `json:"services,omitempty" yaml:"services,omitempty"`
	Status     string   `json:"status,omitempty" yaml:"status,omitempty"`
	Limit      int      `json:"limit,omitempty" yaml:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty" yaml:"offset,omitempty"`
	SortBy     string   `json:"sortBy,omitempty" yaml:"sortBy,omitempty"`
	SortOrder  string   `json:"sortOrder,omitempty" yaml:"sortOrder,omitempty"`
}

// EventsListResponse represents the response from listing events
type EventsListResponse struct {
	Events []Event `json:"events" yaml:"events"`
	Total  int     `json:"total" yaml:"total"`
	Limit  int     `json:"limit" yaml:"limit"`
	Offset int     `json:"offset" yaml:"offset"`
}

// EventsSubscribeRequest represents a request to subscribe to events
type EventsSubscribeRequest struct {
	EventTypes []string `json:"eventTypes,omitempty" yaml:"eventTypes,omitempty"`
	Services   []string `json:"services,omitempty" yaml:"services,omitempty"`
	Statuses   []string `json:"statuses,omitempty" yaml:"statuses,omitempty"`
}
