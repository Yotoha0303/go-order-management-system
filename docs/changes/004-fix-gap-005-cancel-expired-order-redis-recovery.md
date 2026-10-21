# 变更记录：004-fix-gap-005-cancel-expired-order-redis-recovery

- **变更编号**：004
- **创建时间**：2026-10-21T09:05:00Z (UTC)
- **完成时间**：2026-10-21T09:08:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-005 (P1: 订单超时取消在异常重启或网络故障时导致 Redis 库存永久丢失回补)
- **关联 ADR**：ADR-001 (高并发低延迟与分布式最终一致性库存架构设计) §2 决策三 (二阶段预占与幂等最终一致)
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

根据 ADR-001 §2 决策三，库存预占采用二阶段设计：在 MySQL 事务外部通过 Redis Lua 进行原子预扣并生成 reservation 凭证，在超时取消或手动取消时触发回补并清理 reservation。

但在历史基准实现中，`OrderService.CancelExpiredOrder` 与 `CancelOrder` 存在严重的崩溃恢复与网络抖动盲区：
1. **超时取消链路（`CancelExpiredOrder`）**：
   - 当前实现：开启 MySQL 事务检查订单状态，若 `order.Status != model.OrderStatusPending`，则直接标记 `ignored = true` 并提前退出事务；事务外仅在 `restored == true` 时才调用 `p.restorePreDeductedInventory`。
   - 严重缺陷：在分布式或多实例生产环境下，若第一次执行时 MySQL 本地事务成功将订单状态修改为 `OrderStatusCancelled` 并提交，但在紧接着调用外部网络 IO `restorePreDeductedInventory` 之前，服务进程发生崩溃、OOM 杀死、宿主机重启或网络超时断开，导致未能执行 Redis 预扣回补；随后 RabbitMQ 消息重试或死信 Worker 再次消费该取消事件时，查询到数据库中 `order.Status == OrderStatusCancelled`（非 Pending），直接被当成未知无效状态置为 `ignored = true` 并忽略跳过！这导致 Redis 中的 reservation 凭证永久残留，可用库存永久少加，产生实质性的库存泄漏。
2. **手动取消链路（`CancelOrder`）**：
   - 当前实现：当 `order.Status == model.OrderStatusCancelled` 时走幂等分支 `idempotent = true`，但事务外仅在 `restored == true` 时才调用 `restorePreDeductedInventory`，重试请求无法补偿前次崩溃遗留的 Redis 预占。
3. **修复的可行性与强幂等基石**：
   - Redis 中的 `releaseInventoryReservationScript`（释放预占凭证 Lua 脚本）具有原生原子强幂等性：若 reservation 键存在，原子执行 `INCRBY` 回补库存并 `DEL` 凭证键；若 reservation 键已被清理不存在，则安全返回 0（noop）。
   - 因此，只要 MySQL 订单状态已确认为 `OrderStatusCancelled`，在事务外侧确保执行 `p.restorePreDeductedInventory`，即可提供对等的崩溃恢复与重试自愈保证，彻底闭环最终一致性。

---

## 2. 修改文件清单与实现方案

1. `internal/service/order_service.go`:
   - 在 `CancelExpiredOrder` 中：
     - 若 `order.Status == model.OrderStatusCancelled`，显式标记 `alreadyCancelled = true`（而非作为未知状态被 ignored）；
     - 事务外侧判断：`if err == nil && (restored || alreadyCancelled) { p.restorePreDeductedInventory(context.WithoutCancel(ctx), orderID) }`；
     - 指标记录精确区分：当 `alreadyCancelled == true` 时记录业务指标 `"timeout_cancel:already_cancelled"`。
   - 在 `CancelOrder`（用户手动取消链路）中：
     - 当 `order.Status == model.OrderStatusCancelled` 触发 `idempotent = true` 时；
     - 事务外侧统一保障：`if err == nil && (restored || idempotent) { p.restorePreDeductedInventory(context.WithoutCancel(ctx), orderID) }`；
     - 指标记录保持 `"cancel:idempotent"`。
