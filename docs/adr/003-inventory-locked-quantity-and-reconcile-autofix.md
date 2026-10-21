# ADR-003: 健全库存三态数据模型、全局防死锁偏序加锁与缓存对账自愈架构设计

- **Title**: ADR-003: 健全库存三态数据模型、全局防死锁偏序加锁与缓存对账自愈架构设计
- **Status**: Accepted
- **Date**: 2026-10-21
- **Deciders**: 存储与数据流工程师 (`data-engineer`), 分布式系统架构师 (`system-architect`), 后端开发专家 (`backend-dev`), SRE 稳定性专家 (`sre-operator`)
- **Target System**: 订单与库存系统 (`go-order-management-system`)
- **Related GAPs**: GAP-006 (P2, Issue #7)

---

## 1. 上下文与痛点剖析 (Context & Problem Statement)

在现有高并发订单系统中（见 ADR-001），虽然前置了 Redis Lua 内存预扣作为流量削峰护城河，并在 MySQL 本地事务中落地了物理库存扣减，但在底层数据模型、事务加锁拓扑与运维对账维度，依然存在三项阻碍生产级稳定性的关键缺陷：

### 1.1 数据模型二态缺陷：缺少锁定库存（`locked_quantity`）
在当前表结构 `product_inventories` 中，仅存在 `stock_quantity` 字段：
- 下单时直接从 `stock_quantity` 扣减，取消订单时加回；
- **状态语义模糊**：数据库层面无法区分“已被用户下单但未付款预占的库存”与“已付款完成核销出库的库存”。总物理库存数量无法直观反映，盘点与对账时缺乏物理库存与在途预占的严密数学关系；
- **极端状态失控**：缺少针对预占状态的独立约束，无法支持更细粒度的超时关单、部分核销与按件释放。

### 1.2 并发交叉死锁隐患：加锁顺序无序引发哲学家就餐环路
在 `internal/service/order_service.go` 中，多商品事务加锁存在不对称性：
1. `CreateOrder`：按照前端/客户端传入的 `items` 数组顺序，逐个调用 `dao.GetInventoryByProductIDForUpdate(ctx, tx, itemReq.ProductID)`；
2. `cancelPendingOrder`：从 `order_items` 表查出明细，由于 `ListOrderItemsByOrderID` 采用 `ORDER BY id ASC`，其加锁顺序取决于明细自增主键 `id` 的插入顺序；
3. **死锁成环复现**：
   - 订单 A 包含商品 `[P2, P1]`，进入 `CreateOrder`：事务 A 先持有 P2 的行排它锁，试图获取 P1 的锁；
   - 订单 B 包含商品 `[P1, P2]`，在同一时刻创建，或订单 B 被超时取消（明细顺序刚好是 `P1, P2`）：事务 B 先持有 P1 的排它锁，试图获取 P2 的锁；
   - 事务 A 等待事务 B 释放 P1，事务 B 等待事务 A 释放 P2，形成典型的**交叉等待死锁环路**（Circular Wait），触发 MySQL InnoDB `Deadlock found when trying to get lock; try restarting transaction (errno 1213)`，导致订单大面积失败。

### 1.3 缓存对账缺乏自愈与告警能力
后台对账协程 `internal/inventoryreconcile/worker.go` 周期性执行 `ReconcileRedisInventoryStock`：
- 当前实现仅仅通过 `w.logger.Warn` 打印差异日志；
- **无法自动自愈**：当 Redis 发生宕机恢复、网络分区抖动或人为数据订正时，Redis 可售库存与 MySQL 权威库存产生漂移后无法自动纠偏，必须人工介入干预；
- **无监控度量**：对账差异数与自愈结果未暴露 Prometheus 指标，SRE 监控平台与 Alertmanager 无法感知静默的数据漂移。

---

## 2. 架构决策与规范体系 (Decisions)

### 2.1 库存三态流转模型与 DDL 规范

#### 2.1.1 核心数学恒等式
库存系统必须建立在严格的守恒定律之上。定义三大核心指标：
1. **总物理库存 (Total Physical Stock)**：仓库中实际存放的物理实物总量；
2. **可用库存 / 可售库存 (Available Stock, `stock_quantity`)**：用户可立即下单扣减的现货库存；
3. **锁定库存 / 预占库存 (Locked / Reserved Stock, `locked_quantity`)**：已被创建订单预占、处于待支付窗口（如 30 分钟）内的在途库存。

$$TotalPhysicalStock = stock\_quantity + locked\_quantity$$
$$stock\_quantity \ge 0 \quad \land \quad locked\_quantity \ge 0$$

#### 2.1.2 数据库迁移脚本 (`migrations/00016_add_locked_quantity_to_product_inventories.sql`)
```sql
-- +goose Up
ALTER TABLE product_inventories
    ADD COLUMN locked_quantity BIGINT NOT NULL DEFAULT 0 AFTER stock_quantity,
    ADD CONSTRAINT chk_product_inventories_locked_quantity_non_negative CHECK (locked_quantity >= 0);

-- +goose Down
ALTER TABLE product_inventories
    DROP CHECK chk_product_inventories_locked_quantity_non_negative,
    DROP COLUMN locked_quantity;
```

#### 2.1.3 库存生命周期状态机与原子变更公式

```
      +-------------------------------------------------------+
      |               入库/补货 (+Q 到可用库存)                 |
      v                                                       |
+-------------------+      下单预占       +---------------------+      支付核销       +-------------------+
|  可用库存 S       | -----------------> |  锁定库存 L         | -----------------> |  实物出库         |
| (stock_quantity)  |   S = S - Q        | (locked_quantity)   |   L = L - Q        | (已完成销售出库)    |
|                   |   L = L + Q        |                     |   物理总量 - Q      |                   |
+-------------------+ <----------------- +---------------------+                    +-------------------+
                           取消回退 / 超时
                            S = S + Q
                            L = L - Q
```

| 业务动作 | 前置条件 | 原子更新 SQL 公式 | 字段变化量 | 物理库存变化 |
| :--- | :--- | :--- | :--- | :--- |
| **1. 初始入库 / 补货** | `Q > 0` | `UPDATE product_inventories SET stock_quantity = stock_quantity + Q WHERE product_id = ?;` | $\Delta S = +Q, \Delta L = 0$ | $+Q$ |
| **2. 下单预占 (Pending)** | `stock_quantity >= Q` | `UPDATE product_inventories SET stock_quantity = stock_quantity - Q, locked_quantity = locked_quantity + Q WHERE product_id = ? AND stock_quantity >= Q;` | $\Delta S = -Q, \Delta L = +Q$ | $0$ (守恒) |
| **3. 支付核销 (Paid)** | `locked_quantity >= Q` | `UPDATE product_inventories SET locked_quantity = locked_quantity - Q WHERE product_id = ? AND locked_quantity >= Q;` | $\Delta S = 0, \Delta L = -Q$ | $-Q$ (实物出库) |
| **4. 取消/超时释放 (Cancel)** | `locked_quantity >= Q` | `UPDATE product_inventories SET stock_quantity = stock_quantity + Q, locked_quantity = locked_quantity - Q WHERE product_id = ? AND locked_quantity >= Q;` | $\Delta S = +Q, \Delta L = -Q$ | $0$ (守恒归还) |
| **5. 售后退款入库 (Refund)** | `Q > 0` | `UPDATE product_inventories SET stock_quantity = stock_quantity + Q WHERE product_id = ?;` | $\Delta S = +Q, \Delta L = 0$ | $+Q$ (重回可售) |

---

### 2.2 全局严格偏序加锁规范（彻底根治死锁）

#### 2.2.1 偏序理论防死锁证明
死锁产生的四个必要条件为：互斥、持有并等待、不可剥夺、环路等待。
只要破坏**环路等待（Circular Wait）**，死锁在数学上必定不可能发生。
**全系统铁律**：所有涉及多个商品库存行记录加锁（`SELECT ... FOR UPDATE` 或 `UPDATE product_inventories`）的数据库事务，加锁顺序必须严格按照 **`product_id ASC`（主键升序）** 进行！

#### 2.2.2 规范化代码准则
1. **输入项合并去重（Item Deduplication）**：
   下单或批处理请求中可能包含重复商品。必须先按 `product_id` 聚合汇总 `quantity`，避免同一事务中对相同主键重复加锁或导致负扣减；
2. **升序排序切片**：
   在进入事务加锁前，强制对商品列表排序：
   ```go
   sort.Slice(items, func(i, j int) bool {
       return items[i].ProductID < items[j].ProductID
   })
   ```
3. **取消与关单对称对齐**：
   `cancelPendingOrder` 在获取 `order_items` 后，严禁依赖数据库默认返回顺序，必须显式对切片执行 `sort.Slice` 或通过 DAO 保证 `ORDER BY product_id ASC`，保证与 `CreateOrder` 加锁拓扑 100% 对齐。

---

### 2.3 缓存对账自愈与监控度量规范

#### 2.3.1 权威事实源与对账语义
- **权威事实源 (SSOT)**：MySQL `product_inventories.stock_quantity`（可用库存）；
- **缓存护城河**：Redis `inventory:{stock}:available:{product_id}`；
- 对账 Worker 负责定期（默认 5m）扫描检测 Redis 与 MySQL 可用库存的一致性。

#### 2.3.2 自动自愈机制 (AutoFix)
- 配置层增加 `AutoFix bool` 字段：
  - `config.yml`: `inventoryReconcile.autoFix: true`
  - 环境变量: `INVENTORY_RECONCILE_AUTO_FIX: ${INVENTORY_RECONCILE_AUTO_FIX:-false}`
- **自愈算法步骤**：
  1. `ReconcileRedisInventoryStock` 获取所有商品的 MySQL `StockQuantity` 与 Redis 缓存值；
  2. 比对生成差异清单，区分为 `missing`（Redis Key 缺失）与 `mismatch`（数量不相等）；
  3. 若 `AutoFix == true` 且存在差异项：
     - 调用 `stockWriter.SetInventoryStock(ctx, item.ProductID, item.MySQLQuantity)`（或使用 Redis Pipeline/MSET）将 MySQL 权威库存强制回写 Redis；
     - 记录结构化恢复审计日志 `INFO: inventory reconcile autofix applied`；
     - 上报自愈监控指标 `app_inventory_reconcile_autofix_total{result="success"}`；
  4. 若自愈失败，降级记录 `ERROR: inventory reconcile autofix failed`，并上报 `{result="failed"}`；
  5. 若 `AutoFix == false`，记录 `WARN` 告警并上报 `{result="disabled"}`。

#### 2.3.3 Prometheus 业务指标契约
在 `internal/observability/business_metrics.go` 中新增两项指标：

```prometheus
# HELP app_inventory_reconcile_diff_total Total count of inventory reconcile discrepancies detected.
# TYPE app_inventory_reconcile_diff_total counter
app_inventory_reconcile_diff_total{status="missing"} 0
app_inventory_reconcile_diff_total{status="mismatch"} 0

# HELP app_inventory_reconcile_autofix_total Total count of inventory reconcile autofix operations.
# TYPE app_inventory_reconcile_autofix_total counter
app_inventory_reconcile_autofix_total{result="success"} 0
app_inventory_reconcile_autofix_total{result="failed"} 0
app_inventory_reconcile_autofix_total{result="disabled"} 0
```

---

## 3. 交接给 backend-dev 的实现契约清单 (Contract Checklist)

### 3.1 Model 层
- [ ] `internal/model/inventory.go`:
  ```go
  type Inventory struct {
      ID            int64     `gorm:"primaryKey;autoIncrement;type:bigint" json:"id"`
      ProductID     int64     `gorm:"type:bigint;not null;uniqueIndex:uk_inventory_product_id" json:"product_id"`
      StockQuantity int64     `gorm:"type:bigint;not null;default:0;check:stock_quantity >= 0" json:"stock_quantity"`
      LockedQuantity int64    `gorm:"type:bigint;not null;default:0;check:locked_quantity >= 0" json:"locked_quantity"`
      CreatedAt     time.Time `json:"created_at"`
      UpdatedAt     time.Time `json:"updated_at"`
  }
  ```

### 3.2 DAO 层
- [ ] `internal/dao/inventory_dao.go`:
  - `ReserveInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) (int64, error)`:
    `UPDATE product_inventories SET stock_quantity = stock_quantity - ?, locked_quantity = locked_quantity + ? WHERE product_id = ? AND stock_quantity >= ?`
  - `ReleaseLockedInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) (int64, error)`:
    `UPDATE product_inventories SET stock_quantity = stock_quantity + ?, locked_quantity = locked_quantity - ? WHERE product_id = ? AND locked_quantity >= ?`
  - `ConsumeLockedInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) (int64, error)`:
    `UPDATE product_inventories SET locked_quantity = locked_quantity - ? WHERE product_id = ? AND locked_quantity >= ?`
- [ ] `internal/dao/order_dao.go`:
  - 提供 `ListOrderItemsByOrderIDForInventory(ctx context.Context, db *gorm.DB, orderID int64) ([]*model.OrderItem, error)`（带 `ORDER BY product_id ASC`），或统一在内存中进行排序。

### 3.3 Service 层
- [ ] `internal/service/order_service.go`:
  - `CreateOrder`:
    - 针对商品列表进行 `product_id ASC` 排序与去重合并；
    - 执行原子预占 `ReserveInventory`，`stock_quantity` 减少，`locked_quantity` 增加；
  - `cancelPendingOrder`:
    - 明细按 `product_id ASC` 遍历；
    - 执行原子释放 `ReleaseLockedInventory`，`stock_quantity` 回补，`locked_quantity` 减少；
  - `PayOrder`:
    - 订单进入支付态时，按 `product_id ASC` 执行 `ConsumeLockedInventory`，核销锁定库存；
- [ ] `internal/service/inventory_service.go`:
  - 维护 `ReconcileRedisInventoryStock` 返回结构，支持批量修正 `AutoFixRedisInventoryStock`。

### 3.4 Worker & Observability 层
- [ ] `config/config.go` & `config.yml`: 增加 `AutoFix` 字段与环境变量绑定；
- [ ] `internal/inventoryreconcile/worker.go`: 支持根据 `AutoFix` 配置执行自动纠偏，调用指标记录器；
- [ ] `internal/observability/business_metrics.go`: 新增 `RecordInventoryReconcileDiff` 与 `RecordInventoryReconcileAutoFix` 方法。

---

## 4. 收益与权衡 (Consequences & Trade-offs)

### 4.1 收益 (Pros)
1. **状态机严密闭环**：物理库存、可用库存与在途预占库存三态分离，库存报表与盘点完全守恒；
2. **零死锁保证**：全系统严格强制 `product_id ASC` 偏序加锁，在图论层面彻底消除循环等待死锁；
3. **缓存自主愈合**：对账系统由单纯的“事后告警”进化为“告警 + 自主愈合”，极大降低运维排障成本与数据漂移风险。

### 4.2 代价与权衡 (Cons & Mitigation)
1. **写入放大 (Write Amplification)**：
   支付时新增了一次对 `locked_quantity` 的扣减更新。
   - *缓解措施*：支付成功通常非极端突发峰值，且更新走主键精确匹配（`WHERE product_id = ?`），单行更新耗时 < 0.5ms，开销完全可控；
2. **排序开销**：
   在应用层对订单明细进行了升序排序。
   - *缓解措施*：单个电商订单明细通常在 1~50 项之间，内存中切片排序耗时仅数十纳秒，相比死锁回滚重试的百毫秒级开销，收益呈数量级提升。
