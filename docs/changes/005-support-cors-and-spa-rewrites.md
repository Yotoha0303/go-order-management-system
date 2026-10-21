# 变更记录：005-support-cors-and-spa-rewrites

- **变更编号**：005 (端到端跨域与前端 SPA 路由重写专项)
- **创建时间**：2026-10-21T18:18:00Z (UTC)
- **完成时间**：2026-10-21T18:20:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联背景**：Vercel 前端与 Render 后端端到端联调时，浏览器因跨域策略拦截预检请求 (CORS Error: No 'Access-Control-Allow-Origin' header)；直接访问前端子路由时返回 Vercel 404
- **关联 ADR**：ADR-001
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

Render 后端服务成功部署并与 TiDB Cloud Serverless 建立 TLS 握手后，健康检查接口已全部 Live。
但在前端（Vercel 托管：`https://go-order-management-system.vercel.app`）与后端（Render 托管：`https://go-order-management-system.onrender.com`）进行端到端联调时发现两个阻塞问题：

1. **CORS 预检拦截**：
   前端在发起注册或登录请求时，浏览器发送 `OPTIONS /api/v1/auth/register` 预检请求，后端因未配置 CORS 中间件，未返回 `Access-Control-Allow-Origin` 等响应头，导致预检被浏览器拦截。
2. **SPA 刷新 404**：
   在浏览器直接刷新或手动输入 `https://go-order-management-system.vercel.app/sign-up` 时返回 404。这是因为 Vite 单页应用（SPA）依赖客户端路由，在 Vercel 静态托管平台缺少 `vercel.json` 将所有未命中的请求路径重写回 `/index.html`。

方案：
- 后端新增 `internal/middleware/cors_middleware.go`，全面支持跨域请求与 OPTIONS 预检请求极速 204 响应；
- 在 `router/router.go` 最外层挂载 `CORSMiddleware`；
- 编写 `internal/middleware/cors_middleware_test.go` 保证跨域中间件逻辑与竞态安全；
- 前端添加 `fronted/vercel.json` 规则，重写所有子路由至 `/`，解决 SPA 刷新 404。

---

## 2. 拟修改与新增文件清单

1. `internal/middleware/cors_middleware.go`:
   - 实现 `CORSMiddleware()`，支持动态回显 Origin 或通配、Method 与 Header 放行、OPTIONS 204 截断响应。
2. `internal/middleware/cors_middleware_test.go`:
   - 编写针对 OPTIONS 预检请求与普通 GET/POST 请求的 CORS 头断言单元测试。
3. `router/router.go`:
   - 在 `SetupRouters` 引擎初始化入口挂载 `CORSMiddleware`。
4. `fronted/vercel.json`:
   - 新增 Vercel SPA rewrites 配置文件。

---

## 3. 验证方案与待回填实测命令

1. **中间件与路由单元测试（带竞态检测）**：
   - 命令：`go test -v -race ./internal/middleware/... ./router/...`
2. **全仓静态代码检查**：
   - 命令：`go vet ./...`
3. **全仓单元测试回归**：
   - 命令：`go test -race ./...`

---

## 4. 实测验证记录

### 4.1 中间件与路由测试 (`go test -v -race ./internal/middleware/... ./router/...`)

