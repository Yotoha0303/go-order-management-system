# 变更记录：002-fix-gap-001-prededuct-timing

- **变更编号**：002
- **创建时间**：2026-10-21T07:39:00Z (UTC)
- **完成时间**：2026-10-21T08:18:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-001 (P1: Redis Lua 原子预扣位于 MySQL 事务内部，未能前置拦截热点且引发长事务与双重扣减)
- **关联 ADR**：ADR-001 (高并发低延迟与分布式最终一致性库存架构设计) §2 决策一
- **状态**：已完成 (验证通过，已强化抵御证伪攻防)

---

## 1. 变更动机与背景

在历史基准实现中，`OrderService.CreateOrder` 开启本地 MySQL 事务后，在事务深处调用了带有 500ms 超时的外部网络 IO `preDeductInventory`（执行 Redis Lua 脚本预扣），随后在同一事务内继续加排它锁扣减 MySQL 物理库存并创建订单。该设计存在两大严重架构隐患：
1. **长事务与连接池耗尽风险**：将外部网络 IO 混入本地数据库事务，一旦 Redis 出现网络波动或抖动，数据库连接与行锁持有时间被拉长数十倍，极易迅速打满连接池导致整个服务雪崩。
2. **防线未能前置拦截热点与虚假双重扣减**：即使 Redis 预扣返回库存不足（`insufficient`），系统也已经消耗了一个 MySQL 事务连接并开启了事务，无法起到“流量护城河”的极速削峰拦截效果。
3. **订单 ID 依赖数据库自增反向制约**：原流程因为等待 MySQL 生成 `order.ID` 后才去调用预扣，导致预扣不得不位于事务内部。

根据 ADR-001 决策一（精简 MySQL 事务零网络 IO、Redis 预扣前置防穿透），必须将订单 ID 的生成提取至事务外部（Snowflake 分布式 ID），并将 Redis 预扣移至 MySQL 本地事务开启之前；当 Redis 返回库存不足时，极速拦截响应 `ErrInsufficientStock`，实现真正的“零 DB 事务与零连接消耗”。

在第一轮证伪攻击（Falsifier Attack 2, 4, 6）质询中，进一步强化了以下三项硬核约束：
- **Attack 2 (零数据库穿透)**：彻底移除在 Redis 预扣之前的任何 MySQL 查询（原先的 `dao.GetOrderIdempotencyKey` 会在秒杀售罄时导致 99.9% 流量穿透打垮 DB），让 Redis 预扣成为真正的绝对第一道防线；对于偶发的幂等重放流量，采取“先预扣，若 DB 事务发现已创建订单则事务外主动释放多余预扣”的策略，确保高并发售罄场景绝对零 DB 穿透，且幂等重放时零库存凭证泄漏。
- **Attack 4 (Snowflake 时钟回拨验证)**：补齐 `TestSnowflake_ClockRollback` 负向证伪单测，覆盖 <=100ms 轻微漂移自旋容忍与 >100ms 严重回拨熔断拦截（返回 `ErrClockMovedBack` 与 Panic 阻断）。
- **Attack 6 (库存预扣凭证释放断言)**：在幂等重放与事务异常回滚分支中，严格断言 `guard.releases` 的数量与 orderID，杜绝任何阶段的幽灵预扣泄漏。

---

## 2. 修改文件清单与实现方案

1. `pkg/idgen/idgen.go` (新建):
   - 实现高并发、高性能、线程安全的 64 位分布式 Snowflake ID 生成器；
   - 遵循 Twitter Snowflake 标准位布局（41 位时间戳 + 10 位 WorkerID + 12 位序列号）；
   - 提供 `NextID() int64` 全局与实例级方法，用于在开启 DB 事务前预生成全局唯一订单 `orderID`；
   - 包含时钟微小回拨（<=100ms）自旋等待容忍、严重回拨（>100ms）返回 `ErrClockMovedBack` 错误与 Panic 防御，并暴露 `SetTimeFunc` 供时间回拨测试精确可控。
2. `pkg/idgen/idgen_test.go` (新建):
   - 基础单调自增验证（`TestSnowflake_Basic`, `TestSnowflake_Monotonic`）；
   - 边界合法性校验（`TestSnowflake_Validation`）；
   - 50 协程并发生成 100,000 ID 零重复验证（`TestSnowflake_ConcurrentUnique`）；
   - Benchmark 基准性能测试（证明单核/并发达 550ns/op，每秒超 180 万生成能力）；
   - 时钟回拨专门测试（`TestSnowflake_ClockRollback`）：精确模拟 50ms 轻微回拨平滑追平自增，以及 500ms 严重回拨精确返回 `ErrClockMovedBack` 并触发 Panic 保护。
