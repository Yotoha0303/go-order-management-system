# REST Client 自测结果

测试时间：2026-05-19
测试环境：本地 MySQL + Redis
启动命令：go run cmd/main.go

## 1. 商品模块

| 用例 | 结果 | 备注 |
|---|---|---|
| 创建商品 | 通过 | 新建 product_id=1，status=2 |
| 商品上架 | 通过 | `PATCH /products/1/on-sale` 成功 |
| 商品下架 | 通过 | `PATCH /products/1/off-sale` 成功 |

## 2. 库存模块

| 用例 | 结果 | 备注 |
|---|---|---|
| 初始化库存 | 通过 | `POST /inventory/init` 成功，stock_logs 记录 `biz_type=1` |
| 增加库存 | 通过 | `POST /inventory/add` 成功，stock_logs 记录 `biz_type=2` |
| 重复初始化 | 通过 | 返回业务错误码 `2001`（库存已初始化） |

## 3. 订单模块

| 用例 | 结果 | 备注 |
|---|---|---|
| 创建订单 | 通过 | 新建 order_id=4，扣减库存成功，stock_logs 有 `biz_type=3` |
| 支付订单 | 通过 | `pending -> paid` |
| 完成订单 | 通过 | `paid -> finished` |
| 取消订单 | 通过 | 针对 order_id=5 取消成功，库存回滚，stock_logs 有 `biz_type=4` |
| 重复取消 | 通过 | 再次取消 order_id=5 成功，无异常 |

## 4. Redis 缓存

| 用例 | 结果 | 备注 |
|---|---|---|
| 第一次查询商品详情 | 通过 | `GET /products/1` 成功（首次查询） |
| 第二次查询商品详情 | 通过 | `GET /products/1` 成功（重复查询） |
| 商品上下架后删除缓存 | 通过 | 下架后再次查询成功，缓存失效后可重建 |

# Go 自动化测试结果

测试时间：2026-05-23
测试环境：本地 MySQL + Redis
启动命令：go run cmd/main.go

执行命令：

```bash
go test ./...
```

| 测试范围          | 结果 | 备注                   |
| ------------- | -- | -------------------- |
| service 层测试   | 通过 | 覆盖商品、库存、订单状态机等核心业务   |
| bizcache 基础测试 | 通过 | Redis 集成测试默认跳过       |
| 项目整体编译        | 通过 | `go test ./...` 正常完成 |

Redis 集成测试执行命令：

```bash
RUN_REDIS_TEST=1 go test -v ./internal/bizcache
```

| 测试范围               | 结果 | 备注                           |
| ------------------ | -- | ---------------------------- |
| Redis key 测试       | 通过 | `product:detail:{id}`        |
| Redis nil 降级       | 通过 | `global.Redis = nil` 时不影响主流程 |
| Set / Get / Delete | 通过 | 商品详情缓存可写入、读取、删除              |
| TTL 测试             | 通过 | 缓存存在过期时间                     |

## 1. 关键异常链路测试结果

| 用例 | 结果 | 备注 |
|---|---|---|
| 库存不足创建订单 | 通过 | 返回库存不足业务错误，订单不创建 |
| 相同幂等 Key 重放 | 通过 | 返回原订单，不重复扣减库存 |
| 相同幂等 Key 请求冲突 | 通过 | 返回幂等冲突，仅保留原订单 |
| 并发相同幂等 Key | 通过 | 12 个并发请求只创建一笔订单 |
| 创建失败后重试 | 通过 | 失败事务回滚幂等记录，补足库存后可使用原 Key 重试 |
| 已支付订单重复支付 | 通过 | 返回订单已支付错误，状态保持 paid |
| 待支付订单直接完成 | 通过 | 返回订单未支付错误，状态保持 pending |
| 已支付订单取消 | 通过 | 返回订单已支付错误，未写 cancelled_at |
| 已完成订单取消 | 通过 | 返回订单已完成错误，未写 cancelled_at |
| 已取消订单重复取消 | 通过 | 接口成功返回，库存不重复回滚 |
| 重复取消库存流水检查 | 通过 | 回滚流水数量不新增 |

## 2. 测试结论

