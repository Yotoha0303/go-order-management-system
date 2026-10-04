# Go Order & Inventory Consistency Management Service

> A Go order and inventory consistency service focused on transaction consistency, concurrent inventory deduction, idempotent order creation, order state machines, RabbitMQ delayed cancellation, Redis caching, user data isolation, and engineering delivery.

[中文 README](README.md)

## Project Positioning

This is a focused backend engineering project rather than a complete e-commerce platform. The goal is to make the order/inventory critical path technically solid instead of adding peripheral features such as shopping carts, payment, logistics, or coupons.

## Core Flow

```text
User Login → JWT → Product & Inventory → Create Order → Idempotency → MySQL Transaction → Inventory Lock/Conditional Deduction → Order/Stock Records → State Machine
```

## Key Engineering Problems

| Problem | Solution |
| --- | --- |
| Duplicate submission | `(user_id, idempotency_key)` unique index + SHA-256 request hash |
| Hot inventory writes | Redis Lua atomic pre-deduction + reservation compensation |
| Overselling | MySQL transaction + `SELECT ... FOR UPDATE` + conditional update |
| Multi-table inconsistency | Idempotency, order, items, inventory and stock logs in one transaction |
| Unauthorized order access | Current `user_id` enforced on reads and state changes |
| Illegal state transitions | Service-layer state machine + conditional DAO updates |
| Multi-product deadlock risk | Ascending `product_id` lock order |
| Unpaid order timeout | Transactional Outbox + RabbitMQ TTL/DLX + idempotent consumer |
| Redis/MySQL drift | Periodic reconciliation and rebuild capability |

## Technology Stack

- Go 1.25.7 / Gin / GORM
- MySQL 8.4 / Redis 7.2 / RabbitMQ 4.1
- JWT / Goose / YAML / godotenv
- React 19 / TypeScript / Vite / TanStack Router & Query / Zustand / Tailwind CSS
- Docker / Docker Compose / Makefile / GitHub Actions / golangci-lint

## Quick Start

```bash
go mod download
go install github.com/pressly/goose/v3/cmd/goose@v3.27.1
make infra-up
make migrate-up
make run
```

Health endpoints:

```bash
curl http://localhost:8082/ping
curl http://localhost:8082/live
curl http://localhost:8082/readyz
curl http://localhost:8082/metrics
```

## Testing & Observability

The test suite covers concurrent order creation, oversell prevention, transaction rollback, idempotency races, payment/cancellation races, Redis Lua reservation/compensation, cache degradation, and RabbitMQ timeout flow.

Runtime capabilities include HTTP/business metrics, Request ID, trace context, slow SQL logging, bounded HTTP load tests, and Markdown performance reports. Measurements are local engineering baselines, not production capacity guarantees.

## Documentation

See the Chinese README and `docs/` for API, business rules, table design, order flow, idempotency, cache design, testing, observability, performance, deployment, and evidence records.

## Scope Boundary

The project intentionally focuses on the backend order/inventory critical path and does not claim to be a complete e-commerce platform.
