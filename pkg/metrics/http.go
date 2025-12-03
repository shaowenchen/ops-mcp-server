package metrics

import (
	"io"
	"net/http"
	"strconv"
	"time"
)

// HTTPMetricsMiddleware wraps an HTTP handler to collect metrics
func HTTPMetricsMiddleware(next http.Handler, mode string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get endpoint from path
		endpoint := r.URL.Path
		if endpoint == "" {
			endpoint = "/"
		}

		// Track request size
		requestSize := 0
		if r.ContentLength > 0 {
			requestSize = int(r.ContentLength)
		}

		// Wrap response writer to capture status code and size
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Increment in-flight requests
		m := Get()
		if m != nil {
			m.HTTPRequestsInFlight.WithLabelValues(endpoint).Inc()
			defer m.HTTPRequestsInFlight.WithLabelValues(endpoint).Dec()

			// Record request size
			if requestSize > 0 {
				m.HTTPRequestSize.WithLabelValues(r.Method, endpoint).Observe(float64(requestSize))
			}
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Record metrics
		if m != nil {
			duration := time.Since(start).Seconds()
			statusCode := strconv.Itoa(rw.statusCode)

			m.HTTPRequestsTotal.WithLabelValues(r.Method, endpoint, statusCode, mode).Inc()
			m.HTTPRequestDuration.WithLabelValues(r.Method, endpoint, statusCode).Observe(duration)
			m.HTTPResponseSize.WithLabelValues(r.Method, endpoint).Observe(float64(rw.size))
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func (rw *responseWriter) ReadFrom(r io.Reader) (int64, error) {
	n, err := io.Copy(rw.ResponseWriter, r)
	rw.size += int(n)
	return n, err
}

// Flush implements http.Flusher to support streaming responses (e.g., SSE)
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
