# 变更记录：008-fix-gap-006-inventory-locked-quantity

- **变更编号**：008
- **创建时间**：2026-10-21T14:55:00Z (UTC)
- **完成时间**：2026-10-21T15:13:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-006 (P2: 健全库存数据模型与对账自愈，Issue #7)
- **关联 ADR**：ADR-003 (健全库存三态数据模型、全局防死锁偏序加锁与缓存对账自愈架构设计)
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在现有的高并发订单与库存体系中，存在以下三项制约生产级稳定性的缺陷：
1. **数据模型二态缺陷**：
   - `product_inventories` 表仅有 `stock_quantity` 字段，缺少锁定库存（`locked_quantity`），无法区分待支付预占与已付款核销出库，缺乏严格的库存守恒数学关系；
2. **并发交叉死锁隐患**：
   - 下单 `CreateOrder` 与取消 `cancelPendingOrder` 的多商品行锁获取顺序无序，当并发交错处理包含相同商品的订单时，容易引发哲学家就餐环路等待死锁（MySQL 1213 错误）；
3. **缓存对账缺乏自愈与监控度量**：
   - 对账 Worker 仅打印警告日志，无法在检测到 Redis 与 MySQL 差异时自主纠偏，且未暴露 Prometheus 业务监控指标。

根据 ADR-003 设计：
- 引入三态库存模型：$TotalPhysicalStock = stock\_quantity + locked\_quantity$，增加 `locked_quantity` 及非负约束；
- 引入全局严格偏序加锁准则：所有涉及多商品库存操作的事务统一按 `product_id ASC` 升序加锁并去重合并；
- 实现对账 Worker 的 `AutoFix` 自动回写机制与 Prometheus 业务指标（`app_inventory_reconcile_diff_total`, `app_inventory_reconcile_autofix_total`）。

---

## 2. 修改与新增文件清单

1. `internal/model/inventory.go`:
   - 为 `Inventory` 增加 `LockedQuantity int64`，带有 GORM tag `type:bigint;not null;default:0;check:locked_quantity >= 0`。
2. `internal/dao/inventory_dao.go`:
   - 增加 `ReserveInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) error`（扣减 `stock_quantity`，增加 `locked_quantity`）；
   - 增加 `ReleaseLockedInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) error`（回补 `stock_quantity`，扣减 `locked_quantity`）；
   - 增加 `ConsumeLockedInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) error`（核销扣减 `locked_quantity`）。
3. `internal/dao/order_dao.go`:
   - 新增 `ListOrderItemsByOrderIDForInventory(ctx context.Context, db *gorm.DB, orderID int64) ([]*model.OrderItem, error)`（显式携带 `ORDER BY product_id ASC`）。
4. `internal/service/order_service.go`:
   - `CreateOrder`：事务内扣减前，通过 `sortedOrderItems` 按 `product_id ASC` 升序排序与去重，调用 `ReserveInventory` 执行原子预占；
   - `cancelPendingOrder`：获取 order_items 后调用 `aggregateAndSortOrderItems` 保证 `product_id ASC` 偏序遍历，调用 `ReleaseLockedInventory`；
   - `PayOrder`：订单流转为 Paid 时在事务内按 `product_id ASC` 遍历并调用 `ConsumeLockedInventory` 实物核销出库。
5. `internal/service/inventory_service.go`:
   - 暴露 `AutoFixRedisInventoryStock` 与 `SetInventoryStock` 支持批量纠偏与单键回写。
6. `config/config.go` & `config.yml`:
   - 新增 `InventoryReconcileConfig.AutoFix bool` 字段与环境变量 `INVENTORY_RECONCILE_AUTO_FIX` 绑定；在 `config.yml` 中默认配置 `autoFix: true`。
7. `internal/inventoryreconcile/worker.go`:
   - 检测到差异且 `AutoFix == true` 时，自动调用 `stockWriter.SetInventoryStock` 修复 Redis 缓存，记录审计日志与 Prometheus 指标。
8. `internal/observability/business_metrics.go`:
   - 增加 `RecordInventoryReconcileDiff` 与 `RecordInventoryReconcileAutoFix` 指标与收集方法。
9. `internal/app/deps.go`:
   - 在依赖装配中将 `inventoryStockGuard` 与 `metrics.Business` 注入 `inventoryreconcile.Worker`。
10. 测试用例：
   - `internal/dao/inventory_dao_test.go`：测试锁定库存原子三态流转（预占、释放、核销）及 SQL 条件校验；
   - `internal/service/order_deadlock_test.go`：并发交叉订单测试证明 `product_id ASC` 消除哲学家就餐死锁环路；
   - `internal/service/order_inventory_lifecycle_test.go`：测试库存三态数学守恒与流转阶段断言；
   - `internal/inventoryreconcile/worker_test.go`：测试对账 Worker 的 AutoFix 自愈、指标上报与无 writer 容错。

---

## 3. 实测验证命令与真实 stdout

### 3.1 针对性模块测试（带竞态检测）

- 命令：`go test -v -race ./internal/service/... ./internal/dao/... ./internal/inventoryreconcile/...`
- 真实 stdout（摘要）：
```
=== RUN   TestOrderItemsSorting_EliminatesDeadlockCircularWait
--- PASS: TestOrderItemsSorting_EliminatesDeadlockCircularWait (0.00s)
=== RUN   TestConcurrentCrossOrder_PartialOrderLocking_NoDeadlock
--- PASS: TestConcurrentCrossOrder_PartialOrderLocking_NoDeadlock (0.00s)
=== RUN   TestInventoryThreeStateLifecycle_ConservationAndTransitions
--- PASS: TestInventoryThreeStateLifecycle_ConservationAndTransitions (0.00s)
PASS
ok  	go-order-management-system/internal/service	3.265s

=== RUN   TestReserveInventory_Unit
=== RUN   TestReserveInventory_Unit/successful_reservation_updates_stock_and_locked_quantity
=== RUN   TestReserveInventory_Unit/insufficient_stock_returns_ErrReserveInventoryFailed
--- PASS: TestReserveInventory_Unit (0.00s)
=== RUN   TestReleaseLockedInventory_Unit
=== RUN   TestReleaseLockedInventory_Unit/successful_release_restores_stock_and_decrements_locked_quantity
=== RUN   TestReleaseLockedInventory_Unit/insufficient_locked_stock_returns_ErrReleaseLockedInventoryFailed
--- PASS: TestReleaseLockedInventory_Unit (0.00s)
=== RUN   TestConsumeLockedInventory_Unit
=== RUN   TestConsumeLockedInventory_Unit/successful_consume_decrements_locked_quantity
=== RUN   TestConsumeLockedInventory_Unit/insufficient_locked_quantity_returns_ErrConsumeLockedInventoryFailed
--- PASS: TestConsumeLockedInventory_Unit (0.00s)
=== RUN   TestListOrderItemsByOrderIDForInventory_OrdersByProductIDASC
--- PASS: TestListOrderItemsByOrderIDForInventory_OrdersByProductIDASC (0.01s)
PASS
ok  	go-order-management-system/internal/dao	2.580s

=== RUN   TestWorkerRunOnceLogsDifferences
--- PASS: TestWorkerRunOnceLogsDifferences (0.02s)
=== RUN   TestWorkerRunOnce_AutoFixEnabled_RestoresRedisAndRecordsMetrics
--- PASS: TestWorkerRunOnce_AutoFixEnabled_RestoresRedisAndRecordsMetrics (0.00s)
=== RUN   TestWorkerRunOnce_AutoFixEnabled_NoStockWriter_LogsErrorAndRecordsFailedMetric
--- PASS: TestWorkerRunOnce_AutoFixEnabled_NoStockWriter_LogsErrorAndRecordsFailedMetric (0.00s)
=== RUN   TestWorkerRunOnceReturnsAndLogsError
--- PASS: TestWorkerRunOnceReturnsAndLogsError (0.00s)
=== RUN   TestNewWorkerRejectsInvalidConfig
--- PASS: TestNewWorkerRejectsInvalidConfig (0.00s)
PASS
ok  	go-order-management-system/internal/inventoryreconcile	3.261s
```

### 3.2 全仓静态代码分析

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
ok  	go-order-management-system/internal/inventoryreconcile	3.261s
ok  	go-order-management-system/internal/middleware	2.904s
ok  	go-order-management-system/internal/observability	1.600s
ok  	go-order-management-system/internal/ordertimeout	4.655s
ok  	go-order-management-system/internal/service	3.396s
ok  	go-order-management-system/migrations/integration	2.175s
ok  	go-order-management-system/pkg/database	1.704s
ok  	go-order-management-system/pkg/idgen	2.104s
ok  	go-order-management-system/router	3.322s
```
全仓 100% 通过，零回归，零数据竞争。