本轮测试覆盖了商品、库存、库存流水、订单状态机和 Redis 商品详情缓存。REST Client 测试用于验证接口链路和数据库结果，Go 自动化测试用于验证 service 层核心业务规则和 Redis 缓存函数行为。

当前已验证：

- 商品创建、上下架、详情查询流程正常
- 库存初始化、增加库存、库存流水记录正常
- 创建订单时幂等记录、订单、订单项、库存扣减、库存流水在事务内完成
- 创建订单支持同 Key 重放、不同请求冲突和并发单订单保证
- 支付、完成、取消订单状态流转符合规则
- 取消订单能够回滚库存，并记录回滚流水
- 已取消订单重复取消不会重复回滚库存
- Redis 商品详情缓存支持 Set / Get / Delete / TTL
- Redis 不可用时不影响 MySQL 主流程

## 3. 前后端联调补充检查（2026-07-03）

本轮针对 React 管理台接入后端接口进行了源码映射和构建检查，不替代连接真实 MySQL、Redis 的完整 E2E 测试。

| 检查项 | 结果 | 备注 |
|---|---|---|
| 后端普通测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `npm run build` |
| 前端 ESLint | 通过 | `npm run lint` |
| 登录表单定向测试 | 通过 | 4 个用例 |
| 注册表单定向测试 | 通过 | 4 个用例 |
| 后端业务接口源码映射 | 通过 | 认证、用户、商品、库存、流水和订单共 20 个业务接口均有前端调用点 |

健康检查中 `/ping` 用于前端仪表盘；`/live` 与 `/readyz` 保留给容器编排和监控系统，不要求业务页面调用。

## 4. 当前分支回归检查（2026-07-07）

本轮在 `exp/new-feature-test` 执行完整静态检查和前后端回归：

| 检查项 | 结果 | 备注 |
|---|---|---|
| Go 静态检查 | 通过 | `go vet ./...` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `npm run build` |
| 前端自动化测试 | 通过 | 18 个测试文件、105 个测试用例 |
| 文档与源码引用扫描 | 通过 | 已删除功能不存在残留文件、路由、配置或接口说明 |

本节只记录无需外部服务即可复现的回归结果。真实 MySQL、Redis、RabbitMQ 集成测试仍按 `docs/test_plan.md` 中的独立测试环境要求执行。

## 5. 项目完成度复核（2026-07-11）

本轮根据 `docs/project_evolution.md` 复核当前项目完成度，重点确认第九阶段能力是否已由代码、测试和文档支撑，并校正第十阶段为后续增强边界。

| 检查项 | 结果 | 备注 |
|---|---|---|
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

| 项目演进文档一致性 | 通过 | 第九阶段聚焦 RBAC、关键测试、RabbitMQ 超时取消和可部署工程化；第十阶段明确为后续增强 |

结论：

- 当前可复现交付范围已覆盖商品、库存、订单、认证、最小 RBAC、Redis 商品详情缓存、RabbitMQ 超时自动取消、React 管理台和自动化测试。
- 未在本轮声明云服务器长期运行、真实压测、慢 SQL 报告、操作审计、链路追踪、真实支付/退款/发货等能力；这些仍按项目边界归入后续增强。

## 6. 第十阶段进阶边界推进（2026-07-11）

本轮针对 `docs/project_evolution.md` 中“后续进阶边界”做第一轮可落地实现，聚焦两个能直接提升项目工程质量且不引入过度复杂度的事项：多商品锁顺序风险和后台操作审计。

| 检查项 | 结果 | 备注 |
|---|---|---|
| 多商品订单锁顺序 | 通过 | `CreateOrder` 在事务内按 `product_id` 升序处理商品与库存；新增服务层测试覆盖倒序请求 |
| 后台操作审计 | 通过 | 新增 `operation_logs` 表、审计中间件、管理员查询接口和 React 管理台页面 |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

本轮仍未完成的第十阶段事项：

- Redis Lua 库存预扣；
- 线上部署和长期稳定性运行证据；
- Prometheus 指标、链路追踪等完整可观测性；
- 真实压测、慢 SQL 分析和性能报告；
- 完整支付、退款、发货、售后等订单扩展链路。

## 7. 可观测性推进（2026-07-11）

本轮继续推进第十阶段“完善项目可观测性”，新增无第三方依赖的 Prometheus 文本格式 HTTP metrics。

