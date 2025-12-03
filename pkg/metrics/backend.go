package metrics

import (
	"time"
)

// BackendType represents the type of backend service
type BackendType string

const (
	BackendPrometheus   BackendType = "prometheus"
	BackendElasticsearch BackendType = "elasticsearch"
	BackendJaeger       BackendType = "jaeger"
	BackendOps          BackendType = "ops"
)

// RecordBackendRequest records a backend service request
func RecordBackendRequest(backend BackendType, duration time.Duration, success bool) {
	m := Get()
	if m == nil {
		return
	}

	status := "failure"
	if success {
		status = "success"
	}

	m.BackendRequestsTotal.WithLabelValues(string(backend), status).Inc()
	m.BackendRequestDuration.WithLabelValues(string(backend)).Observe(duration.Seconds())
}

// RecordBackendError records a backend service error
func RecordBackendError(backend BackendType, errorType string) {
	m := Get()
	if m != nil {
		m.BackendErrorsTotal.WithLabelValues(string(backend), errorType).Inc()
	}
}

