# 变更记录：003-fix-gap-002-and-011-idempotency-rr

- **变更编号**：003
- **创建时间**：2026-10-21T08:40:00Z (UTC)
- **完成时间**：2026-10-21T08:46:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：
  - GAP-002 (P1: 订单幂等校验在 MySQL RR 隔离级别下的并发同 Key 查询异常与失败回滚状态丢失)
  - GAP-011 (P1: 修复测试套件中并发幂等的错误断言，消除虚假全绿并覆盖并发状态机)
- **关联 ADR**：ADR-001 (高并发低延迟与分布式最终一致性库存架构设计) §2 决策一
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在订单系统的并发幂等设计中，`OrderService.CreateOrder` 依靠 `order_idempotency_keys` 表的主键/唯一索引 `(user_id, idempotency_key)` 保证同请求防重。在业务主事务中，首先执行 `dao.TryCreateOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey, requestHash)`（通过 `clause.OnConflict{DoNothing: true}` 抢占幂等标记）。

然而在 MySQL Repeatable Read (RR，可重复读) 隔离级别下，存在以下两项严重的并发与测试架构缺陷：

1. **GAP-002：MySQL RR 快照读引发的 `RecordNotFound` 异常穿透与库存泄漏风险**：
   - 当多个客户端使用相同 `IdempotencyKey` 并发到达时，先发事务 T1 成功抢占插入记录（状态为 `OrderBeingCreated`），但此时 T1 事务尚未提交；
   - 后到达的并发事务 T2 执行 `TryCreateOrderIdempotencyKey` 因唯一键冲突被忽略（`acquired = false`）；
   - T2 紧接着调用 `dao.GetOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey)` 查询幂等记录状态；
   - **在 MySQL RR 隔离级别下，事务 T2 基于其启动快照进行一致性非锁定读（Consistent Nonlocking Read），根本读不到 T1 尚未提交的新增记录！** 导致 `GetOrderIdempotencyKey` 返回 `gorm.ErrRecordNotFound`；
   - 在历史实现中，`if err != nil { return err }` 将底层 `ErrRecordNotFound` 直接作为内部错误向上冒泡，导致客户端收到非预期的未知错误，而不是业务明确定义的“订单处理中/创建中”错误（`ErrOrderBeingCreated`）；
   - 更严重的是：由于 ADR-001 决策一已将 Redis 原子预扣前置（在开启 MySQL 事务前预扣），如果事务内报错后未能精细拦截与回滚释放多占预扣，极易导致 Redis 预扣库存永久泄漏。

2. **GAP-011：测试套件并发断言错误假设与虚假全绿**：
   - 现有的 `TestCreateOrder_ConcurrentSameIdempotencyKey_CreatesOneOrder` 错误假设并发的全部 12 个请求均必须返回 `err == nil`，完全违背了并发抢占的客观状态流转；
   - 在真实并发场景下，先抢到锁并完成提交的请求返回 `err == nil`，在先发事务执行期间并发到达的后置请求必须稳定得到 `ErrOrderBeingCreated`，待先发事务提交后重放的请求返回已创建订单与 `err == nil`；
   - 严禁向客户端抛出 `RecordNotFound` 或数据库内部未知错误；
   - 此外，缺乏轻量化模拟 `RecordNotFound`（快照隔离边界）的单元测试，导致测试门槛依赖外部全量环境。

---

## 2. 修改文件清单与实现方案

1. `internal/service/order_service.go`:
   - 在 `CreateOrder` 事务内部：
     ```go
     acquired, err := dao.TryCreateOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey, requestHash)
     if err != nil {
         return err
     }
     if !acquired {
         record, err := dao.GetOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey)
         if err != nil {
             if errors.Is(err, gorm.ErrRecordNotFound) {
                 // 在 MySQL RR 隔离级别下，并发先到达的事务已插入记录但尚未提交，后到达的事务快照读不到，必须作为“创建中”处理
                 return ErrOrderBeingCreated
             }
             return err
         }
         ...
     ```
   - 确保当返回 `ErrOrderBeingCreated` 时：
     1. GORM 事务安全中止并回滚；
     2. 事务外 `err != nil` 分支触发 `p.releasePreDeductedInventory(context.WithoutCancel(ctx), orderID)` 释放本次请求占用的 Redis 预扣凭证，杜绝库存凭证泄漏；
     3. 业务指标上报 `classifyOrderCreateError` 精确归类为 `"idempotency_in_progress"`。

