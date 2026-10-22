# GAP台账 — 质量缺口与未竟事项唯一台账

> **单一事实源（D4）**：所有已知架构隐患、并发缺陷与待修复项**必须在此登记并分配唯一编号**。
> 不在台账里的“还没做”等于不存在。
> 关联 GitHub 仓库：[Yotoha0303/go-order-management-system](https://github.com/Yotoha0303/go-order-management-system)

---

## 质量缺口登记表

> **状态枚举（唯一取值，无隐含约定）**：
> - **`CLOSED`** —— 缺口**已解除**：DoD 全部满足，且已过独立测试 / 证伪 / 门禁裁决（细则见下方「缺口细则与完成准则」）。
> - **`IN_PROGRESS`** —— 缺口**未解除**：已登记并指派、正在修复，或其修复尚未通过门禁裁决。
>
> 「**活跃 GAP**」= 状态**非 `CLOSED`** 的条目（`AGENTS.md` §4 的活跃计数按此口径统计）。

| 编号 | 严重度 | 模块 | 核心问题简述 | 对应 Issue | 状态 | 责任角色 |
| :--- | :---: | :--- | :--- | :---: | :---: | :--- |
| **GAP-001** | P1 | 订单/库存 | Redis Lua 原子预扣位于 MySQL 事务内部，未能前置拦截热点且引发长事务与双重扣减 | [#2](https://github.com/Yotoha0303/go-order-management-system/issues/2) | CLOSED | `backend-dev` / `system-architect` |
| **GAP-002** | P1 | 订单/幂等 | 订单幂等校验在 MySQL RR 隔离级别下的并发同 Key 查询异常与失败回滚状态丢失 | [#3](https://github.com/Yotoha0303/go-order-management-system/issues/3) | CLOSED | `backend-dev` / `data-engineer` |
| **GAP-003** | P2 | 延迟投递 | Transactional Outbox 轮询缺乏并发锁，多实例部署下消息重复发布 | [#4](https://github.com/Yotoha0303/go-order-management-system/issues/4) | CLOSED | `sre-operator` / `backend-dev` |
| **GAP-004** | P2 | 队列拓扑 | RabbitMQ 单消息 TTL 存在队头阻塞（Head-of-Line Blocking）隐患 | [#5](https://github.com/Yotoha0303/go-order-management-system/issues/5) | CLOSED | `sre-operator` / `system-architect` |
| **GAP-005** | P1 | 最终一致性 | 订单超时取消在异常重启或网络故障时导致 Redis 库存永久丢失回补 | [#6](https://github.com/Yotoha0303/go-order-management-system/issues/6) | CLOSED | `backend-dev` / `qa-validator` |
| **GAP-006** | P2 | 数据模型 | 库存模型缺少锁定库存（`locked_quantity`）设计，且定时对账 Worker 缺乏自愈机制 | [#7](https://github.com/Yotoha0303/go-order-management-system/issues/7) | CLOSED | `data-engineer` / `backend-dev` |
| **GAP-007** | **P0** | 库存并发 | `AddInventory` 补货使用绝对值覆盖写入 Redis，并发入库存在库存回退竞态 Bug | [#8](https://github.com/Yotoha0303/go-order-management-system/issues/8) | CLOSED | `backend-dev` / `data-engineer` |
| **GAP-008** | P1 | 队列拓扑 | RabbitMQ 死信失败队列 `order.timeout.failed` 缺乏消费者与告警，形成消息堆积黑洞 | [#9](https://github.com/Yotoha0303/go-order-management-system/issues/9) | CLOSED | `sre-operator` / `backend-dev` |
| **GAP-009** | P2 | 业务权限 | 缺少管理员全局订单列表与审计视图，管理后台无法查询与干预全平台订单 | [#10](https://github.com/Yotoha0303/go-order-management-system/issues/10) | CLOSED | `backend-dev` / `sec-auditor` |
| **GAP-010** | P2 | 缓存安全 | 商品详情缓存查询缺乏防穿透（空值缓存/布隆）设计，非法 ID 可打爆底层 MySQL | [#11](https://github.com/Yotoha0303/go-order-management-system/issues/11) | CLOSED | `backend-dev` / `sec-auditor` |
| **GAP-011** | P1 | 测试架构 | 修复测试套件中并发幂等的错误断言，消除 `t.Skip` 导致的本地开发虚假全绿 | [#12](https://github.com/Yotoha0303/go-order-management-system/issues/12) | CLOSED | `qa-validator` / `backend-dev` |
| **GAP-012** | P1 | 订单/接口契约 | 订单主键为 63 位 Snowflake 大整数，JSON 以原生数字输出，前端 `JSON.parse` 精度截断导致详情/支付/完成/取消全部 404 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | IN_PROGRESS | `backend-dev` / `frontend-dev` |
| **GAP-013** | P1 | 订单/前端状态 | 创建订单表单行以数组 index 作 key 致 Radix Select 状态残留；且失败后 `pendingSubmission` 未清空，幂等 Key 跨提交复用；缺显式重置入口 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | IN_PROGRESS | `frontend-dev` |
| **GAP-014** | P1 | 配置/运行 | 云端 Redis 未生效：`RedisConfig` 无 password/TLS 字段，`pkg/redis` 仅支持明文 TCP，云托管 `rediss://` 无法握手，架构首道预扣防线与商品缓存被静默旁路 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | IN_PROGRESS | `backend-dev` / `sre-operator` |
| **GAP-015** | P2 | 数据/接口契约 | `StockLog.BizID` 承载订单 Snowflake 主键（`order_service.go:188,557` 赋 `&order.ID`）却以 JSON 裸数字输出（`stock_log.go:20`），管理员库存流水页 `#{log.biz_id}` 显示被截断的订单号——GAP-012 的同类未收口面 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | IN_PROGRESS | `backend-dev` / `frontend-dev` |
| **GAP-016** | P1 | CI/门禁 | `ci.yml:22` 的 `MYSQL_TEST_DATABASE` 含连字符，被 `test_helper_test.go:95` 的 `^[a-zA-Z0-9_]+$` 拒绝（`:101` `t.Fatalf`）⇒ CI 上 `internal/service` 集成用例必然全红；本机 `RUN_MYSQL_TEST` 未设时同类用例全部 SKIP，造成「虚假全绿」 | [#15](https://github.com/Yotoha0303/go-order-management-system/issues/15) | IN_PROGRESS | `qa-validator` / `backend-dev` |
| **GAP-017** | P1 | CI/门禁 | `.github/workflows/ci.yml` 无任何前端作业，GAP-013 全部前端用例在 PR 上不执行 ⇒ 违反 `AGENTS.md` §2 03 环「新建用例 100% 纳入 CI 矩阵」 | [#16](https://github.com/Yotoha0303/go-order-management-system/issues/16) | IN_PROGRESS | `sre-operator` / `frontend-dev` |
| **GAP-018** | P2 | 数据/接口契约 | 商品主键 `Product.ID` 为**自增** bigint，仍以 JSON 裸数字输出（`product.go:11`），与订单侧 `Order.ID` 的字符串约定不一致；当前不是缺陷（远未及 2^53），但商品 ID 一旦改为 Snowflake 即重演 GAP-012 | —（待建 Issue） | IN_PROGRESS | `data-engineer` / `backend-dev` |
| **GAP-019** | P1 | 文档/编号 SSOT | `docs/changes/` 中编号 `004` 与 `005` **各被两个不同文件占用**（`004-fix-gap-005-…` + `004-support-mysql-tls-connection.md`；`005-fix-gap-003-…` + `005-support-cors-and-spa-rewrites.md`）⇒ 本仓任何「编号唯一」的说法都是**假守卫** | —（待建 Issue） | IN_PROGRESS | `ssot-scribe` |
| **GAP-020** | **P0** | CI/门禁 | **`main` 的 CI 自 `2026-07-07T05:56:15Z` 之后连续 15 次全红、无一次绿**（最近 9 次集中在 `2026-10-21`）。而 `GAP-003/004/006/008/009/010` 均标 `CLOSED`、闭环叙述含「门禁裁决」——**这些 `CLOSED` 从未经过 CI 验证**，其中 5 项的变更记录里自带 `--- SKIP` 日志 | —（待建 Issue） | IN_PROGRESS | `sre-operator` / `ssot-scribe` |

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
- **闭环方案**：落盘 ADR-002 架构决策并选定 Option B（分级固定 TTL 阶梯延时队列），彻底摒弃第三方非标插件依赖；在 `internal/ordertimeout/worker.go` 中声明 `order.timeout.delay.5m` / `15m` / `30m` 独立物理阶梯队列，由 `buildLadderQueueArguments` 统一绑定队列级 `x-message-ttl` 与死信 DLX，并在 `buildOrderTimeoutPublishing` 中实现向上安全匹配阶梯路由与单消息 `Expiration` 严格剥离；彻底消灭由于不同 TTL 混用引发的队头阻塞（Head-of-Line Blocking）。全流程经 qa-validator 审查、falsifier 4 维变异攻击（彻底根除 Expiration 剥离虚假全绿与自编自导测试死穴）与 gate-auditor 门禁放行。见 `docs/changes/007-fix-gap-004-ladder-delay-queue.md`。
- **状态**：CLOSED (2026-10-21)

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
- **解决闭环 (2026-10-21)**：
  - 架构与数据模型锁定：落盘 [ADR-003](file:///d:/MyDocuments/ai-collaboration-framework/go-order-management-system/docs/adr/003-inventory-locked-quantity-and-reconcile-autofix.md)，定义总实物库存守恒定律（$TotalPhysicalStock = stock\_quantity + locked\_quantity$）与加锁偏序 DAG 定理；
  - 数据库迁移：落盘 `migrations/00016_add_locked_quantity_to_product_inventories.sql`，增加 `locked_quantity BIGINT NOT NULL DEFAULT 0` 及 `CHECK(locked_quantity >= 0)` 物理非负约束；
  - 三态流转与死锁消除：在 `inventory_dao.go` 实现 `ReserveInventory`、`ReleaseLockedInventory`、`ConsumeLockedInventory` 原子 CAS；在 `order_service.go` 的 `CreateOrder`、`cancelPendingOrder`、`PayOrder` 统一按 `product_id ASC` 偏序遍历加锁，彻底杜绝哲学家就餐环路死锁；
  - 对账自愈：在 `inventoryreconcile/worker.go` 引入 `AutoFix` 模式，自动纠偏 Redis 缓存并暴露 Prometheus 指标；
  - 验证与审计：通过 `qa-validator` 单测/竞态矩阵、`falsifier` 4 组变异证伪攻击（偏序反转、守恒破解、CAS 条件移除、写回注释均精准红灯拦截），通过 `gate-auditor` 转段门禁裁决。见 `docs/changes/008-fix-gap-006-inventory-locked-quantity.md`。

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
- **解决闭环 (2026-10-21)**：
  - 请求模型与防御：新增 `ListAdminOrdersRequest` 并通过 `NormalizePagination` 进行分页防御（缺省 page=1, pageSize=10, 封顶 100）；
  - 持久层与服务层：在 `order_dao.go` 实现 `ListAdminOrders` 支持 `user_id` 与 `status` 多维动态过滤与稳定降序分页；服务层拦截非法负数 `user_id`；
  - 路由与审计安全网：在 `router.go` 挂载 `admin := rg.Group("")`，使用 `middleware.AdminMiddleware` 强制 RBAC 权限校验（非 admin 返回 403 Forbidden），接入 `middleware.OperationAudit` 自动捕获管理员操作审计流水（UserID、Path、Method、Status）；
  - 验证与审计：通过 `qa-validator` 四项质量门禁复核、`falsifier` 4 组变异证伪攻击（移除 RBAC 拦截、移除审计中间件、放行负数 user_id、清空分页纠偏全拦截），`golangci-lint` 0 issues，通过 `gate-auditor` 转段门禁裁决。见 `docs/changes/010-fix-gap-009-admin-orders-and-audit.md`。

### GAP-010 · 商品详情缓存防穿透加固
- **现状**：查询非法/不存在的 `product_id` 时，Redis 返回 Nil 后穿透查 DB，未写空值缓存。
- **风险**：恶意轮询可打爆 MySQL。
- **DoD**：
  1. 引入空对象缓存（Null Object）机制，设置短 TTL（如 30s~60s）并加随机抖动；
  2. 编写穿透验证测试，证明同非法 ID 后续请求直接命中 Redis 空值。
- **解决闭环 (2026-10-21)**：
  - 架构方案与哨兵设计：遵循 ADR-001 高并发缓存与防穿透设计，定义空值哨兵 `ProductDetailNullSentinel = "__NULL_PRODUCT__"`；
  - 防雪崩抖动：引入 `nullCacheTTLWithJitter()`，设置短 TTL 60s 并附加 0~15s 纳秒时间抖动，杜绝批量空值同时到期雪崩；
  - 零 DB 穿透业务防线：在 `product_service.go` 的 `GetProductByID` 识别三态命中（有效商品返回、命中哨兵秒级直接返回 `ErrProductNotFound` 达成零 DB 查询、未命中回源 DB `gorm.ErrRecordNotFound` 立即触发空哨兵写回）；
  - 验证与审计：单测注入 `nil db` 硬核证明零 DB 访问防穿透；`golangci-lint` 0 issues；通过 `falsifier` 4 组变异证伪攻击（空值写入移除、哨兵误判穿透、抖动算法越界、错误返回遗漏全拦截）与 `gate-auditor` 转段门禁裁决。见 `docs/changes/009-fix-gap-010-product-cache-null-object.md`。

### GAP-011 · 纠正并发幂等测试断言与虚假全绿
- **现状**：`RUN_MYSQL_TEST` 缺省时静默跳过核心测试；并发幂等测试断言错误假设所有请求成功返回 nil。
- **风险**：测试误报全绿；并发状态机逻辑断裂未被暴露。
- **DoD**：
  1. 纠正测试断言：并发同 Key 必须且仅有 1 个成功创建，其余请求断言返回 `ErrOrderBeingCreated`；
  2. 完善集成测试启动指引与轻量单元测试分层。
- **闭环方案**：纠正 `TestCreateOrder_ConcurrentSameIdempotencyKey_CreatesOneOrder` 断言矩阵；新增零依赖 `mockRRConnector` 单测 `TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct`，脱敏外部容器即可 100% 覆盖 RR 快照盲区。见 `docs/changes/003-fix-gap-002-and-011-idempotency-rr.md`。已通过变异测试证伪与门禁审计。

### GAP-012 · 订单主键大整数 JSON 精度截断
- **现状**：订单主键由 `pkg/idgen` 的 Snowflake 生成，为 63 位整数；接口以 JSON 原生数字输出该字段，前端经 `JSON.parse` 解析后超出 IEEE-754 双精度安全整数范围（`Number.MAX_SAFE_INTEGER`），末位被截断；以截断后的 ID 拼装详情 / 支付 / 完成 / 取消请求路径，后端查无此单，四个接口均返回 404。
- **风险**：写入与读取使用两个不同的 ID 事实，且**不会报类型错误**——表现为业务 404 而非序列化告警；订单详情、支付、完成、取消四条主链路对前端整体不可用。
- **DoD**：
  1. `model.Order.ID` 的 JSON 输出改为**字符串**（序列化侧统一约定，不得只在前端做 hack）；
  2. 前端 `Order.id` 类型为 `string`，且构造 URL 路径时**原样透传**，不经过任何数值转换；
  3. 提供**负向用例**证明「数字形式输出」会往返失真：同一 63 位 ID 经 JSON 数字路径往返后与原值不等（被截断），据此可证伪本判据不会静默放行。
- **状态**：IN_PROGRESS —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。CI 实测：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Test, Vet, Validate, Build` **已真跑并执行到 `internal/model` / `internal/handler`**（该作业 `fail`，但失败点为 `TestRebuildRedisInventoryStock_Success`，与本项无关，见 GAP-020）。**故「未在 CI 验证」已不成立**；仍不判 `CLOSED`，因 CI 整体未绿。

### GAP-013 · 创建订单表单行 key 与幂等 Key 生命周期缺陷
- **现状**：创建订单表单的商品行以**数组 index** 作为 React key，删除中间行后 React 复用了错误的 DOM 节点，Radix Select 的内部状态残留到相邻行；提交失败后 `pendingSubmission` 未清空，下一次提交复用同一幂等 Key，被后端判定为重复请求；界面无显式的重置入口。
- **风险**：用户看到的商品行与实际提交内容不一致（选错商品 / 数量），且失败后无法通过重试恢复——重试被幂等键拦成「处理中」，页面却无任何自愈手段。
- **DoD**：
  1. 商品行改用**稳定唯一** `rowId`（非数组 index）作为 key；
  2. 提交**成功与失败均清空** `pendingSubmission`，并断言失败后重新提交会生成**全新**幂等 Key；
  3. 提供显式的「清空」入口；
  4. 有单测覆盖上述三项（含失败后重提生成新 Key 的用例）。
- **状态**：IN_PROGRESS —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。CI 实测：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的**前端作业 `Frontend (tsc, eslint, vitest browser, build)` 已真跑且 `pass`（1m0s）**，本项的前端用例（`order-draft.test.ts` / `order-submission.test.ts` / `qa-order-submission-lifecycle.test.ts` / `qa-orders-form.test.tsx`）在该作业内执行。**故「未在 CI 验证」已不成立**；仍不判 `CLOSED`，因 CI 整体未绿。

### GAP-014 · 云端 Redis 凭据与 TLS 支持缺失
- **现状**：`RedisConfig` 无 password / TLS 字段，`pkg/redis` 的连接构建仅支持明文 TCP；云托管 Redis 以 `rediss://`（TLS）暴露且要求凭据，握手无法完成。
- **风险**：Redis 连接失败后，架构的**首道预扣防线**与**商品详情缓存**被静默旁路——服务仍能启动、部分接口仍能响应，**失效不会显式报错**，与 D3 的「失败即静默」同族。
- **DoD**：
  1. 配置层支持 `REDIS_USERNAME` / `REDIS_PASSWORD` / `REDIS_TLS`，并支持 `REDIS_URL=rediss://…` 形式；
  2. 客户端在 TLS 模式下按 TLS 握手（不得静默退回明文）；
  3. 启动日志可判定 Redis **已连接 / 连接失败**（失败必须显式可见，不得旁路后无声运行）；
  4. 单测覆盖 TLS 分支，并含**负向用例**：非法布尔值（如 `REDIS_TLS=maybe`）必须报错而非被当作 false 静默吞掉。
- **已裁定：维持降级启动（不改为 fail-fast）**——来源与依据如下，**该裁定已落痕，不再计为未决缺口**：
  - **决定**：Redis 不可用时服务**继续启动**，保持现有降级旁路行为；**不**引入 fail-fast。
  - **依据（实测，`docs/changes/013-fix-gap-014-redis-cloud-tls-config.md:307-309` 原文）**：该文件明确记「降级仍是『继续启动』…**未**改为 fail-fast。是否要 fail-fast 属架构决策…需 `system-architect` / `sre-operator` 裁定」——即**该问题原为未决项**。
  - **裁定主体与时点**：由**用户**裁定「维持降级启动」（`2026-10-22`，经总控征询）；属该文件所述「需架构方裁定」的**上位裁定**。
  - **残留义务（不是缺口，是约束）**：因不 fail-fast，**失败必须显式可见**——`redis connection failed … degraded to pass-through bypass`（Error 级日志）与启动日志的 `redis connected` 是这条裁定的**可观测前提**，DoD ③ 因此**不可删减**。
- **状态**：IN_PROGRESS —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。CI 实测：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Test, Vet, Validate, Build` **已真跑**，`pkg/redis` 与 `config` 包执行并通过（失败点为 `internal/service`，与本项无关，见 GAP-020）。**故「未在 CI 验证」已不成立**；仍不判 `CLOSED`，因 CI 整体未绿。

### GAP-015 · 库存流水 `BizID` 的 Snowflake 裸数字输出
- **现状**：`internal/model/stock_log.go:20`（HEAD 口径）为 `BizID *int64` 且 JSON 标签是 `json:"biz_id"`（**无** `,string`）。该字段由订单链路写入并承载订单主键：`internal/service/order_service.go:188`（`StockBizOrderDeduct`，下单扣减）与 `:557`（`StockBizOrderRollback`，订单取消）均赋 `&order.ID`，而 `order.ID` 是 63 位 Snowflake（`internal/model/orders.go:16`）。取值域 = **{nil, 订单 Snowflake}**：`internal/service/inventory_service.go:53-60`（初始化库存）与 `:116-123`（手动入库）不赋 `BizID`，故 nil 分支必须保持 `null`。列表接口挂在管理员组下（`router/router.go:138`，受 `AdminMiddleware`），`internal/handler/stock_log_handler.go:43-49` 直接 `response.Success(c, stockLogs)` 输出 `[]*model.StockLog`，裸数字因此直达响应体。前端 `fronted/src/features/order-inventory/types.ts:52` 声明 `biz_id?: number | null`，`fronted/src/features/order-inventory/stock-logs.tsx:120` 以 `` `#${log.biz_id}` `` 直接渲染。
- **风险**：与 GAP-012 同一失效形态——写入与读取使用两个不同的 ID 事实，且**不会报类型错误**，只表现为流水页显示出错号的订单号；据此反查订单必然查无此单。实测（`node v22.23.2`，`2026-10-22T05:03Z`，探针值 `371447562240004097`）：`JSON.parse('{"biz_id":371447562240004097}')` 得到 `371447562240004100`，往返不相等；同一值以**字符串**形态往返则相等。
- **DoD**：
  1. `BizID` 的 JSON 输出改为**字符串**（序列化侧统一约定，不得只在前端做 hack）；`nil` 仍须序列化为 `null`，**不得**变成 `"0"` 或 `""`；
  2. 前端 `StockLog.biz_id` 类型对齐（`string | null`），`stock-logs.tsx` 渲染路径**不得**做任何数值转换；
  3. 有**原始字节**断言用例：直接对序列化后的 JSON 字节断言 `biz_id` 被引号包裹（不得用「反序列化到 `interface{}` 再比较」的写法——那会同时掩盖数字与字符串两种形态）；
  4. 有**负向用例**证明数字形态必然失真：同一 63 位 ID 经 JSON 数字路径往返后与原值不等，据此可证伪本判据不会静默放行。
- **状态**：IN_PROGRESS —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。**代码与独立测试侧已收口**：`docs/changes/014-fix-gap-015-stock-log-biz-id-precision.md` 记录了修复（`json:"biz_id,string"`，nil 仍为 `null`）与原始字节断言；`qa-validator` 已把 `internal/model/qa_json_surface_test.go` 的钉住用例**翻转**为「set id 必为字符串 + nil 必为 `null`」，实测：
  ```
  go test -count=1 ./internal/model/ -run TestQAStockLogBizIDShapePinsTheOpenContract -v
  --- PASS: TestQAStockLogBizIDShapePinsTheOpenContract (0.00s)
  ```
  **故此前记录的「门禁因该用例变红」已不成立**，`go test ./...` 不再因此有红点。门禁读数见 `014-*.md` §6 / §7（含 §7.2 对 `tsc --noEmit` 空转的订正）。
  **CI 实测**：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Test, Vet, Validate, Build` 已真跑（`internal/model` 包通过），**「未在 CI 验证」已不成立**。
  **仍保持 `IN_PROGRESS`**：按状态枚举，`CLOSED` 需「DoD 全部满足 + 已过独立测试 / 证伪 / **门禁裁决**」，而本轮 `gate-auditor` 裁决为**阻断**——缺的是**门禁裁决这一环**，不是代码或测试。

### GAP-016 · CI 测试库名与本机门禁开关的双向失真
- **现状**（两处独立缺陷，**其一已修、其一仍在**）：
  1. **CI 侧（已修复，历史仍须留痕）**：`4fa68f9`（`main` 当时的 HEAD）的 `ci.yml:22` 为 `MYSQL_TEST_DATABASE: go-order-management-system_test`，**含连字符**；`internal/service/test_helper_test.go:95` 的 `safeDatabaseName`（`^[a-zA-Z0-9_]+$`）在 `:101` 匹配失败后走 `:102` 的 `t.Fatalf`，而 `ci.yml:18` 又设 `RUN_MYSQL_TEST: "1"`，故**必然**进入该分支。**实测（run `39651784273`，`2026-10-21T18:20:12Z`）**：
     ```
     --- FAIL: TestInitInventory_ProductNotFound (0.00s)
         inventory_service_test.go:70: invalid MYSQL_TEST_DATABASE "go-order-management-system_test"
     ```
     `internal/service` 下**全部**用例同因此 Fatal。
     **现已修复**：提交 `d024ef5`（`2026-10-22T05:37:43Z`）把库名改为 `go_order_management_system_test`；**当前 HEAD `3584d1f` 的 `ci.yml:25` 即为下划线形态**。
     **故「CI 上必然全红」这一断言已不再成立**——PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 实测：`internal/service` 已**真跑**，失败点变为 `TestRebuildRedisInventoryStock_Success`（`inventory_service_test.go:203: expected inventories ordered by product_id asc`），**不再是库名 Fatal**。
  2. **本机侧虚假全绿（仍在）**：`RUN_MYSQL_TEST` 未设时，`internal/service/test_helper_test.go:35-37` 等直接 `t.Skip`，包级别仍输出 `ok`。
- **实测（本机 `go1.25.7 windows/amd64`）**：
  - 未设 `RUN_MYSQL_TEST` ⇒ `go test -count=1 ./internal/service/ -run TestInitInventory_CreateStockLog -v` 输出 `--- SKIP ... skip MySQL integration test; set RUN_MYSQL_TEST=1 to run`，`exit 0`；
  - **全仓 SKIP 计数（`2026-10-22`）**：`go test -count=1 -v ./...`（`RUN_MYSQL_TEST` 未设）⇒ **`--- SKIP` 共 85 条**，全部包 `ok`，`exit 0`。⚠️ 这 85 条**同时包含** `RUN_MYSQL_TEST` / `RUN_REDIS_TEST` / `RUN_RABBITMQ_TEST` 三类未设开关导致的跳过，**不等于**「MySQL 集成用例 85 条」；
  - 设 `RUN_MYSQL_TEST=1` 且 `MYSQL_TEST_DATABASE=go-order-management-system_test` ⇒ `--- FAIL ... invalid MYSQL_TEST_DATABASE "go-order-management-system_test"`，`exit 1`（**负向行为有牙，须钉住**）。
- **影响面（须精确，不得笼统写成「全部集成用例」）**：`MYSQL_TEST_DATABASE` 只有两处读者——`internal/service/test_helper_test.go:99`（校验后拼进 `CREATE DATABASE`）与 `pkg/database/testdb.go:48`（仅判非空，据以拼 DSN）。`internal/dao/dao_integration_test.go:147` 与 `migrations/integration/migrations_test.go:38` **不读该变量**，各自新建 `go_order_*_test_<pid>_<nano>` 隔离库（名字由代码生成，天然合法）。故**库名含连字符只让 `internal/service` 变红**；`RUN_MYSQL_TEST` 的 SKIP 分支则同时存在于上述三处。
- **风险**：红灯出现在 CI 上（且根因是一条**配置拼写**而非业务逻辑），本机却永远看不到——两处失真互相掩盖：CI 红被当成「环境问题」，本机绿被当成「已通过」。
- **DoD**：
  1. `ci.yml` 的库名与 `safeDatabaseName` 相容（下划线形态），且**保留**该校验——它是库名拼进 `CREATE DATABASE` 之前的标识符防线，不是可以绕开的障碍；**（已由 `d024ef5` 满足，CI 实测见上）**
  2. `RUN_MYSQL_TEST=1` 时集成用例**真绿**，不得以 SKIP 代替；且须能证明**确实执行了**（例如给出非 SKIP 的执行数，而不是只看包级 `PASS`）；**（仍未满足：CI 至今无一次全绿，见 GAP-020）**
  3. **负向用例**：非法库名必须 `Fatalf` 而非静默跳过——这是当前行为，须**钉住**，防止日后有人把校验改成 skip 来换绿；
  4. 本机与 CI 使用**同一套**开关语义（`AGENTS.md` §5「本地跑的那一套 = CI 跑的那一套」）。
- **状态**：IN_PROGRESS —— 关联 [#15](https://github.com/Yotoha0303/go-order-management-system/issues/15)；Issue 已建档，编号已回填。**① 已满足；② 仍缺一次 CI 全绿**，故不判 `CLOSED`。

### GAP-017 · CI 缺前端作业，03 环用例在 PR 上不执行
- **现状**（**历史缺陷已修，当前状态如实如下**）：
  - **缺陷发生时（`4fa68f9`，`main` 当时的 HEAD）**：`.github/workflows/ci.yml` 共 **111 行**，**只有一个作业** `quality`（`:27`），步骤全部是 Go / goose / golangci-lint / `docker build`；`node` / `npm` / `npx` 关键字命中数 **0**（`Select-String` 实测）。
  - **现已修复**：提交 `d024ef5` 新增 `frontend` 作业；**当前 HEAD `3584d1f` 的 `ci.yml` 共 168 行**，含 `frontend`（`:119`）、`actions/setup-node@v4`（`:137`）、`npm ci`（`:144`）、`npx tsc -b`（`:159`）、`npx eslint .`（`:162`）、`npx vitest run --browser.headless`（`:165`）。
  - **CI 实测**：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Frontend (tsc, eslint, vitest browser, build)` → **pass，1m0s**。**故「前端用例在 PR 上不执行」已不成立。**
  - 前端现有 **22 个测试文件**（`Get-ChildItem fronted/src -Recurse -Include *.test.ts,*.test.tsx`，已排除 `__screenshots__` 下的两个**同名 1 字节产物**——`Test-Path` 为真但 `ReadAllBytes` 被拒、非源码用例），其中含 GAP-013 的相关用例：`order-submission.test.ts`、`order-draft.test.ts`、`qa-order-submission-lifecycle.test.ts`、`qa-orders-form.test.tsx`。`fronted/package.json:14` 的 `test` 脚本为 `vitest run --browser.headless`。
- **风险**：前端用例**存在**但**不在 PR 门禁内**——本地跑与不跑都不影响合并。这正是 `AGENTS.md` §2 03 环「新建用例 **100% 纳入 CI 矩阵**（拒绝门外用例）」所禁止的「门外用例」，且与 D3 同族：**失败不会响**。（该风险现已在 CI 配置层解除；剩余风险见 DoD ②：作业是否被设为 **required check** 尚**未验证**。）
- **DoD**：
  1. CI 有**独立前端作业**，类型检查步骤**必须是 `tsc -b`**——这不是可选替代：`tsc --noEmit` 在本仓**只检查 0 个源文件、恒 `exit 0`**，等同没有类型门禁（证据见下方「① 的判据依据」）。同一步作业还须执行 `eslint` 与 `vitest run`；**（已满足，CI 实测 pass）**
  2. 该作业失败**必须**使 PR 变红（作为 required check 的组成部分，不是 advisory）；**（未验证：需在仓库分支保护里确认为 required，本机命令无法判定）**
  3. 说明**为何不能塞进现有 Go 作业**——两条独立理由：**循环时长**（Go 循环与 `npm ci` + 浏览器二进制安装的耗时互不相关，合并后每次 Go 迭代都要等前端依赖装完）与**失败定位**（一个作业红无法区分是 Go 还是前端坏了）。**（已落实：`ci.yml:116-118` 的注释给出了同一理由）**
- **① 的判据依据（本机实测）**：`fronted/tsconfig.json` 是 solution 风格（`"files": []` + `references`），故以根配置执行时**根本不编译任何文件**。
  - 文件计数（`2026-10-22T05:02Z`，`--listFiles` 过滤 `node_modules` / `typescript/lib`）：`npx tsc --noEmit` 检查 **0** 个源文件、`exit 0`；`npx tsc -p tsconfig.app.json --noEmit --listFiles` 检查 **150** 个；`npx tsc -b` 走 `references`，与后者同集。
  - **注入真实类型错误后的判别力**（`backend-dev` 注入 `src/__tsc_gate_probe.ts` 的 TS2322，探针已撤回、`Test-Path` 现为 `False`）：`npx tsc --noEmit` → `exit 0`（**漏检**）；`npx tsc -b` → `exit 2`（报出 `TS2322`）。
  - **结论**：只有 `tsc -b` 是**真正执行类型检查**的形式；`tsc --noEmit` 在本仓是**空转**，写成它就是一条无牙的判据（D3）。此条为 DoD ① 的判据依据，**不改动本项核心问题与 DoD 编号**。
- **状态**：IN_PROGRESS —— 关联 [#16](https://github.com/Yotoha0303/go-order-management-system/issues/16)；Issue 已建档，编号已回填。**① ③ 已满足且 CI 已真跑 pass；② 未验证**（required check 属仓外配置），故不判 `CLOSED`。

### GAP-018 · 商品主键序列化约定与订单侧不一致
- **现状**：`internal/model/product.go:11` 为 `ID int64 \`gorm:"primaryKey;autoIncrement" json:"id"\``（**自增**，非 Snowflake）；`migrations/00001_create_product.sql:3` 亦为 `id BIGINT NOT NULL AUTO_INCREMENT`。该字段**直接进入响应体**：`internal/handler/product_handler.go:49`（创建）、`:83`（列表）、`:147`（详情）均为 `response.Success(c, product/…products…)`，无 DTO 转换层。
- **风险**：**当前不是缺陷**——自增 ID 远未及 `Number.MAX_SAFE_INTEGER (2^53-1)`，`JSON.parse` 不会截断。真正的风险是**契约不一致的沉默性**：订单侧（`Order.ID` / `OrderItem.OrderID`）与库存流水侧（`GAP-015` 的 `StockLog.BizID`）都已字符串化，商品侧仍是数字；一旦商品主键改为 Snowflake（或引入外部/分布式 ID），将**原地重演 GAP-012**，且同样不报错。来源：`docs/changes/011-fix-gap-012-order-id-json-precision.md:87-89` 与 `:214`（原文建议「由 `ssot-scribe` 决定是否登记为独立条目」）。
- **与 GAP-015 的边界（不得互相替代）**：GAP-015 覆盖的是 `StockLog.BizID`，其取值是**订单** Snowflake；本项的载体是 `Product.ID`，取值是**商品**自增主键。两者载体、取值域、风险性质均不同，**不可用一条结掉另一条**。
- **DoD**：
  1. 明确商品主键的**长期生成策略**（保持自增 / 迁 Snowflake / 换外部 ID），并落 ADR——这是 01 环决策，**不得**由实现方顺手决定；
  2. 若**保持自增**：写明「不字符串化」的理由，并加**守护用例**把「当前不是缺陷」变成可证伪的断言——一旦商品 ID 越过 `2^53` 或生成方式改为非自增，用例必须**主动失败**（不得静默继续）；
  3. 若**迁 Snowflake**：与订单侧同批字符串化，含原始字节断言 + 数字形态往返失真的负向用例（同 GAP-012 / GAP-015 口径）；
  4. 无论选哪条，都须在 `docs/table_design.md` 与前端 `types.ts` 同步该决定（D4：不留两处不同成熟度）。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号；**待架构裁定**（DoD ① 未决，故本项暂不可判 `CLOSED`）。

### GAP-019 · `docs/changes/` 编号 `004` / `005` 各被两个文件占用
- **现状**（全 ref 并集实测，非单看当前检出）：`004` 与 `005` 各有两个文件并存，且**自 `main` 起即如此**（`main`、`origin/main`、`origin/HEAD`、`fix/issue-14-order-module` 四个 ref 均含这四个文件）：
  - `docs/changes/004-fix-gap-005-cancel-expired-order-redis-recovery.md`（GAP-005 变更）
  - `docs/changes/004-support-mysql-tls-connection.md`（MySQL TLS 支持，非 GAP 编号体系）
  - `docs/changes/005-fix-gap-003-outbox-skip-locked.md`（GAP-003 变更）
  - `docs/changes/005-support-cors-and-spa-rewrites.md`（CORS / SPA 重写，非 GAP 编号体系）
  - 复检命令与结论见本节末尾「复检留痕」。
- **风险**：同一编号指向两个不同变更，`docs/changes/004` 与 `005` 在引用中**有歧义**（读者无法确定指的是哪一次改动）；更关键的是它**证伪了「编号唯一」这个隐含前提**——本仓**没有任何**检查 `docs/changes` 编号唯一性的机械守卫，故这类撞车**不会被任何门禁发现**。与 GAP-016（CI 库名） / GAP-017（前端不在 CI）同属**「守卫假绿」家族**：绿着，但没在检查。
- **为何是 `GAP-` 而不是 `DEFER-`**：`DEFER-` 用于「**有理由的主动延期**」；本项是**已发生的既有缺陷**，且其后果（引用歧义 + 无守卫）不会因延期而消失，也没有给出延期的理由。**延期不改变它是缺陷**，故记 `GAP-`。
- **DoD**：
  1. 选定并落定一种解法，三选一：**重命名**（改掉后到者，如 `004-support-mysql-tls-connection.md` → 顺延新号）/ **声明豁免**（在 `docs/changes/README.md` 写明「非 GAP 类变更可不占 GAP 序号」并列出既有例外）/ **引入时间戳前缀**（`YYYYMMDD-NNN-*`，从根上消除手工选号）；
  2. **不重命名既有文件直到引用一并改完**：重命名须全仓更新引用，并给出 `git grep -n '<旧文件名>'` **零命中**的证据（含未跟踪文件），否则就是制造幻影引用；
  3. **加机械守卫**：按「全 ref 并集」逐 `ls-tree` 判重（**不得只看当前检出**，否则并行分支撞号照旧漏），撞车即红；并配**负向用例**——注入「同一编号两个文件」必须触发失败（D3：不注入故障就不算验证过）；
  4. **历史快照不改写**：已在 `docs/changes/*` 内被引用的既有文件名，若必须改，须同时保留指向说明，不得静默消失。
- **复检留痕（`2026-10-22`，本机实测）**：`git for-each-ref --format='%(refname)' refs/heads refs/remotes` 后逐 ref 执行 `git ls-tree -r --name-only <ref> -- docs/changes`，并集去重；编号前缀集合 = `001…010`，其中 `004`、`005` 各出现 **2** 个不同文件名（其余各 1）。工作树另有未跟踪的 `011`–`014`，编号互不重复。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号；本项为**既有缺陷**（非本轮引入），按裁决不改写既有文件名。

### GAP-020 · `main` 的 CI 长期全红，「`CLOSED`」结论从未经过 CI
- **现状（实测 `gh run list --branch main --limit 20`，`2026-10-22T05:53Z` 采集）**：
  - **`2026-07-07T05:56:15Z` 之后连续 15 次 `failure`，无一次 `success`**：`28848964896`（`07-07T07:22`）、`29594894144`、`29596015046`、`29598908734`、`29602411164`、`29646698974`、`39589875294`、`39602099199`、`39606899421`、`39616159527`、`39626627749`、`39630625435`、`39634710946`、`39650366486`、`39651784273`（`2026-10-21T18:20:12Z`，`headSha 4fa68f9`）。
  - ⚠️ **订正一处不准确的说法**：**并非「从未通过过」**——同一次采集里 `28844933317`（`2026-07-07T05:56:15Z`）、`28655645359`、`28648530857`、`28647042143` 四次为 `success`。**准确表述是「最近一次绿在 `2026-07-07T05:56:15Z`，此后 15 次连红」**。
  - **`main` 与当前分支 `fix/issue-14-order-module` 的 CI 从未执行过同一份 `ci.yml`**：`4fa68f9`（`main` 当时的 HEAD）的 `ci.yml` 为 **111 行 / 1 个作业**（`quality`，`:27`）；当前分支为 **168 行 / 2 个作业**（`d024ef5` 才加入 `frontend`）。故「前端作业已在 CI 通过」是**分支侧**的事实，**不是 `main` 的事实**。
- **核心问题**：台账中 `GAP-003` / `GAP-004` / `GAP-006` / `GAP-008` / `GAP-009` / `GAP-010` 均标 `CLOSED`，其闭环叙述含「已通过门禁审计 / 门禁裁决」。但**这些 `CLOSED` 作出时，`main` 的 CI 一次都没绿过**——所谓「门禁」不是 CI。**`CLOSED` 与「CI 通过」在本仓从来不是同一件事，而台账的读法会让人以为它们是。**
- **与 GAP-016 / GAP-017 的边界（不得合并）**：
  - `GAP-016` = **一个具体原因**（库名含连字符导致 `internal/service` Fatal）；`GAP-017` = **另一个具体原因**（缺前端作业）。
  - `GAP-020` = **「门禁整体从未生效」这件事本身**：即使这两个原因都修好，只要 `main` 还没有一次绿，「`CLOSED` 未经 CI 验证」这个系统性缺陷就仍然存在。
  - **判据**：GAP-016/017 修完只说明「原因消除」；GAP-020 关闭需要**一次 `main` 上的成功 run**。**三者不得互相结掉。**
- **召回复核（DoD ③；仅标出，不改其状态）**：
  - 先给定**可检验的否证**（不据「CI 红」直接推断每一项都错——`main` 的红**不是**由 `CLOSED` 内容造成的）：
    - 能**直接否定**「CI 通过」这一说法的：`GAP-003`、`GAP-004`、`GAP-006`、`GAP-008`、`GAP-009`、`GAP-010`（其对应 `main` 提交的 run 全部 `failure`）。
    - 但 `main` 的失败**只被证实为** `internal/service` 的库名 Fatal（run `39651784273` 实测），**该原因与本条目的业务正确性无关**——故**不能**据此判「修复本身错了」。
  - **在变更记录里自带 `--- SKIP` 实证、因而「本地绿」本身不携带信息**（`git grep` 逐文件计数，`2026-10-22`）：
    | 台账编号 | 变更记录 | SKIP 证据 |
    | --- | --- | --- |
    | `GAP-005` | `004-fix-gap-005-cancel-expired-order-redis-recovery.md` | 7 条 `--- SKIP`（含 `TestCancelExpiredOrder_CancelsPendingOrderIdempotently`、`..._CrashRecovery_...`） |
    | `GAP-003` | `005-fix-gap-003-outbox-skip-locked.md` | 4 条（含 `TestClaimPendingOrderTimeoutOutbox_MySQLIntegration`、`TestOrderTimeoutEndToEnd`） |
    | `GAP-004` | `007-fix-gap-004-ladder-delay-queue.md` | 2 条（`TestOrderTimeoutEndToEnd`、`TestRabbitMQDelayAndDeadLetterTopology`） |
    | `GAP-010` | `009-fix-gap-010-product-cache-null-object.md` | 2 条（含 `TestGetProductByID_CachePenetrationPrevention_IntegrationMySQL`——**正是防穿透的集成断言**） |
    | `GAP-001` | `002-fix-gap-001-prededuct-timing.md` | 1 条（`..._ZeroDatabaseAccess/closed_db_connection_when_mysql_enabled`） |
  - **逐项标注（只标出，状态一律不动）**：
    - **P0 级重验（核心断言落在被 SKIP 的用例上）**：`GAP-005`、`GAP-003`、`GAP-010`、`GAP-004`。
    - **P1 级重验（主路径通过、仅边界子用例被 SKIP）**：`GAP-001`。
    - **证据充分性存疑（变更记录内既无 SKIP 也无 CI 记录，本地绿是否覆盖关键分支未可知）**：`GAP-006`、`GAP-008`、`GAP-009`、`GAP-002`、`GAP-011`、`GAP-007`。
    - **注意**：`SKIP` 条数本身是**下界**——它只统计变更记录里**贴出来的**日志行，不等于当时真实的跳过条数。
- **DoD**：
  1. **`main` 的 CI 连续通过**，并在本条目内**贴出 run id 作为证据**（形如 `gh run list --branch main --limit 1 --json databaseId,conclusion`）；在此之前本项不得判 `CLOSED`；
  2. **明确并落定判定口径**：`CLOSED` 的判定**必须以 CI 真实执行为准**，**不得以本机 SKIP 口径的绿读数替代**。本机跑绿只能作为**辅助**，须同时给出「非 SKIP 执行数」与「CI run id」两类证据；
  3. **召回复核**：完成上表所列项目的重验，逐条给出「CI run id + 该用例确为非 SKIP 执行」的证据；**在证据到位前，不放宽任何一条的状态**；
  4. **机械守卫**：让「本机绿」与「CI 绿」在台账里**不可混淆**——条目状态旁须能看出依据来自哪一侧（本机 / CI）；并配负向用例：注入一条「只有本机 SKIP 绿、无 CI run」的记录，守卫必须报警（D3）。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号；本项为 **P0**，是全部 `CLOSED` 结论的**证据基础**问题，优先于其余条目。

> ⚠️ **编号归属**：本节的 `GAP-017` 是**业务项目仓（本仓）**的编号。外层协作框架仓另有一个同名 `GAP-017`（「规则有了、机械检查没有」）——**两者互不相干、不得互引**（框架 `AGENTS.md` §6：跨仓引用编号 = 幻影引用）。
