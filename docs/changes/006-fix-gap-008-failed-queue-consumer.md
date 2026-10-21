# 变更记录：006-fix-gap-008-failed-queue-consumer

- **变更编号**：006
- **创建时间**：2026-10-21T12:59:00Z (UTC)
- **完成时间**：2026-10-21T13:06:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-008 (P1: RabbitMQ 死信失败队列 `order.timeout.failed` 缺乏消费者与告警，形成消息堆积黑洞，Issue #9)
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在订单超时关单系统中，RabbitMQ 拓扑定义了三层队列：
1. `order.timeout.delay`: 延迟缓冲队列，消息具有 TTL，超时后通过 DLX 路由至取消队列；
2. `order.timeout.cancel`: 超时取消消费队列，Worker 消费该队列执行 `CancelExpiredOrder`；
3. `order.timeout.failed`: 失败死信队列。

在历史基准实现中，存在如下严重的稳定性与可观测性缺陷（GAP-008）：
1. **死信队列无消费者成为黑洞**：
   - `order.timeout.failed` 被声明并绑定了路由键，但系统中**没有任何消费者监听该队列**；
   - 当消息在 `cancelQueue` 中多次取消失败或格式畸变被拒时，消息积压在 `failedQueue` 中无声无息，形成不可见的黑洞。
2. **毒丸死循环风险与无限重试**：
   - 历史 `consumeLoop` 在处理 `cancelQueue` 失败时，无条件调用 `delivery.Nack(false, true)`（requeue = true）；
   - 若遇到无法解析的格式畸变毒丸或特定订单取消崩溃，消息将在 `cancelQueue` 中无限循环重试，耗尽 CPU 并刷屏日志。
3. **缺乏结构化告警与持久化排障支撑**：
   - 死信消息缺乏持久化落库表（`order_timeout_dead_letters`）与结构化日志告警（`ORDER_TIMEOUT_DLQ_ALERT`），运维与开发团队无法追溯事故消息、根因与上下文 Payload，更无法提供人工干预入口。

基于 SRE 规范设计，本次变更：
- 新增 `order_timeout_dead_letters` 数据库持久化表及对应 Model 与 DAO；
- 在 `internal/observability/business_metrics.go` 中扩展死信业务指标 `app_order_timeout_dead_letter_total`；
- 在 `ordertimeout.Worker` 中：
  - 在已有的 `consumeLoop` 中利用 `x-death` 提取重试次数，当达到重试上限（`defaultMaxCancelRetries = 3`）时调用 `Nack(false, false)` 将消息流转至 `failedQueue`；
  - 启动独立的 `consumeFailedLoop` 监听 `failedQueue`；
  - 宽容解析 Payload，落库持久化（失败时触发 Emergency Fallback 日志），打印 `ORDER_TIMEOUT_DLQ_ALERT` 结构化日志，上报指标；
  - **绝不调用 Nack(requeue=true)**，终态无条件调用 `delivery.Ack(false)`，彻底杜绝毒丸死循环并清理队列积压。

---

## 2. 修改与新增文件清单

1. `migrations/00015_create_order_timeout_dead_letters.sql` (新建):
   - 创建 `order_timeout_dead_letters` 失败消息表，包含 `message_id`, `event_id`, `order_id`, `user_id`, `retry_count`, `reason`, `payload`, `error_detail`, `status` 字段及索引；
   - 提供标准的 Goose Up 与 Down 迁移。
2. `internal/model/order_timeout_dead_letter.go` (新建):
   - 定义 `OrderTimeoutDeadLetter` GORM 实体模型，映射 `order_timeout_dead_letters` 表。
3. `internal/dao/order_timeout_dead_letter_dao.go` (新建):
   - 提供 `CreateOrderTimeoutDeadLetter(ctx context.Context, db *gorm.DB, dl *model.OrderTimeoutDeadLetter) error` 持久化接口。
4. `internal/dao/order_timeout_dead_letter_dao_test.go` (新建):
   - 验证 DAO 参数边界防御与基于 Mock Connector 的 INSERT SQL 校验。
5. `internal/observability/business_metrics.go` & `business_metrics_test.go`:
   - 扩展 `RecordOrderTimeoutDeadLetter(reason, handled string)` 方法与单元测试断言。
6. `internal/ordertimeout/worker.go`:
   - 引入 `defaultMaxCancelRetries = 3` 常量；
   - 声明 `DeadLetterMetrics` 接口并提供 `SetMetrics` 与 `NewWorkerWithMetrics`；
   - 在 `runSession` 中新增打开 `failedQueue` 消费 channel 并并发启动 `consumeFailedLoop`；
   - 实现 `consumeFailedLoop` 与 `handleFailedDelivery`，保证宽容解析、持久化落库、Fallback 保护、结构化告警日志输出、指标记录以及无条件 `Ack(false)`；
   - 优化 `consumeLoop`，支持基于 `x-death` 检测重试达到阈值时安全 `Nack(false, false)` 流转至 DLX。
7. `internal/ordertimeout/dead_letter_test.go` (新建):
   - 编写死信消费者的针对性测试：
     - 正常死信消费落库、结构化告警日志与终态 Ack 语义（`TestConsumeFailedLoop_SuccessDeadLetter`）；
     - 格式畸变毒丸宽容解析与无死循环 Ack（`TestConsumeFailedLoop_MalformedPayloadPoisonPill`）；
     - 数据库故障下的 Emergency Fallback 保护与日志断言（`TestConsumeFailedLoop_DBFailureEmergencyFallback`）；
     - `cancelQueue` 中重试达到上限流转至 DLX 的针对性单测（`TestConsumeLoop_CancelQueueRetries_ExceededRoutesToDLX`）；
     - `cancelQueue` 中未达到上限时的正常 requeue 单测（`TestConsumeLoop_CancelQueueRetries_BelowThresholdRequeues`）；
     - 覆盖 `x-death` 解析的 5 种子测试（`TestExtractDeathCount`）。
8. `internal/app/deps.go`:
   - 装配 `orderTimeoutWorker` 时注入 `metrics.Business`。

---

## 3. 实测验证命令与真实 stdout

### 3.1 针对性模块测试（带竞态检测）

- **执行命令**：
```bash
go test -v -race ./internal/ordertimeout/... ./internal/dao/... ./internal/observability/...
```
- **执行结果**：`PASS (Exit Code 0)`
- **真实 stdout 关键摘要**：
```text
=== RUN   TestConsumeFailedLoop_SuccessDeadLetter
--- PASS: TestConsumeFailedLoop_SuccessDeadLetter (0.01s)
=== RUN   TestConsumeFailedLoop_MalformedPayloadPoisonPill
--- PASS: TestConsumeFailedLoop_MalformedPayloadPoisonPill (0.00s)
=== RUN   TestConsumeFailedLoop_DBFailureEmergencyFallback
--- PASS: TestConsumeFailedLoop_DBFailureEmergencyFallback (0.00s)
=== RUN   TestConsumeLoop_CancelQueueRetries_ExceededRoutesToDLX
--- PASS: TestConsumeLoop_CancelQueueRetries_ExceededRoutesToDLX (0.00s)
=== RUN   TestConsumeLoop_CancelQueueRetries_BelowThresholdRequeues
--- PASS: TestConsumeLoop_CancelQueueRetries_BelowThresholdRequeues (0.00s)
=== RUN   TestExtractDeathCount
--- PASS: TestExtractDeathCount (0.00s)
=== RUN   TestCalculateExponentialBackoff
--- PASS: TestCalculateExponentialBackoff (0.00s)
=== RUN   TestWorker_PublishBatch_HandlesFailureAndCapping
--- PASS: TestWorker_PublishBatch_HandlesFailureAndCapping (0.00s)
PASS
ok  	go-order-management-system/internal/ordertimeout	3.300s
=== RUN   TestCreateOrderTimeoutDeadLetter
--- PASS: TestCreateOrderTimeoutDeadLetter (0.00s)
=== RUN   TestClaimPendingOrderTimeoutOutbox_UnitMock
--- PASS: TestClaimPendingOrderTimeoutOutbox_UnitMock (0.00s)
=== RUN   TestClaimPendingOrderTimeoutOutbox_Validation
--- PASS: TestClaimPendingOrderTimeoutOutbox_Validation (0.00s)
PASS
ok  	go-order-management-system/internal/dao	1.692s
=== RUN   TestBusinessMetricsRenderPrometheus
--- PASS: TestBusinessMetricsRenderPrometheus (0.00s)
=== RUN   TestHTTPMetricsRenderPrometheus
--- PASS: TestHTTPMetricsRenderPrometheus (0.00s)
PASS
ok  	go-order-management-system/internal/observability	1.496s
```

### 3.2 静态代码检查

- **执行命令**：
```bash
go vet ./...
```
- **执行结果**：`PASS (Exit Code 0)`，全仓无任何告警。

### 3.3 全仓单元测试与竞态检测

- **执行命令**：
```bash
go test -v -race -count=1 ./...
```
- **执行结果**：`PASS (Exit Code 0)`，全仓所有包（config, dao, handler, middleware, observability, ordertimeout, router, service, migrations, database, idgen）单元测试全绿，零竞态。
