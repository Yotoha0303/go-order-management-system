# go-order 读链路压测 + pprof 基线

- URL: `http://127.0.0.1:8082/api/v1/products?page=1&page_size=20`
- Method: `GET`
- Requests: `100000`
- Concurrency: `50`
- Timeout: `10s`
- Started At: `2026-10-23T17:56:19+08:00`
- Duration: `32.538s`

| Metric | Value |
|---|---:|
| Total | 100000 |
| Success | 100000 |
| Failed Status | 0 |
| Errors | 0 |
| RPS | 3073.35 |
| Avg Latency | 16ms |
| P50 Latency | 16ms |
| P95 Latency | 19ms |
| P99 Latency | 23ms |
| Min Latency | 1ms |
| Max Latency | 43ms |

## Status Codes

| Status | Count |
|---:|---:|
| 200 | 100000 |
