package loadtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunCollectsLatencyAndStatus(t *testing.T) {
	var calls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&calls, 1)
		if got := r.Header.Get("X-Test"); got != "load" {
			t.Fatalf("header=%q", got)
		}
		time.Sleep(time.Millisecond)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	report, err := Run(context.Background(), Config{
		Method:      http.MethodPost,
		URL:         server.URL,
		Body:        []byte(`{"ok":true}`),
		Headers:     map[string]string{"X-Test": "load"},
		Requests:    8,
		Concurrency: 2,
		Timeout:     time.Second,
	})
	if err != nil {
		t.Fatalf("run load test: %v", err)
	}
	if calls != 8 {
		t.Fatalf("calls=%d", calls)
	}

	summary := report.Summary()
	if summary.Total != 8 || summary.Success != 8 || summary.Errors != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.StatusCounts[http.StatusAccepted] != 8 {
		t.Fatalf("status counts=%v", summary.StatusCounts)
	}
	if summary.P95 <= 0 || summary.RPS <= 0 {
		t.Fatalf("latency/rps not recorded: %+v", summary)
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	_, err := Run(context.Background(), Config{URL: "http://example.test", Requests: 1, Concurrency: 2})
	if err == nil {
		t.Fatal("expected invalid config error")
	}
}

func TestRenderMarkdown(t *testing.T) {
	report := Report{
		Config:    Config{Method: http.MethodGet, URL: "http://example.test/ping", Requests: 1, Concurrency: 1, Timeout: time.Second},
		StartedAt: time.Unix(100, 0),
		Duration:  time.Second,
		Samples: []Sample{{
			StatusCode: http.StatusOK,
			Latency:    10 * time.Millisecond,
		}},
		StatusCounts: map[int]int{http.StatusOK: 1},
	}
	output := RenderMarkdown("Report", report.Config, report.Summary())
	for _, want := range []string{"# Report", "| Success | 1 |", "| 200 | 1 |"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, output)
		}
	}
}
