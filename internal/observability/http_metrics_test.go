package observability

import (
	"strings"
	"testing"
	"time"
)

func TestHTTPMetricsRenderPrometheus(t *testing.T) {
	metrics := NewHTTPMetrics()
	metrics.Record("GET", "/ping", 200, 10*time.Millisecond)
	metrics.Record("GET", "/ping", 200, 15*time.Millisecond)
	metrics.Record("POST", "", 404, time.Millisecond)

	output := metrics.RenderPrometheus()

	for _, want := range []string{
		`# TYPE app_http_requests_total counter`,
		`app_http_requests_total{method="GET",route="/ping",status="200"} 2`,
		`app_http_requests_total{method="POST",route="unmatched",status="404"} 1`,
		`app_http_request_duration_seconds_sum{method="GET",route="/ping",status="200"} 0.025000`,
		`app_http_request_duration_seconds_count{method="GET",route="/ping",status="200"} 2`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", want, output)
		}
	}
}
