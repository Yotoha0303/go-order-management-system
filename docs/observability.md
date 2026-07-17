# 可观测性设计

## 1. 当前范围

项目当前提供六类基础可观测能力：

- `X-Request-ID`：每个请求生成或透传请求 ID。
- `traceparent`：兼容 W3C Trace Context，生成或透传 `trace_id` 并创建服务端 `span_id`。
- Access Log：记录 method、path、route、status、latency、client_ip、body_size、request_id、trace_id 和 span_id。
- `/metrics`：输出 Prometheus 文本格式指标。
- GORM 慢 SQL 日志：默认记录超过阈值的 SQL 和数据库错误。
- Redis 库存对账日志：周期记录 Redis/MySQL 库存差异情况。

当前没有接入 OpenTelemetry Collector 或完整追踪后端，也没有提供完整告警平台。本文只记录项目内已经实现的指标和可落地告警建议。

## 2. Trace Context

全局中间件会处理请求头中的 `traceparent`：

- 请求已携带合法 `traceparent` 时，复用其中的 `trace_id`，并为本服务生成新的 `span_id`。
- 请求未携带或格式非法时，生成新的 `trace_id` 和 `span_id`。
- 响应头写回新的 `traceparent`。
- Access Log 输出 `trace_id` 和 `span_id`，便于和 request_id 一起定位一次请求。
- Timeout 外层处理器也会补充 `traceparent`，保证超时响应可关联日志。

当前只做 trace context 透传和日志关联，不采集跨服务 span，也不导出到 Jaeger、Tempo 或 OpenTelemetry Collector。

## 3. HTTP 指标

| 指标 | 类型 | 标签 | 说明 |
|---|---|---|---|
| `app_http_requests_total` | counter | method, route, status | HTTP 请求数量 |
| `app_http_request_duration_seconds_sum` | counter | method, route, status | HTTP 请求耗时总和 |
| `app_http_request_duration_seconds_count` | counter | method, route, status | HTTP 请求耗时样本数 |

`route` 使用 Gin 路由模板，例如 `/api/v1/orders/:id`，避免真实订单 ID 或商品 ID 进入 label 造成高基数。

## 4. 业务指标

| 指标 | 类型 | 标签 | 说明 |
|---|---|---|---|
| `app_order_create_total` | counter | result | 创建订单结果 |
| `app_order_state_transition_total` | counter | action, result | 订单支付、完成、取消和超时取消结果 |
| `app_redis_inventory_prededuct_total` | counter | result | Redis Lua 库存预扣结果 |
| `app_redis_inventory_reservation_total` | counter | action, result | Redis reservation 回补、确认结果 |
| `app_redis_inventory_sync_total` | counter | result | 库存初始化/入库后同步 Redis 可售库存结果 |

订单创建 `result` 使用有限枚举，例如：

- `success`
- `idempotent_replay`
- `insufficient_stock`
- `idempotency_conflict`
- `product_off_sale`
- `error`

Redis 预扣 `result` 使用有限枚举，例如：

- `applied`
- `insufficient`
- `skipped_missing_key`
- `error`
- `disabled`

## 5. 慢 SQL 日志

MySQL 初始化时会为 GORM 注入 `slog` logger，默认只输出慢查询和数据库错误，避免普通查询刷屏。

配置项：

| 配置 | 默认值 | 环境变量 | 说明 |
|---|---|---|---|
| `mysql.slowThreshold` | `200ms` | `DB_SLOW_THRESHOLD` | 超过该耗时的 SQL 会记录为 `gorm slow query` |
| `mysql.logLevel` | `warn` | `DB_LOG_LEVEL` | 可选 `silent`、`error`、`warn`、`info` |

慢查询日志字段包括：

- `elapsed_ms`：SQL 执行耗时；
- `threshold_ms`：慢查询阈值；
- `rows`：影响或返回行数；
- `sql`：GORM 输出的 SQL。

数据库错误会记录为 `gorm query error`，常规的 `record not found` 不按错误日志输出。

## 6. Redis 库存对账日志

自动对账 worker 根据 `inventoryReconcile` 配置周期执行 Redis/MySQL 库存差异检查：

- 无差异：记录 `inventory Redis reconcile finished`。
- 有差异：记录 `inventory Redis reconcile found differences`，字段包含 `checked_count` 和 `diff_count`。
- 对账失败：记录 `inventory Redis reconcile failed`，不影响 HTTP 服务。

该 worker 不自动重建 Redis。差异修复仍需要管理员查看差异报告后手动触发重建。

## 7. 告警建议

以下规则是运行建议，是否启用取决于部署环境中的 Prometheus 或其他采集系统。

| 场景 | 建议表达式 | 处理建议 |
|---|---|---|
| 5xx 错误增多 | `sum(rate(app_http_requests_total{status=~"5.."}[5m])) > 0` | 查看 access log 的 request_id，定位异常接口 |
| 下单错误率升高 | `sum(rate(app_order_create_total{result!="success",result!="idempotent_replay"}[5m])) / sum(rate(app_order_create_total[5m])) > 0.2` | 区分库存不足、幂等冲突和系统错误 |
| Redis 预扣频繁失败 | `sum(rate(app_redis_inventory_prededuct_total{result="error"}[5m])) > 0` | 检查 Redis 连通性和 Lua 执行错误 |
| Redis key 缺失过多 | `sum(rate(app_redis_inventory_prededuct_total{result="skipped_missing_key"}[10m])) > 10` | 先查看 Redis/MySQL 库存差异报告，再按 MySQL 当前库存重建 Redis 可售库存 |
| reservation 回补失败 | `sum(rate(app_redis_inventory_reservation_total{result="error"}[5m])) > 0` | 检查 Redis 状态，并用 MySQL 库存作为事实源重建 Redis 可售库存 |
| 慢 SQL 增多 | 查看应用日志中的 `gorm slow query` | 先按 `sql` 定位接口和表，再用 `EXPLAIN` 检查索引和扫描行数 |
| Redis/MySQL 库存差异 | 查看应用日志中的 `inventory Redis reconcile found differences` | 调用管理员差异报告接口确认明细，再决定是否重建 Redis 可售库存 |

## 8. 当前边界

- 已提供 `traceparent` 透传和 access log 关联，但暂未接入 OpenTelemetry 或分布式追踪后端。
- 已提供慢 SQL 日志入口及本地 Docker 读链路压测和 EXPLAIN 报告；写链路和长期运行仍未覆盖。
- 暂未提供 Prometheus 配置文件或线上告警截图证据。
