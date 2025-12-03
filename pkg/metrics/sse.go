package metrics

import (
	"time"
)

// RecordSSEConnection records a new SSE connection
func RecordSSEConnection() {
	m := Get()
	if m != nil {
		m.SSEConnectionsTotal.Inc()
		m.SSEActiveConnections.Inc()
	}
}

// RecordSSEDisconnection records an SSE disconnection
func RecordSSEDisconnection(duration time.Duration) {
	m := Get()
	if m != nil {
		m.SSEActiveConnections.Dec()
		m.SSEConnectionDuration.Observe(duration.Seconds())
	}
}

