package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go-order-management-system/internal/loadtest"
)

type headerFlags map[string]string

func (h headerFlags) String() string {
	values := make([]string, 0, len(h))
	for key, value := range h {
		values = append(values, key+": "+value)
	}
	return strings.Join(values, ", ")
}

func (h headerFlags) Set(value string) error {
	key, headerValue, ok := strings.Cut(value, ":")
	if !ok {
		return fmt.Errorf("header must use Name: value format")
	}
	key = strings.TrimSpace(key)
	headerValue = strings.TrimSpace(headerValue)
	if key == "" {
		return fmt.Errorf("header name is required")
	}
	h[key] = headerValue
	return nil
}

func main() {
	headers := headerFlags{}
	url := flag.String("url", "", "target URL")
	method := flag.String("method", http.MethodGet, "HTTP method")
	body := flag.String("body", "", "request body")
	requests := flag.Int("requests", 100, "total request count")
	concurrency := flag.Int("concurrency", 10, "concurrent workers")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request timeout")
	title := flag.String("title", "HTTP Load Test Report", "markdown report title")
	output := flag.String("output", "", "write markdown report to file; stdout when empty")
	flag.Var(headers, "header", "request header, repeatable, format: Name: value")
	flag.Parse()

	report, err := loadtest.Run(context.Background(), loadtest.Config{
		Method:      strings.ToUpper(strings.TrimSpace(*method)),
		URL:         strings.TrimSpace(*url),
		Body:        []byte(*body),
		Headers:     headers,
		Requests:    *requests,
		Concurrency: *concurrency,
		Timeout:     *timeout,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "load test failed:", err)
		os.Exit(1)
	}

	markdown := loadtest.RenderMarkdown(*title, report.Config, report.Summary())
	if *output == "" {
		fmt.Print(markdown)
		return
	}
	if err := os.WriteFile(*output, []byte(markdown), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write report failed:", err)
		os.Exit(1)
	}
	fmt.Println("report written:", *output)
}