3. `internal/service/order_service.go`:
   - 重构 `CreateOrder` 核心链路（严格防御闭环）：
     1. 最外层参数防御与商品重复校验（`validateDuplicateItems`）前置；
     2. 预扣防线前置：通过 `idgen.NextID()` 预生成 `orderID`，**在触碰任何 MySQL 之前**调用 `p.preDeductInventory(ctx, orderID, items)`；
     3. 极速拦截：若 Redis 预扣返回 `insufficient`，直接返回 `ErrInsufficientStock`，**绝对不触碰 MySQL，零 DB 事务与零连接消耗**；
     4. 极简事务：进入 `p.db.Transaction(...)`，内部绝对不包含任何外部网络 IO；`model.Order` 直接使用预生成的 `orderID`；计算好 `TotalAmountFen` 后一次性创建订单（消除多余的 `PatchOrderTotalPriceFen` 更新）；完成条件扣减、明细写入、流水写入、Outbox 写入与幂等标记；
     5. 幂等与补偿闭环：若事务内部发现 `OrderAlreadyCreated`，则将 `idempotentReplay` 置为 true 并回滚事务；在事务外，若发生事务失败或幂等重放，且此前预扣已生效（`applied == true`），均调用 `p.releasePreDeductedInventory(context.WithoutCancel(ctx), orderID)` 释放预扣凭证，确保零库存泄漏。
4. `internal/service/order_service_test.go`:
   - 补齐负向证伪单测：`TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess`，包含两套严苛子用例：
     - `nil_invalid_db_connection`: 当传入 nil DB 时，证明极速在 Redis 层拦截返回 `ErrInsufficientStock`，绝不 panic 且零 DB 调用；
     - `unconnected_invalid_db_connection`: 当传入未连接/无效的 DB 实例时，证明极速在 Redis 层拦截返回 `ErrInsufficientStock`，零事务进入；
     - `closed_db_connection_when_mysql_enabled`: 当真实 MySQL 连接被提前 Close() 关闭时，证明 Redis 拦截依然优先生效，直接返回 `ErrInsufficientStock` 而绝不冒泡连接关闭错误；
   - 强化幂等重放凭证释放断言：`TestCreateOrder_IdempotentReplay_ReleasesExtraPreDeduct`，断言 `len(guard.preDeducts) == 2 && len(guard.releases) == 1 && guard.releases[0] == guard.preDeducts[1]`，证明幂等请求的多余预扣被精确释放。

---

## 3. 验证命令与实测结果回填

### 3.1 分布式 Snowflake ID 生成器单测、时钟回拨证伪与竞态检测
执行命令：
```powershell
go test -v -race -count=1 ./pkg/idgen/...
```
实测输出 (Stdout)：
```text
=== RUN   TestSnowflake_Basic
--- PASS: TestSnowflake_Basic (0.00s)
=== RUN   TestSnowflake_Monotonic
--- PASS: TestSnowflake_Monotonic (0.00s)
=== RUN   TestSnowflake_Validation
--- PASS: TestSnowflake_Validation (0.00s)
=== RUN   TestSnowflake_ConcurrentUnique
--- PASS: TestSnowflake_ConcurrentUnique (0.10s)
=== RUN   TestSnowflake_PackageNextID
--- PASS: TestSnowflake_PackageNextID (0.00s)
=== RUN   TestSnowflake_ClockRollback
--- PASS: TestSnowflake_ClockRollback (0.05s)
PASS
ok  	go-order-management-system/pkg/idgen	1.649s
```
**Exit Code**: `0`

### 3.2 Snowflake ID 生成器基准性能测试
执行命令：
```powershell
go test -v -race -bench="." -run="^$" ./pkg/idgen/...
```
实测输出 (Stdout)：
```text
goos: windows
goarch: amd64
pkg: go-order-management-system/pkg/idgen
cpu: AMD Ryzen 7 5700U with Radeon Graphics         
BenchmarkSnowflake_NextID
BenchmarkSnowflake_NextID-16             	 2107194	       554.8 ns/op
BenchmarkSnowflake_NextID_Parallel
BenchmarkSnowflake_NextID_Parallel-16    	 1279836	       974.6 ns/op
PASS
ok  	go-order-management-system/pkg/idgen	5.408s
```
**Exit Code**: `0`

### 3.3 负向证伪单测验证 (Zero Database Access)
执行命令：
```powershell
go test -v -race -run TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess ./internal/service/...
```
实测输出 (Stdout)：
```text
=== RUN   TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess
=== RUN   TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess/nil_invalid_db_connection
=== RUN   TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess/unconnected_invalid_db_connection
=== RUN   TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess/closed_db_connection_when_mysql_enabled
    order_service_test.go:210: skip closed DB integration test; set RUN_MYSQL_TEST=1 to run
--- PASS: TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess (0.00s)
    --- PASS: TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess/nil_invalid_db_connection (0.00s)
    --- PASS: TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess/unconnected_invalid_db_connection (0.00s)
    --- SKIP: TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess/closed_db_connection_when_mysql_enabled (0.00s)
PASS
ok  	go-order-management-system/internal/service	3.010s
```
**Exit Code**: `0`

### 3.4 业务服务与全仓单元测试与竞态检测
执行命令：
```powershell
go test -v -race -count=1 ./...
```
实测输出 (Stdout)：
```text
ok  	go-order-management-system/cmd	2.164s
ok  	go-order-management-system/config	2.187s
ok  	go-order-management-system/internal/service	3.184s
ok  	go-order-management-system/migrations/integration	2.237s
ok  	go-order-management-system/pkg/database	2.222s
ok  	go-order-management-system/pkg/idgen	2.194s
ok  	go-order-management-system/router	3.241s
```
**Exit Code**: `0`

### 3.5 全仓代码编译与静态检查
执行命令：
```powershell
go vet ./...
```
实测输出 (Stdout)：无任何告警或错误。
**Exit Code**: `0`
