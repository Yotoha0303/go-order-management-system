package loadtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Config struct {
	Method      string
	URL         string
	Body        []byte
	Headers     map[string]string
	Requests    int
	Concurrency int
	Timeout     time.Duration
}

type Sample struct {
	StatusCode int
	Latency    time.Duration
	Err        error
}

type Report struct {
	Config       Config
	StartedAt    time.Time
	Duration     time.Duration
	Samples      []Sample
	StatusCounts map[int]int
}

type Summary struct {
	Total        int
	Success      int
	FailedStatus int
	Errors       int
	RPS          float64
	Min          time.Duration
	Max          time.Duration
	Average      time.Duration
	P50          time.Duration
	P95          time.Duration
	P99          time.Duration
	StatusCounts map[int]int
	Duration     time.Duration
	StartedAt    time.Time
}

func Run(ctx context.Context, cfg Config) (Report, error) {
	if err := validateConfig(cfg); err != nil {
		return Report{}, err
	}
	if cfg.Method == "" {
		cfg.Method = http.MethodGet
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}

	client := &http.Client{Timeout: cfg.Timeout}
	jobs := make(chan struct{})
	results := make(chan Sample, cfg.Requests)
	startedAt := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				results <- doRequest(ctx, client, cfg)
			}
		}()
	}

enqueue:
	for i := 0; i < cfg.Requests; i++ {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- struct{}{}:
		}
	}
	close(jobs)
	wg.Wait()
	close(results)

	report := Report{
		Config:       cfg,
		StartedAt:    startedAt,
		Duration:     time.Since(startedAt),
		StatusCounts: make(map[int]int),
	}
	for sample := range results {
		report.Samples = append(report.Samples, sample)
		if sample.StatusCode > 0 {
			report.StatusCounts[sample.StatusCode]++
		}
	}
	return report, nil
}

func validateConfig(cfg Config) error {
	if cfg.URL == "" {
		return errors.New("url is required")
	}
	if cfg.Requests <= 0 {
		return errors.New("requests must be positive")
	}
	if cfg.Concurrency <= 0 {
		return errors.New("concurrency must be positive")
	}
	if cfg.Concurrency > cfg.Requests {
		return errors.New("concurrency must not exceed requests")
	}
	return nil
}

func doRequest(ctx context.Context, client *http.Client, cfg Config) Sample {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, cfg.Method, cfg.URL, bytes.NewReader(cfg.Body))
	if err != nil {
		return Sample{Latency: time.Since(start), Err: err}
	}
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Sample{Latency: time.Since(start), Err: err}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return Sample{
		StatusCode: resp.StatusCode,
		Latency:    time.Since(start),
	}
}

func (r Report) Summary() Summary {
	summary := Summary{
		Total:        len(r.Samples),
		StatusCounts: make(map[int]int, len(r.StatusCounts)),
		Duration:     r.Duration,
		StartedAt:    r.StartedAt,
	}
	for status, count := range r.StatusCounts {
		summary.StatusCounts[status] = count
	}
	if summary.Total == 0 {
		return summary
	}

	latencies := make([]time.Duration, 0, len(r.Samples))
	var sum time.Duration
	for _, sample := range r.Samples {
		if sample.Err != nil {
			summary.Errors++
			continue
		}
		if sample.StatusCode >= 200 && sample.StatusCode < 400 {
			summary.Success++
		} else {
			summary.FailedStatus++
		}
		latencies = append(latencies, sample.Latency)
		sum += sample.Latency
	}
	if r.Duration > 0 {
		summary.RPS = float64(summary.Total) / r.Duration.Seconds()
	}
	if len(latencies) == 0 {
		return summary
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	summary.Min = latencies[0]
	summary.Max = latencies[len(latencies)-1]
	summary.Average = sum / time.Duration(len(latencies))
	summary.P50 = percentile(latencies, 0.50)
	summary.P95 = percentile(latencies, 0.95)
	summary.P99 = percentile(latencies, 0.99)
	return summary
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

func RenderMarkdown(title string, cfg Config, summary Summary) string {
	if title == "" {
		title = "HTTP Load Test Report"
	}
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "# %s\n\n", title)
	fmt.Fprintf(&buffer, "- URL: `%s`\n", cfg.URL)
	fmt.Fprintf(&buffer, "- Method: `%s`\n", cfg.Method)
	fmt.Fprintf(&buffer, "- Requests: `%d`\n", cfg.Requests)
	fmt.Fprintf(&buffer, "- Concurrency: `%d`\n", cfg.Concurrency)
	fmt.Fprintf(&buffer, "- Timeout: `%s`\n", cfg.Timeout)
	fmt.Fprintf(&buffer, "- Started At: `%s`\n", summary.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(&buffer, "- Duration: `%s`\n\n", summary.Duration.Round(time.Millisecond))

	buffer.WriteString("| Metric | Value |\n")
	buffer.WriteString("|---|---:|\n")
	fmt.Fprintf(&buffer, "| Total | %d |\n", summary.Total)
	fmt.Fprintf(&buffer, "| Success | %d |\n", summary.Success)
	fmt.Fprintf(&buffer, "| Failed Status | %d |\n", summary.FailedStatus)
	fmt.Fprintf(&buffer, "| Errors | %d |\n", summary.Errors)
	fmt.Fprintf(&buffer, "| RPS | %.2f |\n", summary.RPS)
	fmt.Fprintf(&buffer, "| Avg Latency | %s |\n", summary.Average.Round(time.Millisecond))
	fmt.Fprintf(&buffer, "| P50 Latency | %s |\n", summary.P50.Round(time.Millisecond))
	fmt.Fprintf(&buffer, "| P95 Latency | %s |\n", summary.P95.Round(time.Millisecond))
	fmt.Fprintf(&buffer, "| P99 Latency | %s |\n", summary.P99.Round(time.Millisecond))
	fmt.Fprintf(&buffer, "| Min Latency | %s |\n", summary.Min.Round(time.Millisecond))
	fmt.Fprintf(&buffer, "| Max Latency | %s |\n\n", summary.Max.Round(time.Millisecond))

	buffer.WriteString("## Status Codes\n\n")
	buffer.WriteString("| Status | Count |\n")
	buffer.WriteString("|---:|---:|\n")
	statuses := make([]int, 0, len(summary.StatusCounts))
	for status := range summary.StatusCounts {
		statuses = append(statuses, status)
	}
	sort.Ints(statuses)
	for _, status := range statuses {
		fmt.Fprintf(&buffer, "| %d | %d |\n", status, summary.StatusCounts[status])
	}
	return buffer.String()
}
