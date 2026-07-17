package observability

import (
	"strings"
	"testing"
)

func TestBusinessMetricsRenderPrometheus(t *testing.T) {
	metrics := NewBusinessMetrics()
	metrics.RecordOrderCreate("success")
	metrics.RecordOrderCreate("success")
	metrics.RecordOrderCreate("insufficient_stock")
	metrics.RecordOrderStateTransition("pay", "success")
	metrics.RecordRedisInventoryPreDeduct("applied")
	metrics.RecordRedisInventoryReservation("release", "success")
	metrics.RecordRedisInventorySync("error")

	output := metrics.RenderPrometheus()
	for _, want := range []string{
		`app_order_create_total{result="success"} 2`,
		`app_order_create_total{result="insufficient_stock"} 1`,
		`app_order_state_transition_total{action="pay",result="success"} 1`,
		`app_redis_inventory_prededuct_total{result="applied"} 1`,
		`app_redis_inventory_reservation_total{action="release",result="success"} 1`,
		`app_redis_inventory_sync_total{result="error"} 1`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", want, output)
		}
	}
}
