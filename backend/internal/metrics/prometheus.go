package metrics

import (
	"fmt"
	"io"
)

func (c *Collector) WritePrometheus(w io.Writer, pool PoolSource, mode string) {
	snapshot := c.Snapshot(1)
	current := snapshot.Current
	usage := pool()
	fmt.Fprintf(w, "# TYPE mongoscale_requests_total counter\nmongoscale_requests_total %d\n", snapshot.Totals.Requests)
	fmt.Fprintf(w, "# TYPE mongoscale_errors_total counter\nmongoscale_errors_total %d\n", snapshot.Totals.Errors)
	fmt.Fprintf(w, "# TYPE mongoscale_throttled_total counter\nmongoscale_throttled_total %d\n", snapshot.Totals.Throttled)
	fmt.Fprintf(w, "# TYPE mongoscale_requests_per_second gauge\nmongoscale_requests_per_second %f\n", current.RPS)
	fmt.Fprintf(w, "# TYPE mongoscale_latency_ms gauge\n")
	fmt.Fprintf(w, "mongoscale_latency_ms{quantile=\"0.5\"} %f\n", current.P50)
	fmt.Fprintf(w, "mongoscale_latency_ms{quantile=\"0.95\"} %f\n", current.P95)
	fmt.Fprintf(w, "mongoscale_latency_ms{quantile=\"0.99\"} %f\n", current.P99)
	fmt.Fprintf(w, "# TYPE mongoscale_error_rate_percent gauge\nmongoscale_error_rate_percent %f\n", current.ErrorRate)
	fmt.Fprintf(w, "# TYPE mongoscale_active_users gauge\nmongoscale_active_users %d\n", current.ActiveUsers)
	fmt.Fprintf(w, "# TYPE mongoscale_inflight_requests gauge\nmongoscale_inflight_requests %d\n", current.Inflight)
	fmt.Fprintf(w, "# TYPE mongoscale_mongo_reads_per_second gauge\nmongoscale_mongo_reads_per_second %f\n", current.MongoReads)
	fmt.Fprintf(w, "# TYPE mongoscale_mongo_writes_per_second gauge\nmongoscale_mongo_writes_per_second %f\n", current.MongoWrites)
	fmt.Fprintf(w, "# TYPE mongoscale_mongo_slow_queries_total counter\nmongoscale_mongo_slow_queries_total %d\n", current.SlowQueries)
	fmt.Fprintf(w, "# TYPE mongoscale_pool_connections gauge\n")
	fmt.Fprintf(w, "mongoscale_pool_connections{state=\"open\"} %d\n", usage.Open)
	fmt.Fprintf(w, "mongoscale_pool_connections{state=\"in_use\"} %d\n", usage.InUse)
	fmt.Fprintf(w, "mongoscale_pool_connections{state=\"waiting\"} %d\n", usage.Waiting)
	fmt.Fprintf(w, "mongoscale_pool_connections{state=\"max\"} %d\n", usage.Max)
	fmt.Fprintf(w, "# TYPE mongoscale_process_cpu_percent gauge\nmongoscale_process_cpu_percent %f\n", current.CPUPercent)
	fmt.Fprintf(w, "# TYPE mongoscale_process_memory_mb gauge\nmongoscale_process_memory_mb %f\n", current.MemoryMB)
	fmt.Fprintf(w, "# TYPE mongoscale_mode_info gauge\nmongoscale_mode_info{mode=%q} 1\n", mode)
}
