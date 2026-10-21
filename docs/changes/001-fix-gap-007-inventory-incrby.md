# 变更记录：001-fix-gap-007-inventory-incrby

- **变更编号**：001
- **创建时间**：2026-10-21T06:38:00Z (UTC)
- **完成时间**：2026-10-21T06:48:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-007 (P0 Critical: AddInventory 补货使用绝对值覆盖写入 Redis 导致库存回退竞态 Bug)
- **关联 ADR**：ADR-001 (高并发低延迟与分布式最终一致性库存架构设计) §2 决策二 & §3.2
- **状态**：已完成 (验证通过)

---

## 1. 变更动机与背景

在原实现中，`InventoryService.AddInventory` 在完成本地 MySQL 事务计算出 `afterQuantity` 之后，直接通过 `p.setInventoryStock(ctx, req.ProductID, afterQuantity)`（底层执行 Redis `SET` 命令）进行绝对值覆盖写入。
在高并发补货或网络传输抖动场景下，早开始执行的请求由于网络延迟较晚到达 Redis，其较小的绝对值会覆盖后开始但较早到达 Redis 的较大绝对值，导致补货库存凭空蒸发（例如并发 +10 与 +20，期望最终累加 +30，可能最终被置为仅 +10）。

根据 ADR-001 决策二，必须废除绝对值覆盖写入，改由 Redis 原生原子增量（`INCRBY`）操作同步补货增量；若 Redis 缓存不存在（冷启动），则使用 `SET` 将计算后的 `fallbackQuantity` 写入，确保原子性与最终一致性。

---

## 2. 修改文件清单

1. `internal/service/inventory_interface.go`:
   - 在 `InventoryStockWriter` 接口中增加原子增量方法：
     `IncrInventoryStock(ctx context.Context, productID int64, delta int64, fallbackQuantity int64)`
2. `internal/bizcache/inventory_stock_guard.go`:
   - 定义 Lua 脚本 `incrInventoryStockScript`：
     - 若 `EXISTS key == 1`，执行 `INCRBY key delta`；
     - 若 `EXISTS key == 0`，执行 `SET key fallbackQuantity`；
   - 实现 `InventoryStockGuard.IncrInventoryStock(...)` 方法，调用上述 Lua 脚本并打点业务指标（`app_redis_inventory_sync_total`）。
3. `internal/service/inventory_service.go`:
   - 将 `AddInventory` 中的 `p.setInventoryStock(ctx, req.ProductID, afterQuantity)` 替换为 `p.incrInventoryStock(ctx, req.ProductID, req.Quantity, afterQuantity)`；
   - 增加辅助方法 `p.incrInventoryStock(...)`。
4. `internal/service/inventory_service_test.go`:
   - 在 `fakeInventoryStockWriter` 中实现 `IncrInventoryStock(...)` 并记录调用入参；
   - 更新 `TestAddInventory_SyncsRedisStock` 验证 `IncrInventoryStock` 被精准调用（`delta=5`, `fallback=15`），且断言绝对值覆盖方法 `SetInventoryStock` 绝对未被调用；
   - 新增 `TestAddInventory_Validation_NoDB` 单元测试，验证参数校验拦截与防零调用的负向用例。
5. `internal/bizcache/inventory_stock_guard_test.go`:
   - 更新 `TestInventoryStockGuard_NoRedis` 覆盖 `IncrInventoryStock`；
   - 新增 `TestInventoryStockGuard_IncrInventoryStock_DisabledValidation` 覆盖所有参数非法与空指针场景，证伪指标统计；
   - 新增 `TestInventoryStockGuard_IncrInventoryStock_WithRedis` 覆盖冷启动 fallbackQuantity 兜底与增量 INCRBY 双分支；
   - 新增 `TestInventoryStockGuard_IncrInventoryStock_Concurrent_WithRedis` 启动 20 并发协程同时对 Redis 执行原子递增，验证最终一致性无竞态丢数。

---

## 3. 验证命令与实测结果回填