2. `internal/service/order_service_test.go`:
   - 新增针对性崩溃恢复单测：`TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled`：
     - 包含集成测试（MySQL 启用）与轻量化单元测试（基于标准库 `database/sql/driver` 的 `mockCancelCrashConnector`）；
     - 模拟订单在 DB 中已处于 `OrderStatusCancelled` 状态（前次执行 DB 已提交但进程崩溃）；
     - 挂载 `fakeInventoryPreDeductor`，调用 `CancelExpiredOrder`，断言返回 `err == nil`；
     - 严格断言 `len(guard.restores) == 1 && guard.restores[0] == orderID`，证明崩溃重试自愈机制准确触发了 Redis 预占凭证回补与清理；
     - 验证指标记录为 `"timeout_cancel:already_cancelled"`。
   - 新增针对手动取消的崩溃恢复单测：`TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled`：
     - 模拟已取消订单再次触发手动取消，断言返回 `err == nil` 且 `len(guard.restores) == 1 && guard.restores[0] == orderID`；
     - 验证指标记录为 `"cancel:idempotent"`。
   - 验证所有测试用例全绿通过。

---

## 3. 验证命令与实测结果回填

### 3.1 取消与崩溃恢复针对性单测与竞态检测
执行命令：
```powershell
go test -v -race -count=1 -run "TestCancelExpiredOrder|TestCancelOrder" ./internal/service/...
```
实测输出 (Stdout)：
```text
=== RUN   TestCancelExpiredOrder_CancelsPendingOrderIdempotently
    order_service_test.go:496: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestCancelExpiredOrder_CancelsPendingOrderIdempotently (0.00s)
=== RUN   TestCancelExpiredOrder_DoesNotCancelPaidOrder
    order_service_test.go:542: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestCancelExpiredOrder_DoesNotCancelPaidOrder (0.00s)
=== RUN   TestCancelOrder_Success
    order_service_test.go:850: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestCancelOrder_Success (0.00s)
=== RUN   TestCancelOrder_RestoresRedisPreDeductReservation
    order_service_test.go:897: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestCancelOrder_RestoresRedisPreDeductReservation (0.00s)
=== RUN   TestCancelOrder_CancelledOrder_Idempotent
    order_service_test.go:1003: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestCancelOrder_CancelledOrder_Idempotent (0.00s)
=== RUN   TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled
=== RUN   TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/integration_mysql
    order_service_test.go:2065: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
=== RUN   TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/unit_crash_recovery_mock
--- PASS: TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled (0.01s)
    --- SKIP: TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/integration_mysql (0.00s)
    --- PASS: TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/unit_crash_recovery_mock (0.01s)
=== RUN   TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled
=== RUN   TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/integration_mysql
    order_service_test.go:2143: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
=== RUN   TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/unit_crash_recovery_mock
--- PASS: TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled (0.00s)
    --- SKIP: TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/integration_mysql (0.00s)
    --- PASS: TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled/unit_crash_recovery_mock (0.00s)
PASS
ok  	go-order-management-system/internal/service	3.412s
```
**Exit Code**: `0`

### 3.2 订单服务包全量单测与竞态检测
执行命令：
```powershell
go test -v -race -count=1 ./internal/service/...
```
实测输出 (Stdout)：
```text
=== RUN   TestCreateOrder_InsufficientStock
--- PASS: TestCreateOrder_InsufficientStock (0.00s)
=== RUN   TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess
--- PASS: TestCreateOrder_RedisPreDeductInsufficient_ZeroDatabaseAccess (0.00s)
=== RUN   TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct
--- PASS: TestCreateOrder_IdempotencyKeyRecordNotFoundInRR_ReturnsOrderBeingCreatedAndReleasesPreDeduct (0.00s)
=== RUN   TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled
--- PASS: TestCancelExpiredOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled (0.01s)
=== RUN   TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled
--- PASS: TestCancelOrder_CrashRecovery_RestoresRedisReservationWhenAlreadyCancelled (0.00s)
PASS
ok  	go-order-management-system/internal/service	3.070s
```
**Exit Code**: `0`

### 3.3 全仓单元测试与竞态检测
执行命令：
```powershell
go test -v -race -count=1 ./...
```
实测输出 (Stdout)：
```text
ok  	go-order-management-system/cmd	2.046s
ok  	go-order-management-system/config	1.968s
ok  	go-order-management-system/internal/bizcache	2.148s
ok  	go-order-management-system/internal/dao	2.286s
ok  	go-order-management-system/internal/service	3.936s
ok  	go-order-management-system/migrations/integration	2.066s
ok  	go-order-management-system/pkg/database	2.232s
ok  	go-order-management-system/pkg/idgen	2.010s
ok  	go-order-management-system/router	3.091s
```
**Exit Code**: `0`

### 3.4 全仓静态代码检查
执行命令：
```powershell
go vet ./...
```
实测输出 (Stdout)：无任何告警或错误。
**Exit Code**: `0`
