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
| **GAP-012** | P1 | 订单/接口契约 | 订单主键为 63 位 Snowflake 大整数，JSON 以原生数字输出，前端 `JSON.parse` 精度截断导致详情/支付/完成/取消全部 404 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | CLOSED | `backend-dev` / `frontend-dev` |
| **GAP-013** | P1 | 订单/前端状态 | 创建订单表单行以数组 index 作 key 致 Radix Select 状态残留；且失败后 `pendingSubmission` 未清空，幂等 Key 跨提交复用；缺显式重置入口 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | CLOSED | `frontend-dev` |
| **GAP-014** | P1 | 配置/运行 | 云端 Redis 未生效：`RedisConfig` 无 password/TLS 字段，`pkg/redis` 仅支持明文 TCP，云托管 `rediss://` 无法握手，架构首道预扣防线与商品缓存被静默旁路 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | CLOSED | `backend-dev` / `sre-operator` |
| **GAP-015** | P2 | 数据/接口契约 | `StockLog.BizID` 承载订单 Snowflake 主键（`order_service.go:188,557` 赋 `&order.ID`）却以 JSON 裸数字输出（`stock_log.go:20`），管理员库存流水页 `#{log.biz_id}` 显示被截断的订单号——GAP-012 的同类未收口面 | [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14) | CLOSED | `backend-dev` / `frontend-dev` |
| **GAP-016** | P1 | CI/门禁 | **（库名已修；DoD ①②③ 均已满足；④ 见下方处置）** `4fa68f9` 的 `ci.yml:22` 库名含连字符，被 `test_helper_test.go:95` 的 `^[a-zA-Z0-9_]+$` 拒绝（`:101` `t.Fatalf`）⇒ 当时 CI 上 `internal/service` 必然全红。**残余**：本机 `RUN_MYSQL_TEST` 未设时仍全 SKIP。② 已由 run `39733467897`（`passes=358 skips=0`）与 `39758849047`（`passes=367 skips=0`）满足；③ 载体已建（`internal/service/qa_database_name_guard_test.go`，`4d37f29`）**且变异有效性已实测**；**④ 判为「语义一致、形式不同」**（理由见条目）；门禁裁决仍阻断 | [#15](https://github.com/Yotoha0303/go-order-management-system/issues/15) | IN_PROGRESS | `qa-validator` / `backend-dev` |
| **GAP-017** | P1 | CI/门禁 | **（前端作业已加，CI 已连续六轮 pass）** `4fa68f9` 的 `.github/workflows/ci.yml` 无任何前端作业，GAP-013 全部前端用例在 PR 上不执行 ⇒ 违反 `AGENTS.md` §2 03 环「新建用例 100% 纳入 CI 矩阵」；**残余（已实测确定）**：`main` **无分支保护**（`branches/main/protection` 返回 `"Branch not protected"` / 404），该作业**未被设为 required check** ⇒ 失败仍可合并。**解除需仓外平台动作，AI 不能代办** | [#16](https://github.com/Yotoha0303/go-order-management-system/issues/16) | IN_PROGRESS | `sre-operator` / `frontend-dev` |
| **GAP-018** | P2 | 数据/接口契约 | 商品主键 `Product.ID` 为**自增** bigint，仍以 JSON 裸数字输出（`product.go:11`），与订单侧 `Order.ID` 的字符串约定不一致；当前不是缺陷（远未及 2^53），但商品 ID 一旦改为 Snowflake 即重演 GAP-012 | —（待建 Issue） | IN_PROGRESS | `data-engineer` / `backend-dev` |
| **GAP-019** | P1 | 文档/编号 SSOT | `docs/changes/` 中编号 `004` 与 `005` **各被两个不同文件占用**（`004-fix-gap-005-…` + `004-support-mysql-tls-connection.md`；`005-fix-gap-003-…` + `005-support-cors-and-spa-rewrites.md`）⇒ 本仓任何「编号唯一」的说法都是**假守卫** | —（待建 Issue） | IN_PROGRESS | `ssot-scribe` |
| **GAP-020** | **P0** | CI/门禁 | **`main` 的 CI 自 `2026-07-07T05:56:15Z` 之后连续 15 次全红、无一次绿**（最近 9 次集中在 `2026-10-21`）。而 `GAP-003/004/006/008/009/010` 均标 `CLOSED`、闭环叙述含「门禁裁决」——**这些 `CLOSED` 从未经过 CI 验证**，其中 5 项的变更记录里自带 `--- SKIP` 日志 | —（待建 Issue） | IN_PROGRESS | `sre-operator` / `ssot-scribe` |
| **GAP-022** | P3 | 测试质量 | `TestCreateOrder_MultipleItemsProcessedByProductID`（`order_service_test.go:406`）用「种子顺序」隐式假定 `productA.ID < productB.ID`，断言 `items[0]=A`。生产侧**确实**按 `ProductID` 排序（`order_service.go:250-254`、`:595-596`），故当前成立；但商品主键一旦改为非单调（如 Snowflake，见 GAP-018），断言会在**实现仍正确**的情况下失败 | —（待建 Issue） | IN_PROGRESS | `qa-validator` |
| **GAP-023** | P3 | 测试质量 | 同一用例内用 `Order("id ASC")` 读回 items / stock_logs（`order_service_test.go:425`、`:436`），隐含「自增 `id` 顺序 = 处理顺序」；把**排序契约**押在实现细节上，改批量插入或改主键生成即失效 | —（待建 Issue） | IN_PROGRESS | `qa-validator` |
| **GAP-024** | P2 | 测试质量 | **（已由 `4d37f29` 处置：空壳用例已删除）** 原 `TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain`（`:1703`）函数体只有一行转调 `…_ReleasesExtraPreDeduct`（`:1663`），而后者断言「重放**会再次**预扣 2 次并释放 1 次」——**用例名与断言语义相反**，且同一逻辑被执行两遍 | —（待建 Issue） | CLOSED | `qa-validator` |
| **GAP-025** | P3 | 工具链/假信号 | `internal/service` 下 **11 个**文件在工作树为 CRLF（`git ls-files --eol` 实测，其中 1 个还是 `w/mixed`），`gofmt -l` 把它们**全部**列出；但索引中这些文件**全是 `i/lf`**，且 LF 版本经 `gofmt -l` 实测**干净**——即 `gofmt -l` 在本机输出的是**行尾符噪声，不是格式缺陷**，开发者无法用它判断真实格式问题。CI 不含 `gofmt` 步骤 | —（待建 Issue） | IN_PROGRESS | `ssot-scribe` / `qa-validator` |
| **GAP-026** | P2 | 配置/运行 | **（c）推断，未实测运行期行为**：`compose.yml` 三处库名默认值 `go-order-management-system` **含连字符**（`:29` / `:64` / `:125`），与 `ci.yml:25` 刚因同一约束改成下划线形态的情形**形态不一致**；`ci.yml:22-24` 注释明写连字符是**故意**被 `safeDatabaseName`（`test_helper_test.go:95`）拒绝的（它要拼进 `CREATE DATABASE`）。CI 侧是**硬防线**，compose 侧**毫无防护**——migrate 容器是否 `exit 0` 无任何机械判据 | [#18](https://github.com/Yotoha0303/go-order-management-system/issues/18) | IN_PROGRESS | `sre-operator` / `backend-dev` |
| **GAP-027** | P2 | 测试/门禁 | `AGENTS.md` §2 03 环要求「覆盖率**配置阈值**」，而**两栈均不存在**：全仓 `git grep -i thresholds` **零命中**、`.github/workflows/` 内 `git grep -i cover` **零命中**；Go 侧 `make coverage` 只产 `coverage.out`（`Makefile:181-182`）、前端 `vite.config.ts:75-85` 的 `coverage` 块**只有 `include`/`exclude`、无 `thresholds`**，`package.json:18` 的 `test:coverage` **不在 CI**。即该准出条件**从未被持有，也从未被登记** | [#19](https://github.com/Yotoha0303/go-order-management-system/issues/19) | IN_PROGRESS | `qa-validator` / `sre-operator` |
| **GAP-028** | P2 | 测试/门禁 | `qa_database_name_guard_test.go:259` 的 AST 钉住断言 `len(rejectCalls) != 1 \|\| rejectCalls[0] != "Fatalf"`——`rejectCalls` 由 `qaIsTestingCall` **收集该分支内一切 `t.<method>()` 调用**（`:247-248`）。故只要在该分支加一句 `t.Logf` / `t.Helper` / `t.Errorf`，`rejectCalls` 即变 `["Fatalf","Logf"]` ⇒ **误报失败**。属「**误报的判据比没有判据更糟**」：它会把人推向删掉这条钉住 | [#20](https://github.com/Yotoha0303/go-order-management-system/issues/20) | IN_PROGRESS | `qa-validator` |
| **GAP-029** | P1 | 构建/发布（**04 环**） | **04 环在本仓无任何对应物**：`AGENTS.md` §2 该环准出为「镜像**不可变 Digest** + 运行时断言 `REVISION == Git HEAD`」。实测全仓 `git grep -iE 'REVISION\|@sha256\|digest'` **零命中**（唯一命中 `LICENSE:43` 的英文单词 "revisions"，非制品）；`Dockerfile:14/36/50` 三个 `FROM` **全部是可变 tag**（`${GO_IMAGE}` / `${RUNTIME_IMAGE}` → `golang:1.25.7-alpine` / `alpine:3.22`，见 `compose.yml:18-19`）；`compose.yml:60/75/90` 的 mysql/redis/rabbitmq 亦为可变 tag。**无 digest 固定、无 REVISION 断言** | [#21](https://github.com/Yotoha0303/go-order-management-system/issues/21) | IN_PROGRESS | `sre-operator` / `sec-auditor` |
| **GAP-030** | P1 | 部署（**05 环**） | **05 环准出为「部署脚本全链路幂等 + 配置与环境变量双向对齐无残留」**，本仓两半均无判据：`deploy.sh`（**451 行**）未见幂等自检（无「连续执行无状态突变」的断言），仅 `:216` 的 `compose up -d --build --wait`；**配置双向对齐**这一半只靠 `AGENTS.md` §2 的 `docker compose config -q`（**纯语法校验，不执行任何容器**，`Makefile:189-190`）。**注意**：`GAP-026`（compose 三处库名含连字符）是**这半边的一个具体面**，**不得互相结掉**——库名只是配置对齐里的一条 | [#21](https://github.com/Yotoha0303/go-order-management-system/issues/21) | IN_PROGRESS | `sre-operator` / `backend-dev` |
| **GAP-031** | P1 | 观测/排障/恢复/复盘/自动化（**06–10 环**） | **06–10 环在本仓无任何对应物**（合并为一条，按裁决口径）。实测：`alertmanager` **零命中**；`severity` / `rto` / `rpo` **仅在散文里出现**（`docs/adr/003-*.md:35` 一处提到 Alertmanager 无法感知漂移——**是问题陈述，不是制品**）；`git ls-files` 过滤 `prometheus\|alertmanager\|grafana\|dashboard\|runbook\|slo\|sli` **零制品命中**（`fronted/src/features/order-inventory/dashboard.tsx` 是**业务页面**，与 SRE 仪表盘无关）；过滤 `drill\|failover\|backup\|restore\|chaos` **零命中** ⇒ **无停机演练、无 RTO/RPO 实测、无 Runbook、无告警规则、无复盘模板、无定时归档** | [#21](https://github.com/Yotoha0303/go-order-management-system/issues/21) | IN_PROGRESS | `sre-operator` / `rca-investigator` |
| **GAP-032** | P2 | 架构决策/一致性 | **ADR-001 的决策三（TTL 租期）与决策四（Redis `SET NX` 幂等互斥）未落地**——且**不是「实施未完成」，是闭环走了别的路径**：`GAP-005` 用状态无关的防御性释放闭环、`GAP-002` 用 MySQL `ErrRecordNotFound` 处理。**核心风险**：决策三的 TTL 是预扣凭证的**兜底回收机制**，当前 `SET` **无 `EX`** ⇒ 若补偿链路本身失败，reservation 键**永不自动过期**、**永久占用库存**；现仅依赖「补偿必定执行」这一假设，**无任何时间兜底** | [#23](https://github.com/Yotoha0303/go-order-management-system/issues/23) | IN_PROGRESS | `system-architect` / `backend-dev` |

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
- **状态**：**`CLOSED`** —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。CI 实测：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Test, Vet, Validate, Build` **已真跑并执行到 `internal/model` / `internal/handler`**（该作业当时 `fail`，但失败点为 `TestRebuildRedisInventoryStock_Success`，与本项无关，见 GAP-020）。
  **剩余条件（`2026-10-22` 订正——此前写的「因 CI 整体未绿」现已为假）**：`main` 的 CI **已首次转绿**（run `39774132552` @ `adc1f21`，`event=push`，`passes=367 skips=0`；详见 `GAP-020`）。**该条件已满足。**
  **门禁裁决已放行**——**载体指针（以章节名为准，行号仅供导航）**：本文件 `GAP-020` 的「⚠️ 方法论」**第 6 条「门禁裁决载体（业务仓内）」**下的**裁决表「第四轮」行**，结论「**`GAP-012` / `GAP-013` / `GAP-014` 可转 `CLOSED`**」、所审 HEAD `adc1f21`。**反查命令**：`git grep -n '门禁裁决载体' -- docs/GAP台账.md`。
  **此为总控依裁决作出的状态翻转（总控推荐，用户可否决）。**
- **⚠️ 裁决方对 `GAP-012` 的错判与主动撤回（`2026-10-22`；**本条本身即一个方法论实例，故留在本条正文**）**：
  - **错判形态**：裁决方**上一轮**判定 `GAP-012` 有缺失项，依据是**只 `Glob` 了 `internal/model/*_test.go`**、**未跨包 `Grep` 函数名**，因而**漏掉了** `internal/handler/order_handler_test.go` 中 `TestOrderIDSerializedAsString`（**`:254-381`**）内的两条往返用例。
  - **被漏掉的两条（实测存在）**：
    - **`:335-343`** —— `t.Run("javascript float64 round trip diverges from original id")` 与 `t.Run("javascript float64 round trip diverges from original order_id")`，均调用 `assertPrecisionLostByFloat64`，**正对应 DoD ③ 的负向用例要求**；
    - **`:345-380`** —— `t.Run("unquoted json number is truncated by float64 decode")`：端到端构造**未加引号**的 JSON 数字载荷并断言**必被 float64 截断**（`:361-363`），再对**加引号**形态断言**往返相等**（`:377-379`）——**这是 DoD ③ 的完整实现**，且**带 `t.Logf` 打印 original / truncated / delta**。
  - **撤回已由裁决方主动作出**，本轮据此重裁为「**无缺失项、可转**」。
  - **为何把它写进本条**：这是**「据局部静态读码下结论」的又一实例**，与 `GAP-020` 方法论**第 1~3 条同形**（那三条是「推断冒充实测」；本条是「**局部检索冒充全量检索**」——**检索范围不写清，零命中就与没查同形**）。
    **可执行的推论**：凡以「**没找到**」为据的结论，**必须给出检索范围**（命令原文 + 路径范围），否则该结论**与「没查」在观测上不可区分**。**本条正是该推论的证据。**

### GAP-013 · 创建订单表单行 key 与幂等 Key 生命周期缺陷
- **现状**：创建订单表单的商品行以**数组 index** 作为 React key，删除中间行后 React 复用了错误的 DOM 节点，Radix Select 的内部状态残留到相邻行；提交失败后 `pendingSubmission` 未清空，下一次提交复用同一幂等 Key，被后端判定为重复请求；界面无显式的重置入口。
- **风险**：用户看到的商品行与实际提交内容不一致（选错商品 / 数量），且失败后无法通过重试恢复——重试被幂等键拦成「处理中」，页面却无任何自愈手段。
- **DoD**：
  1. 商品行改用**稳定唯一** `rowId`（非数组 index）作为 key；
  2. 提交**成功与失败均清空** `pendingSubmission`，并断言失败后重新提交会生成**全新**幂等 Key；
  3. 提供显式的「清空」入口；
  4. 有单测覆盖上述三项（含失败后重提生成新 Key 的用例）。
- **状态**：**`CLOSED`** —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。CI 实测：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的**前端作业 `Frontend (tsc, eslint, vitest browser, build)` 已真跑且 `pass`**，本项的前端用例（`order-draft.test.ts` / `order-submission.test.ts` / `qa-order-submission-lifecycle.test.ts` / `qa-orders-form.test.tsx`）在该作业内执行。
  **剩余条件（`2026-10-22` 订正——此前写的「因 CI 整体未绿」现已为假）**：`main` 的 CI **已首次转绿**（run `39774132552` @ `adc1f21`，`event=push`，`passes=367 skips=0`）。**该条件已满足。**
  **门禁裁决已放行**——**载体指针（以章节名为准，行号仅供导航）**：本文件 `GAP-020` 的「⚠️ 方法论」**第 6 条「门禁裁决载体（业务仓内）」**下的**裁决表「第四轮」行**，结论「**`GAP-012` / `GAP-013` / `GAP-014` 可转 `CLOSED`**」、所审 HEAD `adc1f21`。**反查命令**：`git grep -n '门禁裁决载体' -- docs/GAP台账.md`。
  **此为总控依裁决作出的状态翻转（总控推荐，用户可否决）。**

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
- **状态**：**`CLOSED`** —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。CI 实测：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Test, Vet, Validate, Build` **已真跑**，`pkg/redis` 与 `config` 包执行并通过（失败点为 `internal/service`，与本项无关，见 GAP-020）。
  **剩余条件（`2026-10-22` 订正——此前写的「因 CI 整体未绿」现已为假）**：`main` 的 CI **已首次转绿**（run `39774132552` @ `adc1f21`，`event=push`，`passes=367 skips=0`）。**该条件已满足。**
  **门禁裁决已放行**——**载体指针（以章节名为准，行号仅供导航）**：本文件 `GAP-020` 的「⚠️ 方法论」**第 6 条「门禁裁决载体（业务仓内）」**下的**裁决表「第四轮」行**，结论「**`GAP-012` / `GAP-013` / `GAP-014` 可转 `CLOSED`**」、所审 HEAD `adc1f21`。**反查命令**：`git grep -n '门禁裁决载体' -- docs/GAP台账.md`。
  **此为总控依裁决作出的状态翻转（总控推荐，用户可否决）。**
  ⚠️ **另有一项不在 DoD 内、但须与本项合读的残留**：本条的 **DoD ④ 负向用例**（`REDIS_TLS=maybe` 必须报错而非静默当 false）**未由本轮单独复跑**——「已真跑」指的是 CI 执行了 `pkg/redis` / `config` 包，**不等于**该负向用例被单独验证过（同 `GAP-020` 方法论第 5 条：「执行了」≠「判据有效」）。**本条仍依裁决转 `CLOSED`（裁决判定「无缺失项」），但该残留如实保留在此**——**若日后要复核本条的 `CLOSED`，这是首先该跑的一条命令。**

### GAP-015 · 库存流水 `BizID` 的 Snowflake 裸数字输出
- **现状**：`internal/model/stock_log.go:20`（HEAD 口径）为 `BizID *int64` 且 JSON 标签是 `json:"biz_id"`（**无** `,string`）。该字段由订单链路写入并承载订单主键：`internal/service/order_service.go:188`（`StockBizOrderDeduct`，下单扣减）与 `:557`（`StockBizOrderRollback`，订单取消）均赋 `&order.ID`，而 `order.ID` 是 63 位 Snowflake（`internal/model/orders.go:16`）。取值域 = **{nil, 订单 Snowflake}**：`internal/service/inventory_service.go:53-60`（初始化库存）与 `:116-123`（手动入库）不赋 `BizID`，故 nil 分支必须保持 `null`。列表接口挂在管理员组下（`router/router.go:138`，受 `AdminMiddleware`），`internal/handler/stock_log_handler.go:43-49` 直接 `response.Success(c, stockLogs)` 输出 `[]*model.StockLog`，裸数字因此直达响应体。前端 `fronted/src/features/order-inventory/types.ts:52` 声明 `biz_id?: number | null`，`fronted/src/features/order-inventory/stock-logs.tsx:120` 以 `` `#${log.biz_id}` `` 直接渲染。
- **风险**：与 GAP-012 同一失效形态——写入与读取使用两个不同的 ID 事实，且**不会报类型错误**，只表现为流水页显示出错号的订单号；据此反查订单必然查无此单。实测（`node v22.23.2`，`2026-10-22T05:03Z`，探针值 `371447562240004097`）：`JSON.parse('{"biz_id":371447562240004097}')` 得到 `371447562240004100`，往返不相等；同一值以**字符串**形态往返则相等。
- **DoD**：
  1. `BizID` 的 JSON 输出改为**字符串**（序列化侧统一约定，不得只在前端做 hack）；`nil` 仍须序列化为 `null`，**不得**变成 `"0"` 或 `""`；
  2. 前端 `StockLog.biz_id` 类型对齐（`string | null`），`stock-logs.tsx` 渲染路径**不得**做任何数值转换；
  3. 有**原始字节**断言用例：直接对序列化后的 JSON 字节断言 `biz_id` 被引号包裹（不得用「反序列化到 `interface{}` 再比较」的写法——那会同时掩盖数字与字符串两种形态）；
  4. 有**负向用例**证明数字形态必然失真：同一 63 位 ID 经 JSON 数字路径往返后与原值不等，据此可证伪本判据不会静默放行。
- **状态**：**`CLOSED`** —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。**代码与独立测试侧已收口**：`docs/changes/014-fix-gap-015-stock-log-biz-id-precision.md` 记录了修复（`json:"biz_id,string"`，nil 仍为 `null`）与原始字节断言；`qa-validator` 已把 `internal/model/qa_json_surface_test.go` 的钉住用例**翻转**为「set id 必为字符串 + nil 必为 `null`」，实测：
  ```
  go test -count=1 ./internal/model/ -run TestQAStockLogBizIDShapePinsTheOpenContract -v
  --- PASS: TestQAStockLogBizIDShapePinsTheOpenContract (0.00s)
  ```
  **故此前记录的「门禁因该用例变红」已不成立**，`go test ./...` 不再因此有红点。门禁读数见 `014-*.md` §6 / §7（含 §7.2 对 `tsc --noEmit` 空转的订正）。
  **CI 实测**：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Test, Vet, Validate, Build` 已真跑（`internal/model` 包通过），**「未在 CI 验证」已不成立**。
- **状态**：**`CLOSED`** —— 关联 [#14](https://github.com/Yotoha0303/go-order-management-system/issues/14)。**DoD ①②③④ 全部满足**（代码与测试侧收口见上）；**门禁裁决已放行**——**载体指针（以章节名为准，行号仅供导航）**：本文件 `GAP-020` 的「⚠️ 方法论」**第 6 条「门禁裁决载体（业务仓内）」**下的**三轮裁决表「第三轮」行**，结论「**放行合并**」、所审 HEAD `aac6908`、并载明「`GAP-015` / `GAP-024` 转 `CLOSED` 的前置（裁决载体）已满足」。
  **可复跑的反查命令（不依赖行号，故不会随编辑漂移）**：
  ```
  git grep -n '门禁裁决载体' -- docs/GAP台账.md      # 第 6 条标题所在行
  git grep -n '第三轮' -- docs/GAP台账.md           # 三轮裁决表中的第三轮行
  ```
  **此为总控依裁决作出的状态翻转（总控推荐，用户可否决）**。

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
  4. 本机与 CI 使用**同一套**开关语义（`AGENTS.md` §5「本地跑的那一套 = CI 跑的那一套」）。**（已处置：判为「语义一致、形式不同」——理由见下方「④ 的处置」，走裁决给的第 (a) 路）**
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
  - ✅ **证据分级已升级为 (a) 实测（`2026-10-22` 由总控代跑坐实）**：注入 `t.Fatalf → t.Skipf` 后，`TestQADatabaseNameGuardAbortsInsteadOfSkipping` **变红**，失败信息含「must abort … skipping makes every integration test in this package silently green」。**此前的 (c) 推断已作废**——该判据**确实有牙**。
    > ⚠️ **`2026-10-22` 复查（`ssot-scribe`）**：本条目的「变异有效」现为**(a)**；但同文件 `:259` 的钉住写法本身**另有一个误报面**（`rejectCalls` 收集的是该分支内**全部** `t.<method>()`，加一句 `t.Logf` 即误报）——**已另开 `GAP-028`**，两者不得互相结掉。
- **④ 的处置（`2026-10-22`；裁决给的第 (a) 路：说明「为何允许不同」）**：
  - **实测差异（a）**：
    - **本地**：`Makefile:155-174` **分步**——`test-service` / `test-dao` / `test-migrations` / `test-order-timeout` 各自 `export RUN_MYSQL_TEST := 1`（`test-redis` 为 `RUN_REDIS_TEST := 1`，`test-order-timeout` 另加 `RUN_RABBITMQ_TEST := 1`），且各自带 `-v`；`make ci` 也是逐目标串行调用。
    - **CI**：**单次** `go test -v ./...`（`ci.yml:101-102`）**加**独立断言步骤 `Assert no integration test was skipped`（`:109`）。
  - **判为满足的理由（三点，均可核）**：
    1. **开关语义逐字相同**：`ci.yml:18-20` 的 `RUN_MYSQL_TEST` / `RUN_REDIS_TEST` / `RUN_RABBITMQ_TEST` 与 `Makefile:155/167/171-172` **同名、同值 `1`**。DoD ④ 要求的是「同一套**开关语义**」，而**不是**「同一串命令」——**文字上确实不同，语义上一致**。
    2. **两者的差异是各自目的所必需**：本地分步是为了**失败定位**（红在哪一组）与按需只跑一组；CI 单次是为了**总耗时**与**「一次跑完、无遗漏」**——若 CI 也分步，反而会**丢失**「全仓一次跑完」这一覆盖性质，并引入「某一步被漏加进矩阵」的新缺口（这正是 GAP-017 的形态）。**两种形式各自都在防一种失败，合并任一种都会引入另一种。**
    3. **同一套判据已由 CI 侧钉住**：`skips=0` 断言（`:109-120`）是**形式无关**的——它只看 `go test -v` 的日志，**不论**命令是分步还是一次性。故「不许用 SKIP 代替」这条**在两种形式下都成立**。
  - **残留（不是缺口，是约束）**：本地**没有**等价于 CI `Assert no integration test was skipped` 的步骤——本机 `make test-*` 仍可能因开关未设而 SKIP 并报 `ok`（见「现状」第 2 条）。**但**本地分步目标本身**已显式 export 开关**，故按 `Makefile` 走不会 SKIP；**只有绕过 `make` 直接跑 `go test` 才会**。该残留由 `GAP-011`（「消除 `t.Skip` 导致的本地虚假全绿」）承接，**不另开条目**。
- **状态**：IN_PROGRESS —— 关联 [#15](https://github.com/Yotoha0303/go-order-management-system/issues/15)。
  **① ② ③ ④ 均已满足**（② 的证据见上：run `39733467897` / `39758849047`，`skips=0`；③ 的载体与**变异实测**见上；④ 的处置见上）。
  **剩余条件（`2026-10-22` 订正——此前写的「因 CI 整体未绿 / GAP-020 未解除」现已为假）**：`main` 的 CI **已首次转绿**（run `39774132552` @ `adc1f21`，`event=push`，`passes=367 skips=0`），**该条件已满足**；`GAP-020` 的 **DoD ① 亦已满足**（该条仍因 DoD ③④ 保持 `IN_PROGRESS`，**但这不再构成本项证据基础的缺口**——本项要的是 `main` 的一次绿 run，已拿到）。
  **故本项现存的唯一剩余条件是「门禁裁决」**——**是否可转由门禁裁决，总控正在另派**，本台账**不预判**。

### GAP-017 · CI 缺前端作业，03 环用例在 PR 上不执行
- **现状**（**历史缺陷已修，当前状态如实如下**）：
  - **缺陷发生时（`4fa68f9`，`main` 当时的 HEAD）**：`.github/workflows/ci.yml` 共 **111 行**，**只有一个作业** `quality`（`:27`），步骤全部是 Go / goose / golangci-lint / `docker build`；`node` / `npm` / `npx` 关键字命中数 **0**（`Select-String` 实测）。
  - **现已修复**：提交 `d024ef5` 新增 `frontend` 作业；**`a6439d2` 起 `ci.yml` 为 2 个作业**（`a6439d2`/`cb3c4ac` 共 **195 行**；`e37da30` 加缓存后 **219 行**），含作业 **`frontend:`**、步骤 `actions/setup-node@v4` / `npm ci` / **`Type check` 的 `run: npx tsc -b`** / `Lint` 的 `npx eslint .` / `Unit and browser tests` 的 `npx vitest run --browser.headless`。
    > 📌 **本处原写「`frontend`（`:146`）、`npx tsc -b`（`:186`）」——两处行号均已因 `e37da30` 的 +24 行而下移**（实测：`frontend:` 在 **`:146`→ 不变**，`run: npx tsc -b` 在 **`:186` → `:210`**）。**按本节确立的新口径，此处改为「作业名 + 步骤名 + `run:` 内容」定位，不再依赖行号**（口径说明见本条目 DoD ③）。
  - **CI 实测**：PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 `Frontend (tsc, eslint, vitest browser, build)` → **pass 六轮**：run `39729484909`（`3584d1f`）**1m0s**、run `39732018029`（`6fa11f8`）**1m9s**、run `39733467897`（`a6439d2`）**56s**、run `39758849047`（`4d37f29`）**51s**、run `39761963731`（`aac6908`）**64s**、run `39772049248`（`e37da30`）**60s**。**故「前端用例在 PR 上不执行」已不成立。**
  - **✅ Playwright 浏览器缓存（`e37da30` 引入；这是 DoD 的可复跑证据，`2026-10-22` 实测）**：
    - **改动**：提交 `e37da30`（`ci(frontend): cache Playwright browsers between runs`）对 `ci.yml` **+24 行**，新增两步 `Resolve Playwright version`（读 `node_modules/playwright/package.json` 的 version）与 `Cache Playwright browsers`（`actions/cache@v4`，`path: ~/.cache/ms-playwright`）。
    - **缓存键**：**`Linux-playwright-1.59.1`**（= `${{ runner.os }}-playwright-${{ steps.playwright.outputs.version }}`）。**键可复现的依据**：`playwright` 在 `package.json` 中**精确锁定**为 `"1.59.1"`（非范围），故版本号即 revision，**不需要 `hashFiles`**（代码注释原文即此理由）。
    - **三态日志实测（`gh run view --log` 逐行实取）**：
      | 状态 | run | 日志行 |
      | --- | --- | --- |
      | **miss** | `39774132552`（`adc1f21`，首次带缓存） | `Cache not found for input keys: Linux-playwright-1.59.1` |
      | **saved** | `39774132552`（同一次，步骤收尾） | `Cache saved with key: Linux-playwright-1.59.1` |
      | **hit** | `39772049248`（`e37da30`） | `Cache hit for: Linux-playwright-1.59.1` → `Cache restored successfully` |
    - **效果（实测耗时对比）**：
      | 指标 | 无缓存 run `39767723671`（`e8d5311`） | 有缓存 run `39772049248`（`e37da30`） |
      | --- | --- | --- |
      | `Install Playwright Chromium` 步骤 | **781s** | **12s** |
      | `Frontend` 作业总计 | **835s（13m55s）** | **60s（1m0s）** |
      > ⚠️ **两处须精确，勿沿用近似值**：① 合并 run `39774132552` 因**缓存 miss**，该步骤为 **20s**（非 12s）、作业 **63s**——**12s 属 cache-hit 场景**；② 无缓存基线**以 `39767723671` 为准（781s / 835s）**——`39767723671` 的 `Frontend` 是**唯一**作业（该 run 无 Go 作业），**故「Go 386s / Frontend 63s」是 `39774132552` 的数，与「781s / 835s」不可混为一谈**。
  - 前端现有 **22 个测试文件**（`Get-ChildItem fronted/src -Recurse -Include *.test.ts,*.test.tsx`，已排除 `__screenshots__` 下的两个**同名 1 字节产物**——`Test-Path` 为真但 `ReadAllBytes` 被拒、非源码用例），其中含 GAP-013 的相关用例：`order-submission.test.ts`、`order-draft.test.ts`、`qa-order-submission-lifecycle.test.ts`、`qa-orders-form.test.tsx`。`fronted/package.json:14` 的 `test` 脚本为 `vitest run --browser.headless`。
- **风险**：前端用例**存在**但**不在 PR 门禁内**——本地跑与不跑都不影响合并。这正是 `AGENTS.md` §2 03 环「新建用例 **100% 纳入 CI 矩阵**（拒绝门外用例）」所禁止的「门外用例」，且与 D3 同族：**失败不会响**。（该风险在 **CI 配置层**已解除；但 **DoD ② 经实测确定未满足**——`main` **无任何分支保护**，故前端作业**失败也不会挡住合并**，见下。）
- **DoD**：
  1. CI 有**独立前端作业**，类型检查步骤**必须是 `tsc -b`**——这不是可选替代：`tsc --noEmit` 在本仓**只检查 0 个源文件、恒 `exit 0`**，等同没有类型门禁（证据见下方「① 的判据依据」）。同一步作业还须执行 `eslint` 与 `vitest run`；**（已满足，CI 实测 pass）**
  2. 该作业失败**必须**使 PR 变红（作为 required check 的组成部分，不是 advisory）；**❌ 确定未满足（`2026-10-22` 实测；此前写「本机命令无法判定」——该表述是错的）**
     **证据（a，可复跑）**：`gh api repos/Yotoha0303/go-order-management-system/branches/main/protection` 返回
     ```
     {"message":"Branch not protected","documentation_url":"https://docs.github.com/rest/branches/branch-protection#get-branch-protection","status":"404"}
     ```
     **这不是「无法判定」，是 GitHub 的明确回答**——`main` **没有任何分支保护规则**。
     **判据：404 与 403 必须分开读**——**403（权限不足）才是「无法判定」**；**404 + 该文案 = 确定未满足**。
     **两处对照实验（排除误读，同一次采集）**：
     - **不是权限不足**：`gh api repos/Yotoha0303/go-order-management-system` 的 `permissions` 为 `{"admin":true,"maintain":true,"push":true,"triage":true,"pull":true}` —— **当前 token 有 admin**，故 404 不可能来自权限；
     - **API 能区分「不存在」与「无保护」**：对不存在的分支请求同一端点，返回的是 `{"message":"Branch not found",…,"status":"404"}` —— **文案不同**，说明 `"Branch not protected"` 是**该分支确实存在但无保护**的专属回答。
     **⇒ 结论：前端作业存在、CI 六轮 pass，但没有任何机制强制它——一个前端作业失败的 PR 仍可合并。**
     **解除方式（仓外平台动作，AI 不能代办）**：由**仓库管理员**在 GitHub 上开启 `main` 的分支保护，并把**两个作业**（`Test, Vet, Validate, Build` 与 `Frontend (tsc, eslint, vitest browser, build)`）设为 **required status checks**。
     **归属**：属框架 `AGENTS.md` §5 的「**待执行指令**」类别——**「谁做」栏只能是「人」或「平台」，没有一条 AI 能代办**；且须**单列**，**不混进 `GAP-` 台账的代码类缺口**（本条记录缺口事实，**动作**由那份待执行指令承载）。
     **可复核命令**：`gh api repos/Yotoha0303/go-order-management-system/branches/main/protection`（**保护开启后应返回 200 + 规则 JSON，而不是 404**——这即是该 DoD 的正向判据）。
  3. 说明**为何不能塞进现有 Go 作业**——两条独立理由：**循环时长**（Go 循环与 `npm ci` + 浏览器二进制安装的耗时互不相关，合并后每次 Go 迭代都要等前端依赖装完）与**失败定位**（一个作业红无法区分是 Go 还是前端坏了）。**（已落实：`ci.yml` 中 `frontend:` 作业**声明处正上方**的注释给出了同一理由——原文 `# Frontend gate. Kept as a separate job on purpose: it shares no steps with the / # Go loop, and folding it into \`quality\` would make every Go iteration wait on / # npm install + Playwright.`）**
     > 📌 **引用方式说明（`2026-10-22` 改）**：本处**不再写行号**。原写法为「`ci.yml:116-118`」，而 `e37da30` 给该文件加了 Playwright 缓存（**+24 行**）后，同一注释**整体下移到 `:143-145`**——**行号引用本身就是漂移源**（同 `GAP-020` 方法论第 5 条的精神：一个读数若不写清采集口径，就会静默过期）。
     > **本台账自本处起改口径**：引用 CI 配置一律用「**作业名 / 步骤名 + 关键行内容**」定位（如「`Type check` 步骤的 `run: npx tsc -b`」），**不写行号**；确需行号时**必须同时注明所依据的 commit**。**理由**：作业名与 `run:` 内容**只在语义变更时失效**，而行号**每次文件增删都失效**。
- **① 的判据依据（本机实测）**：`fronted/tsconfig.json` 是 solution 风格（`"files": []` + `references`），故以根配置执行时**根本不编译任何文件**。
  - 文件计数（`2026-10-22T05:02Z`，`--listFiles` 过滤 `node_modules` / `typescript/lib`）：`npx tsc --noEmit` 检查 **0** 个源文件、`exit 0`；`npx tsc -p tsconfig.app.json --noEmit --listFiles` 检查 **150** 个；`npx tsc -b` 走 `references`，与后者同集。
  - **注入真实类型错误后的判别力**（`backend-dev` 注入 `src/__tsc_gate_probe.ts` 的 TS2322，探针已撤回、`Test-Path` 现为 `False`）：`npx tsc --noEmit` → `exit 0`（**漏检**）；`npx tsc -b` → `exit 2`（报出 `TS2322`）。
  - **结论**：只有 `tsc -b` 是**真正执行类型检查**的形式；`tsc --noEmit` 在本仓是**空转**，写成它就是一条无牙的判据（D3）。此条为 DoD ① 的判据依据，**不改动本项核心问题与 DoD 编号**。
- **状态**：IN_PROGRESS —— 关联 [#16](https://github.com/Yotoha0303/go-order-management-system/issues/16)。**① ③ 已满足且 CI 已真跑 pass（六轮：`39729484909` / `39732018029` / `39733467897` / `39758849047` / `39761963731` / `39772049248`）；② 确定未满足**（`main` 无分支保护，实测见 DoD ②；**解除需仓外平台动作，AI 不能代办**），故不判 `CLOSED`。

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
- **现状（初采 `gh run list --branch main --limit 20` @ `2026-10-22T05:53Z`；**本段已于 `2026-10-22` 合并后订正**）**：
  - **⚠️ 分支侧 vs `main` 侧（合并前此区分是关键；`2026-10-22` 合并后已成历史，保留以备考古）**：
    - **分支侧 `fix/issue-14-order-module`：合并前连续五轮全绿** —— run `39732018029`（`6fa11f8`）、run `39733467897`（`a6439d2`）、run `39734691195`（`cb3c4ac`）、run `39758849047`（`4d37f29`，`Frontend` **51s** / Go **3m31s**）与 run `39761963731`（`aac6908`，`Frontend` **64s** / Go **239s**），五轮均为 `Frontend` + `Test, Vet, Validate, Build` **双作业 pass**。**该分支已合并入 `main`**（`adc1f21`）。
    - **`main`：已首次转绿（`2026-10-22T11:54:33Z`）** —— run **`39774132552`** @ `adc1f21`，**`event=push`**（即真正的主干 push 事件，非 PR 事件），结论 **`success`**；跳过断言读数 **`passes=367 skips=0`**；作业耗时 `Test, Vet, Validate, Build` **386s** / `Frontend` **63s**。
      **该次 green 之前连续 `15` 次 `failure`**（实测逐条计数，自 `39651784273` @ `4fa68f9` 往前）；**上一次 green = `28844933317` @ `2026-07-07T05:56:15Z`，间隔约 `107` 天（3.5 个月）**。
      > ⚠️ **合并方式**：PR #17 **以 merge commit 合并**——`adc1f21` **父数 = 2**（`4fa68f9` + `e37da30`），13 个提交**完整保留**在 `main` 历史中（非 squash）。实测：`git rev-list --count 4fa68f9..adc1f21` = **14**（13 个分支提交 + 1 个 merge commit）。
    - **故「分支绿」不满足本项 DoD** —— 本项要的是 **`main` 上的一次成功 run**，**该 run 已于 `39774132552` 取得**（见上）。**「分支绿 ≠ `main` 绿」这一区分在合并前是关键，合并后已成为历史。**
  - **`2026-07-07T05:56:15Z` 之后连续 15 次 `failure`，无一次 `success`**：`28848964896`（`07-07T07:22`）、`29594894144`、`29596015046`、`29598908734`、`29602411164`、`29646698974`、`39589875294`、`39602099199`、`39606899421`、`39616159527`、`39626627749`、`39630625435`、`39634710946`、`39650366486`、`39651784273`（`2026-10-21T18:20:12Z`，`headSha 4fa68f9`）。
  - ⚠️ **订正一处不准确的说法**：**并非「从未通过过」**——同一次采集里 `28844933317`（`2026-07-07T05:56:15Z`）、`28655645359`、`28648530857`、`28647042143` 四次为 `success`。**准确表述是「最近一次绿在 `2026-07-07T05:56:15Z`，此后 15 次连红」**。
  - **`main` 与分支此前从未执行过同一份 `ci.yml`（`2026-10-22` 订正：合并后此差异已消除）**：`4fa68f9`（合并前 `main` 的 HEAD）的 `ci.yml` 为 **111 行 / 1 个作业**（`quality`，`:27`）；分支侧 `a6439d2` 起为 **195 行 / 2 个作业 + 跳过断言步骤**（`Assert no integration test was skipped`，`:109`）。**PR #17 合并后，`main` 上跑的已是那份 195 行的 `ci.yml`** —— run `39774132552` @ `adc1f21` 即其首次执行，**故「前端作业在 `main` 上不执行」与「`main` 无跳过断言」两条同时解除**。
    > 实测口径：`git diff --stat a6439d2..e37da30 -- .github/workflows/ci.yml` 显示 `e37da30` **+24 行**（Playwright 缓存，见 `GAP-017`），其余自 `a6439d2` 起未变。
- **核心问题**：台账中 `GAP-003` / `GAP-004` / `GAP-006` / `GAP-008` / `GAP-009` / `GAP-010` 均标 `CLOSED`，其闭环叙述含「已通过门禁审计 / 门禁裁决」。但**这些 `CLOSED` 作出时，`main` 的 CI 一次都没绿过**——所谓「门禁」不是 CI。**`CLOSED` 与「CI 通过」在本仓从来不是同一件事，而台账的读法会让人以为它们是。**
  > ⚠️ **`2026-10-22` 订正（勿误读为「已关闭」）**：`main` **已首次转绿**（run `39774132552` @ `adc1f21`），故上句的「作出时一次都没绿过」**是历史事实，不是当前状态**。但**本条的核心问题并未因此消失**——它要的是**判定口径落定 + 召回重验 + 机械守卫**（DoD ②③④），而这三项**至今未做**。**「有了一次绿 run」只满足 DoD ①。**
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

> ⚠️ **关于「证据充分性存疑」那 6 项（`GAP-006` / `GAP-008` / `GAP-009` / `GAP-002` / `GAP-011` / `GAP-007`）：本次不回填，且这是决定，不是遗漏。**
> **原因**：本表回填**只用一类证据**——**该用例在 CI 日志里的具名 `--- PASS` 行**。而那 6 项**没有对应的点名读数**：它们的变更记录里**既无 `--- SKIP`、也无 CI 记录**（这正是它们当初被标为「存疑」的理由），故**无从点名**。
> **不得用「`passes=367 skips=0` 总数」去填它们**——那正是本条目方法论第 5 条明令禁止的推理跳跃（**计数不回答「哪条跑了」，更不回答「断言对不对」**）。
> **故保持未回填，状态一律不放宽。** 它们的解除条件是：**各自补出可点名的 CI 读数**（即指出该 GAP 的关键用例名，并在某次 run 的日志里查到它的 `--- PASS` 行）。**为了表格整齐而填，比空着更坏**——空着是「未知」，填了是「假的已知」。

##### 该证据**不能**证明什么（本轮教训的核心）

1. **不能证明这些用例的断言本身正确。** 用例通过只说明「它与当前实现一致」，**不说明它断言的是正确的行为**。**本轮实例（实测）**：`GAP-024` 的空壳用例 `TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain` 在 run `39734691195`（`cb3c4ac`）中是
   `--- PASS: TestCreateOrder_IdempotentReplayDoesNotPreDeductRedisAgain (0.13s)` —— 它**真的执行了、也真的通过了**，但**它声称守护的性质（「重放不重复预扣」）根本不存在**（其函数体只转调另一用例，断言的是「重放会二次预扣再释放」）。**这是「执行了 ≠ 判据有效」的实证**：绿着，却没有那条保护。
   ⚠️ **订正一处转述（勿再沿用）**：本轮曾以 `TestRebuildRedisInventoryStock_Success` 作为「此前也是『执行且通过』而断言是错的」的例证——**该前提与实测不符**。实测：该用例在断言未修时的 run `39729484909`（`3584d1f`）中读数是 **`--- FAIL: TestRebuildRedisInventoryStock_Success (0.11s)`**，**不是 PASS**；它恰是**被 CI 抓出来的**（`a6439d2` 的 `-v` + 断言步骤使 `internal/service` 真跑），修好后在 run `39734691195` 才转为 `--- PASS`。
   **故准确表述是**：该错误断言**从未「执行且通过」**——它此前**根本没被执行**（本机 SKIP、CI 因库名 Fatal），属于「潜伏的假绿」；**而「执行了却仍无信息」的实例是 GAP-024 空壳用例**。两类问题不同，不得混为一谈。
2. **不能证明这些用例覆盖了该 GAP 的完整 DoD。** 用例通过只说明它断言的那条性质成立；它**没有断言**的性质（如 `GAP-001` 的释放补偿链路、`GAP-003` 的租期互斥在真并发下的表现）仍然无判据。
3. **不能证明 `CLOSED` 成立。** 本条 DoD ① 明写「`main` 的 CI 连续通过」——回填所据的是**分支侧**的 run。**分支绿 ≠ `main` 绿**（本项开篇即已区分）。
   > ⚠️ **`2026-10-22` 订正**：本条**已不适用于 `main`** —— run `39774132552` @ `adc1f21`（`event=push`）已使 `main` 首次转绿。**但这条订正只解除「分支绿不算数」这一层**：下面 1 / 2 / 4 / 5 条**依然成立**，且**「用例在 `main` 上执行过」同样不等于「断言正确」**。
4. **不能把「当时的 SKIP」判为「当时的结论错误」。** 本次回填只解除**证据缺陷**；各 `CLOSED` 条目的**业务正确性**既未被本次证实、也未被本次否证——那需要各自的 DoD 逐条重验（DoD ③ 的其余部分）。
5. **不能外推到时点**。本读数只对 `4d37f29` 有效；`passes` 是随提交变化的量（`358` @ `a6439d2` → `367` @ `4d37f29`）。**引用时必须带 run id 与 headSha。**
- **DoD**：
  1. **`main` 的 CI 连续通过**，并在本条目内**贴出 run id 作为证据**（形如 `gh run list --branch main --limit 1 --json databaseId,conclusion`）；在此之前本项不得判 `CLOSED`；
     **✅ 已满足（`2026-10-22T11:54:33Z`）** —— run **`39774132552`** @ **`adc1f21`**，`event=push`，`conclusion=success`，跳过断言 **`passes=367 skips=0`**（`gh run view --log` 实取）。
     **判据说明（须与本 DoD 的字面合读）**：本 DoD 原文写「**连续**通过」。当前**只有 1 次** `main` 绿 run（此前连续 15 次 `failure`，上一次绿在 `2026-07-07T05:56:15Z`，间隔约 107 天）。
     **故严格按字面，「连续」尚未成立**——**是否把「1 次」视为满足，属口径裁定，本台账不预判**；此处如实记录**读数**与**口径歧义**，两者都留痕。
  2. **明确并落定判定口径**：`CLOSED` 的判定**必须以 CI 真实执行为准**，**不得以本机 SKIP 口径的绿读数替代**。本机跑绿只能作为**辅助**，须同时给出「非 SKIP 执行数」与「CI run id」两类证据；
  3. **召回复核**：完成上表所列项目的重验，逐条给出「CI run id + 该用例确为非 SKIP 执行」的证据；**在证据到位前，不放宽任何一条的状态**；
     **（`2026-10-22` 部分回填：上表 5 项已给出 run `39758849047` @ `4d37f29` 的逐条点名 `--- PASS` 证据，见上方「召回重验·回填」。**
     **该回填只解除「被 SKIP」这层证据缺陷**；「证据充分性存疑」那 6 项（`GAP-006/008/009/002/011/007`）**仍未回填**，且**各 `CLOSED` 的断言正确性一律未被本次证明**——见方法论第 5 条。）**
  4. **机械守卫**：让「本机绿」与「CI 绿」在台账里**不可混淆**——条目状态旁须能看出依据来自哪一侧（本机 / CI）；并配负向用例：注入一条「只有本机 SKIP 绿、无 CI run」的记录，守卫必须报警（D3）。
- **状态**：IN_PROGRESS —— 尚无 Issue 编号；本项为 **P0**，是全部 `CLOSED` 结论的**证据基础**问题，优先于其余条目。
  **DoD 逐项（`2026-10-22` 订正；此前笼统写「①②④ 未满足」已不准确）**：
  - **DoD ①：已满足** —— run `39774132552` @ `adc1f21`（`event=push`，`success`，`passes=367 skips=0`）。**唯一保留**：原文写「**连续**通过」，当前**只有 1 次** `main` 绿 run；**「1 次是否算连续」属口径裁定，本台账不预判**（见 DoD ① 下的判据说明）。
  - **DoD ②：部分满足** —— 「判定口径」已落定（方法论第 4、5 条 + 本条的 `(a)/(b)/(c)` 分类法）；但「须同时给出非 SKIP 执行数与 CI run id」这条**已可执行且已有先例**（召回重验表），**尚未写成机械规则**。
  - **DoD ③：未满足（部分回填）** —— 上表 5 项已回填（证据 run `39758849047`）；**「证据充分性存疑」那 6 项（`GAP-006/008/009/002/011/007`）仍未回填**（原因见上方引用块，**不是遗漏**）。
  - **DoD ④：未满足** —— 机械守卫**未建**（让「本机绿」与「CI 绿」在台账里不可混淆，并配负向用例）。
  **⇒ 故本项保持 `IN_PROGRESS`**：**DoD ③④ 是仅剩的两个未满足项**（①已满足，②部分满足）。
  **裁决状态（第三轮，`aac6908`）：放行合并**——**该放行已执行**：`GAP-015` / `GAP-024` 转 `CLOSED` 的**前置（裁决载体）已满足**（见方法论第 6 条，`:424` / `:431`），两条**已据本裁决转 `CLOSED`**。**本条自身的 DoD 不受该放行影响。**
  ⚠️ **「放行合并」与「本条 DoD」是两件事**（`2026-10-22` 订正）：此前写「`main` 仍一次未绿」——**该表述现已为假**，`main` 已首次转绿。**但本条仍不解除**，因为 **DoD ③④ 仍未满足**；**「合并完成」不替代「召回重验」与「机械守卫」。**
  ⚠️ **也不得反向推断**：`main` 绿了 **≠** 台账里那些 `CLOSED` 的断言因此正确（方法论第 5 条）——**本条的 DoD ③ 正是为此而设**。
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
  6. **门禁裁决载体（业务仓内）—— 第 4 条的落地**（`2026-10-22` 落盘；**这是 `GAP-012` / `GAP-013` / `GAP-014` / `GAP-015` / `GAP-024` 转 `CLOSED` 的前置**）：
     第 4 条要求「任一判 `CLOSED` 的条目须能反查到裁决原文」。此前各轮裁决**只落在框架仓的 `record.md`**，而该文件被框架仓 `.gitignore` 排除、**不入库**——**业务仓读者 clone 后够不到**，故第 4 条的判据在本仓**无法满足**。**本节即其载体**（按第 4 条的边界：**不新建文件**）。

     | 轮次 | 结论 | 阻断项 / 收尾项 | 所审 HEAD | 出具时点 |
     | --- | --- | --- | --- | --- |
     | **第一轮** | **阻断** | 03 环：`E-1` 独立测试资产未入库（`qa_*` 全为未跟踪）、`E-2`/`E-3` CI 矩阵从未执行且 `go test` 无 `-v` 致「SKIP=0」为假读数；另 `E-4`~`E-8` 文档漂移；05 环缺实测 | ⚠️ **未记录**（见下方「本表的缺口」） | ⚠️ **未记录**（同上） |
     | **第二轮** | **仍阻断** | `E-1`/`E-2`/`E-3` **已解除**；新增：`GAP-016` DoD③ 无钉住用例、`GAP-024` 空壳用例使 `passes` 虚高；04 环缺不可变 Digest 与 `REVISION` 断言、05 环仅有 `compose config -q` 语法验证；新发现 `E-9` / `E-12` / `E-13` / `E-14` | `cb3c4ac` | `2026-10-22T06:33Z`~`06:43Z` 之间（**区间值，非精确值**） |
     | **第三轮** | **放行合并** | **无阻断项**；四项「不阻断合并、必须登记」的收尾：`E-11`（基线落后）、`G-5`（`GAP-016` DoD④ 从未处置）、`G-2`（**本条**：裁决原文在业务仓无载体）、`G-3`（`qa_database_name_guard_test.go` AST 钉住有误报面）；另要求把 04 / 05 / 06–10 环登记为独立 GAP | **`aac6908`** | ⚠️ **未单独记录**；本载体于 `2026-10-22T10:36Z` 后落盘 |
     | **第四轮** | **`GAP-012` / `GAP-013` / `GAP-014` 可转 `CLOSED`；`GAP-016` / `GAP-017` / `GAP-020` 维持 `IN_PROGRESS`** —— **本裁决即三条所需的那次门禁裁决** | ① `GAP-016` DoD④ 判据口径未落定（记 **`无法判定`**）；② `GAP-017` DoD② **实测未满足**（`main` 无分支保护）；③ `GAP-020` DoD④ 未建 **+** DoD① 的「**连续**」口径未定 | **`adc1f21`**（merge commit，**父数 = 2**） | **`2026-10-22T12:19:31Z`**（UTC；**由总控代填**，理由见下） |

     **第四轮的解除条件**：**见各条各自的 DoD** —— 本裁决**未新增**统一解除条件，三条转 `CLOSED` 者已满足，维持 `IN_PROGRESS` 者按各自 DoD 的未满足项推进。

     **第四轮的两处须留痕之事（都与方法论直接相关）**：
     1. **裁决方主动撤回对 `GAP-012` 的上一轮错判** —— 它**只 `Glob` 了 `internal/model/*_test.go`**、**未跨包 `Grep` 函数名**，因而**漏掉了 `internal/handler/order_handler_test.go` 中 `TestOrderIDSerializedAsString`（`:254-381`）内的两条往返用例**（`:335-343` 的 `float64` 往返失真、`:345-380` 的**未加引号 JSON 数字被 float64 截断**的端到端负向对照）。**这是「据局部静态读码下结论」的又一实例**——与本节方法论第 1~3 条**同形**（那三条是「推断冒充实测」，本条是「**局部检索冒充全量检索**」），**已记入 `GAP-012` 正文**。
     2. **裁决方拒绝自填 UTC 时点** —— 它明确指出：台账明写「**补造即违规**」，而它作为**无 shell 的只读角色**，**编不出可信的 UTC**。**故本行时点由总控代填，并在此注明来源。**
        **这条拒绝本身是方法论的正面执行**，值得留痕：**它没有为了让表格「看起来完整」而填一个数**——这正是本表下方「本表的缺口」一节所要求的行为（宁可留白并说明，不可补造）。
        **同时也印证了只读角色的分工**：其产出是**攻击清单 / 裁决表 + 命令清单**，**执行与取证由总控代跑**（框架 `AGENTS.md` §4.1「协同纪律」）——**时点即属代跑项**。

     **第三轮的解除条件（逐项，均已在本轮处置）**：
     1. `E-11` → `AGENTS.md` §4 重采，并**在 §4 内写明采集口径以打破「采集即过期」的循环**（见该节）；
     2. `G-5` → `GAP-016` DoD ④ 给出「语义一致、形式不同」的裁定与理由（见该条目「④ 的处置」）；
     3. `G-2` → **本节**（载体落进业务仓）；
     4. `G-3` → 登记为 **`GAP-028`**（**只登记、不改代码**，按裁决要求）；
     5. 04 / 05 / 06–10 环 → 登记为 **`GAP-029`** / **`GAP-030`** / **`GAP-031`**（DoD 均含「**阻断上生产，不阻断合并**」这一区分）。

     ⚠️ **本表的缺口（如实，不粉饰）**：**第一轮的出具时点与所审 HEAD 未在任何地方留下记录**——这正是第 4 条要治的病，而它**在治它之前就已经发生过一次**。本表**不补造**这两个值（补造即为 (c) 推断冒充实测）。**第二轮的时点也只能给区间**（由框架仓运行态记录的落盘时点 `2026-10-22T06:43:15Z` 与其所审 `cb3c4ac` 的 run 时点反推）。**故本表自身的完备性，只能从第三轮起成立**——这不是瑕疵，是这条纪律**建立得太晚**的直接证据。
     **「不补」是决定，且已确认（`2026-10-22`）**——理由两条，须与上一段合读：
     1. **补造即违规**：第一轮裁决出具时总控**未落痕**，原始载体在**框架仓的运行态记录**里（该文件被框架仓 `.gitignore` 排除、**不入库**、**业务仓读者不可达**）。从那个载体反推出一个「时点 + HEAD」再写进本表，是**把仓外运行态的数字伪装成本仓实测**——正是第 1~3 条要治的形态。
     2. **这个缺失本身就是证据**：它**恰恰是 `G-2` 的成因与实证**——「裁决原文只落在不入库的运行态里」导致的结果，就是**下一轮连上一轮审的是哪个提交都不知道**。**补一个数会让这条证据消失，而缺口留着会一直提醒后来者**。故**保持「未记录」**，并在此写明原因。
     **后续轮次的义务**：每轮裁决**落盘时**必须同时写全**五项**（结论 / 阻断项 / 解除条件 / UTC 时点 / 所审 HEAD），**缺一即视为未落痕**，下轮不得据以放行。
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
- **状态**：**`CLOSED`** —— 尚无 Issue 编号。P2。**DoD ①②③ 全部满足**（处置见上）；**门禁裁决已放行**——**载体指针（以章节名为准，行号仅供导航）**：本文件 `GAP-020` 的「⚠️ 方法论」**第 6 条「门禁裁决载体（业务仓内）」**下的**三轮裁决表「第三轮」行**，结论「**放行合并**」、所审 HEAD `aac6908`。**反查命令同 `GAP-015`**（`git grep -n '门禁裁决载体' -- docs/GAP台账.md`）。**此为总控依裁决作出的状态翻转（总控推荐，用户可否决）**。
  ⚠️ **与 `GAP-015` 的关系**：两条的裁决载体**同一处**（`GAP-020` 方法论第 6 条 / 三轮裁决表「第三轮」行），故**同轮转 `CLOSED`**。

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

### GAP-028 · AST 钉住的 `rejectCalls` 断言过窄，会因无害调用误报失败
- **现状（实测，`2026-10-22`，HEAD `aac6908`）**：`internal/service/qa_database_name_guard_test.go` 的 `TestQADatabaseNameGuardAbortsInsteadOfSkipping`：
  - `:241-251` 以 `ast.Inspect` 遍历 `ensureTestDatabase` 的**非法名分支**，用 `qaIsTestingCall(call, tParam)` 收集**该分支内一切形如 `t.<method>(...)` 的调用**（`qaIsTestingCall` 见同文件：只校验接收者是 `*testing.T` 参数，**不区分方法名**）；
  - `:252` `sort.Strings(rejectCalls)`；`:259` 断言 **`len(rejectCalls) != 1 || rejectCalls[0] != "Fatalf"`**。
- **误报路径（可复现）**：在该分支内加**任何**一句无害的 `t.Logf(...)`（或 `t.Helper()`、`t.Errorf(...)`），`rejectCalls` 即变为 `["Fatalf","Logf"]` ⇒ **`len != 1` ⇒ 该用例失败**，而**被守护的性质（「非法名必须 `Fatalf` 中止，不得 `Skip`」）完好无损**。
  > 注意：该断言**不是「恰好一个测试调用」**，而是「**该分支内恰好只有一个 `t.*` 调用，且它是 `Fatalf`**」——这个「恰好」是**偶然的**（当前分支里确实只有一句 `t.Fatalf`），**不是契约**。
- **风险（P2，属 D3 的反面）**：本台账反复强调「**判据无牙**」（永远绿，不携带信息）；本条是**另一种坏形态**——**判据有牙，但会咬错人**。其后果比没有判据更糟：开发者遇到**假红**，最省事的处置是**删掉或放宽这条钉住**，于是**真正该守的「不得改成 Skip」也随之消失**。**误报会把守卫推向被移除**，而移除是静默的（此后没有任何东西会失败）。
- **与 `GAP-016` 的边界（不得互相结掉）**：`GAP-016` 的 DoD③ 已被证明**有牙**（变异 `Fatalf → Skipf` 实测变红，见该条「③ 的满足证据」）；本条说的是**同一文件里另一处写法**的**误报面**。**「有牙」与「不误报」是两件事**——一个判据可以同时**有牙且误报**。
- **DoD**：
  1. **改断言口径**：从「**恰好一个且为 `Fatalf`**」改为「**必须包含 `Fatalf`**」（`slices.Contains(rejectCalls, "Fatalf")`），使无害调用不再导致失败；**`skipCalls` 那一半（`:254` 断言 `len(skipCalls) == 0`）保持不变**——「不得出现任何 `Skip*`」本来就是对**全量**的要求，不存在同类误报；
  2. **保留判别力（关键，不得只放宽不加固）**：把「必须 `Fatalf`」与「不得 `Skip`」**分开断言**后，须**重新验证负向用例**：注入 `t.Fatalf → t.Skipf`，该用例**仍必须变红**（D3：放宽判据后**必须重跑负向用例**，否则「放宽」与「拆掉」在观测上同形）；
  3. **另加一条防「空集通过」的判据**：断言 `rejectCalls` **非空**——否则若有人把整个分支删空/重构，`Contains` 也会失败（这条已由 `:236` 的 `guardBody == nil` 部分覆盖，但**分支存在 ≠ 分支里有调用**，故须显式）；
  4. **补一条误报的回归用例**：在测试内以**合成 AST**（或临时改写副本）构造「分支里含 `Fatalf` + 一句 `t.Logf`」，断言**改后的口径不报错**——把「不误报」也变成可证伪的判据，而不是靠这次修完就算。
- **状态**：IN_PROGRESS —— 关联 [#20](https://github.com/Yotoha0303/go-order-management-system/issues/20)。P2。**只登记，不改代码**（按第三轮裁决 `G-3` 的要求）。
  ⚠️ **两处同步义务（D4）**：本条正文与 Issue #20 正文**承载同一份实测证据**（`rejectCalls` 的收集口径 / `:259` 的断言写法 / `t.Logf` 即误报的复现路径）——**任一方改动，另一方必须同步**。

### GAP-029 · 04 环（构建）在本仓无任何对应物
- **要求出处**：`AGENTS.md` §2 04 环准出条件 = 「镜像**不可变 Digest** + 运行时断言 `REVISION == Git HEAD`」。
- **现状（实测，`2026-10-22`，HEAD `aac6908`）**：
  - **零命中（a）**：`git grep -iE 'REVISION|@sha256|digest'` 全仓**仅一处命中**——`LICENSE:43` 的英文单词 `revisions`，**是散文，不是制品**。故「REVISION 断言」与「digest 固定」**均不存在**。
  - **基础镜像全部是可变 tag（b）**：`Dockerfile:14` / `:36` / `:50` 三个 `FROM` 全部走变量——`${GO_IMAGE}`、`${RUNTIME_IMAGE}`（`Dockerfile:14/36` 为 `GO_IMAGE`，`:50` 为 `RUNTIME_IMAGE`），默认值由 `compose.yml:18-19` 提供：`golang:1.25.7-alpine` / `alpine:3.22`。
  - **服务镜像同为可变 tag**：`compose.yml:60` `mysql:8.4`、`:75` `redis:7.2-alpine`、`:90` `rabbitmq:4.1-management-alpine`；应用镜像 `compose.yml:11` / `:109` 为 `go-order-management-system:local`（本地 tag）。
  - **结论**：**同一份 `compose.yml` 在不同时间可能拉起不同的镜像内容**，且**无从校验**「跑的是哪个提交构建的」。
- **风险**：与 `GAP-025` 同族的「**静默**」形态——不是构建失败，而是**构建成功但内容不可复现、不可追溯**。它使 `GAP-020` 要的「CI 通过 = 可发布」这一等式**缺一环**：CI 绿只说明**当时那台 runner 上**的构建通过了，**不说明**部署侧拉到的镜像与之同源。
- **DoD**：
  1. **基础镜像按 digest 固定**（`FROM alpine@sha256:…`），或在 `compose.yml` / `Dockerfile` 内以 digest 覆盖 tag；**不得**只固定 major.minor；
  2. **构建时注入 `REVISION`**（= `git rev-parse HEAD`，经 `--build-arg` 或 `ldflags -X`），并在**运行时**暴露可查询面（如 `/version` 或启动日志）；
  3. **加运行时断言**：`REVISION` 与**部署所声明的** Git HEAD **不一致即失败**（不得只打印不校验——「打印了」与「断言了」是两件事，同 `GAP-016` 的教训）；
  4. **负向用例（D3）**：注入一个**伪造的 `REVISION`**，断言**必须失败**；并注入**一个不带 digest 的 tag**，断言镜像固定检查**必须报警**；
  5. **本项的通过与否，不阻断 PR #17 的合并**——见下方「与合并前提的关系」。
- **与合并前提的关系（用户已裁定）**：**04 环不作为 PR #17 的合并前提**。但**「不作为合并前提」不等于「不存在」**——本条目照常登记、照常计入活跃 GAP，只是**其解除不阻塞当前 PR**。
  **DoD 的通过与否**：**阻断上生产，不阻断合并**。判据一句话：**合并**是把已通过 CI 的改动并入 `main`（可回退）；**上生产**是把镜像交付给真实流量（`GAP-029` 的缺陷在那一刻才变成**不可追溯的线上事实**）。
- **状态**：IN_PROGRESS —— 关联 [#21](https://github.com/Yotoha0303/go-order-management-system/issues/21)。P1（**阻断上生产，不阻断合并**）。
  ⚠️ **同步义务（D4）**：本条正文与 Issue #21 正文**承载同一份实测证据**（`REVISION|@sha256|digest` 的唯一命中是 `LICENSE:43` 散文、三个 `FROM` 全为可变 tag、服务镜像同为可变 tag）——**任一方改动，另一方必须同步**。

### GAP-030 · 05 环（部署）的两条准出条件在本仓均无判据
- **要求出处**：`AGENTS.md` §2 05 环准出条件 = 「部署脚本**全链路幂等**（连续执行无状态突变）+ 配置与环境变量**双向对齐无残留**」。
- **现状（实测，`2026-10-22`，HEAD `aac6908`）**：
  - **幂等那一半**：`deploy.sh` 共 **451 行**，含 `set -euo pipefail`（`:6`）与 `compose up -d --build --wait`（`:216`），并有若干**防御性**分支（`:84` "Docker is already available."、`:108` "`/etc/docker/daemon.json` already exists; leave it unchanged."）。**但这些是逐点防御，不是幂等判据**——**没有任何**「连续执行两次、状态无突变」的断言或演练记录。**故「全链路幂等」未被持有。**
  - **配置对齐那一半**：唯一载体是 `AGENTS.md` §2 的 `docker compose config -q`（`Makefile:189-190` 的 `compose-config`）。**它只做语法校验、不执行任何容器**——**语法合法 ≠ 配置与环境变量真对齐**（这正是 D3 的「判据夹缝」）。
- **与 `GAP-026` 的边界（不得互相结掉）**：`GAP-026`（`compose.yml` 三处 `DB_NAME` 默认值含连字符）**是「配置对齐」这半边的一个具体面**——它指出一处**形态不一致**。本条说的是**该环整体缺少判据**。**修好库名不等于 05 环准出**；反之亦然。
- **风险**：部署是**不可逆动作**里最贵的一类。幂等未验证的后果是**「第二次执行会怎样」无人知道**——而重跑部署恰是**故障时最先做的事**（`AGENTS.md` 08 环的恢复路径）。与 `GAP-029` 同族：**失败发生在最不方便的时刻**。
- **DoD**：
  1. **幂等演练（a）**：在干净环境**连续执行两次** `deploy.sh`，第二次**不得**产生状态突变（容器无重建、迁移无重复应用、`/etc/docker/daemon.json` 等被「already exists」分支跳过）；**须留真实 stdout / 退出码**，不得只声明；
  2. **配置双向对齐**：枚举**代码读取的每一个环境变量**与**`compose.yml` / `.env.example` / `config.yml` 提供的每一个**，两侧求差集；**两个方向都必须为空**（代码要但没人给 ⇒ 运行时缺；给了但没人读 ⇒ 残留）。`docker compose config -q` **不足以**替代这一步（它看不到「代码读不读」）；
  3. **负向用例（D3）**：注入一个**代码读取但配置缺失**的变量，断言**必须显式失败**（不得静默用零值启动——同 `GAP-014` 的「失败必须显式可见」）；
  4. **`GAP-026` 并入本项的验证**：库名形态一致性作为 ② 的一个检查点，**但不替代** ② 的其余部分。
- **与合并前提的关系（用户已裁定）**：**05 环不作为 PR #17 的合并前提**；但**「不作为合并前提」不等于「不存在」**。
  **DoD 的通过与否**：**阻断上生产，不阻断合并**（理由同 `GAP-029`：合并可回退，上生产不可逆）。
- **状态**：IN_PROGRESS —— 关联 [#21](https://github.com/Yotoha0303/go-order-management-system/issues/21)。P1（**阻断上生产，不阻断合并**）。
  ⚠️ **同步义务（D4）**：本条正文与 Issue #21 正文**承载同一份实测证据**（`deploy.sh` 无幂等断言、配置对齐只靠 `compose config -q` 语法校验、`GAP-026` 是其中一面）——**任一方改动，另一方必须同步**。

### GAP-031 · 06–10 环（观测/排障/恢复/复盘/自动化）在本仓无任何对应物
- **要求出处**：`AGENTS.md` §2 的 **06 观测 / 07 排障 / 08 恢复 / 09 复盘 / 10 自动化** 五环准出条件（Alertmanager active 区分 + 严重度参与 + 真实停机演练 · 秒级时间线 + 对照实验 + 根因回写 · 实测 RTO/RPO · 行动项带 DoD · 定时器前置条件断言 + 独立归档报告）。
- **现状（实测，`2026-10-22`，HEAD `aac6908`）**——**合并为一条**（五环在本仓的状态同形：**均无对应物**）：
  - **`alertmanager` 零命中**（全仓，跟踪文件）；
  - **`severity` / `rto` / `rpo` 仅在散文里出现**：唯一相关命中是 `docs/adr/003-inventory-locked-quantity-and-reconcile-autofix.md:35`——原文说的是「对账差异数与自愈结果**未**暴露 Prometheus 指标，SRE 监控平台与 Alertmanager **无法感知**静默的数据漂移」。**这是问题陈述，不是制品**——它恰恰在说「06 环缺什么」。
  - **制品过滤零命中**：`git ls-files` 过滤 `prometheus|alertmanager|grafana|dashboard|runbook|slo|sli` ⇒ **零 SRE 制品**。
    > ⚠️ **两处命中必须排除，否则会得出相反结论**：`fronted/src/features/order-inventory/dashboard.tsx` 是**业务页面**（库存看板），**不是** SRE 仪表盘；`fronted/eslint.config.js` 的命中是 `@eslint/js` / `eslint-plugin-*` 里的子串，**纯误报**。
  - **恢复/演练制品过滤零命中**：`git ls-files` 过滤 `drill|failover|backup|restore|chaos` ⇒ **零命中** ⇒ **无停机演练、无 RTO/RPO 实测、无备份/恢复脚本**。
  - **故**：06（无告警规则、无 SLI/SLO）、07（无 Runbook、无时间线还原）、08（无恢复演练、无 RTO/RPO 实测）、09（无复盘模板、无行动项载体）、10（无定时任务前置断言、无归档报告）**五环全部为「该环尚未建设」**。
- **风险**：这五环的**共同性质是「只在出事后才被用到」**——平时缺席**没有任何症状**（不像 03 环缺用例会立刻红）。故它们最容易**长期停在「未满足」而无人注意**：`AGENTS.md` §2 明写「**对应物不存在时该环判「未满足」，不是「不适用」**」。
  **本仓现状的准确表述**：这些环**不是被豁免**，而是**尚未建设**——两者在台账里必须区分（豁免是决定，未建设是缺口）。
- **DoD（按环，各自独立，不得互相结掉）**：
  - **06 观测**：① 有可被 Alertmanager 消费的**告警规则文件**（入库）；② 告警**区分 active / 已恢复**；③ 每条告警带**严重度**并参与路由；④ 至少一次**真实停机演练**（不是 mock）；
  - **07 排障**：① 有一次真实排障的**秒级时间线**；② 有**对照实验**（同条件开/关对照）；③ **根因回写**到本台账或设计文档；④ 改进项**转为 `GAP-` 编号**；
  - **08 恢复**：① **实测** RTO/RPO（不是估算）；② **业务级**可用性核验（不是「进程起来了」）；③ 备份可恢复性演练——**损坏的备份必须被拒绝**（不得「恢复成功但数据是空的」）；
  - **09 复盘**：① 行动项具备**承载形式**（编号 + 责任角色 + 截止）；② 每条有**可度量 DoD**；③ 依赖关系写明；
  - **10 自动化**：① 定时任务**前置条件断言**（条件不满足即失败，不得静默跳过）；② **独立归档报告**（不得只写日志）。
  - **五环共同**：**每一项都须配负向用例**（D3）——注入故障后**必定**触发，否则「有告警文件」与「告警真的会响」是两件事。
- **与合并前提的关系**：用户已裁定 **04/05 不作为 PR #17 的合并前提**；**06–10 环同属「非合并前提」**（它们连构建与部署都不是，更不影响代码合并）。
  **DoD 的通过与否**：**阻断上生产，不阻断合并**。**判据**：合并的后果限于 `main` 的代码内容（可 `git revert`）；上生产的后果是**真实用户与真实数据**——06–10 环缺位意味着**出事时既看不见、也查不出、也恢复不了**，这三件事**都无法靠回退代码解决**。
- **状态**：IN_PROGRESS —— 关联 [#21](https://github.com/Yotoha0303/go-order-management-system/issues/21)。P1（**阻断上生产，不阻断合并**）。
  ⚠️ **同步义务（D4）**：本条正文与 Issue #21 正文**承载同一份实测证据**（`alertmanager` 零命中、`severity`/`rto`/`rpo` 仅散文、制品过滤零命中、**以及下方两处误报的排除**）——**任一方改动，另一方必须同步**。
  > ⚠️ **两处误报的排除必须保留（否则下一轮会被当命中重新翻案）**：
  > - `fronted/src/features/order-inventory/dashboard.tsx` —— **业务页面**（库存看板），**与 SRE 仪表盘无关**；
  > - `fronted/eslint.config.js` —— 命中是 `@eslint/js` / `eslint-plugin-*` 里的**子串**，**纯误报**。
  > **排除理由已同步写入 Issue #21 正文**；两处**均不得**计入「06–10 环有对应物」。

### GAP-032 · ADR-001 决策三（TTL 租期）与决策四（Redis `SET NX` 幂等互斥）未落地
- **发现路径**：`2026-10-22` 修 `ADR-001` 文件头 `Proposed` / 索引表 `Accepted` 的 D4 漂移时，复核「关联 GAP 已闭环 ⇒ 决策已落地」这一前提，**发现该前提不成立**（见下）。
- **现状（实测，`2026-10-22`，HEAD `e37da30`）**——ADR-001 共 **四条**决策（`:76` / `:87` / `:98` / `:120`），**前两条落地、后两条未落地**：

  | 决策 | 原文位置 | 实测 | 证据 |
  | --- | --- | --- | --- |
  | 一 · 预扣前置 + 事务内零网络 IO | `:76` | ✅ 落地 | `internal/service/order_service.go:47-50`（预扣在 `:72` 的 `db.Transaction` **之前**）、`:58` 注释 |
  | 二 · 原生原子增量（废黜绝对值覆盖） | `:87` | ✅ 落地 | `internal/bizcache/inventory_stock_guard.go:88`、`:98` 的 `redis.call("INCRBY", …)` |
  | 三 · **带 TTL 的二阶段预占** | `:98` | ❌ **未落地** | 全仓**无任何** `Expire` / `SetEx` / `TTL` 作用于 reservation 键；`inventory_stock_guard.go:74` 为 `redis.call("SET", KEYS[n + 1], cjson.encode(reservation))` —— **无 `EX` 参数** |
  | 四 · **Redis `SET NX` 幂等前置互斥** | `:120` | ❌ **未落地** | `SetNX` / `NX` / `PROCESSING` 在 `internal/`、`pkg/` **零命中**；实际幂等走 MySQL —— `internal/dao/order_idempotency_key.go:18` 的 `clause.OnConflict{DoNothing: true}` + `order_service.go:80-83` 把 `gorm.ErrRecordNotFound` 转 `ErrOrderBeingCreated` |

- **⚠️ 定性（这一句是本条的核心，不得省略）**：**这不是「实施未完成」，是「闭环走了别的路径」。**
  - `GAP-005` 的闭环**不是** TTL，而是**状态无关的防御性释放**（ADR-001 决策三的**第 2 点**）——即「即便订单已是 `Cancelled`，仍无条件尝试释放」。**TTL 这条第 1 点从未被采用。**
  - `GAP-002` 的闭环**不是** Redis 互斥，而是 MySQL 侧处理 `ErrRecordNotFound`。且 `GAP-002` 的 **DoD 本就给了两个选项**——「幂等 Key 获取解耦或优化查询方式（**采用当前读 或 前置分布式锁**）」，代码取了**前者**。**故 DoD 满足 ≠ ADR 决策四被遵循。**
  - **⇒ 因此「关联 GAP 已 `CLOSED`」推不出「ADR 决策已落地」。这两件事不等价**，本条即其反例。
  - **对后人的提醒**：**不要**把本项读成「当时漏做了」——决策三第 2 点与决策四的替代路径**各自都是有效闭环**（`GAP-005` / `GAP-002` 均已过独立测试、证伪与门禁裁决）。**本项要裁的是「那两条决策本身还算不算数」，不是「当时的修复对不对」。**
- **风险（本条的价值所在）**：ADR-001 决策三的 **TTL 是预扣凭证的兜底回收机制**。当前 `SET` 无 `EX`，意味着：
  - **若补偿链路本身失败**——进程在 `preDeductInventory` 成功之后、`releasePreDeductedInventory` 之前**崩溃**（或该调用自身失败、或消息丢失导致重试链断裂）——则该 reservation 键**永不自动过期**；
  - 后果是 `inventory:{stock}:reservation:<orderID>` **永久残留**，对应的 Redis 可用库存**被永久占用**（少卖），且**没有任何机制会发现它**（无 TTL 到期、无对账覆盖——对账 `inventoryreconcile` 修的是 `stock_quantity` 与 DB 的差异，**不清理孤儿 reservation**）。
  - **当前实际依赖的是一个假设**：「补偿必定执行」。而**凡是「必定」的东西，都是没有兜底的**——这正是 D3 的形态：**失败不会响，只会静默地少卖**。
  - **与 `GAP-005` 的边界（不得互相结掉）**：`GAP-005` 覆盖的是「订单已 `Cancelled` 时的幂等回补」；**本项覆盖的是「补偿链路整体失败」**——后者恰恰是 `GAP-005` 的修复**够不到**的场景（它的修复本身就在补偿链路**内部**）。**两者是「链路内正确」与「链路整体失效」的区别。**
- **DoD（二选一，由 `system-architect` 裁定；本台账不预判）**：
  - **(a) 认定有意放弃** ⇒ 走 `Superseded` / 修订 **ADR-001 决策三**，**并写明放弃后由什么机制兜底**；
    - **硬要求**：若**无**替代兜底，**必须在 ADR 内显式记录该风险**（即「孤儿 reservation 不会被自动回收，需人工巡检」），**不得**只是把决策划掉。**「划掉 + 不写兜底」比不划更糟**——它会让后人以为该风险已被处置。
  - **(b) 认定仍需实施** ⇒ 转实施任务，DoD 须含：
    1. reservation 键**带 TTL**（如原文的 15 分钟支付窗口 + 5 分钟网络漂移冗余 = 20 分钟，**具体值由架构方定**），且 TTL **不得短于**订单超时窗口（否则未超时的预占会被提前回收，造成**超卖**——这是反向风险，须一并论证）；
    2. **负向用例（D3）**：**模拟补偿链路失败**（跳过 / 注入 `releasePreDeductedInventory` 失败），验证凭证仍能**在 TTL 到期后自动回收**、且回收后可用库存恢复。**不注入这个故障就不算验证过**；
    3. **正反两侧都要**：既验证「补偿失败时 TTL 兜底生效」，也验证「补偿成功时 TTL **不**误伤正常预占」（即正常支付窗口内不得被回收）。
  - **决策四的处置**：同批裁定——若认定 MySQL 侧处理已是**等价替代**，须在 ADR 内写明**替代关系与理由**（现状是「原文写 Redis 互斥、代码用 MySQL 冲突处理」，**两者不等价**：前者是内存级前置拦截，后者是事务内仲裁，**对 RR 快照盲区的处理路径不同**）；若仍需前置互斥，同样转实施任务。
- **严重度：`P2`（判断依据，非转述）**：
  - **不判 P1**：触发需**两个条件叠加**——① 预扣成功，**且** ② 补偿链路整体失败。单看任一条件都不触发；且崩溃窗口窄（预扣与补偿之间）。
  - **不判 P3**：后果**不可自愈且不可见**——永久库存泄漏、无 TTL、无对账覆盖、无告警。P3 项（如 `GAP-022`–`GAP-025`）的共同特征是「只影响测试健壮性或开发者体验，不影响生产数据」；**本项影响生产库存账实相符**，性质不同。
  - **与同类对比**：`GAP-014`（Redis 失效被静默旁路）判 **P1**，因其**无需叠加条件**即可发生；本项需叠加，故**低一档**。
- **状态**：IN_PROGRESS —— 关联 [#23](https://github.com/Yotoha0303/go-order-management-system/issues/23)。**待架构裁定**（DoD 二选一未决，故本项暂不可判 `CLOSED`）。P2（判据见上，**非「后果严重就升级」**）。
- **关联载体**：本项的发现与证据**同时记于** `docs/adr/001-high-concurrency-distributed-consistency-inventory-architecture.md` 文件头的「状态订正」注（含决策三 TTL / 决策四未落地的实测），**以及** `docs/adr/README.md` 索引表 ADR-001 行的指针。**三处须同步**（D4）——任一方改动，其余两方必须同步。
- ⚠️ **同步义务（D4）**：本条正文与 **Issue [#23](https://github.com/Yotoha0303/go-order-management-system/issues/23)** 正文**承载同一份实测证据**（四条决策的落地对照表 / `inventory_stock_guard.go:74` 的 `SET` 无 `EX` / `SetNX`·`NX`·`PROCESSING` 零命中 / `order_idempotency_key.go:18` 的 `OnConflict{DoNothing}` / `order_service.go:80-83` 的 `ErrRecordNotFound` 转换 / 以及对账不清理孤儿 reservation）——**任一方改动，另一方必须同步**，否则立即产生两处不同成熟度的事实。