| 检查项 | 结果 | 备注 |
|---|---|---|
| HTTP metrics 聚合 | 通过 | 按 method、route、status 聚合请求数、耗时总和和样本数 |
| `/metrics` 端点 | 通过 | 不要求 Bearer Token，输出 `text/plain; version=0.0.4` |
| 高基数控制 | 通过 | route label 使用 Gin 路由模板，未匹配路由记为 `unmatched` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

本轮仍未完成的第十阶段事项：

- Redis Lua 库存预扣；
- 线上部署和长期稳定性运行证据；
- 业务指标、告警规则和链路追踪；
- 真实压测、慢 SQL 分析和性能报告；
- 完整支付、退款、发货、售后等订单扩展链路。

## 8. Redis Lua 库存预扣推进（2026-07-11）

本轮继续推进第十阶段“Redis 进阶使用”和评估重点“库存扣减加一层 Redis Lua 做预扣”。

| 检查项 | 结果 | 备注 |
|---|---|---|
| Redis Lua 预扣 | 通过 | Lua 原子检查多个商品可售库存并批量 `DECRBY` |
| Redis reservation | 通过 | 预扣成功写 `inventory:{stock}:reservation:{order_id}`，用于失败补偿和取消回补 |
| 降级策略 | 通过 | Redis nil、Redis 异常或库存 key 缺失时降级走 MySQL |
| 库存不足 | 通过 | Redis 判断库存不足时提前返回库存不足，不扣减 Redis |
| 事务补偿 | 通过 | MySQL 事务失败后释放 reservation 并回补 Redis |
| 状态流转补偿 | 通过 | 支付成功清理 reservation；取消/超时取消成功按 reservation 回补 |
| Go 针对性测试 | 通过 | `go test ./internal/service ./internal/bizcache ./internal/app` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

仍需后续补充：

- 真实压测验证 Redis 预扣对热点库存的削峰效果。

## 9. 业务指标和告警建议推进（2026-07-11）

本轮继续推进第十阶段“业务指标、告警规则和链路追踪”中的业务指标和告警建议部分。

| 检查项 | 结果 | 备注 |
|---|---|---|
| 订单创建指标 | 通过 | `app_order_create_total{result=...}` |
| 订单状态流转指标 | 通过 | `app_order_state_transition_total{action=...,result=...}` |
| Redis 预扣指标 | 通过 | `app_redis_inventory_prededuct_total{result=...}` |
| Redis reservation 指标 | 通过 | `app_redis_inventory_reservation_total{action=...,result=...}` |
| Redis 库存同步指标 | 通过 | `app_redis_inventory_sync_total{result=...}` |
| 告警建议文档 | 通过 | `docs/observability.md` |
| Go 针对性测试 | 通过 | `go test ./internal/observability ./internal/bizcache ./internal/service ./router ./internal/app` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

仍需后续补充：

- Prometheus 线上配置文件和告警截图证据；
- OpenTelemetry 或其他链路追踪；
- 慢 SQL 自动采集和性能报告；

## 10. Redis 可售库存对账和重建推进（2026-07-11）

本轮继续推进第十阶段中 Redis/MySQL 库存对账相关能力，实现管理员可查看差异报告，并可手动重建 Redis 可售库存。

| 检查项 | 结果 | 备注 |
|---|---|---|
| DAO 查询 | 通过 | 按 `product_id asc` 查询 MySQL 当前库存 |
| 差异报告 | 通过 | 返回 `checked_count`、`diff_count` 和差异项 |
| Service 编排 | 通过 | 从 MySQL 读取库存后调用 Redis 读取器或重建器 |
| 管理员接口 | 通过 | `GET /api/v1/inventory/redis/reconcile` 和 `POST /api/v1/inventory/redis/rebuild` |
| 前端入口 | 通过 | 库存管理页新增“Redis 库存对账”和“重建 Redis 库存”按钮 |
| 权限边界 | 通过 | 路由测试覆盖该接口要求管理员权限 |
| Go 针对性测试 | 通过 | `go test ./internal/service ./internal/bizcache ./internal/handler ./router ./internal/app` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

## 11. 慢 SQL 日志入口推进（2026-07-11）

