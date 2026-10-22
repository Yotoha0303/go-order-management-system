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
| **GAP-016** | P1 | CI/门禁 | **（库名已修；DoD ①②③ 均已满足）** `4fa68f9` 的 `ci.yml:22` 库名含连字符，被 `test_helper_test.go:95` 的 `^[a-zA-Z0-9_]+$` 拒绝（`:101` `t.Fatalf`）⇒ 当时 CI 上 `internal/service` 必然全红。**残余**：本机 `RUN_MYSQL_TEST` 未设时仍全 SKIP。② 已由 run `39733467897`（`passes=358 skips=0`）与 `39758849047`（`passes=367 skips=0`）满足；③ 载体已建（`internal/service/qa_database_name_guard_test.go`，`4d37f29`）；但门禁裁决仍阻断 | [#15](https://github.com/Yotoha0303/go-order-management-system/issues/15) | IN_PROGRESS | `qa-validator` / `backend-dev` |
| **GAP-017** | P1 | CI/门禁 | **（前端作业已加，CI 已连续四轮 pass）** `4fa68f9` 的 `.github/workflows/ci.yml` 无任何前端作业，GAP-013 全部前端用例在 PR 上不执行 ⇒ 违反 `AGENTS.md` §2 03 环「新建用例 100% 纳入 CI 矩阵」；**残余**：该作业是否为 required check 未验证 | [#16](https://github.com/Yotoha0303/go-order-management-system/issues/16) | IN_PROGRESS | `sre-operator` / `frontend-dev` |
| **GAP-018** | P2 | 数据/接口契约 | 商品主键 `Product.ID` 为**自增** bigint，仍以 JSON 裸数字输出（`product.go:11`），与订单侧 `Order.ID` 的字符串约定不一致；当前不是缺陷（远未及 2^53），但商品 ID 一旦改为 Snowflake 即重演 GAP-012 | —（待建 Issue） | IN_PROGRESS | `data-engineer` / `backend-dev` |
| **GAP-019** | P1 | 文档/编号 SSOT | `docs/changes/` 中编号 `004` 与 `005` **各被两个不同文件占用**（`004-fix-gap-005-…` + `004-support-mysql-tls-connection.md`；`005-fix-gap-003-…` + `005-support-cors-and-spa-rewrites.md`）⇒ 本仓任何「编号唯一」的说法都是**假守卫** | —（待建 Issue） | IN_PROGRESS | `ssot-scribe` |
| **GAP-020** | **P0** | CI/门禁 | **`main` 的 CI 自 `2026-07-07T05:56:15Z` 之后连续 15 次全红、无一次绿**（最近 9 次集中在 `2026-10-21`）。而 `GAP-003/004/006/008/009/010` 均标 `CLOSED`、闭环叙述含「门禁裁决」——**这些 `CLOSED` 从未经过 CI 验证**，其中 5 项的变更记录里自带 `--- SKIP` 日志 | —（待建 Issue） | IN_PROGRESS | `sre-operator` / `ssot-scribe` |
| **GAP-022** | P3 | 测试质量 | `TestCreateOrder_MultipleItemsProcessedByProductID`（`order_service_test.go:406`）用「种子顺序」隐式假定 `productA.ID < productB.ID`，断言 `items[0]=A`。生产侧**确实**按 `ProductID` 排序（`order_service.go:250-254`、`:595-596`），故当前成立；但商品主键一旦改为非单调（如 Snowflake，见 GAP-018），断言会在**实现仍正确**的情况下失败 | —（待建 Issue） | IN_PROGRESS | `qa-validator` |
| **GAP-023** | P3 | 测试质量 | 同一用例内用 `Order("id ASC")` 读回 items / stock_logs（`order_service_test.go:425`、`:436`），隐含「自增 `id` 顺序 = 处理顺序」；把**排序契约**押在实现细节上，改批量插入或改主键生成即失效 | —（待建 Issue） | IN_PROGRESS | `qa-validator` |
| **GAP-024** | P2 | 测试质量 | **（已由 `4d37f29` 处置：空壳用例已删除）** 原 `TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain`（`:1703`）函数体只有一行转调 `…_ReleasesExtraPreDeduct`（`:1663`），而后者断言「重放**会再次**预扣 2 次并释放 1 次」——**用例名与断言语义相反**，且同一逻辑被执行两遍 | —（待建 Issue） | IN_PROGRESS | `qa-validator` |
| **GAP-025** | P3 | 工具链/假信号 | `internal/service` 下 **11 个**文件在工作树为 CRLF（`git ls-files --eol` 实测，其中 1 个还是 `w/mixed`），`gofmt -l` 把它们**全部**列出；但索引中这些文件**全是 `i/lf`**，且 LF 版本经 `gofmt -l` 实测**干净**——即 `gofmt -l` 在本机输出的是**行尾符噪声，不是格式缺陷**，开发者无法用它判断真实格式问题。CI 不含 `gofmt` 步骤 | —（待建 Issue） | IN_PROGRESS | `ssot-scribe` / `qa-validator` |
| **GAP-026** | P2 | 配置/运行 | **（c）推断，未实测运行期行为**：`compose.yml` 三处库名默认值 `go-order-management-system` **含连字符**（`:29` / `:64` / `:125`），与 `ci.yml:25` 刚因同一约束改成下划线形态的情形**形态不一致**；`ci.yml:22-24` 注释明写连字符是**故意**被 `safeDatabaseName`（`test_helper_test.go:95`）拒绝的（它要拼进 `CREATE DATABASE`）。CI 侧是**硬防线**，compose 侧**毫无防护**——migrate 容器是否 `exit 0` 无任何机械判据 | [#18](https://github.com/Yotoha0303/go-order-management-system/issues/18) | IN_PROGRESS | `sre-operator` / `backend-dev` |
| **GAP-027** | P2 | 测试/门禁 | `AGENTS.md` §2 03 环要求「覆盖率**配置阈值**」，而**两栈均不存在**：全仓 `git grep -i thresholds` **零命中**、`.github/workflows/` 内 `git grep -i cover` **零命中**；Go 侧 `make coverage` 只产 `coverage.out`（`Makefile:181-182`）、前端 `vite.config.ts:75-85` 的 `coverage` 块**只有 `include`/`exclude`、无 `thresholds`**，`package.json:18` 的 `test:coverage` **不在 CI**。即该准出条件**从未被持有，也从未被登记** | [#19](https://github.com/Yotoha0303/go-order-management-system/issues/19) | IN_PROGRESS | `qa-validator` / `sre-operator` |

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
     **现已修复**：提交 `d024ef5`（`2026-10-22T05:37:43Z`）把库名改为 `go_order_management_system_test`；**`a6439d2` 起 `ci.yml:25` 即为下划线形态**（`ci.yml` 自 `a6439d2` 至当前 `cb3c4ac` **未再变动**：`git diff --stat a6439d2..HEAD -- .github/workflows/ci.yml` 为空输出）。
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
  2. `RUN_MYSQL_TEST=1` 时集成用例**真绿**，不得以 SKIP 代替；且须能证明**确实执行了**（例如给出非 SKIP 的执行数，而不是只看包级 `PASS`）；**（已满足，见下方「② 的满足证据」）**
  3. **负向用例**：非法库名必须 `Fatalf` 而非静默跳过——这是当前行为，须**钉住**，防止日后有人把校验改成 skip 来换绿；**（已建载体：`internal/service/qa_database_name_guard_test.go`，由 `4d37f29` 落盘，267 行；此前本 DoD 只有「一次性手工命令」作证据，那不是门禁——详见下方「③ 的满足证据」）**
  4. 本机与 CI 使用**同一套**开关语义（`AGENTS.md` §5「本地跑的那一套 = CI 跑的那一套」）。
- **⚠️ 一次绿不等于已验证（`2026-10-22T06:07:35Z` 的 run `39732018029`）**：该 run 属 PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17)，`headSha 6fa11f8`，**两个作业全绿**（`Frontend` 1m9s / `Test, Vet, Validate, Build` 7m3s）。**但它不能作为 ② 的满足证据**：`6fa11f8` 的 `ci.yml:96` 是 `run: go test ./...`，**没有 `-v`**；不带 `-v` 时 `go test` **不打印** `--- PASS` / `--- SKIP`，故「日志里 `--- SKIP` 计数 = 0」是**因为没有打印，而不是因为没有跳过**。
  旁证（本机实测）：`internal/service` 在 `RUN_MYSQL_TEST` 未设时 `--- SKIP` = **69 条**、`--- PASS` = 24 条、包结果仍为 `ok`。**结构上「全部执行过」与「全被跳过」在包级 `ok` 下不可区分**——这正是 D3 的判据夹缝。
- **② 的满足证据（载体已建成并产生真实读数）**：提交 `a6439d2` 把 `ci.yml` 的测试步骤改为
  `set -o pipefail` + `go test -v ./... 2>&1 | tee /tmp/test.log`（`:101-102`），并新增独立步骤
  **`Assert no integration test was skipped`**（`:109`），其中 `:112-114` 统计 `--- PASS` / `--- SKIP` 并在
  `skips != 0` 时直接 `exit 1`（`:117-120`）、在 `passes == 0` 时亦 `exit 1`（防止「根本没跑」被当成绿）。
  **CI 实测（run `39733467897`，`headSha a6439d2`，`2026-10-22T06:21:51Z`，结论 `success`）**，该步骤输出：
  ```
  passes=358 skips=0
  ```
  这是**带 `-v` 的日志**上的读数（`go test -v` 才会打印 `--- PASS` / `--- SKIP`），因此
  **`skips=0` 这次是「真的没有跳过」，而不是上一轮那种「没打印所以为 0」**。**DoD ② 据此满足。**
  **读数随提交推进（须与 `headSha` 一起读）**：`a6439d2` ⇒ `passes=358 skips=0`（run `39733467897`）；
  `4d37f29` ⇒ `passes=367 skips=0`（run `39758849047`，`gh run view --log` 实取原始行 `passes=367 skips=0`）。
  **`passes` 是随提交变化的量，引用时必须带 run id 与 headSha**——单独引一个数字，下一轮就会过期（本条自身即为此前踩过的坑）。
- **③ 的满足证据（载体已建成，`4d37f29`）**：`internal/service/qa_database_name_guard_test.go`（**267 行**，已入库——`git ls-files --error-unmatch` 实测命中）。其文件头注释自述两层钉住，**两层都必要**：
  - **① 表驱动分类用例** `TestQADatabaseNameClassification`——证明正则对库名的分类正确；但**它证不了「被拒绝的名字真的会中止」**（它只看一个 bool）；
  - **② AST 钉住** `TestQADatabaseNameGuardAbortsInsteadOfSkipping`——解析 `test_helper_test.go` 源码，断言 `ensureTestDatabase` 用 `t.Fatalf` 拒绝非法名；文件头注明「Mutation-verified: changing :102 to `t.Skipf` must turn it red」。
  - **为何需要 ②**（该文件头原文的理由，与本台账 D3 同源）：若有人为了「修红 CI」把 `test_helper_test.go:102` 的 `Fatalf` 改成 `t.Skipf`，**该包全部集成用例会静默转绿且没有任何东西失败**——这正是 D3 警告的「失败即静默」。
  - ⚠️ **标注（c）**：上述「两次变异自检均变红」是**该文件自述**，本轮**未由我复跑变异**；我实测到的只有「文件已入库 + 两个顶层 Test 函数存在」。**「已入库」是实测，「变异有效」是转述。**
- **状态**：IN_PROGRESS —— 关联 [#15](https://github.com/Yotoha0303/go-order-management-system/issues/15)。
  **① ② ③ 均已满足**（② 的证据见上：run `39733467897` / `39758849047`，`skips=0`；③ 的载体见上）。
  **仍不判 `CLOSED`**：`CLOSED` 还要求**门禁裁决**放行，而本轮 `gate-auditor` 裁决仍为**阻断**；
  且 GAP-020 未解除（`main` 尚无绿 run），本条目的证据基础仍待最终确认。

### GAP-017 · CI 缺前端作业，03 环用例在 PR 上不执行
- **现状**（**历史缺陷已修，当前状态如实如下**）：
  - **缺陷发生时（`4fa68f9`，`main` 当时的 HEAD）**：`.github/workflows/ci.yml` 共 **111 行**，**只有一个作业** `quality`（`:27`），步骤全部是 Go / goose / golangci-lint / `docker build`；`node` / `npm` / `npx` 关键字命中数 **0**（`Select-String` 实测）。
  - **现已修复**：提交 `d024ef5` 新增 `frontend` 作业；**`a6439d2` 起 `ci.yml` 共 195 行 / 2 个作业（`cb3c4ac` 实测同值）**，含 `frontend`（`:146`）、`actions/setup-node@v4`、`npm ci`、`npx tsc -b`（`:186`）、`npx eslint .`、`npx vitest run --browser.headless`。
  - **CI 实测**：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Frontend (tsc, eslint, vitest browser, build)` → **pass 四轮**：run `39729484909`（`3584d1f`）**1m0s**、run `39732018029`（`6fa11f8`）**1m9s**、run `39733467897`（`a6439d2`）**56s**、run `39758849047`（`4d37f29`）**51s**。**故「前端用例在 PR 上不执行」已不成立。**
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
- **状态**：IN_PROGRESS —— 关联 [#16](https://github.com/Yotoha0303/go-order-management-system/issues/16)。**① ③ 已满足且 CI 已真跑 pass（四轮：`39729484909` / `39732018029` / `39733467897` / `39758849047`）；② 未验证**（required check 属仓外配置），故不判 `CLOSED`。

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
  - **⚠️ 分支侧 vs `main` 侧（这条区分不写清，本项会被误读成「已关闭」）**：
    - **分支侧 `fix/issue-14-order-module`：已连续四轮全绿** —— run `39732018029`（`6fa11f8`）、run `39733467897`（`a6439d2`）、run `39734691195`（`cb3c4ac`）与 run `39758849047`（`4d37f29`，`2026-10-22T09:53:04Z`，`gh run view` 实测：`Frontend` **51s**、`Test, Vet, Validate, Build` **3m31s**），四轮均为 `Frontend` + `Test, Vet, Validate, Build` **双作业 pass**。
    - **`main`：仍然一次未绿**（`gh run list --branch main --limit 3` 于 `2026-10-22T06:45Z` 复采：最近一条仍是 `39651784273` @ `4fa68f9`，`failure`）。原因很直接：**PR #17 尚未合并**，`main` 上跑的仍是那 111 行 / 1 个作业的旧 `ci.yml`。
    - **故「分支绿」不满足本项 DoD**。本项要的是 **`main` 上的一次成功 run**。
  - **`2026-07-07T05:56:15Z` 之后连续 15 次 `failure`，无一次 `success`**：`28848964896`（`07-07T07:22`）、`29594894144`、`29596015046`、`29598908734`、`29602411164`、`29646698974`、`39589875294`、`39602099199`、`39606899421`、`39616159527`、`39626627749`、`39630625435`、`39634710946`、`39650366486`、`39651784273`（`2026-10-21T18:20:12Z`，`headSha 4fa68f9`）。
  - ⚠️ **订正一处不准确的说法**：**并非「从未通过过」**——同一次采集里 `28844933317`（`2026-07-07T05:56:15Z`）、`28655645359`、`28648530857`、`28647042143` 四次为 `success`。**准确表述是「最近一次绿在 `2026-07-07T05:56:15Z`，此后 15 次连红」**。
  - **`main` 与当前分支 `fix/issue-14-order-module` 的 CI 从未执行过同一份 `ci.yml`**：`4fa68f9`（`main` 当次的 HEAD）的 `ci.yml` 为 **111 行 / 1 个作业**（`quality`，`:27`）；分支侧 `a6439d2` / `cb3c4ac` / `4d37f29` 为 **195 行 / 2 个作业 + 跳过断言步骤**（`Assert no integration test was skipped`，`:109`；`git diff --stat a6439d2..HEAD -- .github/workflows/ci.yml` 实测**空输出**，即 `ci.yml` 自 `a6439d2` 至今未再变动）。故「前端作业已在 CI 通过」「`passes=367 skips=0`」都是**分支侧**的事实，**不是 `main` 的事实**。
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
    | 台账编号 | 变更记录 | 当时 SKIP 证据 |
    | --- | --- | --- |
    | `GAP-005` | `004-fix-gap-005-cancel-expired-order-redis-recovery.md` | 7 条 `--- SKIP`（含 `TestCancelExpiredOrder_CancelsPendingOrderIdempotently`、`..._CrashRecovery_...`） |
    | `GAP-003` | `005-fix-gap-003-outbox-skip-locked.md` | 4 条（含 `TestClaimPendingOrderTimeoutOutbox_MySQLIntegration`、`TestOrderTimeoutEndToEnd`） |
    | `GAP-004` | `007-fix-gap-004-ladder-delay-queue.md` | 2 条（`TestOrderTimeoutEndToEnd`、`TestRabbitMQDelayAndDeadLetterTopology`） |
    | `GAP-010` | `009-fix-gap-010-product-cache-null-object.md` | 2 条（含 `TestGetProductByID_CachePenetrationPrevention_IntegrationMySQL`——**正是防穿透的集成断言**） |
    | `GAP-001` | `002-fix-gap-001-prededuct-timing.md` | 1 条（`..._ZeroDatabaseAccess/closed_db_connection_when_mysql_enabled`） |
  - **原逐项标注（只标出，状态一律不动）**：
    - **P0 级重验（核心断言落在被 SKIP 的用例上）**：`GAP-005`、`GAP-003`、`GAP-010`、`GAP-004`。
    - **P1 级重验（主路径通过、仅边界子用例被 SKIP）**：`GAP-001`。
    - **证据充分性存疑（变更记录内既无 SKIP 也无 CI 记录，本地绿是否覆盖关键分支未可知）**：`GAP-006`、`GAP-008`、`GAP-009`、`GAP-002`、`GAP-011`、`GAP-007`。
    - **注意**：`SKIP` 条数本身是**下界**——它只统计变更记录里**贴出来的**日志行，不等于当时真实的跳过条数。

#### 召回重验·回填（`2026-10-22`，**E-14 的处置**）

**证据（a）实测**：CI run **`39758849047`**（`headSha 4d37f29`，`2026-10-22T09:53:04Z`，结论 `success`）。跳过断言步骤原始输出 **`passes=367 skips=0`**（`gh run view 39758849047 --log` 实取）；且**逐条点名**在日志中查到 `--- PASS` 行，**不是靠聚合数字演绎**：

| 台账编号 | 被点名的用例（原 SKIP 行） | 本次 CI 读数（`39758849047` @ `4d37f29`） |
| --- | --- | --- |
| `GAP-005` | `TestCancelExpiredOrder_CancelsPendingOrderIdempotently` | `--- PASS ... (0.14s)` |
| `GAP-003` | `TestClaimPendingOrderTimeoutOutbox_MySQLIntegration` | `--- PASS ... (0.32s)` |
| `GAP-003` / `GAP-004` | `TestOrderTimeoutEndToEnd` | `--- PASS ... (0.50s)` |
| `GAP-004` | `TestRabbitMQDelayAndDeadLetterTopology` | `--- PASS ... (0.41s)` |
| `GAP-010` | `TestGetProductByID_CachePenetrationPrevention_IntegrationMySQL` | `--- PASS ... (0.08s)` |
| `GAP-001` | `TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess/closed_db_connection_when_mysql_enabled` | `--- PASS ... (0.09s)`（子用例逐条 PASS，含该具名子用例） |

**结论：上表 5 项（P0 ×4 + P1 ×1）的「被 SKIP」问题已在 CI 上解除** —— 上述用例在 CI 上**确为非 SKIP 执行，且通过**。

##### 该证据能证明到什么程度（**不得多证**）

1. **能证明**：这些用例在 CI 上**被执行了**（`--- PASS` 行存在，非 `--- SKIP`；且该 run 的断言步骤在 `skips != 0` 时必然 `exit 1`，而该步骤 `success`）；
2. **能证明**：它们在**该次运行的环境中通过**（`success`）；
3. **能证明**：此前「本地绿建立在 SKIP 上、CI 从未执行过它们」这一**证据缺陷**已消除——`SKIP` 与 `PASS` 现在是**可区分**的（这正是 `a6439d2` 加 `-v` + 断言步骤的目的）。

##### 该证据**不能**证明什么（本轮教训的核心）

1. **不能证明这些用例的断言本身正确。** 用例通过只说明「它与当前实现一致」，**不说明它断言的是正确的行为**。**本轮实例（实测）**：`GAP-024` 的空壳用例 `TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain` 在 run `39734691195`（`cb3c4ac`）中是
   `--- PASS: TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain (0.13s)` —— 它**真的执行了、也真的通过了**，但**它声称守护的性质（「重放不重复预扣」）根本不存在**（其函数体只转调另一用例，断言的是「重放会二次预扣再释放」）。**这是「执行了 ≠ 判据有效」的实证**：绿着，却没有那条保护。
   ⚠️ **订正一处转述（勿再沿用）**：本轮曾以 `TestRebuildRedisInventoryStock_Success` 作为「此前也是『执行且通过』而断言是错的」的例证——**该前提与实测不符**。实测：该用例在断言未修时的 run `39729484909`（`3584d1f`）中读数是 **`--- FAIL: TestRebuildRedisInventoryStock_Success (0.11s)`**，**不是 PASS**；它恰是**被 CI 抓出来的**（`a6439d2` 的 `-v` + 断言步骤使 `internal/service` 真跑），修好后在 run `39734691195` 才转为 `--- PASS`。
   **故准确表述是**：该错误断言**从未「执行且通过」**——它此前**根本没被执行**（本机 SKIP、CI 因库名 Fatal），属于「潜伏的假绿」；**而「执行了却仍无信息」的实例是 GAP-024 空壳用例**。两类问题不同，不得混为一谈。
2. **不能证明这些用例覆盖了该 GAP 的完整 DoD。** 用例通过只说明它断言的那条性质成立；它**没有断言**的性质（如 `GAP-001` 的释放补偿链路、`GAP-003` 的租期互斥在真并发下的表现）仍然无判据。
3. **不能证明 `CLOSED` 成立。** 本条 DoD ① 明写「`main` 的 CI 连续通过」——这是**分支侧**的 run，`main` 仍无绿 run。**分支绿 ≠ `main` 绿**（本项开篇即已区分）。
4. **不能把「当时的 SKIP」判为「当时的结论错误」。** 本次回填只解除**证据缺陷**；各 `CLOSED` 条目的**业务正确性**既未被本次证实、也未被本次否证——那需要各自的 DoD 逐条重验（DoD ③ 的其余部分）。
5. **不能外推到时点**。本读数只对 `4d37f29` 有效；`passes` 是随提交变化的量（`358` @ `a6439d2` → `367` @ `4d37f29`）。**引用时必须带 run id 与 headSha。**
- **DoD**：
  1. **`main` 的 CI 连续通过**，并在本条目内**贴出 run id 作为证据**（形如 `gh run list --branch main --limit 1 --json databaseId,conclusion`）；在此之前本项不得判 `CLOSED`；
  2. **明确并落定判定口径**：`CLOSED` 的判定**必须以 CI 真实执行为准**，**不得以本机 SKIP 口径的绿读数替代**。本机跑绿只能作为**辅助**，须同时给出「非 SKIP 执行数」与「CI run id」两类证据；
  3. **召回复核**：完成上表所列项目的重验，逐条给出「CI run id + 该用例确为非 SKIP 执行」的证据；**在证据到位前，不放宽任何一条的状态**；
     **（`2026-10-22` 部分回填：上表 5 项已给出 run `39758849047` @ `4d37f29` 的逐条点名 `--- PASS` 证据，见上方「召回重验·回填」。**
     **该回填只解除「被 SKIP」这层证据缺陷**；「证据充分性存疑」那 6 项（`GAP-006/008/009/002/011/007`）**仍未回填**，且**各 `CLOSED` 的断言正确性一律未被本次证明**——见方法论第 5 条。）**
  4. **机械守卫**：让「本机绿」与「CI 绿」在台账里**不可混淆**——条目状态旁须能看出依据来自哪一侧（本机 / CI）；并配负向用例：注入一条「只有本机 SKIP 绿、无 CI run」的记录，守卫必须报警（D3）。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号；本项为 **P0**，是全部 `CLOSED` 结论的**证据基础**问题，优先于其余条目。
  **DoD ③ 已部分回填**（上表 5 项，证据 run `39758849047`）；**DoD ①②④ 未满足**，故不判 `CLOSED`。
  **不得因分支侧四轮全绿而降级**：DoD ① 明写「`main` 的 CI 连续通过」，**PR 合并前不可能满足**。
- **⚠️ 方法论（本轮第三次踩到，故写死在台账里）**：**没有真跑证据时，结论必须标注为「推断」，不得与实测读数并列书写。**
  本轮已发生三次，形态完全相同——**静态读码 / 读配置得出一个听起来合理的因果，被真跑或原文推翻**：
  1. **`qa-validator`**：据 `product_handler.go:87-88` 的 `(0,0)` 推断 `TestListProducts_Pagination` 应为 P1 缺陷；实际该用例**不经 handler**，直接传 `PageSize: 2`，分页分支生效、断言正确（详见下方 `GAP-021` 撤销说明）。
  2. **`ssot-scribe`（我）**：曾把 `4fa68f9` 的 `ci.yml` 记为「90 行」——那是 `Measure-Object -Line` 的**非空行计数**，真实 `Count` 为 **111**。**口径写错，数字就错，而不报错**。
  3. **总控**：据最近 20 条 run 推断 `main` 的 CI「从未通过」；实际为「最近一次绿在 `2026-07-07T05:56:15Z`，此后 15 连红」。
  **可执行的判据**：凡台账条目中的结论，须能指出它属于哪一类——
  **（a）实测**（附命令与 stdout / run id）· **（b）原文引用**（附 `文件:行号`）· **（c）推断**（必须显式写「推断」二字，且**不得**作为 `CLOSED` 的依据）。
  4. **门禁裁决必须落痕（`2026-10-22` 补；来源：`gate-auditor` 第二轮的复核失败）** —— **注意：本条与上面第 1~3 条属不同族**（1~3 是「推断冒充实测」，本条是「结论没有证据载体」），同属 D1/D3 的证据纪律，故并入本节而不单开条目。
     - **实测形态（a）**：`git grep -i gate-auditor -- docs` 的命中**全部是一句话转述**（如 `GAP-015` 的「本轮 `gate-auditor` 裁决为**阻断**」、`GAP-003/004/006/009/010` 的「通过 `gate-auditor` 转段门禁裁决」），**没有一份裁决原文**（阻断项清单 / 解除条件 / 所审 HEAD 三者齐备的载体）。
     - **后果已经发生**：第二轮审计**无法复核自己上一轮列出的 E-4~E-8**，只能记「无法判定（缺原文）」——**上一轮的阻断项因此既无法被解除，也无法被确认已解除**。**与 1~3 条同属一条根纪律（D1：读到才算）**：那三条是「结论没有实测算」，本条是「结论没有证据载体」，**都不满足「读到才算」**。
     - **判据（可证伪）**：任一判 `CLOSED` 的条目，须能反查到**承载其裁决原文的载体**（裁决结论 + 阻断项 + 解除条件 + UTC 时点 + 所审 HEAD）；**反查不到 ⇒ 该 `CLOSED` 的「已过门禁裁决」不成立**。**只写「已通过裁决」不算落痕——那是结论，不是证据。**
     - **边界（不新建文件）**：裁决原文随该轮所属的 `docs/changes/NNN-*.md` 或本台账对应条目落盘，**不为此新建文件**。**未入库的运行态记录（memory / record 类）不得作为唯一载体**——它们不进版本库，下一轮与外部读者都够不到（同 D4「仓库内 > 仓库外」）。
  5. **「执行了」≠「判据有效」（`2026-10-22` 补；本轮最重要的教训，与第 4 条同族）** —— **注意：这条是「证据强度」问题，与第 1~3 条（推断冒充实测）不同**，故与第 4 条并列。
     - **形态**：一个用例**真的执行了、也真的通过了**，但**它断言的命题是错的**。此时「绿」不携带信息——它证明的是「这条断言与当前实现一致」，**不是**「这条断言描述了正确的行为」。
     - **本轮实例（a）**：`GAP-024` 的空壳用例 `TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain` 在 run `39734691195`（`cb3c4ac`）中 **`--- PASS (0.13s)`**——**真的执行了、也真的通过了**，但**它声称守护的性质（「重放不重复预扣」）根本不存在**（函数体只转调另一用例，而后者断言的是「重放会二次预扣再释放」）。绿着，却没有那条保护。
       ⚠️ **不得再沿用的一处例证（本轮已订正）**：`TestRebuildRedisInventoryStock_Success` **不是**「执行且通过而断言错」的实例——实测其在 run `39729484909`（`3584d1f`）为 **`--- FAIL`**，即它是**被执行后抓出来的**，修好后（`6fa11f8`）才转 PASS。它属「**此前根本没执行**」那一族（本机 SKIP + CI 库名 Fatal），**与「执行了却无信息」是两类问题**。
     - **为何这条必须写进方法论**：本轮「把召回重验表回填为『已在 CI 验证』」这一动作，**只解除了「被 SKIP」这层证据缺陷**；若把它读成「这些 `CLOSED` 的判据因此被证明有效」，就会**重演同一个错误**——**换了一个更硬的读数，做同一个推理跳跃**。`skips=0` 比「没打印的 0」硬，但它**同样不检查断言的正确性**。
     - **判据（可证伪）**：凡以「用例通过」为据的结论，须能回答**该用例断言的是哪条命题**、以及**该命题被谁定义**（DoD / 契约 / ADR 的哪一行）。
       答不出 ⇒ 该「通过」**只能作为「执行证据」，不得作为「正确性证据」**。
     - **可操作推论**：`passes=N` 是**计数**，不是**证明**。计数回答「跑了多少」，不回答「断言对不对」。
       **引用计数时不得省略 run id 与 headSha**（`358` @ `a6439d2` → `367` @ `4d37f29` 已实测变化）。
- **⚠️ 本轮追加的一条实测反例（关于「SKIP=0」这类读数的有效性）**：run `39732018029`（`6fa11f8`）**两作业全绿**，若据「日志里 `--- SKIP` 计数 = 0」判定「集成用例全部真跑」，**该判定无效**——`6fa11f8` 的 `ci.yml:96` 是 `run: go test ./...`，**无 `-v`**；不带 `-v` 时 `go test` **根本不打印** `--- PASS` / `--- SKIP` 行（本机已实测该行为）。**故「SKIP=0」只说明「没打印」，不说明「没跳过」**。这正是 D3 的「判据夹缝」：一个恒为零的读数看起来永远合格。解除方式见 DoD ④。
  **订正（实测 `2026-10-22T06:45Z`，HEAD `cb3c4ac`）**：本条此前曾写「且工作树已在补 `-v` + `set -o pipefail` + 断言 `skips -eq 0` 的步骤（**尚未提交**）」——**该表述已过期且会误导人**。实测：
  - `git show a6439d2 --stat` ⇒ `.github/workflows/ci.yml` **+28 / −1**，提交标题 `ci: fail the build when integration tests are skipped (GAP-016)`；
  - `git show HEAD:.github/workflows/ci.yml` ⇒ `Run test` 步骤为 `set -o pipefail` + `go test -v ./... 2>&1 | tee /tmp/test.log`，其下为**独立步骤** `Assert no integration test was skipped`，含 `skips -ne 0` 与 `passes -eq 0` **双出口** `exit 1`；
  - 该改动已在 run `39733467897`（`headSha a6439d2`，`success`）**生效**，读数 `passes=358 skips=0`（满足证据见上方 `GAP-016`）；
  - `git diff --stat a6439d2..HEAD -- .github/workflows/ci.yml` ⇒ **空输出**，即 `ci.yml` 自 `a6439d2` 至今未再变动。
  故「**尚未提交**」**不成立**；本段与同节上半部分（`GAP-016` 的「② 的满足证据」）表述一致。

> ⚠️ **编号归属**：本节的 `GAP-017` 是**业务项目仓（本仓）**的编号。外层协作框架仓另有一个同名 `GAP-017`（「规则有了、机械检查没有」）——**两者互不相干、不得互引**（框架 `AGENTS.md` §6：跨仓引用编号 = 幻影引用）。

### GAP-021 · 【已撤销】`TestListProducts_Pagination` 分页断言

> **`GAP-021` 已撤销：经总控复核为误判。编号保留、不复用**（留此行以免后人考古时以为漏号）。

- **原报告**：`qa-validator` 判 `internal/service/product_service_test.go:280 TestListProducts_Pagination` 为 P1——理由是 `dao.ListProducts` 只在 `pageSize>0` 时分页（`product_dao.go:21-26`），而 handler 空参返回 `(0,0)`（`product_handler.go:87-88`），「0 = 不分页」只存在于 dao，故断言 `len==1` 应失败。
- **撤销依据（逐条复核）**：
  1. **调用路径读错**：该用例**不经过 handler**。原文 `product_service_test.go:286-288` 为
     `productSvc.ListProducts(ctx, request.ListProductsRequest{Page: 2, PageSize: 2})`——**`PageSize=2 > 0`**，
     故 `product_dao.go:21` 的分页分支**确实生效**；3 条商品、第 2 页每页 2 条 ⇒ `len==1`、`total==3`，断言**正确**。
  2. **`(0,0)` 是另一条路径**：`parseOptionalPagination` 返回 `(0,0)` 只在**两个查询参数都缺省**时发生，
     语义是「不带分页参数 = 返回全部」（`product_dao.go:30-31` 亦配合把 `total` 置为 `len`），**与该用例无关**。
  3. **真跑旁证**：该用例属 `internal/service`，在 CI 的 `go test ./...` 中执行且整体 `ok`。
- **状态**：**已撤销**（非 `CLOSED`——它**不是缺口**，从未成立）。

### GAP-022 · 用例依赖「种子插入顺序」而非显式排序契约
- **现状**：`internal/service/order_service_test.go:406 TestCreateOrder_MultipleItemsProcessedByProductID`——先 `seedProduct` 建 A、再建 B（`:408-409`），随后以**请求顺序 B 在前**下单（`:416-417`），最后断言 `items[0].ProductID == productA.ID`（`:431`）、`logs[0].ProductID == productA.ID`（`:442`），失败信息写「expected … processed by product_id asc」。
- **关键事实（必须与实现对齐，否则会把它误判成生产缺陷）**：生产侧**确实**按 `ProductID` 升序处理——
  `internal/service/order_service.go:250-254`（预扣去重排序）与 `:595-596`（加锁顺序）均为 `sort.Slice(... ProductID < ...)`。
  故本用例**当前是正确的**：它在验证「与请求顺序无关、按 product_id 升序」这一真实契约。
- **风险**：**契约本身没问题，脆的是「A 的 ID 一定小于 B」这个前提**——它由 `seedProduct` 的插入顺序 + `products.id` 自增共同保证。一旦商品主键改为非单调（如 Snowflake，见 `GAP-018`），`productA.ID < productB.ID` 不再必然成立，用例会在**实现仍然正确**的情况下失败（假红）；反之若有人改动排序实现却又恰好沿用同一顺序，用例也可能**假绿**。
- **DoD**：
  1. 不再依赖插入顺序生成「更小 / 更大」的 ID——显式构造两个已知 ID 的种子（或断言时按 `ProductID` 排序后的语义而非硬编码 A/B 位置）；
  2. 至少补一条**反向**用例：请求顺序与 `ProductID` 顺序**相反**（现已如此，须保留并加注释说明意图）；
  3. 用例注释写明「本用例守护的是 `order_service.go:250-254 / :595-596` 的升序契约」，避免它被当成「顺序无关」的普通断言。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号。P3（纯测试健壮性，不影响生产行为）。

### GAP-023 · 用例以 `Order("id ASC")` 把处理顺序押在自增主键上
- **现状**：`order_service_test.go:425`（`Order("id ASC")` 读回 `order_items`）与 `:436`（同法读回 `stock_logs`）——两者都用**自增 `id` 的顺序**代表「处理顺序」，再据此断言 `[0]=A, [1]=B`。
- **风险**：`id` 是**插入序**，不是契约序。若将来改成批量插入、改用非自增主键、或引入重试写入，`id` 顺序与业务处理顺序脱钩，断言即失真。它把「按 `product_id` 升序」这条**业务契约**悄悄替换成了「按 `id` 升序」这条**实现细节**。
- **同类观察（原 `C3`「隐式前置无注释」，并入本条，不单开编号）**：该用例还依赖若干**未在注释中声明**的隐式前置——商品必须处于 `ProductStatusOnSale`、库存必须已 `seedInventory`、用户必须已由 `setupTestDB` 种入。这些前提**只存在于代码里**，读者从用例名看不出来；与「依赖 `id` 顺序」同属**测试依赖隐式前提**这一族。**不为此单开编号**：它纯属文档级、无独立判据，单开会稀释编号空间（总控裁定）。
  > ⚠️ **`2026-10-22` 订正**：此句原写「**不单开 `GAP-026`**」。该写法在当时即已不妥（它把一个**未使用的编号**写进了正文），而在 `GAP-026` 被 `compose.yml` 库名条目占用后，会变成**指向另一条目的幻影引用**（同 `GAP-019` 的歧义形态）。故改为不带编号的表述。**`GAP-026` 的归属以登记表为准，与本句无关。**
- **DoD**：
  1. 断言的目标顺序改为**显式**依据（按 `product_id` 排序后再比对，或直接断言集合与关键字段），不再借 `Order("id ASC")` 表达语义；
  2. 若确实需要验证「写入顺序」，则须在注释中写明它验证的是**存储顺序**而非**业务处理顺序**，两者不得混用同一条断言；
  3. （承接 `C3`）用例开头补齐隐式前置**清单式注释**：所需商品状态 / 库存 / 用户种子，缺一即失败——让「前提」从代码里显形到文字上。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号。P3。

### GAP-024 · 用例名与断言语义相反，且重复执行同一逻辑
- **现状**：`order_service_test.go:1703 TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain` 的**函数体只有一行**：转调 `TestCreateOrder_IdempotentReplay_ReleasesExtraPreDeduct(t)`（`:1704`）。而后者（`:1663-1701`）断言的是
  `len(guard.preDeducts) == 2`（`:1695`）与 `len(guard.releases) == 1`（`:1698`）——即**重放会再次预扣，然后把多扣的那次释放掉**。
- **风险（这是本组里唯一可能掩盖真实缺失的一条）**：用例名声称「重放不再预扣 Redis」，**断言却证明相反**。读者按名字检索时会以为「不重复预扣」这条性质**已有测试守护**，实际**没有**——而它恰恰是幂等链路的**主链路**性质。这属于「该测的没测，但看起来测了」，故**标 P2**（其余三条 P3）。
- **DoD**：
  1. 二者取一：**删掉**这个纯转调的空壳（把语义交由 `_ReleasesExtraPreDeduct` 承载），或**改名**为与其断言一致的表述（如 `TestCreateOrder_IdempotentReplay_PreDeductsTwiceThenReleases`）；
  2. 若业务上**确实**要求「重放不得二次预扣」，则须**新增**一条真正断言 `len(guard.preDeducts) == 1` 的用例——**名字不能代替断言**；
  3. 不得保留「一个用例函数体内只调用另一个用例」的写法：`go test` 会把同一逻辑执行两遍，且失败时 `t` 归属混乱、报告计数虚高。
- **处置（`4d37f29`，`2026-10-22T09:53Z` 前的提交，已入库）**：
  - **走 DoD ① 的第一条路：删除，不是改名。** 实测：`git show 4d37f29 -- internal/service/order_service_test.go` 删除 `func TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain` 及其**唯一的转调行**；`git grep "TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain"` ⇒ **零命中**（`exit 1`）。语义交由 `TestCreateOrder_IdempotentReplay_ReleasesExtraPreDeduct`（`order_service_test.go:1684`）承载。
  - **DoD ② 以「契约澄清」而非「新增用例」结掉**：`4d37f29` 在该用例上补了 14 行契约注释，写明**架构上就该「先预扣、后补偿」**——重放**确实**会二次预扣，事务回滚后在事务外释放；并明确「**不得**在不改架构的前提下新增断言 `len(preDeducts)==1` 的用例」。故 DoD ② 的条件句（「若业务上**确实**要求重放不得二次预扣」）**不成立**，无需新增用例。
  - **DoD ③ 已满足**：空壳删除后不再存在「一个用例函数体只调用另一个用例」的写法。
  - **顺带纠正一处计数，并给出可复核的差额分解**：本项此前导致 `passes` 计数虚高（同一逻辑执行两遍）；删除后该重复不再计入。`passes` 由 **358**（`a6439d2`）升至 **367**（`4d37f29`），**净增 9**，**恰好等于该提交的增删差**：
    - **删 1 个顶层用例**：`TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain`（`git show 4d37f29 -- internal/service/order_service_test.go` 实测 `-func Test`）；
    - **增 2 个顶层用例 + 8 个子用例**：`TestQADatabaseNameClassification`（**表驱动，8 个子用例**：`lowercase` / `upper_and_digits_and_underscore` / `digits_only` / `hyphen_is_rejected` / `space_is_rejected` / `statement_injection_is_rejected` / `empty_is_rejected` / `backtick_escape_is_rejected`）与 `TestQADatabaseNameGuardAbortsInsteadOfSkipping`（run `39758849047` 日志逐行实取）；
    - **10 − 1 = +9** ⇒ `358 + 9 = 367`，**闭合**。
    **不得**把 367 读成「本项修复后仍为 358」；也**不得**把净增归因于本项（净增全部来自新增的 guard 用例）。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号。P2。**DoD ①②③ 均已满足**（处置见上）；
  **仍不判 `CLOSED`**：按状态枚举，`CLOSED` 还要求**门禁裁决**放行，而本轮 `gate-auditor` 裁决仍为**阻断**——缺的是**裁决这一环**，不是代码或测试（同 `GAP-012` / `GAP-015` / `GAP-016` 的处置口径）。

### GAP-025 · `internal/service` 的 CRLF 让 `gofmt -l` 变成恒噪声
- **现状（本机实测，`2026-10-22`）**：
  - `gofmt -l internal/service/` 列出 **11 个**文件：`authorization_service.go`、`inventory_interface.go`、`operation_log_interface.go`、`operation_log_service.go`、`order_interface.go`、`service_error.go`、`stock_log_interface.go`、`stock_log_service.go`、`test_helper_test.go`、`user_service.go`、`user_service_test.go`。
  - `git ls-files --eol internal/service/` 显示这些文件多为 `w/crlf`（`inventory_interface.go` 为 **`w/mixed`**），而索引侧**全部是 `i/lf`**；`.gitattributes` 为 `* text=auto eol=lf`。
  - **决定性证据**：把 HEAD 的 blob 写成纯 LF 后再跑 `gofmt -l` ⇒ **全部干净（无输出）**；对 CRLF 文件跑 `gofmt -d`，其 diff 的 `-` 行与 `+` 行**内容完全相同**（仅行尾符不同）。
- **结论（必须与「gofmt 报格式问题」区分开）**：**不存在格式缺陷**；`gofmt -l` 报出的是**行尾符噪声**。因此：
  - 「若 CI 跑 `gofmt` 会全红」——**在 CI 上不成立**（CI 检出的是索引里的 LF 内容，已实测干净）；
  - 真正的问题是**开发者本机**：`gofmt -l` 永远有输出 ⇒ 无法用它判断真实格式问题，**这是一个恒为「红」的假信号**，与 D3 的「判据无牙」互补（一个是永远绿，一个是永远红，同样不携带信息）。
- **风险**：低（不影响构建与运行）。但会持续消耗开发者的判读成本，并诱使人们把「gofmt 有输出」当成常态而忽略**真正的**格式漂移。
- **DoD**：
  1. 选择一种解并落定：**(a)** 把 `internal/service` 下 11 个文件统一转 LF（工作树侧，一次性，索引已为 LF 故无内容变更）；或 **(b)** 在 CI 增加 `gofmt -l` 检查（CI 侧天然为 LF，可直接作为**真**判据）；
  2. **(b) 与 (a) 建议同时做**：只有 (b) 无 (a)，本机噪声仍在；只有 (a) 无 (b)，缺少机械守卫；
  3. 若增加 `gofmt -l` 步骤，须配**负向用例**：故意注入一处未格式化代码，CI 必须红（D3：不注入故障就不算验证过）；
  4. 顺带修正 `inventory_interface.go` 的 `w/mixed`（混合行尾是最容易被工具误判的形态）。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号。P3。

### GAP-026 · `compose.yml` 库名默认值含连字符，与 CI 侧同一约束不一致
- **现状（实测，`2026-10-22T06:45Z`，HEAD `cb3c4ac`）**：
  - `compose.yml:29`（`app.environment.DB_NAME`）、`:64`（`mysql.environment.MYSQL_DATABASE`）、`:125`（`migrate.environment.DB_NAME`）**三处**默认值均为 `${DB_NAME:-go-order-management-system}` —— **含连字符**。`:133` 的 migrate 命令把 `$${DB_NAME}` 直接拼进 DSN。
  - **同一仓库的同一约束，另一侧刚被修过**：`ci.yml:25` 的 `MYSQL_TEST_DATABASE` 为 `go_order_management_system_test`（**下划线**），`ci.yml:22-24` 的注释原文：
    「Must stay `[a-zA-Z0-9_]+` — internal/service/test_helper_test.go enforces safeDatabaseName and t.Fatalf's otherwise. Hyphens are rejected there on purpose (it is a SQL identifier interposed into CREATE DATABASE).」
  - **口径不一致**：`config.yml:8`（`database: go_order_management_system`）与 `Makefile:16`（`DB_NAME ?= go_order_management_system`）都是**下划线**；只有 `compose.yml` 三处是**连字符**。
- **风险（必须按方法论分类书写）**：本条主体属 **(c) 推断**——**未实测** GORM / MySQL 是否接受含连字符的库名（`CREATE DATABASE` 与 DSN 两条路径均未真跑）。**不得据此声称「compose 下 migrate 必然失败」**。
  - **已实测的部分（a）**：三处默认值形态为连字符（`Select-String` 实跑）；`ci.yml:22-25` 的硬防线与注释原文（`git show HEAD:.github/workflows/ci.yml` 实跑）。
  - **仍然成立的风险是「防护不对称」**：CI 侧有 `safeDatabaseName` + `t.Fatalf` 的硬防线（**负向行为有牙**，GAP-016 已实测），而 compose 侧**没有任何校验**——库名一旦非法，**失败会发生在容器内**（migrate 容器 `exit != 0`），而 `AGENTS.md` §2 05 环的准出命令只跑 `docker compose config -q`（**纯语法校验**，`Makefile:189-190` 的 `compose-config` 同），**不执行任何容器**。故这是 D3 的「判据夹缝」形态：**语法合法 ≠ 能起来**。
- **DoD**：
  1. **合并前实测一次**：`docker compose up` 并确认 `migrate` 容器 **`exit 0`**（本机 Docker 可用：`Docker Compose version v5.6.0`；`compose.yml:49-50` 中 `app` 依赖 `migrate: condition: service_completed_successfully`，故 migrate 失败会使整栈起不来）。若 `exit 0`，本条降级为「形态一致性问题」，仍须做 ②；
  2. **或**把三处默认值改为**下划线**形态（如 `go_order_management_system`），并在 `compose.yml` 注释里写明**理由**（与 `ci.yml:22-24` 同一理由：库名是拼进 SQL / DSN 的标识符）；`Makefile:16` 的 `DB_NAME`（供 `make migrate-*` 拼 `MIGRATION_DSN` 使用）**已是下划线**，改 compose 即可两边一致；
  3. **负向用例（D3）**：显式 `DB_NAME=go-order-management-system` 跑一次，确认失败**显式可见**（migrate 容器非零退出 / 错误日志），而不是静默降级或被 `config -q` 放行；
  4. 若最终决定**保留连字符**并证明可运行，须在条目里写明实测证据，并把「为什么 compose 可以用而 CI 不能」讲清（两者用库名的路径不同：CI 是 `CREATE DATABASE` 标识符，compose 是预先建库 + DSN）——**不得**只把默认值改掉而不解释。
- **状态**：IN_PROGRESS —— 关联 [#18](https://github.com/Yotoha0303/go-order-management-system/issues/18)。P2。
  ⚠️ **两处同步义务（D4）**：本条正文与 Issue #18 正文**承载同一份实测证据**（三处默认值 / `ci.yml:22-25` 注释原文 / 与 `config.yml:8`、`Makefile:16` 的形态对比）——**任一方改动，另一方必须同步**，否则立即产生两处不同成熟度的事实。

### GAP-027 · 两栈均无覆盖率配置阈值，03 环该准出条件从未被持有
- **现状（实测，`2026-10-22T06:45Z`，HEAD `cb3c4ac`）**：
  - **判据一（零命中）**：`git grep -i thresholds`（全仓、跟踪文件）⇒ **`exit 1`、零输出**。
  - **判据二（零命中）**：`git grep -i cover -- .github/workflows/` ⇒ **`exit 1`、零输出**——CI **从不生成也从不检查**覆盖率。
  - **Go 侧形态**：`Makefile:181-182` 的 `coverage` 目标只执行 `go test -covermode=atomic -coverprofile=coverage.out`，**无地板值、无比较步骤**；`Makefile:185` 的 `coverage-html` 仅生成报告。`make ci`（`Makefile` 末尾）**不含** `coverage`。
  - **前端侧形态**：`fronted/vite.config.ts:75-85` 的 `test.coverage` 块**只有 `exclude`**（另有一行被注释掉的 `include`），**无 `thresholds`**；`fronted/package.json:18` 的 `test:coverage` 脚本存在（`vitest run --coverage --browser.headless`），但 CI 的 `Unit and browser tests` 步骤（`ci.yml:191-192`）跑的是 `npx vitest run --browser.headless`——**不带 `--coverage`**。
  - **要求出处**：`AGENTS.md` §2 03 环准出条件原文为「新建用例 **100% 纳入 CI 矩阵**（拒绝门外用例）+ 覆盖率**配置阈值**」。前半句已由 GAP-017 承接，**后半句从未被承接**：既不存在，也未登记。
- **风险**：与 GAP-017 / GAP-020 同族——**准出条件写在规则里、却不落在任何判据上**。它的失效形态是「无声」：没有阈值时，覆盖率下滑不会有任何红点；而团队会以为 03 环已经过了（因为它「看起来」有 `make coverage` 与 `test:coverage` 两个入口）。**入口存在 ≠ 门槛存在。**
- **为何不能凭感觉设地板值**：地板值的功能是「**防回退**」，不是「评优劣」。它必须钉在**当前实测水位**上（否则第一天就红，规则随即被人为绕过——绕过比没有更糟），并且要能区分「真的掉下来」与「地板设错了」。凭感觉取一个整数（如 80%）无法回答任何一个问题。
- **DoD**：
  1. **两栈各出一份实测水位**（命令与 stdout 一并留痕）：
     - Go：`make coverage` 后 `go tool cover -func=coverage.out | tail -1` 取总覆盖率；并注意 `AGENTS.md`（框架）§5 的既定教训——**读数必须取自 LF 工作树**，CRLF 会虚高（本仓 GAP-025 已实测 `internal/service` 工作树为 CRLF）；
     - 前端：`cd fronted && npx vitest run --coverage --browser.headless` 取 `All files` 行；**须先确认 `vite.config.ts:76` 的 `include` 注释意味着什么**——当前只统计「被测试触达的文件」，未测模块**根本不进分母**，这会系统性**高估**覆盖率（该注释原文即「Uncomment to expand the report to all src/**/* so untested modules appear as 0% coverage」）。**地板值应在决策 `include` 口径之后才定**，否则定的是一个假水位。
  2. **地板值 = 当前实测水位向下取整到整数**（不得高于实测值），并写明**采集时点 + 命令 + 原始读数**；地板值写在配置里（Go：`Makefile` 的目标内比较，或独立脚本；前端：`vite.config.ts` 的 `test.coverage.thresholds`）；
  3. **进入 CI**：新增覆盖率步骤，`AGENTS.md`（框架）§5「本地跑的那一套 = CI 跑的那一套」——本地命令与 CI 命令**逐字相同**；且**缺前置就失败，不得静默跳过**（无 `coverage.out` 时须报错，不得当作 0 通过）；
  4. **负向用例（D3）**：**注入一次覆盖率下降**（删掉一个包/一个模块的用例，或把地板临时抬高 1 个点），门禁**必须红**；并给出「空分母」用例——被测集为空时**必须失败**，不得报 100%（框架 `AGENTS.md` §1 D3 记录过真实案例：空分母被当成 100%）。**这两条缺一即不算验证过。**
  5. 地板值的**后续调整**须走记录（`docs/changes/`），不得静默下调——下调地板等于放弃防回退。
- **状态**：IN_PROGRESS —— 关联 [#19](https://github.com/Yotoha0303/go-order-management-system/issues/19)。P2。
  ⚠️ **两处同步义务（D4）**：本条正文与 Issue #19 正文**承载同一份实测证据**（`git grep -i thresholds` 零命中 / `vite.config.ts:75-85` 无 `thresholds` / **`vite.config.ts:76` 的 `include` 被注释 ⇒ 分母口径使前端覆盖率系统性高估**这一前置）——**任一方改动，另一方必须同步**。