2. `internal/service/order_service_test.go`:
   - 纠正 `TestCreateOrder_ConcurrentSameIdempotencyKey_CreatesOneOrder` 断言（修复 GAP-011）：
     - 允许请求返回 `err == nil`（首次创建或幂等重放）或 `service.ErrOrderBeingCreated`（并发抢占中）；
     - 严禁出现除 `ErrOrderBeingCreated` 之外的任何错误（坚决拒绝 `RecordNotFound`）；
     - 断言至少有 1 个请求创建成功且所有成功返回的订单 ID 严格一致；
     - 断言 `successCount + inProgressCount == requests`（全部 12 个请求合法收敛）；
     - 断言数据库中订单总量为 1，库存流水记录为 1，库存总扣减为 1。
   - 新增零依赖轻量化单元测试 `TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct`：
     - 利用 Go 标准库 `database/sql/driver` 构建 `mockRRConnector`，模拟 `TryCreateOrderIdempotencyKey` 返回未抢占（`RowsAffected = 0`）、且 `GetOrderIdempotencyKey` 返回 0 行触发 `gorm.ErrRecordNotFound`（精准映射 MySQL RR 快照读场景）；
     - 严格验证 `CreateOrder` 稳定返回 `ErrOrderBeingCreated`，绝不冒泡底层的 `ErrRecordNotFound`；
     - 严格断言 Redis 预扣凭证已被 `fakeInventoryPreDeductor` 正确释放（`len(guard.releases) == 1 && guard.releases[0] == guard.preDeducts[0]`），证明高并发冲突下零库存泄漏；
     - 验证业务指标被精确记录为 `"idempotency_in_progress"`。

---

## 3. 验证命令与实测结果回填

### 3.1 MySQL RR 隔离级别快照未见单测与预扣释放断言
执行命令：
```powershell
go test -v -race -count=1 -run TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct ./internal/service/...
```
实测输出 (Stdout)：
```text
=== RUN   TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct
--- PASS: TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct (0.00s)
PASS
ok  	go-order-management-system/internal/service	3.303s
```
**Exit Code**: `0`

### 3.2 订单服务包全量单元测试与竞态检测
执行命令：
```powershell
go test -v -race -count=1 ./internal/service/...
```
实测输出 (Stdout)：
```text
=== RUN   TestCreateOrder_InsufficientStock
--- PASS: TestCreateOrder_InsufficientStock (0.00s)
=== RUN   TestCreateOrder_RecordsBusinessMetrics
--- PASS: TestCreateOrder_RecordsBusinessMetrics (0.00s)
=== RUN   TestCreateOrder_RecordsFailureBusinessMetrics
--- PASS: TestCreateOrder_RecordsFailureBusinessMetrics (0.00s)
=== RUN   TestCreateOrder_RedisPreDeductInsufficient
--- PASS: TestCreateOrder_RedisPreDeductInsufficient (0.00s)
=== RUN   TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess
--- PASS: TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess (0.00s)
=== RUN   TestCreateOrder_ReleasesRedisPreDeductOnTransactionFailure
--- PASS: TestCreateOrder_ReleasesRedisPreDeductOnTransactionFailure (0.00s)
=== RUN   TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct
--- PASS: TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct (0.00s)
PASS
ok  	go-order-management-system/internal/service	2.997s
```
**Exit Code**: `0`

### 3.3 全仓单元测试与竞态检测
执行命令：
```powershell
go test -v -race -count=1 ./...
```
实测输出 (Stdout)：
```text
ok  	go-order-management-system/cmd	2.164s
ok  	go-order-management-system/config	1.846s
ok  	go-order-management-system/internal/service	2.979s
ok  	go-order-management-system/migrations/integration	1.948s
ok  	go-order-management-system/pkg/database	2.005s
ok  	go-order-management-system/pkg/idgen	1.648s
ok  	go-order-management-system/router	3.175s
```
**Exit Code**: `0`

### 3.4 静态代码检查
执行命令：
```powershell
go vet ./...
```
实测输出 (Stdout)：无任何告警或错误。
**Exit Code**: `0`