本轮继续推进第十阶段中“慢 SQL 分析和性能报告”的本地可落地前置能力，新增 GORM 慢查询日志入口。当前仅提供日志采集入口，不声明已经完成真实压测或慢 SQL 分析报告。

| 检查项 | 结果 | 备注 |
|---|---|---|
| MySQL 配置 | 通过 | 新增 `mysql.slowThreshold`、`mysql.logLevel` 和环境变量覆盖 |
| GORM 慢查询日志 | 通过 | 超过阈值时输出 `gorm slow query` |
| GORM 错误日志 | 通过 | 数据库错误输出 `gorm query error` |
| 常规未命中处理 | 通过 | `record not found` 不按数据库错误日志输出 |
| Go 针对性测试 | 通过 | `go test ./config ./pkg/database ./internal/app` |
| Go 全量测试 | 通过 | `go test ./...` |

## 12. Redis/MySQL 定时自动对账推进（2026-07-11）

本轮继续推进第十阶段中 Redis/MySQL 定时自动对账能力，新增后台 worker 周期执行库存差异检查。该 worker 只记录差异，不自动重建 Redis，避免隐藏库存数据问题。

| 检查项 | 结果 | 备注 |
|---|---|---|
| 自动对账配置 | 通过 | 新增 `inventoryReconcile.enabled`、`interval`、`timeout` 和环境变量覆盖 |
| 后台 worker | 通过 | 应用启动后周期调用 `ReconcileRedisInventoryStock` |
| 差异处理 | 通过 | `diff_count > 0` 时写 warn 日志，不自动改 Redis |
| 错误处理 | 通过 | Redis/MySQL 对账失败仅记录错误，不影响 HTTP 服务 |
| 优雅退出 | 通过 | 与订单超时 worker 共用 cancel 和 WaitGroup 关闭 |
| Go 针对性测试 | 通过 | `go test ./config ./internal/inventoryreconcile ./internal/app ./internal/service` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

## 13. Trace Context 链路关联推进（2026-07-11）

本轮继续推进第十阶段中链路追踪相关能力，新增轻量 `traceparent` 透传和 access log 关联字段。当前不声明已经接入 OpenTelemetry 或完整分布式追踪后端。

| 检查项 | 结果 | 备注 |
|---|---|---|
| Trace Context | 通过 | 合法 `traceparent` 复用 trace_id，并生成服务端 span_id |
| 非法输入处理 | 通过 | 非法或缺失 `traceparent` 自动重生成 |
| Access Log | 通过 | 输出 `trace_id` 和 `span_id` |
| Timeout 响应 | 通过 | 外层超时响应保留 `traceparent` |
| Go 针对性测试 | 通过 | `go test ./internal/middleware ./router ./internal/app` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

## 14. Redis Cluster hash tag 推进（2026-07-11）

本轮继续推进 Redis 进阶边界，将库存预扣相关 Redis key 调整为固定 hash tag，降低 Redis Cluster 下多商品 Lua 预扣跨 slot 风险。

| 检查项 | 结果 | 备注 |
|---|---|---|
| Key 命名 | 通过 | `inventory:{stock}:available:{product_id}` 和 `inventory:{stock}:reservation:{order_id}` |
| Lua 补偿前缀 | 通过 | reservation 回补使用新格式库存 key 前缀 |
| 升级影响 | 通过 | MySQL 无迁移；旧 Redis key 需通过管理员重建接口刷新 |
| Go 针对性测试 | 通过 | `go test ./internal/bizcache ./internal/service ./internal/app` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

## 15. HTTP 压测工具推进（2026-07-11）

本节记录压测工具完成时的前置能力和当时阻塞；Docker 恢复后的真实运行结果见第 16 节。

