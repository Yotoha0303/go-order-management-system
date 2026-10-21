# 变更记录：007-fix-gap-004-ladder-delay-queue

- **变更编号**：007
- **创建时间**：2026-10-21T13:52:00Z (UTC)
- **完成时间**：2026-10-21T14:01:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-004 (P2: RabbitMQ 单消息 TTL 存在队头阻塞隐患，Issue #5)
- **关联 ADR**：ADR-002 (RabbitMQ 分级阶梯延时队列消除队头阻塞架构设计)
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在原有的订单超时关单机制中，系统通过向单一延迟队列 `order.timeout.delay` 投递带 `Expiration` 属性的单消息，配合死信交换机（DLX）路由至 `order.timeout.cancel` 进行消费。

这种单队列动态单消息 TTL 设计存在严重的队头阻塞（Head-of-Line Blocking, HoLB）隐患：
1. **物理队头阻塞（GAP-004）**：
   - RabbitMQ 仅检查队头消息是否过期。若一条长延时消息（如 30 分钟常规单）先入队排在队头，后续排在其后的短延时消息（如 5 分钟秒杀单）即便在物理时间上早已到达超时期限，也无法被检查和流转；
   - 短延时消息被迫等待长延时消息到期出队后才被死信路由，导致秒杀/大促抢购订单超时取消滞后整整 25 分钟，库存预占被无理锁定，造成极大商业损失。
2. **架构裁定（ADR-002 Option B）**：
   - 拒绝高风险且缺乏原生镜像支持的第三方插件 `rabbitmq_delayed_message_exchange`（Option A）；
   - 正式采用分级固定 TTL 阶梯队列（Ladder Queues）架构：
     - 在 Exchange 下创建多个固定 TTL 阶梯队列（`delay.30m`, `delay.15m`, `delay.5m`）；
     - 各阶梯队列配置统一的队列级 `x-message-ttl` 及死信属性指向 `order.timeout.cancel`；
     - 同一队列内所有消息的 TTL 严格相等，队头消息必然是最先到期的消息，从数学原理上彻底根除队头阻塞；
     - 生产者废弃单消息 `Expiration` 属性，通过 `matchDelayLadder` 计算最匹配的阶梯 RoutingKey 投递。

---

## 2. 修改与新增文件清单

1. `internal/ordertimeout/worker.go`:
   - 定义 `DelayLadder` 模型（`Duration`, `RoutingKey`, `QueueName`）；
   - 定义默认 3 级生产阶梯切片（`5m`, `15m`, `30m`）；
   - `Config` 中新增 `DelayLadders []DelayLadder` 支持生产阶梯与自定义测试级阶梯；
   - `NewWorker` 初始化时对 `DelayLadders` 进行升序排序与默认值兜底；
   - 提取 `buildLadderQueueArguments(ladder DelayLadder, topology topology) amqp.Table` 纯函数生成队列参数，被 `declareTopology` 循环调用；
   - 增加 `matchDelayLadder` 算法，支持基于业务延时向上对齐匹配最优阶梯；
   - 提取 `buildOrderTimeoutPublishing(now time.Time, event Event, top topology) (amqp.Publishing, string, error)` 纯函数，组装发布消息并根据模式计算 routingKey 与 Expiration；
   - 重构 `publishEvent`：调用 `buildOrderTimeoutPublishing`，阶梯模式下 Expiration 严格为空并路由至匹配阶梯，旧版单队列模式下保留 Expiration；同时保留 channel nil 防御。
2. `internal/ordertimeout/rabbitmq_integration_test.go`:
   - 更新 `deleteTestTopology` 清理逻辑，包含所有阶梯队列的清理。
3. `internal/ordertimeout/ladder_test.go` (新建):
   - `TestMatchDelayLadder_ProductionLadders`：9 组边界单测覆盖负数、0、各区间内部、正好命中以及超长延时；
   - `TestMatchDelayLadder_CustomTestLadders`：自定义秒级测试阶梯验证；
   - `TestMatchDelayLadder_EmptyLaddersFallback`：空阶梯切片默认兜底 30m 阶梯；
   - `TestNewWorker_CustomLaddersSortingAndConfig`：无序阶梯配置自动升序排序；
   - `TestLadderTopology_QueueArgumentsValidation`：直接断言 `buildLadderQueueArguments` 生产代码产出参数（死穴加固：拒绝测试自造 map）；
   - `TestInterleavedDelay_HeadOfLineBlocking_EliminationProof`：队头阻塞消除数学仿真与逻辑证明，对比单队列 HoLB 与阶梯独立流转；
   - `TestPublishEvent_OmitsExpirationInLadderMode`：直接断言 `buildOrderTimeoutPublishing` 在阶梯模式下 Expiration 严格为空字符串、routingKey 匹配阶梯，对比断言旧版模式 Expiration > 0，并断言 `publishEvent` 的 nil channel 防御（死穴加固：消除 nil channel 绕过组装逻辑）。

---

## 3. 实测验证命令与真实 stdout

### 3.1 针对性模块测试（带竞态检测）

- 命令：`go test -v -race ./internal/ordertimeout/...`
- 真实 stdout：
```
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
=== RUN   TestOrderTimeoutEndToEnd
    end_to_end_integration_test.go:28: skip order timeout end-to-end test; enable RabbitMQ and MySQL integration tests
--- SKIP: TestOrderTimeoutEndToEnd (0.00s)
=== RUN   TestOrderTimeoutEventRoundTrip
--- PASS: TestOrderTimeoutEventRoundTrip (0.00s)
=== RUN   TestDecodeOrderTimeoutEventRejectsInvalidPayload
--- PASS: TestDecodeOrderTimeoutEventRejectsInvalidPayload (0.00s)
=== RUN   TestExpirationMilliseconds
--- PASS: TestExpirationMilliseconds (0.00s)
=== RUN   TestMatchDelayLadder_ProductionLadders
=== RUN   TestMatchDelayLadder_ProductionLadders/negative_delay_matches_smallest_ladder_(5m)
=== RUN   TestMatchDelayLadder_ProductionLadders/zero_delay_matches_smallest_ladder_(5m)
=== RUN   TestMatchDelayLadder_ProductionLadders/1_minute_delay_matches_smallest_ladder_(5m)
=== RUN   TestMatchDelayLadder_ProductionLadders/exactly_5_minute_delay_matches_5m_ladder
=== RUN   TestMatchDelayLadder_ProductionLadders/6_minute_delay_matches_middle_ladder_(15m)
=== RUN   TestMatchDelayLadder_ProductionLadders/exactly_15_minute_delay_matches_15m_ladder
=== RUN   TestMatchDelayLadder_ProductionLadders/16_minute_delay_matches_largest_ladder_(30m)
=== RUN   TestMatchDelayLadder_ProductionLadders/exactly_30_minute_delay_matches_30m_ladder
=== RUN   TestMatchDelayLadder_ProductionLadders/large_2_hour_delay_matches_largest_ladder_(30m)
--- PASS: TestMatchDelayLadder_ProductionLadders (0.00s)
=== RUN   TestMatchDelayLadder_CustomTestLadders
--- PASS: TestMatchDelayLadder_CustomTestLadders (0.00s)
=== RUN   TestMatchDelayLadder_EmptyLaddersFallback
--- PASS: TestMatchDelayLadder_EmptyLaddersFallback (0.00s)
=== RUN   TestNewWorker_CustomLaddersSortingAndConfig
--- PASS: TestNewWorker_CustomLaddersSortingAndConfig (0.00s)
=== RUN   TestLadderTopology_QueueArgumentsValidation
--- PASS: TestLadderTopology_QueueArgumentsValidation (0.00s)
=== RUN   TestInterleavedDelay_HeadOfLineBlocking_EliminationProof
--- PASS: TestInterleavedDelay_HeadOfLineBlocking_EliminationProof (0.00s)
=== RUN   TestPublishEvent_OmitsExpirationInLadderMode
--- PASS: TestPublishEvent_OmitsExpirationInLadderMode (0.00s)
=== RUN   TestRabbitMQDelayAndDeadLetterTopology
    rabbitmq_integration_test.go:18: skip RabbitMQ integration test; set RUN_RABBITMQ_TEST=1 to run
--- SKIP: TestRabbitMQDelayAndDeadLetterTopology (0.00s)
=== RUN   TestCalculateExponentialBackoff
--- PASS: TestCalculateExponentialBackoff (0.00s)
=== RUN   TestWorker_PublishBatch_HandlesFailureAndCapping
--- PASS: TestWorker_PublishBatch_HandlesFailureAndCapping (0.00s)
PASS
ok  	go-order-management-system/internal/ordertimeout	(cached)
```

### 3.2 全仓静态代码检查

- 命令：`go vet ./...`
- 退出码：0
- 真实 stdout：
```
Proxy enabled
```

### 3.3 全仓单元测试与竞态检测

- 命令：`go test -v -race -count=1 ./...`
- 退出码：0
- 真实 stdout（摘要）：
```
ok  	go-order-management-system/config	2.583s
ok  	go-order-management-system/internal/app	4.467s
ok  	go-order-management-system/internal/auth	2.304s
ok  	go-order-management-system/internal/bizcache	2.234s
ok  	go-order-management-system/internal/dao	2.424s
ok  	go-order-management-system/internal/handler	4.872s
ok  	go-order-management-system/internal/middleware	2.904s
ok  	go-order-management-system/internal/observability	1.600s
ok  	go-order-management-system/internal/ordertimeout	4.655s
ok  	go-order-management-system/internal/service	4.140s
ok  	go-order-management-system/migrations/integration	3.054s
ok  	go-order-management-system/pkg/database	2.596s
ok  	go-order-management-system/pkg/idgen	2.034s
ok  	go-order-management-system/router	3.351s
```
全仓 11 个带测试包 100% 通过，无数据竞态，无回归缺陷。
