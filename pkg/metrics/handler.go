package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns the HTTP handler for the /metrics endpoint
func Handler() http.Handler {
	return promhttp.Handler()
}

