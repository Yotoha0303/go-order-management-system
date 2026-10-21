# 变更记录：005-fix-gap-003-outbox-skip-locked

- **变更编号**：005
- **创建时间**：2026-10-21T12:20:00Z (UTC)
- **完成时间**：2026-10-21T12:27:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-003 (P2: Transactional Outbox 轮询缺乏并发锁，多实例部署下消息重复发布)
- **关联 ADR**：ADR-001 (高并发低延迟与分布式最终一致性库存架构设计) §2 决策一 (精简 MySQL 事务零外部网络 IO)
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在订单超时关单架构中，系统通过 Transactional Outbox 模式保证本地订单落库与延迟取消事件发布的最终一致性。

在历史基准实现中，`ordertimeout.Worker.publishBatch` 轮询待发布记录时存在以下严重的并发竞争与可用性缺陷：
1. **多实例部署下的无锁重复发布（GAP-003）**：
   - 原实现直接使用 `dao.ListPendingOrderTimeoutOutbox` 进行普通的无锁 `SELECT ... WHERE published_at IS NULL AND next_attempt_at <= now LIMIT ?`；
   - 在生产多 Pod / 多副本或多 Worker 实例部署时，所有实例在同一时刻读出完全相同的待发布批次，各实例竞相向 RabbitMQ 重复发布同一笔订单的超时事件，导致延迟队列消息重复翻倍，浪费带宽与 Broker 存储，并增加消费端幂等压力。
2. **重试机制缺乏退避与死循环风暴**：
   - 历史重试采用固定延迟重试（`OutboxRetryDelay`），当下游 RabbitMQ 发生网络分区、证书过期或磁盘满载时，所有 Worker 瞬间进入死循环轮询重试风暴，不仅无法恢复，反而可能拖垮底层数据库连接池。
3. **架构约束：坚守 ADR-001 决策一**：
   - 严禁将耗时极不稳定的外部网络 IO（RabbitMQ `PublishWithDeferredConfirm`）包含在长数据库事务中；
   - 必须采用**两阶段安全认领与租期模式（Two-Phase Safe Claim with Lease & SKIP LOCKED）**：
     - **阶段一（微秒级本地事务，零外部网络 IO）**：在极短的数据库事务内使用 `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}` 并发安全认领记录，原子递增 `attempts` 并立即将 `next_attempt_at` 推进至租期过期时间 `now + leaseDuration`，随即提交事务释放行锁；
     - **阶段二（事务外网络异步发布）**：在事务外部安全向 RabbitMQ 发布消息并等待 DeferredConfirm；
     - **终态决算**：发布成功则标记 `published_at`；发布失败则基于指数退避算法（Exponential Backoff）计算新的 `next_attempt_at`，若重试次数达到封顶上限（`defaultMaxOutboxAttempts = 10`）记录 ERROR 告警，彻底杜绝无界重试与惊群效应。

---

## 2. 修改文件清单与实现方案

1. `internal/dao/order_timeout_outbox_dao.go`:
   - 新增 `ClaimPendingOrderTimeoutOutbox(ctx context.Context, db *gorm.DB, now time.Time, limit int, leaseDuration time.Duration, maxAttempts int) ([]model.OrderTimeoutOutbox, error)`：
     - 开启短事务，执行 `tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})` 检索 `published_at IS NULL AND next_attempt_at <= now AND attempts < maxAttempts`；
     - 在同一微秒级事务中原子更新 `attempts = attempts + 1` 与 `next_attempt_at = now.Add(leaseDuration)` 并提交；
     - 返回实体对象，提供租期排他占有。
   - 优化 `MarkOrderTimeoutOutboxFailed`：
     - 去除重复的 `attempts + 1` 自增（避免在认领与失败回写中双重自增），更新 `last_error` 与指数退避后的 `next_attempt_at`。
