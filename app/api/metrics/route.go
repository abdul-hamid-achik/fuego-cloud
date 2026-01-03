package metrics

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/abdul-hamid-achik/fuego/pkg/fuego"
)

// Metrics holds application metrics
var (
	requestCount   atomic.Uint64
	requestLatency atomic.Uint64 // in microseconds
	errorCount     atomic.Uint64
	startTime      = time.Now()
)

// IncrementRequests increments the request counter
func IncrementRequests() {
	requestCount.Add(1)
}

// AddLatency adds latency to the total (for averaging)
func AddLatency(d time.Duration) {
	requestLatency.Add(uint64(d.Microseconds()))
}

// IncrementErrors increments the error counter
func IncrementErrors() {
	errorCount.Add(1)
}

// Get returns Prometheus-formatted metrics
func Get(c *fuego.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime).Seconds()
	requests := requestCount.Load()
	latency := requestLatency.Load()
	errors := errorCount.Load()

	avgLatency := float64(0)
	if requests > 0 {
		avgLatency = float64(latency) / float64(requests) / 1000.0 // convert to ms
	}

	// Prometheus exposition format
	metrics := fmt.Sprintf(`# HELP fuego_cloud_uptime_seconds Total uptime in seconds
# TYPE fuego_cloud_uptime_seconds gauge
fuego_cloud_uptime_seconds %.2f

# HELP fuego_cloud_requests_total Total number of HTTP requests
# TYPE fuego_cloud_requests_total counter
fuego_cloud_requests_total %d

# HELP fuego_cloud_request_latency_avg_ms Average request latency in milliseconds
# TYPE fuego_cloud_request_latency_avg_ms gauge
fuego_cloud_request_latency_avg_ms %.2f

# HELP fuego_cloud_errors_total Total number of errors
# TYPE fuego_cloud_errors_total counter
fuego_cloud_errors_total %d

# HELP fuego_cloud_goroutines Current number of goroutines
# TYPE fuego_cloud_goroutines gauge
fuego_cloud_goroutines %d

# HELP fuego_cloud_memory_alloc_bytes Currently allocated memory in bytes
# TYPE fuego_cloud_memory_alloc_bytes gauge
fuego_cloud_memory_alloc_bytes %d

# HELP fuego_cloud_memory_sys_bytes Total memory obtained from OS in bytes
# TYPE fuego_cloud_memory_sys_bytes gauge
fuego_cloud_memory_sys_bytes %d

# HELP fuego_cloud_gc_pause_total_ns Total GC pause time in nanoseconds
# TYPE fuego_cloud_gc_pause_total_ns counter
fuego_cloud_gc_pause_total_ns %d

# HELP fuego_cloud_gc_num_gc Total number of GC cycles
# TYPE fuego_cloud_gc_num_gc counter
fuego_cloud_gc_num_gc %d
`,
		uptime,
		requests,
		avgLatency,
		errors,
		runtime.NumGoroutine(),
		m.Alloc,
		m.Sys,
		m.PauseTotalNs,
		m.NumGC,
	)

	c.Response.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	return c.String(200, metrics)
}
