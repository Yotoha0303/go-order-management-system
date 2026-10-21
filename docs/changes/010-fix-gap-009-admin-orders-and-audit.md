# 变更记录：010-fix-gap-009-admin-orders-and-audit

- **变更编号**：010
- **创建时间**：2026-10-21T16:05:00Z (UTC)
- **完成时间**：2026-10-21T16:15:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-009 (P2: 缺少管理员全局订单列表与审计视图，管理后台无法查询与干预全平台订单，Issue #10)
- **关联 ADR**：ADR-001
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在现有的订单模块中，系统仅提供了面向普通用户的订单查询接口（`GET /api/v1/orders`），且强制绑定了当前登录用户的 `user_id`。
导致的问题（GAP-009）：
1. 管理员后台缺少全局跨租户/跨用户的订单列表查询与审计视图，运维与客服人员无法查询全平台的订单状态；
2. 缺乏支持多维过滤（按 `user_id`、按订单状态 `status`）及分页能力的全局接口；
3. 未将管理员订单查询路由纳入统一的 RBAC 权限控制（`AdminMiddleware`）与操作审计（`OperationAudit`）链路中，存在管理盲区与安全合规隐患。

为彻底根治此缺陷，本变更构建完整的管理员全局订单查询与审计闭环：
- 新增 `ListAdminOrdersRequest` 请求 DTO，具备健壮的分页参数防御（缺省 page=1, pageSize=10, 上限 100）；
- 在 DAO 层实现 `ListAdminOrders`，基于 GORM 构建多维动态过滤（动态拼接 `user_id` 与 `status`），按 `id DESC` 稳定排序；
- 在 Service 层实现 `ListAdminOrders`，负责参数规范化与业务规则校验；
- 在 Handler 层实现 `ListAdminOrders`，返回结构化的 `response.OrderListResponse`；
- 在 Router 注册 `GET /api/v1/admin/orders`，挂载 `middleware.AdminMiddleware(roleChecker)` 与 `middleware.OperationAudit(auditRecorder)`；
- 编写完善的单元测试与权限安全测试，保证普通用户访问被 403 阻断，管理员访问成功并留存审计日志。

---

## 2. 变更实施清单

1. `internal/request/order_request.go`:
   - 新增 `ListAdminOrdersRequest` 结构体（包含 `UserID *int64`、`Status *int`、`Page int`、`PageSize int`）；
   - 提供 `NormalizePagination()` 防御方法，确保缺省 page=1、pageSize=10 且不超过上限 100。
2. `internal/dao/order_dao.go`:
   - 新增 `ListAdminOrders(ctx context.Context, db *gorm.DB, userID *int64, status *int, pageSize, offset int) ([]*model.Order, int64, error)` 函数，支持多维条件动态过滤与排序分页。
3. `internal/service/order_service.go`:
   - 新增 `ListAdminOrders(ctx context.Context, req request.ListAdminOrdersRequest) ([]*model.Order, int64, error)` 方法，包含 `UserID` 正整性防御与分页越界检查。
4. `internal/handler/order_handler.go`:
   - 在 `OrderService` 接口声明 `ListAdminOrders`；
   - 在 `OrderHandler` 实现 `ListAdminOrders(c *gin.Context)`，负责 Query 解析、分页规范化与返回 `response.OrderListResponse`。
5. `router/router.go`:
   - 更新 `registerOrderAPIRouter`，接入 `roleChecker middleware.RoleChecker` 与 `auditRecorder middleware.OperationAuditRecorder`；
   - 挂载 `admin := rg.Group("")`，使用 `middleware.AdminMiddleware` 与 `middleware.OperationAudit`，注册 `admin.GET("/admin/orders", orderHandler.ListAdminOrders)`。
6. 测试套件更新与新增：
   - `router/router_test.go`：
     - 在 `TestAdminRoutesRequireAdminRole` 中新增 `{method: http.MethodGet, path: "/api/v1/admin/orders"}`，严格断言非 admin 用户请求返回 403 Forbidden；
     - 新增 `TestAdminOrdersRoute_AllowedForAdminAndAudited`，断言 admin 角色返回 200 及订单数据，且操作审计中间件生成精确审计日志。
   - `internal/handler/order_handler_test.go`：
     - 新增 `TestOrderHandlerListAdminOrders`，覆盖参数解析、分页缺省回退与异常映射。
   - `internal/service/order_service_test.go`：
     - 新增 `TestListAdminOrders_ValidationAndPagination`，验证负数用户 ID 拦截与分页上限限制。

---

## 3. 实测验证日志 (纪律 D1: 真实 stdout 回填)

### 3.1 路由层 RBAC 权限与操作审计测试（带 `-race`）

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go test -v -race ./router/... ./internal/handler/...
=== RUN   TestBusinessRoutesRequireAuthentication
--- PASS: TestBusinessRoutesRequireAuthentication (0.01s)
=== RUN   TestAdminRoutesRequireAdminRole
--- PASS: TestAdminRoutesRequireAdminRole (0.00s)
=== RUN   TestAdminOrdersRoute_AllowedForAdminAndAudited
--- PASS: TestAdminOrdersRoute_AllowedForAdminAndAudited (0.00s)
=== RUN   TestMetricsEndpointIsPublicAndRecordsRequests
--- PASS: TestMetricsEndpointIsPublicAndRecordsRequests (0.00s)
PASS
ok  	go-order-management-system/router	3.309s
=== RUN   TestReadyzHandler
--- PASS: TestReadyzHandler (0.00s)
=== RUN   TestInventoryHandlerRebuildRedisInventoryStock
--- PASS: TestInventoryHandlerRebuildRedisInventoryStock (0.00s)
=== RUN   TestInventoryHandlerReconcileRedisInventoryStock
--- PASS: TestInventoryHandlerReconcileRedisInventoryStock (0.00s)
=== RUN   TestOrderHandlerCreateOrder
--- PASS: TestOrderHandlerCreateOrder (0.00s)
=== RUN   TestOrderHandlerListAdminOrders
=== RUN   TestOrderHandlerListAdminOrders/binds_query_params_and_returns_order_list
=== RUN   TestOrderHandlerListAdminOrders/normalizes_pagination_defaults_when_query_params_omitted
=== RUN   TestOrderHandlerListAdminOrders/maps_service_error_to_HTTP_response
--- PASS: TestOrderHandlerListAdminOrders (0.00s)
    --- PASS: TestOrderHandlerListAdminOrders/binds_query_params_and_returns_order_list (0.00s)
    --- PASS: TestOrderHandlerListAdminOrders/normalizes_pagination_defaults_when_query_params_omitted (0.00s)
    --- PASS: TestOrderHandlerListAdminOrders/maps_service_error_to_HTTP_response (0.00s)
=== RUN   TestParseProductStatus
--- PASS: TestParseProductStatus (0.00s)
=== RUN   TestParseOptionalPagination
--- PASS: TestParseOptionalPagination (0.00s)
PASS
ok  	go-order-management-system/internal/handler	3.295s
```

### 3.2 静态代码检查 (golangci-lint & go vet)

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> golangci-lint run ./...
0 issues.
```

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go vet ./...
# 退出码 0，全仓零告警
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

1. **越权请求负向证伪**：
   在 `TestAdminRoutesRequireAdminRole` 中，普通用户（`normal-user`，`denyRoleChecker`）携带有效 JWT Token 访问 `/api/v1/admin/orders`，响应状态码严格断言为 `403 Forbidden`。若路由组未接入 `AdminMiddleware`，用例必定失败。
2. **非法入参拦截证伪**：
   在 `TestListAdminOrders_ValidationAndPagination` 中注入负数 `UserID = -1`，服务层严格拦截并断言返回 `ErrInvalidUserID`。
3. **审计留痕断言证伪**：
   在 `TestAdminOrdersRoute_AllowedForAdminAndAudited` 中，断言管理员成功访问后，审计日志组件 `OperationAudit` 必须捕获包含 `UserID=7`, `Username="admin-user"`, `Path="/api/v1/admin/orders"`, `HTTPStatus=200` 的记录。若审计记录被吞或缺失，测试立即报错。

---

## 5. 下游影响与未竟项

- **GAP-009**：本变更已彻底闭环修复，全局订单查询与操作审计链路全量打通。
- **项目缺陷清零状态**：至此，项目中登记的最后一个未竟缺陷（GAP-009）已完全闭环交付。
