package observability

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HTTPMetrics struct {
	mu       sync.RWMutex
	requests map[httpMetricKey]httpMetricStats
}

type httpMetricKey struct {
	Method string
	Route  string
	Status int
}

type httpMetricStats struct {
	Count              uint64
	DurationSecondsSum float64
}

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		requests: make(map[httpMetricKey]httpMetricStats),
	}
}

func (m *HTTPMetrics) Record(method, route string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	route = strings.TrimSpace(route)
	if route == "" {
		route = "unmatched"
	}
	key := httpMetricKey{
		Method: method,
		Route:  route,
		Status: status,
	}

	m.mu.Lock()
	stats := m.requests[key]
	stats.Count++
	stats.DurationSecondsSum += duration.Seconds()
	m.requests[key] = stats
	m.mu.Unlock()
}

func (m *HTTPMetrics) RenderPrometheus() string {
	if m == nil {
		return ""
	}
	snapshots := m.snapshot()
	var builder strings.Builder
	builder.WriteString("# HELP app_http_requests_total Total HTTP requests by method, route and status.\n")
	builder.WriteString("# TYPE app_http_requests_total counter\n")
	for _, snapshot := range snapshots {
		fmt.Fprintf(
			&builder,
			"app_http_requests_total{method=%q,route=%q,status=%q} %d\n",
			snapshot.Key.Method,
			snapshot.Key.Route,
			strconv.Itoa(snapshot.Key.Status),
			snapshot.Stats.Count,
		)
	}
	builder.WriteString("# HELP app_http_request_duration_seconds_sum Total HTTP request duration in seconds by method, route and status.\n")
	builder.WriteString("# TYPE app_http_request_duration_seconds_sum counter\n")
	for _, snapshot := range snapshots {
		fmt.Fprintf(
			&builder,
			"app_http_request_duration_seconds_sum{method=%q,route=%q,status=%q} %.6f\n",
			snapshot.Key.Method,
			snapshot.Key.Route,
			strconv.Itoa(snapshot.Key.Status),
			snapshot.Stats.DurationSecondsSum,
		)
	}
	builder.WriteString("# HELP app_http_request_duration_seconds_count HTTP request duration sample count by method, route and status.\n")
	builder.WriteString("# TYPE app_http_request_duration_seconds_count counter\n")
	for _, snapshot := range snapshots {
		fmt.Fprintf(
			&builder,
			"app_http_request_duration_seconds_count{method=%q,route=%q,status=%q} %d\n",
			snapshot.Key.Method,
			snapshot.Key.Route,
			strconv.Itoa(snapshot.Key.Status),
			snapshot.Stats.Count,
		)
	}
	return builder.String()
}

type httpMetricSnapshot struct {
	Key   httpMetricKey
	Stats httpMetricStats
}

func (m *HTTPMetrics) snapshot() []httpMetricSnapshot {
	m.mu.RLock()
	snapshots := make([]httpMetricSnapshot, 0, len(m.requests))
	for key, stats := range m.requests {
		snapshots = append(snapshots, httpMetricSnapshot{Key: key, Stats: stats})
	}
	m.mu.RUnlock()

	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].Key.Method != snapshots[j].Key.Method {
			return snapshots[i].Key.Method < snapshots[j].Key.Method
		}
		if snapshots[i].Key.Route != snapshots[j].Key.Route {
			return snapshots[i].Key.Route < snapshots[j].Key.Route
		}
		return snapshots[i].Key.Status < snapshots[j].Key.Status
	})
	return snapshots
}
