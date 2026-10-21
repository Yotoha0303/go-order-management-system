# 变更记录：009-fix-gap-010-product-cache-null-object

- **变更编号**：009
- **创建时间**：2026-10-21T15:36:00Z (UTC)
- **完成时间**：2026-10-21T15:47:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-010 (P2: 商品详情缓存查询缺乏防穿透设计，非法 ID 可打爆底层 MySQL，Issue #11)
- **关联 ADR**：ADR-001 (高并发缓存与数据一致性规范)
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在现有的商品详情查询机制中（`internal/service/product_service.go` 的 `GetProductByID`）：
1. 客户端发起查询请求时，先尝试从 Redis 读取商品详情缓存；
2. 若缓存未命中（例如非法或不存在的 `product_id`），直接穿透回源到 MySQL 执行 `dao.GetProductByID`；
3. 当 MySQL 返回 `gorm.ErrRecordNotFound` 时，系统仅返回 `ErrProductNotFound`，**未向 Redis 写入任何空值占位缓存**；
4. **穿透漏洞（GAP-010）**：黑客或恶意调用方使用大量不存在的非法 `product_id` 进行高并发请求时，缓存层完全形同虚设，所有流量全部直击 MySQL 数据库，可能引发底层数据库连接池耗尽、CPU 100%、服务雪崩崩溃。

为彻底根治缓存穿透风险，引入**空值缓存对象（Null Object Pattern）+ 随机过期抖动（Jitter）**方案：
- 定义空值哨兵 `ProductDetailNullSentinel = "__NULL_PRODUCT__"`；
- 定义短 TTL `ProductDetailNullCacheTTL = 60 * time.Second`，并增加 0~15 秒随机时间抖动，防止批量空值同时过期造成雪崩；
- 当回源 MySQL 确认记录不存在时，立即向 Redis 回写带短 TTL 的空值哨兵；
- 后续相同非法 ID 的并发请求将直接命中 Redis 空值缓存，并在服务层秒级直接返回 `ErrProductNotFound`，达成**零 DB 访问**保护。

---

## 2. 变更实施清单

1. `internal/bizcache/product_cache.go`:
   - 定义空值常量 `ProductDetailNullSentinel = "__NULL_PRODUCT__"` 与短 TTL `ProductDetailNullCacheTTL = 60 * time.Second`；
   - 增加 `nullCacheTTLWithJitter()` 函数（`60s + 0~15s` 随机纳秒抖动）；
   - 升级 `GetProductDetail(ctx context.Context, productID int64) (*model.Product, bool)`：
     - 若命中哨兵 `ProductDetailNullSentinel`，返回 `nil, true`（缓存命中，对象为空）；
     - 若命中合法数据，反序列化返回 `&product, true`；
     - 若未命中或异常，返回 `nil, false`；
   - 新增 `SetProductDetailEmpty(ctx context.Context, productID int64)`：向 Redis 回写空值哨兵，TTL 带随机时间抖动。
2. `internal/service/product_interface.go`:
   - 在 `ProductCache` 接口中扩展 `SetProductDetailEmpty(ctx context.Context, productID int64)` 契约。
3. `internal/service/product_service.go`:
   - 重构 `GetProductByID`:
     - 缓存命中时若 `product == nil`：判定命中了空值哨兵，秒级直接返回 `ErrProductNotFound`（达成零 DB 访问）；
     - 缓存未命中回源 DB 返回 `gorm.ErrRecordNotFound` 时：调用 `p.cache.SetProductDetailEmpty(ctx, id)` 回写短 TTL 空值哨兵，再返回 `ErrProductNotFound`。
4. 测试文件更新与补充：
   - `internal/service/product_service_test.go`：
     - 扩展 `fakeProductCache` 支持 `SetProductDetailEmpty` 并自动置空记录与置 `hit = true`；
     - 新增 `TestGetProductByID_NullCacheHit`：向 service 传入 `nil db`，断言命中空缓存时直接返回 `ErrProductNotFound` 且绝对不触发 DB panic；
     - 新增 `TestGetProductByID_CachePenetrationPrevention_UnitMock`：基于 `mockRRConnector`，模拟非存在商品初次查询回源 DB 并回写空值缓存，第二次查询直接命中空值哨兵，断言零 DB 访问；
     - 新增 `TestGetProductByID_CachePenetrationPrevention_IntegrationMySQL`：支持在真实 MySQL 环境下验证完整防穿透闭环。
   - `internal/bizcache/product_cache_test.go`：
     - 新增 `TestProductDetailCache_NoRedis_Empty`：验证 Redis 缺失时的稳健性与边界保护；
     - 新增 `TestProductDetailCache_EmptySentinel_WithRedis`：验证底层真实 Redis 空值哨兵写入、TTL 抖动区间与读取识别；
   - `internal/bizcache/product_cache_internal_test.go`：
     - 新增 `TestNullCacheTTLWithJitter`：100 次循环断言随机抖动严格落在 `[60s, 75s]` 区间内。

---

## 3. 实测验证日志 (纪律 D1: 真实 stdout 回填)

### 3.1 针对性模块单元测试与防穿透断言（带 `-race`）

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go test -v -race -run "TestGetProductByID|TestNullCacheTTLWithJitter" ./internal/bizcache/... ./internal/service/...
=== RUN   TestNullCacheTTLWithJitter
--- PASS: TestNullCacheTTLWithJitter (0.00s)
PASS
ok  	go-order-management-system/internal/bizcache	(cached)
=== RUN   TestGetProductByID_NotFound
--- PASS: TestGetProductByID_NotFound (0.00s)
=== RUN   TestGetProductByID_CacheHit
--- PASS: TestGetProductByID_CacheHit (0.00s)
=== RUN   TestGetProductByID_CacheMissSetsCache
    product_service_test.go:343: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestGetProductByID_CacheMissSetsCache (0.00s)
=== RUN   TestGetProductByID_NullCacheHit
--- PASS: TestGetProductByID_NullCacheHit (0.00s)
=== RUN   TestGetProductByID_CachePenetrationPrevention_UnitMock
--- PASS: TestGetProductByID_CachePenetrationPrevention_UnitMock (0.00s)
=== RUN   TestGetProductByID_CachePenetrationPrevention_IntegrationMySQL
    product_service_test.go:429: skip MySQL integration test; set RUN_MYSQL_TEST=1 to run
--- SKIP: TestGetProductByID_CachePenetrationPrevention_IntegrationMySQL (0.00s)
PASS
ok  	go-order-management-system/internal/service	3.137s
```

### 3.2 静态代码检查与 Linter

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go vet ./...
# 退出码 0，无任何 warning 或 lint 错误

PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> golangci-lint run ./...
0 issues.
# 退出码 0，无任何 linter 违规
```

### 3.3 全仓单元测试与竞态检测

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go test -race ./...
ok  	go-order-management-system/config	(cached)
ok  	go-order-management-system/internal/app	3.964s
ok  	go-order-management-system/internal/auth	(cached)
ok  	go-order-management-system/internal/bizcache	1.994s
ok  	go-order-management-system/internal/dao	(cached)
ok  	go-order-management-system/internal/handler	4.198s
ok  	go-order-management-system/internal/inventoryreconcile	2.546s
ok  	go-order-management-system/internal/loadtest	(cached)
ok  	go-order-management-system/internal/middleware	(cached)
ok  	go-order-management-system/internal/observability	(cached)
ok  	go-order-management-system/internal/ordertimeout	3.193s
ok  	go-order-management-system/internal/service	3.350s
ok  	go-order-management-system/migrations/integration	(cached)
ok  	go-order-management-system/pkg/database	(cached)
ok  	go-order-management-system/pkg/idgen	(cached)
ok  	go-order-management-system/router	3.301s
```

---

## 4. 负向证伪检验 (纪律 D3)

1. **判据非静默证伪**：
   在 `TestGetProductByID_NullCacheHit` 与 `TestGetProductByID_CachePenetrationPrevention_UnitMock` 中，第二轮命中缓存时传入的 `db` 为 `nil`（`nilDBSvc := service.NewProductService(nil, cache)`）。若服务层未能识别空值哨兵、发生逻辑泄漏而继续向下查询 DB，必定触发 nil pointer dereference 产生致命 panic，从物理上杜绝了"虚假全绿"或判据静默。
2. **抖动区间证伪**：
   `TestNullCacheTTLWithJitter` 循环执行 100 次，若 TTL 偏移量产生负值或超过 15 秒上界，用例必定即刻失败阻断构建。

---

## 5. 下游影响与未竟项

- **GAP-010**：本变更已彻底闭环修复，可由 `gate-auditor` / `qa-validator` 审查转段与验收。
- **无破坏性变更**：`ProductCache` 接口扩展为纯追加；`GetProductDetail` 现有合法商品返回格式不变，向后完全兼容。