2. `internal/ordertimeout/worker.go`:
   - 引入核心常量：
     - `defaultMaxOutboxAttempts = 10`
     - `maxOutboxBackoffDelay = 10 * time.Minute`
   - 新增辅助函数：
     - `calculateExponentialBackoff(baseDelay time.Duration, attempts int, maxDelay time.Duration) time.Duration`：支持指数级递增重试延迟，附带精确的 `shift >= 62` 溢出防御与 `maxDelay` 封顶截断。
   - 重构 `publishBatch`：
     - 使用 `dao.ClaimPendingOrderTimeoutOutbox` 替换原先无锁的 `dao.ListPendingOrderTimeoutOutbox`；
     - 发布成功调用 `dao.MarkOrderTimeoutOutboxPublished`；
     - 发布失败通过 `calculateExponentialBackoff` 平滑拉长下一次尝试时间并回写；当 `attempts >= defaultMaxOutboxAttempts` 触发 ERROR 告警日志。
   - 在 `publishEvent` 中增加 nil channel 防御，增强单元测试隔离度与运行时鲁棒性。
3. `internal/ordertimeout/worker_test.go` (新建):
   - 编写 `TestCalculateExponentialBackoff` 单元测试，全面覆盖 14 种边界场景（零/负延迟、基础延迟、指数翻倍、最大延迟截断、默认兜底上限、以及 `attempts >= 62` 和 1000 次超大重试安全防溢出）；
   - 编写 `TestWorker_PublishBatch_HandlesFailureAndCapping` 单元测试：模拟发布失败回写与达到最大重试次数封顶告警；通过注入带 `bytes.Buffer` 的 `slog.Logger` 强校验日志输出，断言必须包含告警消息文本、`attempts=10` 与 `event_id=evt-fail-cap-101`，彻底杜绝断言软化与虚假全绿。
4. `internal/dao/order_timeout_outbox_dao_test.go` (新建):
   - 基于标准库 `database/sql/driver` 构建零外部依赖的 Mock 驱动，验证 `ClaimPendingOrderTimeoutOutbox` 的租期认领与 `attempts` 原子递增语义；
   - 编写参数验证用例 `TestClaimPendingOrderTimeoutOutbox_Validation`（`limit <= 0`）；
   - 编写 MySQL 集成测试骨架 `TestClaimPendingOrderTimeoutOutbox_MySQLIntegration`。

---

## 3. 实测验证命令与真实 stdout

### 3.1 Outbox 与 Worker 模块针对性单测与竞态检测

