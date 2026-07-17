package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type BusinessMetrics struct {
	mu       sync.RWMutex
	counters map[businessMetricKey]uint64
}

type businessMetricKey struct {
	Name   string
	Labels string
}

func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		counters: make(map[businessMetricKey]uint64),
	}
}

func (m *BusinessMetrics) RecordOrderCreate(result string) {
	m.increment("app_order_create_total", map[string]string{"result": normalizeLabelValue(result)})
}

func (m *BusinessMetrics) RecordOrderStateTransition(action, result string) {
	m.increment("app_order_state_transition_total", map[string]string{
		"action": normalizeLabelValue(action),
		"result": normalizeLabelValue(result),
	})
}

func (m *BusinessMetrics) RecordRedisInventoryPreDeduct(result string) {
	m.increment("app_redis_inventory_prededuct_total", map[string]string{"result": normalizeLabelValue(result)})
}

func (m *BusinessMetrics) RecordRedisInventoryReservation(action, result string) {
	m.increment("app_redis_inventory_reservation_total", map[string]string{
		"action": normalizeLabelValue(action),
		"result": normalizeLabelValue(result),
	})
}

func (m *BusinessMetrics) RecordRedisInventorySync(result string) {
	m.increment("app_redis_inventory_sync_total", map[string]string{"result": normalizeLabelValue(result)})
}

func (m *BusinessMetrics) increment(name string, labels map[string]string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.counters[businessMetricKey{Name: name, Labels: formatLabels(labels)}]++
	m.mu.Unlock()
}

func (m *BusinessMetrics) RenderPrometheus() string {
	if m == nil {
		return ""
	}
	snapshots := m.snapshot()
	if len(snapshots) == 0 {
		return ""
	}
	var builder strings.Builder
	currentMetric := ""
	for _, snapshot := range snapshots {
		if snapshot.Key.Name != currentMetric {
			currentMetric = snapshot.Key.Name
			fmt.Fprintf(&builder, "# HELP %s Business counter.\n", currentMetric)
			fmt.Fprintf(&builder, "# TYPE %s counter\n", currentMetric)
		}
		fmt.Fprintf(&builder, "%s%s %d\n", snapshot.Key.Name, snapshot.Key.Labels, snapshot.Count)
	}
	return builder.String()
}

type businessMetricSnapshot struct {
	Key   businessMetricKey
	Count uint64
}

func (m *BusinessMetrics) snapshot() []businessMetricSnapshot {
	m.mu.RLock()
	snapshots := make([]businessMetricSnapshot, 0, len(m.counters))
	for key, count := range m.counters {
		snapshots = append(snapshots, businessMetricSnapshot{Key: key, Count: count})
	}
	m.mu.RUnlock()

	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].Key.Name != snapshots[j].Key.Name {
			return snapshots[i].Key.Name < snapshots[j].Key.Name
		}
		return snapshots[i].Key.Labels < snapshots[j].Key.Labels
	})
	return snapshots
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, labels[key]))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func normalizeLabelValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
