package events

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Event represents a CloudEvent format event (can be K8s or other types)
type Event struct {
	SpecVersion     string          `json:"specversion" yaml:"specversion"`
	ID              string          `json:"id" yaml:"id"`
	Source          string          `json:"source" yaml:"source"`
	Type            string          `json:"type" yaml:"type"`
	Subject         string          `json:"subject" yaml:"subject"`
	DataContentType string          `json:"datacontenttype" yaml:"datacontenttype"`
	Time            string          `json:"time" yaml:"time"`
	Data            json.RawMessage `json:"data" yaml:"data"` // Keep as RawMessage for flexibility
	Cluster         string          `json:"cluster" yaml:"cluster"`
}

// EventWrapper represents the complete event structure from the API
type EventWrapper struct {
	Subject string `json:"subject" yaml:"subject"`
	Event   Event  `json:"event" yaml:"event"`
}

// ParsedEventInfo contains extracted information from the event subject
type ParsedEventInfo struct {
	Cluster     string `json:"cluster" yaml:"cluster"`
	Namespace   string `json:"namespace" yaml:"namespace"`
	Resource    string `json:"resource" yaml:"resource"`                             // e.g., "pods", "deployments", "nodes", etc.
	Name        string `json:"name" yaml:"name"`                                     // resource name
	EventType   string `json:"event_type" yaml:"event_type"`                         // kubernetes, application, infrastructure, etc.
	SubCategory string `json:"sub_category,omitempty" yaml:"sub_category,omitempty"` // for non-k8s events
}

// EnhancedEvent includes parsed information for quick access
type EnhancedEvent struct {
	EventWrapper
	ParsedInfo ParsedEventInfo `json:"parsed_info" yaml:"parsed_info"`
}

// ParseSubject parses the subject string to extract event information
// Supports multiple formats:
// Kubernetes events:
// - Pods/Deployments: ops.clusters.{cluster}.namespaces.{namespace}.{resource}.{name}.event
// - Nodes: ops.clusters.{cluster}.nodes.{name}.event
// Other event formats can be added here
func ParseSubject(subject string) ParsedEventInfo {
	info := ParsedEventInfo{}

	// Check if it's a Kubernetes cluster event (contains "ops.clusters")
	if strings.Contains(subject, "ops.clusters") {
		info.EventType = "kubernetes"

		// Try nodes pattern first (no namespace)
		// Pattern: ops.clusters.{cluster}.nodes.{name}.event
		nodesPattern := `^ops\.clusters\.([^.]+)\.nodes\.([^.]+)\.event$`
		nodesRe := regexp.MustCompile(nodesPattern)

		matches := nodesRe.FindStringSubmatch(subject)
		if len(matches) == 3 {
			info.Cluster = matches[1]
			info.Resource = "nodes"
			info.Name = matches[2]
			// Nodes don't have namespace
			info.Namespace = ""
			return info
		}

		// Try namespaced resources pattern
		// Pattern: ops.clusters.{cluster}.namespaces.{namespace}.{resource}.{name}.event
		namespacedPattern := `^ops\.clusters\.([^.]+)\.namespaces\.([^.]+)\.([^.]+)\.([^.]+)\.event$`
		namespacedRe := regexp.MustCompile(namespacedPattern)

		matches = namespacedRe.FindStringSubmatch(subject)
		if len(matches) == 5 {
			info.Cluster = matches[1]
			info.Namespace = matches[2]
			info.Resource = matches[3] // Accept any resource type
			info.Name = matches[4]
			return info
		}

		// Fallback for Kubernetes events: try to extract parts manually
		parts := strings.Split(subject, ".")
		for i, part := range parts {
			switch part {
			case "clusters":
				if i+1 < len(parts) {
					info.Cluster = parts[i+1]
				}
			case "namespaces":
				if i+1 < len(parts) {
					info.Namespace = parts[i+1]
				}
			case "nodes":
				info.Resource = part
				if i+1 < len(parts) && parts[i+1] != "event" {
					info.Name = parts[i+1]
				}
				return info
			default:
				// For other resources (pods, deployments, services, etc.)
				if i > 0 && (parts[i-1] == "namespaces" || parts[i-1] == "clusters") {
					continue // Skip already processed parts
				}
				if i+1 < len(parts) && parts[i+1] != "event" && !strings.Contains(part, "ops") {
					// This might be a resource type
					if parts[i-1] != "clusters" && parts[i-1] != "namespaces" {
						info.Resource = part
						info.Name = parts[i+1]
					}
				}
			}
		}
	} else {
		// Handle other event types (application, infrastructure, etc.)
		info.EventType = "other"
		// Extract basic information for non-kubernetes events
		parts := strings.Split(subject, ".")
		if len(parts) > 0 {
			info.SubCategory = parts[0] // First part as category
		}
		// Try to extract meaningful information from other event formats
		// This can be extended based on actual event formats you receive
	}

	return info
}

// EventsListRequest represents a request to list events
type EventsListRequest struct {
	StartTime string `json:"startTime,omitempty" yaml:"startTime,omitempty"`
	Limit     int    `json:"limit,omitempty" yaml:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty" yaml:"offset,omitempty"`
	// Raw subject pattern for direct NATS queries (takes precedence over structured fields)
	SubjectPattern string `json:"subjectPattern,omitempty" yaml:"subjectPattern,omitempty"`
	// Kubernetes-specific fields
	Cluster      string `json:"cluster,omitempty" yaml:"cluster,omitempty"`
	Namespace    string `json:"namespace,omitempty" yaml:"namespace,omitempty"`       // not applicable for nodes
	Resource     string `json:"resource,omitempty" yaml:"resource,omitempty"`         // pods, deployments, or nodes
	ResourceName string `json:"resourceName,omitempty" yaml:"resourceName,omitempty"` // specific resource name (optional)
}

// EventsListResponse represents the response from listing events
type EventsListResponse struct {
	Code    int    `json:"code" yaml:"code"`
	Message string `json:"message" yaml:"message"`
	Data    struct {
		PageSize int             `json:"page_size" yaml:"page_size"`
		Page     int             `json:"page" yaml:"page"`
		List     []EnhancedEvent `json:"list" yaml:"list"`
		Total    int             `json:"total" yaml:"total"`
	} `json:"data" yaml:"data"`
}