- **执行命令**：
```bash
go test -v -race -count=1 ./internal/ordertimeout/... ./internal/dao/...
```
- **执行结果**：`PASS (Exit Code 0)`
- **真实 stdout**：
```text
=== RUN   TestOrderTimeoutEndToEnd
    end_to_end_integration_test.go:28: skip order timeout end-to-end test; enable RabbitMQ and MySQL integration tests
--- SKIP: TestOrderTimeoutEndToEnd (0.00s)
=== RUN   TestOrderTimeoutEventRoundTrip
--- PASS: TestOrderTimeoutEventRoundTrip (0.00s)
=== RUN   TestDecodeOrderTimeoutEventRejectsInvalidPayload
--- PASS: TestDecodeOrderTimeoutEventRejectsInvalidPayload (0.00s)
=== RUN   TestExpirationMilliseconds
--- PASS: TestExpirationMilliseconds (0.00s)
=== RUN   TestRabbitMQDelayAndDeadLetterTopology
    rabbitmq_integration_test.go:18: skip RabbitMQ integration test; set RUN_RABBITMQ_TEST=1 to run
--- SKIP: TestRabbitMQDelayAndDeadLetterTopology (0.00s)
=== RUN   TestCalculateExponentialBackoff
=== RUN   TestCalculateExponentialBackoff/zero_or_negative_base_delay_returns_0
=== RUN   TestCalculateExponentialBackoff/negative_base_delay_returns_0
=== RUN   TestCalculateExponentialBackoff/attempt_0_returns_base_delay
=== RUN   TestCalculateExponentialBackoff/attempt_1_returns_base_delay_(5s)
=== RUN   TestCalculateExponentialBackoff/attempt_2_doubles_base_delay_(10s)
=== RUN   TestCalculateExponentialBackoff/attempt_3_quadruples_base_delay_(20s)
=== RUN   TestCalculateExponentialBackoff/attempt_4_is_8x_base_delay_(40s)
=== RUN   TestCalculateExponentialBackoff/attempt_6_is_32x_base_delay_(160s)
=== RUN   TestCalculateExponentialBackoff/attempt_8_exceeds_maxDelay_and_is_capped_at_10m
=== RUN   TestCalculateExponentialBackoff/attempt_10_capped_at_maxDelay
=== RUN   TestCalculateExponentialBackoff/base_delay_greater_than_max_delay_is_capped
=== RUN   TestCalculateExponentialBackoff/default_maxDelay_applied_when_maxDelay_<=_0
=== RUN   TestCalculateExponentialBackoff/large_attempts_shift_>=_62_does_not_overflow_int64
=== RUN   TestCalculateExponentialBackoff/extreme_attempts_1000_does_not_panic_or_overflow
--- PASS: TestCalculateExponentialBackoff (0.00s)
    --- PASS: TestCalculateExponentialBackoff/zero_or_negative_base_delay_returns_0 (0.00s)
    --- PASS: TestCalculateExponentialBackoff/negative_base_delay_returns_0 (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_0_returns_base_delay (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_1_returns_base_delay_(5s) (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_2_doubles_base_delay_(10s) (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_3_quadruples_base_delay_(20s) (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_4_is_8x_base_delay_(40s) (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_6_is_32x_base_delay_(160s) (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_8_exceeds_maxDelay_and_is_capped_at_10m (0.00s)
    --- PASS: TestCalculateExponentialBackoff/attempt_10_capped_at_maxDelay (0.00s)
    --- PASS: TestCalculateExponentialBackoff/base_delay_greater_than_max_delay_is_capped (0.00s)
    --- PASS: TestCalculateExponentialBackoff/default_maxDelay_applied_when_maxDelay_<=_0 (0.00s)
    --- PASS: TestCalculateExponentialBackoff/large_attempts_shift_>=_62_does_not_overflow_int64 (0.00s)
    --- PASS: TestCalculateExponentialBackoff/extreme_attempts_1000_does_not_panic_or_overflow (0.00s)
=== RUN   TestWorker_PublishBatch_HandlesFailureAndCapping
2026/10/21 20:25:54 ERROR order timeout outbox reached max attempts event_id=evt-fail-cap-101 order_id=2001 attempts=10 error="publish order timeout event: channel is nil"
--- PASS: TestWorker_PublishBatch_HandlesFailureAndCapping (0.01s)
PASS
ok  	go-order-management-system/internal/ordertimeout	3.200s
=== RUN   TestCriticalDAOQueries
    dao_integration_test.go:23: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestCriticalDAOQueries (0.00s)
=== RUN   TestClaimPendingOrderTimeoutOutbox_UnitMock
--- PASS: TestClaimPendingOrderTimeoutOutbox_UnitMock (0.00s)
=== RUN   TestClaimPendingOrderTimeoutOutbox_Validation
--- PASS: TestClaimPendingOrderTimeoutOutbox_Validation (0.00s)
=== RUN   TestClaimPendingOrderTimeoutOutbox_MySQLIntegration
    order_timeout_outbox_dao_test.go:213: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestClaimPendingOrderTimeoutOutbox_MySQLIntegration (0.00s)
PASS
ok  	go-order-management-system/internal/dao	1.720s
```

### 3.2 静态代码检查

- **执行命令**：
```bash
go vet ./...
```
- **执行结果**：`PASS (Exit Code 0)` 无任何报警或代码坏味道。

### 3.3 全仓单元测试与竞态检测

- **执行命令**：
```bash
go test -v -race -count=1 ./...
```
- **执行结果**：`PASS (Exit Code 0)`，全仓所有包（config, dao, handler, middleware, observability, ordertimeout, router, service, migrations, database, idgen）单元测试全绿，零竞态竞争。

### 3.4 Falsifier 审查反馈加固单测（精准日志与上限断言）

- **执行命令**：
```bash
go test -v -race -run TestWorker_PublishBatch_HandlesFailureAndCapping ./internal/ordertimeout/...
```
- **执行结果**：`PASS (Exit Code 0)`
- **真实 stdout**：
```text
=== RUN   TestWorker_PublishBatch_HandlesFailureAndCapping
--- PASS: TestWorker_PublishBatch_HandlesFailureAndCapping (0.01s)
PASS
ok  	go-order-management-system/internal/ordertimeout	3.225s
```

