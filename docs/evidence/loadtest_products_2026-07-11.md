# Product List Load Test

- URL: `http://127.0.0.1:8082/api/v1/products?page=1&page_size=20`
- Method: `GET`
- Requests: `2000`
- Concurrency: `20`
- Timeout: `5s`
- Started At: `2026-07-11T19:56:56+08:00`
- Duration: `1.428s`

| Metric | Value |
|---|---:|
| Total | 2000 |
| Success | 2000 |
| Failed Status | 0 |
| Errors | 0 |
| RPS | 1400.31 |
| Avg Latency | 14ms |
| P50 Latency | 14ms |
| P95 Latency | 23ms |
| P99 Latency | 28ms |
| Min Latency | 4ms |
| Max Latency | 40ms |

## Status Codes

| Status | Count |
|---:|---:|
| 200 | 2000 |
