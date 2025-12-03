package metrics

import (
	"time"
)

// RecordAuthRequest records an authentication request
func RecordAuthRequest(success bool, skipped bool) {
	m := Get()
	if m == nil {
		return
	}

	status := "failure"
	if skipped {
		status = "skipped"
	} else if success {
		status = "success"
	}

	m.AuthRequestsTotal.WithLabelValues(status).Inc()
}

// RecordAuthValidationDuration records authentication validation duration
func RecordAuthValidationDuration(duration time.Duration) {
	m := Get()
	if m != nil {
		m.AuthValidationDuration.Observe(duration.Seconds())
	}
}