### 3.1 本地竞态检测 (`-race`)
执行命令：
```powershell
go test -v -race ./internal/bizcache/... ./internal/service/...
```
实测输出 (Stdout)：
```text
=== RUN   TestInventoryStockGuardKeyNames
--- PASS: TestInventoryStockGuardKeyNames (0.00s)
=== RUN   TestInventoryStockGuard_NoRedis
--- PASS: TestInventoryStockGuard_NoRedis (0.00s)
=== RUN   TestInventoryStockGuard_RecordsDisabledMetrics
--- PASS: TestInventoryStockGuard_RecordsDisabledMetrics (0.00s)
=== RUN   TestInventoryStockGuard_RebuildUnavailableNoRedis
--- PASS: TestInventoryStockGuard_RebuildUnavailableNoRedis (0.00s)
=== RUN   TestInventoryStockGuard_GetInventoryStocksUnavailableNoRedis
--- PASS: TestInventoryStockGuard_GetInventoryStocksUnavailableNoRedis (0.00s)
=== RUN   TestInventoryStockGuard_IncrInventoryStock_DisabledValidation
--- PASS: TestInventoryStockGuard_IncrInventoryStock_DisabledValidation (0.00s)
=== RUN   TestProductDetailCacheKey
--- PASS: TestProductDetailCacheKey (0.00s)
=== RUN   TestProductDetailCache_NoRedis
--- PASS: TestProductDetailCache_NoRedis (0.00s)
PASS
ok  	go-order-management-system/internal/bizcache	2.211s
...
=== RUN   TestAddInventory_Validation_NoDB
--- PASS: TestAddInventory_Validation_NoDB (0.00s)
=== RUN   TestGetProductByID_CacheHit
--- PASS: TestGetProductByID_CacheHit (0.00s)
PASS
ok  	go-order-management-system/internal/service	3.147s
```
**Exit Code**: `0`

### 3.2 真实 Redis 环境集成与并发竞态证伪实测
执行命令：
```powershell
$env:REDIS_PASSWORD="pass4Redis"; $env:RUN_REDIS_TEST="1"; go test -v -race -run TestInventoryStockGuard ./internal/bizcache/...
```
实测输出 (Stdout)：
```text
=== RUN   TestInventoryStockGuardKeyNames
--- PASS: TestInventoryStockGuardKeyNames (0.00s)
=== RUN   TestInventoryStockGuard_NoRedis
--- PASS: TestInventoryStockGuard_NoRedis (0.00s)
=== RUN   TestInventoryStockGuard_RecordsDisabledMetrics
--- PASS: TestInventoryStockGuard_RecordsDisabledMetrics (0.00s)
=== RUN   TestInventoryStockGuard_RebuildUnavailableNoRedis
--- PASS: TestInventoryStockGuard_RebuildUnavailableNoRedis (0.00s)
=== RUN   TestInventoryStockGuard_GetInventoryStocksUnavailableNoRedis
--- PASS: TestInventoryStockGuard_GetInventoryStocksUnavailableNoRedis (0.00s)
=== RUN   TestInventoryStockGuard_PreDeductAndRelease_WithRedis
--- PASS: TestInventoryStockGuard_PreDeductAndRelease_WithRedis (0.01s)
=== RUN   TestInventoryStockGuard_GetInventoryStocks_WithRedis
--- PASS: TestInventoryStockGuard_GetInventoryStocks_WithRedis (0.01s)
=== RUN   TestInventoryStockGuard_InsufficientDoesNotDeduct_WithRedis
--- PASS: TestInventoryStockGuard_InsufficientDoesNotDeduct_WithRedis (0.01s)
=== RUN   TestInventoryStockGuard_MissingKeySkips_WithRedis
--- PASS: TestInventoryStockGuard_MissingKeySkips_WithRedis (0.01s)
=== RUN   TestInventoryStockGuard_ConfirmDeletesReservation_WithRedis
--- PASS: TestInventoryStockGuard_ConfirmDeletesReservation_WithRedis (0.01s)
=== RUN   TestInventoryStockGuard_IncrInventoryStock_DisabledValidation
--- PASS: TestInventoryStockGuard_IncrInventoryStock_DisabledValidation (0.00s)
=== RUN   TestInventoryStockGuard_IncrInventoryStock_WithRedis
--- PASS: TestInventoryStockGuard_IncrInventoryStock_WithRedis (0.01s)
=== RUN   TestInventoryStockGuard_IncrInventoryStock_Concurrent_WithRedis
--- PASS: TestInventoryStockGuard_IncrInventoryStock_Concurrent_WithRedis (0.02s)
PASS
ok  	go-order-management-system/internal/bizcache	2.111s
```
**Exit Code**: `0`

---

## 4. 下游联动与状态

- **GAP 状态**：GAP-007 (P0 Critical) 核心代码实现与单测已修复完成，可进入 03 环测试验收。
- **约束核验**：
  - 遵守 Clean Architecture / DDD 分层边界；
  - 严格传递 `context.Context`，采用带超时的子上下文与 `context.WithoutCancel` 兜底；
  - 并发安全保证：通过 Redis 原生 Lua 脚本原子执行 `INCRBY`，单测并发 20 协程无竞争通过；
  - 指标联动：精确打点 `app_redis_inventory_sync_total{result="success|disabled|error"}`；
  - 严禁篡改 `memory.md`（硬约束已遵守）。
