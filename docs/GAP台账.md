# GAP台账 — 质量缺口与未竟事项唯一台账

> **单一事实源（D4）**：所有已知架构隐患、并发缺陷与待修复项**必须在此登记并分配唯一编号**。
> 不在台账里的“还没做”等于不存在。
> 关联 GitHub 仓库：[Yotoha0303/go-order-management-system](https://github.com/Yotoha0303/go-order-management-system)

---

## 质量缺口登记表

| 编号 | 严重度 | 模块 | 核心问题简述 | 对应 Issue | 状态 | 责任角色 |
| :--- | :---: | :--- | :--- | :---: | :---: | :--- |
| **GAP-001** | P1 | 订单/库存 | Redis Lua 原子预扣位于 MySQL 事务内部，未能前置拦截热点且引发长事务与双重扣减 | [#2](https://github.com/Yotoha0303/go-order-management-system/issues/2) | CLOSED | `backend-dev` / `system-architect` |
| **GAP-002** | P1 | 订单/幂等 | 订单幂等校验在 MySQL RR 隔离级别下的并发同 Key 查询异常与失败回滚状态丢失 | [#3](https://github.com/Yotoha0303/go-order-management-system/issues/3) | CLOSED | `backend-dev` / `data-engineer` |
| **GAP-003** | P2 | 延迟投递 | Transactional Outbox 轮询缺乏并发锁，多实例部署下消息重复发布 | [#4](https://github.com/Yotoha0303/go-order-management-system/issues/4) | CLOSED | `sre-operator` / `backend-dev` |
| **GAP-004** | P2 | 队列拓扑 | RabbitMQ 单消息 TTL 存在队头阻塞（Head-of-Line Blocking）隐患 | [#5](https://github.com/Yotoha0303/go-order-management-system/issues/5) | OPEN | `sre-operator` / `system-architect` |
| **GAP-005** | P1 | 最终一致性 | 订单超时取消在异常重启或网络故障时导致 Redis 库存永久丢失回补 | [#6](https://github.com/Yotoha0303/go-order-management-system/issues/6) | CLOSED | `backend-dev` / `qa-validator` |
| **GAP-006** | P2 | 数据模型 | 库存模型缺少锁定库存（`locked_quantity`）设计，且定时对账 Worker 缺乏自愈机制 | [#7](https://github.com/Yotoha0303/go-order-management-system/issues/7) | OPEN | `data-engineer` / `backend-dev` |
| **GAP-007** | **P0** | 库存并发 | `AddInventory` 补货使用绝对值覆盖写入 Redis，并发入库存在库存回退竞态 Bug | [#8](https://github.com/Yotoha0303/go-order-management-system/issues/8) | CLOSED | `backend-dev` / `data-engineer` |
| **GAP-008** | P1 | 队列拓扑 | RabbitMQ 死信失败队列 `order.timeout.failed` 缺乏消费者与告警，形成消息堆积黑洞 | [#9](https://github.com/Yotoha0303/go-order-management-system/issues/9) | CLOSED | `sre-operator` / `backend-dev` |
| **GAP-009** | P2 | 业务权限 | 缺少管理员全局订单列表与审计视图，管理后台无法查询与干预全平台订单 | [#10](https://github.com/Yotoha0303/go-order-management-system/issues/10) | OPEN | `backend-dev` / `sec-auditor` |
| **GAP-010** | P2 | 缓存安全 | 商品详情缓存查询缺乏防穿透（空值缓存/布隆）设计，非法 ID 可打爆底层 MySQL | [#11](https://github.com/Yotoha0303/go-order-management-system/issues/11) | OPEN | `backend-dev` / `sec-auditor` |
| **GAP-011** | P1 | 测试架构 | 修复测试套件中并发幂等的错误断言，消除 `t.Skip` 导致的本地开发虚假全绿 | [#12](https://github.com/Yotoha0303/go-order-management-system/issues/12) | CLOSED | `qa-validator` / `backend-dev` |

---

## 缺口细则与完成准则 (DoD)

### GAP-001 · Redis 预扣前置与解耦
- **现状**：`order_service.go` 的 `CreateOrder` 在开启 MySQL 事务且写入订单表后，才调用 Redis 预扣；紧接着又在同一事务内使用 `SELECT ... FOR UPDATE` 扣减 MySQL。
- **风险**：事务内网络 IO 拉长连接持有；热点请求即便库存不足也经历了 MySQL 事务开启与回滚。
- **DoD**：
  1. 将 Redis 预扣移至 MySQL 开启事务之前，使用全局预生成的业务编号（UUID / 分布式 ID）作为预扣标识；
  2. 若 Redis 预扣返回库存不足，直接响应客户端错误，绝对不触碰 MySQL；
  3. 完善扣减失败时的释放补偿链路。
- **解决闭环 (2026-10-21)**：
  - 落盘 ADR-001 决策一，确立精简 MySQL 事务零外部网络 IO、Redis 预扣前置防穿透。
  - 在 `pkg/idgen` 实现 64 位分布式 Snowflake ID 生成器，提取 `orderID` 生成至事务外。
  - 彻底移除预扣前对 MySQL 的所有探测查询，售罄时 100% 极速拦截（零 DB 连接与零事务）。
  - 完善事务异常及重放时多占预扣凭证的 `releasePreDeductedInventory` 自动补偿闭环。
  - 经 `qa-validator` 独立测试与 `falsifier` 6 项攻击全面检验（含零 DB 穿透、时钟回拨异常、重放防泄漏）全项 PASS。
  - 关联记录：`docs/changes/002-fix-gap-001-prededuct-timing.md`。状态：**CLOSED**。

### GAP-002 · 幂等校验在 RR 隔离级别下的并发防御
- **现状**：`TryCreateOrderIdempotencyKey` 在 MySQL 事务中使用 `ON CONFLICT DO NOTHING`，并发同 Key 时后到达的线程查询记录，但在 RR 隔离级别下读不到未提交快照，报 `RecordNotFound`。
- **风险**：客户端收到内部服务器错误，而非预期的 `ErrOrderBeingCreated`。
- **DoD**：
  1. 幂等 Key 获取解耦或优化查询方式（采用当前读或前置分布式锁）；
  2. 补齐并发同 Key 的自动化测试用例，断言并发下稳定返回“订单处理中”。
- **闭环方案**：在 `order_service.go` 中捕获 `gorm.ErrRecordNotFound` 并转换为领域错误 `ErrOrderBeingCreated`；在事务回滚出口自动触发 `releasePreDeductedInventory` 释放多占预扣，杜绝库存凭证泄漏。见 `docs/changes/003-fix-gap-002-and-011-idempotency-rr.md`。已通过变异测试证伪与门禁审计。

### GAP-003 · Transactional Outbox 多实例防重发布
- **现状**：`ListPendingOrderTimeoutOutbox` 使用普通 `SELECT` 捞取待发布记录，无锁保护。
- **风险**：多 Pod / 多副本部署时同时捞取并重复发布超时消息。
- **DoD**：
  1. 查询 SQL 增加 `FOR UPDATE SKIP LOCKED`，或引入轻量分布式锁；
  2. 增加指数退避重试与最大重试次数归档。
- **闭环方案**：在 `internal/dao/order_timeout_outbox_dao.go` 中实现 `ClaimPendingOrderTimeoutOutbox`，采用短事务配合 `clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}` 与租期（`leaseDuration`）安全互斥认领，原子自增 `attempts` 并前移 `next_attempt_at`，坚守 ADR-001 事务内零外部网络 IO 决策；在 `internal/ordertimeout/worker.go` 中实现具备 `shift >= 62` 溢出防御与最大 10 分钟封顶的指数退避重试（`calculateExponentialBackoff`），并在达到 10 次上限时记录 ERROR 告警日志。已完成针对性单测、falsifier 变异测试（日志断言硬化）与 gate-auditor 门禁审计。见 `docs/changes/005-fix-gap-003-outbox-skip-locked.md`。
- **状态**：CLOSED (2026-10-21)

### GAP-004 · 延迟消息队头阻塞防范
- **现状**：采用单消息设置 `Expiration` 属性投递至延迟队列。
- **风险**：RabbitMQ 仅检查队头消息 TTL，长消息阻塞后续短消息。
- **DoD**：
  1. 引入延迟消息插件（`rabbitmq-delayed-message-exchange`）或固定延迟阶梯队列设计；
  2. 验证多批次交错超时的消息能否准时进入死信取消链路。

### GAP-005 · 订单超时取消最终一致性加固
- **现状**：`CancelExpiredOrder` 在 MySQL 事务提交后调用 `restorePreDeductedInventory`。若此时服务崩溃，重试消费时因订单状态为 Cancelled 导致直接退出，Redis 库存永远少加。
- **风险**：分布式系统崩溃恢复时产生永久性可用库存泄漏。
- **DoD**：
  1. 消费端逻辑优化：即便订单已是 `Cancelled`，依然校验并清理该订单关联的 Redis reservation，确保回补幂等执行；
  2. 编写模拟事务提交后注入故障的异常测试用例。
- **闭环方案**：在 `CancelExpiredOrder` 与 `CancelOrder` 中，当状态为 `Cancelled` 时显式标记 `alreadyCancelled` / `idempotent`，并在事务外侧无条件触发 `restorePreDeductedInventory`。结合 Redis Lua 预占清理脚本的原生原子幂等性，实现崩溃恢复与重试自愈；新增基于 `mockCancelCrashConnector` 的双轨单元/集成单测与变异测试。见 `docs/changes/004-fix-gap-005-cancel-expired-order-redis-recovery.md`。已通过变异测试证伪与门禁审计。

### GAP-006 · 健全库存数据模型与对账自愈
- **现状**：`product_inventories` 仅有 `stock_quantity`，无 `locked_quantity`；取消回补按 `id ASC` 加锁而非 `product_id ASC`；对账 Worker 仅 Warn 日志。
- **风险**：库存状态语义不清；取消订单存在边缘死锁风险；对账无法自愈。
- **DoD**：
  1. 数据表增加 `locked_quantity` 字段并设计状态迁移；
  2. 取消订单回补库存显式按 `product_id ASC` 加锁；
  3. 对账 Worker 增加可配置的自动修正或告警推送。

### GAP-007 · 并发补货绝对值覆盖写导致库存回退修复
- **现状**：`AddInventory` 在 DB 成功后调用 `g.redisClient.Set(key, afterQuantity, 0)` 绝对值覆盖。
- **风险**：并发入库时先计算的旧值在网络延迟后覆盖新值，导致补货库存凭空蒸发。
- **DoD**：
  1. 改为 Redis `INCRBY` 原子递增指令；
  2. 编写多 Goroutine 并发补货测试，验证最终 Redis 库存与 MySQL 严格一致。
- **解决闭环 (2026-10-21)**：
  - 落盘 ADR-001 决策二，定义原生原子增量规范。
  - 在 `InventoryStockGuard` 引入 Lua 脚本原子执行 `INCRBY`（冷启动带 fallback 兜底），彻底消除应用层覆盖写。
  - 经 `qa-validator` 独立测试与 `falsifier` 变异攻击验证（变异注入绝对值覆盖必定爆红，判据真实有牙）。
  - 关联记录：`docs/changes/001-fix-gap-007-inventory-incrby.md`。状态：**CLOSED**。

### GAP-008 · RabbitMQ 死信失败队列治理
- **现状**：`order.timeout.failed` 队列声明后没有任何消费者监听或告警通道。
- **风险**：异常与坏消息永久堆积成黑洞。
- **DoD**：
  1. 编写对 `failedQueue` 的消费或告警监控逻辑；
  2. 支持结合重试次数将消息转为人工排障记录。
- **闭环方案**：建立 Goose 迁移 `migrations/00015_create_order_timeout_dead_letters.sql` 与 GORM 模型，持久化死信消息元数据；在 `internal/ordertimeout/worker.go` 中通过独立消费者通道启动 `consumeFailedLoop`，对进入 `failedQueue` 的消息进行宽容解析并落库，DB 异常平滑降级至 Emergency Fallback 日志，严格遵守终态无条件调用 `delivery.Ack(false)`（绝不 requeue，杜绝毒丸死循环风暴）；输出结构化 `ORDER_TIMEOUT_DLQ_ALERT` ERROR 告警，并暴露 Prometheus 业务指标 `app_order_timeout_dead_letter_total`；在 `consumeLoop` 中利用 `x-death` 累计重试达到 3 次上限时调用 `delivery.Nack(false, false)` 自然流转至 DLX。全流程通过针对性单测、三项变异证伪攻击与门禁审计放行。见 `docs/changes/006-fix-gap-008-failed-queue-consumer.md`。
- **状态**：CLOSED (2026-10-21)

### GAP-009 · 管理员全局订单列表与审计视图补齐
- **现状**：`/api/v1/orders` 强绑定 `current_user_id`，管理员无法查看全平台订单。
- **风险**：管理后台订单履约、排障与审计功能不可用。
- **DoD**：
  1. 新增 `/api/v1/admin/orders` 路由及分页筛选 Handler；
  2. 接入 `operation_logs` 审计记录。

### GAP-010 · 商品详情缓存防穿透加固
- **现状**：查询非法/不存在的 `product_id` 时，Redis 返回 Nil 后穿透查 DB，未写空值缓存。
- **风险**：恶意轮询可打爆 MySQL。
- **DoD**：
  1. 引入空对象缓存（Null Object）机制，设置短 TTL（如 30s~60s）并加随机抖动；
  2. 编写穿透验证测试，证明同非法 ID 后续请求直接命中 Redis 空值。

### GAP-011 · 纠正并发幂等测试断言与虚假全绿
- **现状**：`RUN_MYSQL_TEST` 缺省时静默跳过核心测试；并发幂等测试断言错误假设所有请求成功返回 nil。
- **风险**：测试误报全绿；并发状态机逻辑断裂未被暴露。
- **DoD**：
  1. 纠正测试断言：并发同 Key 必须且仅有 1 个成功创建，其余请求断言返回 `ErrOrderBeingCreated`；
  2. 完善集成测试启动指引与轻量单元测试分层。
- **闭环方案**：纠正 `TestCreateOrder_ConcurrentSameIdempotencyKey_CreatesOneOrder` 断言矩阵；新增零依赖 `mockRRConnector` 单测 `TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct`，脱敏外部容器即可 100% 覆盖 RR 快照盲区。见 `docs/changes/003-fix-gap-002-and-011-idempotency-rr.md`。已通过变异测试证伪与门禁审计。