| 检查项 | 结果 | 备注 |
|---|---|---|
| 压测命令 | 通过 | `go run ./cmd/loadtest` 支持 URL、方法、Header、Body、请求数、并发数和超时 |
| Markdown 报告 | 通过 | 输出 RPS、成功数、错误数、状态码分布和 p50/p95/p99 |
| Makefile 入口 | 通过 | `make load-test` 默认写入 `docs/evidence/loadtest_report.md` |
| 慢 SQL 分析流程 | 通过 | `docs/performance.md` 记录慢 SQL 日志和 EXPLAIN 分析步骤 |
| 真实运行环境压测 | 当时阻塞，后续解除 | 工具验证时 Docker Desktop Linux Engine 未运行；后续实测见第 16 节 |
| Go 针对性测试 | 通过 | `go test ./internal/loadtest ./cmd/loadtest` |
| Go 全量测试 | 通过 | `go test ./...` |
| 前端生产构建 | 通过 | `cd fronted && npm run build` |
| 前端自动化测试 | 通过 | `cd fronted && npm test`，18 个测试文件、105 个测试用例 |

## 16. 本地 Docker 真实压测与 SQL 分析（2026-07-11）

Docker Desktop 恢复后，使用独立 Compose 项目 `go-order-loadtest` 和全新数据卷启动完整栈。原开发数据卷未删除。迁移容器成功执行到版本 14，应用、MySQL、Redis 和 RabbitMQ 均通过健康检查。

| 检查项 | 结果 | 备注 |
|---|---|---|
| `/ping` 压测 | 通过 | 5000 请求、50 并发，5000 次 HTTP 200，无请求错误，RPS 3645.19，p95 30ms |
| 商品列表压测 | 通过 | 1000 条商品，2000 请求、20 并发，2000 次 HTTP 200，无请求错误，RPS 1400.31，p95 23ms |
| Prometheus 指标核对 | 通过 | 路由模板、状态码和请求计数与预检及压测请求吻合 |
| 应用日志核对 | 通过 | 9071 行日志中 error、panic、`gorm slow query` 均为 0 |
| 商品总数查询 EXPLAIN | 通过 | 使用 `idx_products_status`，`type=ref`，`Extra=Using index` |
| 商品分页查询 EXPLAIN | 通过 | 使用 `idx_products_status`，`type=ref`，`Extra=Backward index scan` |
| 完整证据 | 通过 | `docs/evidence/loadtest_summary_2026-07-11.md` 及两份原始报告 |
| Go 全量测试 | 通过 | `go test ./...` |
| Go 静态检查 | 通过 | `go vet ./...` |
| 差异格式检查 | 通过 | `git diff --check`，仅有工作区 LF/CRLF 提示 |

边界：结果仅是 Windows + Docker Desktop 同机基线，不代表生产容量；订单写链路、热点库存竞争、长期稳定性和资源峰值尚未覆盖。

## 17. 第十阶段状态标注复核（2026-07-11）

本轮只整理文档状态，不修改 Go、前端、数据库迁移或公开 API。复核后，第十阶段整体状态标记为“部分完成”，并把遗留内容拆成可验收任务。

| 能力 | 状态 | 复核结论 |
|---|---|---|
| Redis 进阶库存保护 | 已完成 | Lua 预扣、reservation、固定 hash tag、对账和重建均有代码与测试证据 |
| 后台操作审计 | 已完成 | 数据表、中间件、管理员接口和前端页面已形成闭环 |
| 多商品锁顺序 | 已完成 | 已按 `product_id` 升序处理并有服务层测试 |
| 基础可观测性 | 部分完成 | metrics、Trace Context 和日志已完成；OpenTelemetry/追踪后端未完成 |
| 性能与稳定性证据 | 部分完成 | 本地读链路报告和 EXPLAIN 已完成；写链路、热点库存、资源峰值和长期运行未完成 |
| 线上部署与运行证据 | 未完成 | 尚无云主机、TLS、外部监控、故障恢复和持续运行证据 |
| 完整支付与履约链路 | 未完成 | 尚无支付网关/回调、退款、发货和售后完整实现 |
| 文档同步 | 持续事项 | 后续每次接口、迁移、配置和验收证据变化都需要同步更新 |
| 本地文档链接检查 | 通过 | 本轮新增或修改的相对链接目标均存在 |
| 差异格式检查 | 通过 | `git diff --check`，仅有工作区 LF/CRLF 提示 |

后续优先级：P0 先完成订单写链路和热点库存压测；P1 再补 OpenTelemetry 与线上运行证据；P2 在业务规则评审后扩展支付和履约链路。

本轮没有代码、配置、迁移和前端行为变更，因此未重复执行 Go 或前端测试；最近一次完整回归结果见第 16 节。