```text
=== RUN   TestAdminMiddleware
=== RUN   TestAdminMiddleware/missing_identity
=== RUN   TestAdminMiddleware/role_check_failed
=== RUN   TestAdminMiddleware/permission_denied
=== RUN   TestAdminMiddleware/permission_granted
--- PASS: TestAdminMiddleware (0.00s)
    --- PASS: TestAdminMiddleware/missing_identity (0.00s)
    --- PASS: TestAdminMiddleware/role_check_failed (0.00s)
    --- PASS: TestAdminMiddleware/permission_denied (0.00s)
    --- PASS: TestAdminMiddleware/permission_granted (0.00s)
=== RUN   TestAdminMiddlewareRejectsNilRoleChecker
--- PASS: TestAdminMiddlewareRejectsNilRoleChecker (0.00s)
=== RUN   TestAuthMiddlewareRejectsMissingToken
--- PASS: TestAuthMiddlewareRejectsMissingToken (0.00s)
=== RUN   TestAuthMiddlewareAddsIdentity
--- PASS: TestAuthMiddlewareAddsIdentity (0.00s)
=== RUN   TestCORSMiddleware_PreflightOptions
--- PASS: TestCORSMiddleware_PreflightOptions (0.00s)
=== RUN   TestCORSMiddleware_ActualRequestWithOrigin
--- PASS: TestCORSMiddleware_ActualRequestWithOrigin (0.00s)
=== RUN   TestCORSMiddleware_RequestWithoutOrigin
--- PASS: TestCORSMiddleware_RequestWithoutOrigin (0.00s)
=== RUN   TestHTTPMetricsMiddlewareRecordsRouteStatus
--- PASS: TestHTTPMetricsMiddlewareRecordsRouteStatus (0.00s)
=== RUN   TestOperationAuditRecordsAdminOperation
--- PASS: TestOperationAuditRecordsAdminOperation (0.00s)
=== RUN   TestOperationAuditSkipsMissingUser
--- PASS: TestOperationAuditSkipsMissingUser (0.00s)
=== RUN   TestTraceContextUsesIncomingTraceIDAndCreatesServerSpan
--- PASS: TestTraceContextUsesIncomingTraceIDAndCreatesServerSpan (0.00s)
=== RUN   TestTraceContextRegeneratesInvalidTraceparent
--- PASS: TestTraceContextRegeneratesInvalidTraceparent (0.00s)
=== RUN   TestAccessLogIncludesTraceContext
--- PASS: TestAccessLogIncludesTraceContext (0.01s)
=== RUN   TestTimeoutHandler_PassesThroughCompletedResponse
--- PASS: TestTimeoutHandler_PassesThroughCompletedResponse (0.00s)
=== RUN   TestTimeoutHandler_ReturnsTimeoutResponseAndCancelsContext
--- PASS: TestTimeoutHandler_ReturnsTimeoutResponseAndCancelsContext (0.02s)
=== RUN   TestTimeoutHandler_Disabled
--- PASS: TestTimeoutHandler_Disabled (0.00s)
PASS
ok  	go-order-management-system/internal/middleware	2.794s
=== RUN   TestBusinessRoutesRequireAuthentication
--- PASS: TestBusinessRoutesRequireAuthentication (0.01s)
=== RUN   TestAdminRoutesRequireAdminRole
--- PASS: TestAdminRoutesRequireAdminRole (0.00s)
=== RUN   TestAdminOrdersRoute_AllowedForAdminAndAudited
--- PASS: TestAdminOrdersRoute_AllowedForAdminAndAudited (0.00s)
=== RUN   TestMetricsEndpointIsPublicAndRecordsRequests
--- PASS: TestMetricsEndpointIsPublicAndRecordsRequests (0.00s)
PASS
ok  	go-order-management-system/router	3.218s
```

### 4.2 全仓静态检查与单元测试 (`go vet ./...` & `go test -race ./...`)

```text
ok  	go-order-management-system/config	(cached)
ok  	go-order-management-system/internal/app	3.282s
ok  	go-order-management-system/internal/auth	(cached)
ok  	go-order-management-system/internal/bizcache	(cached)
ok  	go-order-management-system/internal/dao	(cached)
ok  	go-order-management-system/internal/handler	3.739s
ok  	go-order-management-system/internal/inventoryreconcile	(cached)
ok  	go-order-management-system/internal/loadtest	(cached)
ok  	go-order-management-system/internal/middleware	2.217s
ok  	go-order-management-system/internal/observability	(cached)
ok  	go-order-management-system/internal/ordertimeout	(cached)
ok  	go-order-management-system/internal/service	(cached)
ok  	go-order-management-system/migrations/integration	(cached)
ok  	go-order-management-system/pkg/database	(cached)
ok  	go-order-management-system/pkg/idgen	(cached)
ok  	go-order-management-system/router	3.136s
```
全项测试通过，静态检查 0 告警，无竞态隐患。

