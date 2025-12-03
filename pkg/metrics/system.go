package metrics

import (
	"runtime"
	"time"

	"go.uber.org/zap"
)

// StartSystemMetricsCollector starts a goroutine that periodically collects system metrics
func StartSystemMetricsCollector(logger *zap.Logger) {
	if logger == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			collectSystemMetrics()
		}
	}()

	logger.Info("System metrics collector started")
}

// collectSystemMetrics collects current system metrics
func collectSystemMetrics() {
	m := Get()
	if m == nil {
		return
	}

	// Goroutine count
	m.ProcessGoroutines.Set(float64(runtime.NumGoroutine()))

	// Memory stats
	var mStats runtime.MemStats
	runtime.ReadMemStats(&mStats)

	m.ProcessMemoryBytes.WithLabelValues("heap").Set(float64(mStats.HeapAlloc))
	m.ProcessMemoryBytes.WithLabelValues("stack").Set(float64(mStats.StackInuse))
	m.ProcessMemoryBytes.WithLabelValues("sys").Set(float64(mStats.Sys))
}

