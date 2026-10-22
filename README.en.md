<div align="right">
  <strong>Languages:</strong>
  <a href="README.md">简体中文</a> |
  <a href="README.en.md">English</a>
</div>

# Go Order & Inventory Consistency Service

[![CI](https://github.com/Yotoha0303/go-order-management-system/actions/workflows/ci.yml/badge.svg)](https://github.com/Yotoha0303/go-order-management-system/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8?logo=go&logoColor=white)](go.mod)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)](fronted/package.json)
[![MySQL](https://img.shields.io/badge/MySQL-8.4-4479A1?logo=mysql&logoColor=white)](compose.yml)
[![Redis](https://img.shields.io/badge/Redis-7.2-DC382D?logo=redis&logoColor=white)](compose.yml)
[![RabbitMQ](https://img.shields.io/badge/RabbitMQ-4.1-FF6600?logo=rabbitmq&logoColor=white)](compose.yml)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](compose.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

> A Go order-and-inventory service built as a backend portfolio piece. It focuses on transaction consistency, concurrent stock deduction, per-user idempotency, the order state machine, RabbitMQ-driven delayed cancellation, per-user data isolation, Redis caching, and baseline production engineering.

This is deliberately not a full e-commerce platform - no cart, payment gateway, logistics, or coupon subsystems. The point is to make the order-and-inventory critical path solid: the parts of backend work that get asked about most and that show engineering judgement best.

> 📖 **Entry points**
> - 📘 [Core business design](#core-business-design): order creation, idempotency, caching, business rules
> - 🧪 [Testing and verification](#testing-and-verification): local gate commands and coverage
> - 🚀 [Getting started](#getting-started): local, Docker, and cloud host
> - 📚 [Project documentation](#project-documentation): full design-doc index

---

## At a glance

| Dimension | Description |
| --- | --- |
| What it is | A Go service for order and inventory consistency |
| Good for | Go backend portfolio, practice on transactions/locks/idempotency/state machines |
| Critical path | Login → create order → idempotency check → transactional stock deduction → order + items + stock log → state transitions |
| Backend stack | Gin, GORM, MySQL transactions, row locks, conditional updates, RabbitMQ, Outbox, JWT, Redis cache-aside, Docker, CI |
| What it demonstrates | Business consistency, concurrency safety, data isolation, and engineering completeness - not CRUD volume |

### Live demo

The deployment is hosted in the cloud and can be opened directly. Demo credentials are available from the maintainer on request - **credentials are never committed to the repository**.

| Entry | URL | Notes |
| --- | --- | --- |
| Admin console | `https://go-order-management-system.vercel.app` | React SPA covering products, inventory, orders, operation logs |
| Backend readiness probe | `https://go-order-management-system.onrender.com/readyz` | Returns `{"code":0,"data":{"status":"ready"}}` when dependencies are ready |

> The cloud instance runs on a free tier. After a period of inactivity it cold-starts, so the first request may take tens of seconds.

## Core business flows

**Synchronous path** (order creation through state transitions):

```mermaid
flowchart LR
    A[Register / login] --> B[JWT auth]
    B --> C[Products and inventory setup]
    C --> D[Submit order]
    D --> E[Per-user idempotency check]
    E --> F[Begin transaction]
    F --> G[Lock and conditionally deduct stock]
    G --> H[Create order and order items]
    H --> I[Write stock log]
    I --> J{Order state machine}
    J -->|pay| K[Paid]
    K -->|finish| L[Finished]
    J -->|cancel| M[Cancelled, stock restored]
```

**Asynchronous timeout path** (auto-cancel unpaid orders):

```mermaid
flowchart LR
    N[Create order] -->|same transaction| O[order_timeout_outbox]
    O -->|poll and claim| P[Publish to delay queue]
    P --> Q[Ladder queues 5m / 15m / 30m]
    Q -->|TTL expires via DLX| R[Timeout consumer]
    R -->|still pending| S[Cancel order and restore stock]
    R -->|paid / cancelled| T[Idempotent no-op]
    R -->|repeated failures| U[Failed queue order.timeout.failed]
```

## Problems this project solves

| Problem | Approach | Why it matters |
| --- | --- | --- |
| Duplicate submissions create duplicate orders | `(user_id, idempotency_key)` unique index + SHA-256 request digest | Absorbs double-clicks and client retries without double-charging stock |
| Hot stock rows hit the database directly | Redis Lua atomic pre-deduction + reservation compensation | Adds a soft barrier before the MySQL transaction |
| Concurrent orders oversell | MySQL transaction + `SELECT ... FOR UPDATE` + `stock_quantity >= quantity` | Stock can never go negative |
| Multi-table writes drift apart | Idempotency record, order, items, inventory, stock log in one transaction | Any failure rolls the whole thing back |
| Cross-user order access | Reads and state changes match the current `user_id` | A user cannot touch another user's orders |
| Illegal state transitions | Service-layer state machine + conditional DAO updates | Only legal transitions between pending/paid/finished/cancelled |
| Deadlock risk on multi-item orders | Items processed in ascending `product_id` order inside the transaction | Removes circular wait caused by inconsistent lock ordering |
| Repeated product-detail reads | Redis cache-aside + invalidation on status change | Cuts repeated MySQL reads on hot products |
| Unpaid orders hold stock forever | Transactional Outbox + RabbitMQ TTL/DLX + idempotent consumer | Auto-cancels and restores stock 30 minutes after creation |
| Admin actions untraceable | Admin endpoints write `operation_logs`, correlated with the request ID | Supports audit and troubleshooting |
| Requests and business state invisible | `/metrics` exposes HTTP, order-creation, state-transition and Redis pre-deduction metrics | Baseline observability locally and in containers |
| Request chains hard to correlate | W3C `traceparent` compatible; access log emits `trace_id` / `span_id` | Ready for OpenTelemetry or a tracing backend |
| Slow SQL hard to locate | GORM + `slog` slow-query logging with configurable threshold and level | Entry point for EXPLAIN analysis and performance reports |
| Redis stock keys lost | Admin can view a Redis/MySQL drift report and rebuild Redis sellable stock from MySQL | Recovers pre-deduction quickly after a Redis restart |
| Redis/MySQL drift invisible | Background worker reconciles on a schedule and logs differences | Surfaces divergence between the soft layer and the source of truth |
| Multi-key Lua across Redis Cluster slots | Stock keys share a fixed hash tag `{stock}` | Multi-product pre-deduction lands in one slot |
| Different TTLs blocking each other in a queue | Fixed-TTL ladder queues instead of per-message `Expiration` | Removes RabbitMQ head-of-line blocking (see [ADR-002](docs/adr/002-rabbitmq-ladder-delay-queue-topology.md)) |

## Technology keyword matrix

> Diagrams cannot be indexed by search engines or code search, so this table supplies the **English keywords** for each core technique along with where it lives.

| Component | Keywords | Layer / form | Protocol & port | Role |
| :--- | :--- | :--- | :--- | :--- |
| **Distributed pre-deduction cache** | `Redis Lua Pre-deduct`, `Atomic DECRBY / INCRBY`, `Hash Tag {stock}` | Cache, `internal/bizcache` | RESP `6379` | Lua atomically checks stock and writes a reservation before any MySQL access |
| **Transactional deduction** | `MySQL Transaction`, `SELECT FOR UPDATE`, `Row-Level Lock`, `Conditional Update` | Storage, `internal/dao` | MySQL `3306` | Exclusive row lock plus conditional update; overselling is impossible |
| **Distributed unique ID** | `Snowflake ID Generator`, `Distributed ID`, `64-bit Int` | Library, `pkg/idgen` | In-process | 41-bit timestamp + 10-bit worker + 12-bit sequence; IDs allocated before the DB write |
| **Idempotency** | `Idempotency Key`, `SHA-256 Request Hash`, `MySQL RR Isolation` | Domain, `internal/service` | Digest + unique index | Composite unique index plus request hash; blocks double submits and replays |
| **Reliable delayed cancellation** | `Transactional Outbox`, `RabbitMQ TTL + DLX`, `Dead Letter Queue`, `Lease-based Claim` | Messaging, `internal/ordertimeout` | AMQP `5672` | Outbox row committed with the order; `FOR UPDATE SKIP LOCKED` + lease prevents duplicate publishing |
| **Ladder delay queues** | `Ladder Delay Queue`, `Queue-level x-message-ttl`, `Head-of-Line Blocking` | Messaging, `internal/ordertimeout` | AMQP `5672` | Three fixed-TTL queues (5m/15m/30m) eliminate per-message TTL blocking |
| **Three-state inventory, deadlock-free** | `Locked Quantity`, `Three-State Model`, `Deadlock Prevention`, `Ordered Locking` | Storage/model, `migrations/` | SQL DDL / ACID | Splits sellable and locked stock; ascending `product_id` ordering breaks the cycle |
| **Observability** | `Prometheus Metrics`, `W3C Traceparent`, `Structured Slog`, `Slow Query Log` | Observability, `/metrics` `/readyz` | HTTP `8082` | Golden-signal metrics, request/trace correlation, configurable slow-query threshold |
| **Cache penetration defence** | `Null Object Cache`, `TTL Jitter`, `Cache Penetration` | Cache, `internal/bizcache` | RESP `6379` | Null sentinel with short jittered TTL keeps invalid product IDs off MySQL |
| **Reconciliation and self-healing** | `Reconcile Worker`, `Auto-Fix`, `Redis/MySQL Drift` | Background, `internal/inventoryreconcile` | Scheduled task | Periodic Redis/MySQL comparison with optional auto-fix and Prometheus metrics |

## Features

### Users and authentication

- Register, log in, fetch the current user
- Change nickname and password
- HS256 JWT issuing, expiry validation, Bearer auth
- Every business endpoint requires JWT except health checks, registration, and login
- Order reads and writes are isolated by the current user

### Products

- Create a product
- List products and fetch product details
- Put a product on sale or take it off sale
- Prices use `price_fen` (integer cents) to avoid floating-point drift
- New products start off-sale so they cannot enter the order path without stock

### Inventory

- Initialise stock for a product
- Add stock
- Query stock
- Record every stock change
- Trace the origin of each change (init, manual add, order deduction, cancellation rollback)

### Orders

- Idempotent creation via `idempotency_key`
- Redis Lua pre-deduction; falls back to MySQL when Redis is unavailable or the key is missing
- Stock keys use a fixed `inventory:{stock}:...` hash tag to reduce cross-slot risk on Redis Cluster
- Deducts stock at creation time
- Rolls back the transaction when stock is insufficient
- List orders and fetch order detail
- Pay, finish, cancel
- Cancelling a pending order restores stock
- The state machine rejects illegal transitions
- Orders unpaid after 30 minutes are auto-cancelled and their stock restored via RabbitMQ
- Order creation and the timeout Outbox row commit in the same transaction; publishing uses Publisher Confirm, consumption uses manual ACK
- A successful pre-deduction writes a reservation; transaction failure or cancellation restores Redis, payment clears it
- Admins can request a Redis/MySQL drift report and rebuild Redis sellable stock from MySQL
- A background worker reconciles Redis against MySQL on a schedule, with optional auto-fix

### Admin console

- React console wired to products, inventory, stock logs, operation logs, orders, and user profile
- The landing page presents backend health, order summary, stock activity, and engineering highlights
- Walks through the full flow: product setup, stock init, ordering, payment, cancellation, stock rollback

## Tech stack

### Backend

- Go 1.25.7
- Gin + GORM
- MySQL 8.4
- Redis 7.2
- RabbitMQ 4.1
- JWT
- Goose migrations
- YAML + godotenv configuration

### Admin console

- React 19 + TypeScript + Vite
- TanStack Router / Query
- Axios + Zustand
- Tailwind CSS + Shadcn UI

### Engineering

- Docker + Docker Compose
- Makefile
- GitHub Actions
- golangci-lint
- go test / go test -race / go vet / go build

## Project layout

```text
.github/workflows/ci.yml  Continuous integration
cmd/                      Entry point
config/                   YAML loading, env overrides, validation
docs/                     Design docs, REST Client requests, evidence
internal/apperror/        Business errors and error-code mapping
internal/app/             Dependency wiring, HTTP server, graceful shutdown
internal/auth/            JWT issuing, validation, auth context
internal/bizcache/        Redis business caches
internal/dao/             Data access
internal/handler/         HTTP layer
internal/middleware/      Request ID, logging, timeout, recovery
internal/model/           GORM models
internal/ordertimeout/    RabbitMQ delay topology, Outbox publishing, timeout consumer
internal/request/         Request payloads and validation
internal/response/        Unified response envelope
internal/service/         Business rules, state machine, transactions
fronted/                  React admin console and API adapter
migrations/               Goose SQL migrations
pkg/database/             MySQL init and connection pool
pkg/redis/                Redis client init
router/                   Route registration
compose.yml               App, MySQL, Redis, RabbitMQ orchestration
Dockerfile                Multi-stage image build (local and cloud)
Makefile                  Dev, test, Docker, migration entry points
```

## Layering

| Layer | Responsibility |
| --- | --- |
| handler | HTTP handling, binding, error mapping, unified responses |
| service | Business rules, state transitions, transactions, cross-table work |
| dao | CRUD, conditional queries, conditional updates |
| model | Table mapping |
| request / response | Payload validation and response shapes |
| bizcache | Cache reads/writes, key rules, invalidation |
| apperror | Business errors, codes, messages |

Core rule: **handlers hold no business rules, services never build HTTP responses, DAOs never decide business state.**

## Core tables

| Table | Purpose | Key design |
| --- | --- | --- |
| `users` | Users | Unique `username`, password hash, status, soft delete |
| `roles` | Roles | Unique `role_name`; built-in `admin` and `user` |
| `user_roles` | User-role mapping | Unique `user_id`; one role per user today |
| `products` | Products | `price_fen` cents, status, off-sale by default |
| `product_inventories` | Stock | Unique `product_id`; `stock_quantity` sellable + `locked_quantity` reserved |
| `stock_logs` | Stock ledger | Before/change/after quantities, business type and ID |
| `orders` | Orders | `user_id` isolation, unique `order_no`, state machine |
| `order_items` | Order lines | Product name, price snapshot, quantity |
| `order_idempotency_keys` | Idempotency | Composite unique `(user_id, idempotency_key)` + `request_hash` |
| `order_timeout_outbox` | Timeout outbox | Written in the order transaction; records publish time, attempts, deadline |
| `order_timeout_dead_letters` | Dead letters | Persists messages routed to the failed queue for manual triage |
| `operation_logs` | Admin audit | Action, route, HTTP status, request ID, origin |

Full schema: [docs/table_design.md](docs/table_design.md)

## Core business design

| Topic | Document | Contents |
| --- | --- | --- |
| Order flow | [docs/order_flow.md](docs/order_flow.md) | Creation, cancellation, state transitions |
| Idempotency | [docs/idempotency.md](docs/idempotency.md) | Idempotency key, request hash, concurrent conflicts |
| Caching | [docs/cache_design.md](docs/cache_design.md) | Product-detail cache and penetration defence |
| Business rules | [docs/business_rules.md](docs/business_rules.md) | Full rule set |
| API list | [docs/api_list.md](docs/api_list.md) | REST endpoints |
| Decisions | [docs/adr](docs/adr) | ADR-001 inventory architecture / ADR-002 ladder queues / ADR-003 three-state inventory |

## Engineering notes

- HTTP server sets ReadTimeout, WriteTimeout, IdleTimeout, ReadHeaderTimeout, MaxHeaderBytes
- MySQL connection pool configured on init: MaxOpenConns, MaxIdleConns, ConnMaxLifetime, ConnMaxIdleTime
- Connectivity checked with PingContext at startup
- GORM slow-query logging above `mysql.slowThreshold`, level configurable
- Request ID, trace context, access log, recovery, and timeout middleware
- Access log emits request_id, trace_id, span_id; timeout responses also carry `traceparent`
- Product-detail caching degrades gracefully when Redis is down; **the degradation logs at ERROR level rather than silently**
- Fixed hash tag on stock keys; legacy keys can be refreshed via the admin rebuild endpoint
- RabbitMQ TTL + DLX; Publisher Confirm and manual ACK; duplicates absorbed by conditional state updates
- `/metrics` exposes Prometheus text metrics for HTTP, order creation, transitions, and Redis pre-deduction
- Single multi-stage Dockerfile shared by local Compose and cloud, running as non-root; configurable `GOPROXY`/`GOSUMDB`/`APK_MIRROR`
- Compose orchestrates app, MySQL, Redis, RabbitMQ with health-check-gated startup ordering
- Goose manages schema versions; the Makefile wraps dev, test, Docker and migration commands
- Built-in HTTP load test producing a Markdown report; a [local Docker read-path baseline](docs/evidence/loadtest_summary_2026-07-11.md) is committed (same-machine numbers only)
- CI runs go test, go test -race, go vet, golangci-lint, goose validate, go build; **a separate frontend job** covers `tsc -b`, eslint, vitest (browser mode) and build
- CI **asserts zero skips** when the integration switches are on: any `t.Skip` turns the pipeline red

## Getting started

### Prerequisites

- Go 1.25.7
- GNU Make
- Docker and Docker Compose
- Goose v3.27.1

```bash
go mod download
go install github.com/pressly/goose/v3/cmd/goose@v3.27.1
```

### Configuration

The app loads `.env` first, then [config.yml](config.yml). Environment variables override the connection settings that vary by environment.

```env
MYSQL_PASSWORD=your-password
DB_SLOW_THRESHOLD=200ms
DB_LOG_LEVEL=warn
JWT_SECRET=replace-with-at-least-32-random-characters
JWT_EXPIRE_HOURS=24
# Redis: use REDIS_URL for managed/cloud endpoints (rediss:// enables TLS),
# or the discrete REDIS_ADDR/REDIS_USERNAME/REDIS_PASSWORD/REDIS_TLS set.
REDIS_URL=
REDIS_ADDR=
REDIS_USERNAME=
REDIS_PASSWORD=
REDIS_TLS=
RABBITMQ_URL=amqp://order_app:order_dev_password@127.0.0.1:5672/
ORDER_TIMEOUT_DELAY=30m
INVENTORY_RECONCILE_ENABLED=true
INVENTORY_RECONCILE_INTERVAL=5m
INVENTORY_RECONCILE_TIMEOUT=3s
```

Never commit a real `.env`; copy [.env.example](.env.example) and edit it.

> `order_dev_password` in `RABBITMQ_URL` is a **local development default** (the Compose fallback), not a real credential. Override it via environment variables in the cloud.

### Run the backend locally

PowerShell:

```powershell
Copy-Item .env.example .env
$env:MYSQL_PASSWORD = "your-password"
$env:JWT_SECRET = "replace-with-at-least-32-random-characters"

make infra-up
make migrate-up
make run
```

Default address: `http://localhost:8082`

Health checks:

```bash
curl http://localhost:8082/ping
curl http://localhost:8082/live
curl http://localhost:8082/readyz
curl http://localhost:8082/metrics
```

### Run everything in Docker

```powershell
$env:MYSQL_PASSWORD = "your-password"
$env:JWT_SECRET = "replace-with-at-least-32-random-characters"

make docker-up
```

`make docker-up` builds the image, starts MySQL, Redis, RabbitMQ and the app, and runs migrations.

Image builds default to mainland-China-friendly mirrors: `GOPROXY=goproxy.cn` (with goproxy.io fallback), `GOSUMDB=sum.golang.google.cn`, Aliyun Alpine mirrors. Override for other regions:

```bash
GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org APK_MIRROR=dl-cdn.alpinelinux.org make docker-up
```

If base images are unreachable, set `GO_IMAGE` / `RUNTIME_IMAGE` in `.env`; see [docs/deploy.md](docs/deploy.md).

### One-shot cloud deployment

On Ubuntu/Debian hosts, the root script installs Docker, brings up the backend stack, builds the frontend, and reverse-proxies it through Nginx:

```bash
chmod +x deploy.sh
sudo ./deploy.sh
```

See [docs/deploy.md](docs/deploy.md). The script creates or fills `.env`, then serves the frontend, `/api`, `/readyz` and `/metrics`.

### Run the frontend locally

```powershell
Set-Location fronted
npm install
npm run dev
```

The console runs on `http://127.0.0.1:8880`; Vite proxies `/api` and `/ping` to `http://localhost:8082`.

## Common commands

| Command | Purpose |
| --- | --- |
| `make infra-up` | Start MySQL, Redis, RabbitMQ only, waiting for health |
| `make infra-down` | Stop the Compose project |
| `make migrate-validate` | Statically validate migration files |
| `make migrate-status` | Show migration status |
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Roll back the latest migration |
| `make docker-build` | Build the application image |
| `make docker-up` | Build and start app, MySQL, Redis, RabbitMQ |
| `make docker-down` | Stop and remove containers, keeping volumes |
| `make load-test` | HTTP load test against `LOADTEST_URL`, Markdown report |
| `make test` | Run all Go tests |
| `make test-service` | MySQL service integration tests |
| `make test-dao` | Critical DAO MySQL integration tests |
| `make test-migrations` | Apply and roll back migrations on an isolated database |
| `make test-redis` | Redis integration tests |
| `make test-order-timeout` | RabbitMQ + MySQL order-timeout end-to-end tests |
| `make test-all` | Plain tests plus MySQL service/DAO/migration and Redis tests |
| `make test-race` | Run tests under the race detector |
| `make check` | Format, module verification, vet, and tests |

## Testing and verification

Service tests truncate business tables in the connected database. **Always point `DB_NAME` at a dedicated test database** - never at development or production data.

Coverage today:

- Products, inventory, order creation, state machine, and error branches
- Concurrent ordering without overselling
- Multi-item transaction rollback
- Ascending `product_id` processing to reduce lock-ordering risk
- Idempotency: replay with the same key, conflict with a different payload, concurrent same-key, retry after rollback
- Concurrent order state: parallel pay, parallel cancel, pay-vs-cancel races
- Redis product-detail cache hits, invalidation, and degradation
- Redis Lua pre-deduction, insufficient-stock rejection, reservation restore and payment cleanup
- Order ID serialization contract: Snowflake keys are emitted as JSON strings to avoid frontend precision loss
- Redis config precedence: `REDIS_URL` vs discrete env vars, and fail-closed TLS parsing

Manual request files live in [docs/http](docs/http); the full flow is [docs/http/demo_flow.http](docs/http/demo_flow.http). Test plan: [docs/test_plan.md](docs/test_plan.md).

## Project documentation

| Document | Contents |
| --- | --- |
| [docs/api_list.md](docs/api_list.md) | API list |
| [docs/business_rules.md](docs/business_rules.md) | Business rules |
| [docs/table_design.md](docs/table_design.md) | Table design |
| [docs/order_flow.md](docs/order_flow.md) | Order creation, cancellation, transitions |
| [docs/idempotency.md](docs/idempotency.md) | Idempotency design |
| [docs/cache_design.md](docs/cache_design.md) | Redis product-detail caching |
| [docs/adr](docs/adr) | Architecture decision records |
| [docs/GAP台账.md](docs/GAP台账.md) | Quality-gap ledger (single source of truth) |
| [docs/test_plan.md](docs/test_plan.md) | Test plan |
| [docs/test_result.md](docs/test_result.md) | Test results |
| [docs/observability.md](docs/observability.md) | Metrics and alerting notes |
| [docs/performance.md](docs/performance.md) | Load testing and slow-SQL analysis |
| [docs/deploy.md](docs/deploy.md) | Cloud deployment script |
| [docs/project_evolution.md](docs/project_evolution.md) | Roadmap |
| [docs/interview_guide.md](docs/interview_guide.md) | Resume notes and interview walkthrough |
| [docs/evidence](docs/evidence) | Runtime, test and business evidence |

## Boundaries and roadmap

The focus is backend fundamentals; the project is not packaged as a full e-commerce platform.

Possible next steps:

- Order-list filters by status and time range
- Broader handler boundary tests and DAO integration coverage
- Alerting evidence, structured log field conventions, and tracing
- Expiry/cleanup policy for idempotency records
- Advanced operation-log filtering, export, and retention
- Coverage thresholds in CI (reports exist today; no floor is enforced yet)

## License

[Apache License 2.0](LICENSE). The `fronted/` subdirectory derives from an upstream template and keeps its own [MIT license](fronted/LICENSE).
