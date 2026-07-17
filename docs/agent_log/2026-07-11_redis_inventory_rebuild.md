# 2026-07-11 Redis 可售库存对账和重建执行日志

## 目标

继续执行 `docs/project_evolution.md` 中 Redis/MySQL 库存对账和重建相关边界，补齐管理员差异报告和手动重建能力。

## 本轮完成内容

1. 新增 DAO 查询：
   - `ListAllInventories` 按 `product_id asc` 读取 MySQL 当前库存。

2. 新增 Redis 批量读取和重建：
   - `InventoryStockGuard.GetInventoryStocks` 使用 `MGET` 批量读取 Redis 可售库存；
   - `InventoryStockGuard.RebuildInventoryStocks` 使用 Redis pipeline 批量写入 `inventory:available:{product_id}`。
   - Redis 不可用时返回明确错误，不影响普通下单主流程。

3. 新增 Service 编排：
   - `InventoryService.ReconcileRedisInventoryStock` 对比 MySQL 当前库存和 Redis 可售库存，只返回差异项。
   - `InventoryService.RebuildRedisInventoryStock` 从 MySQL 读取库存并调用 Redis 重建器。

4. 新增管理员接口：
   - `GET /api/v1/inventory/redis/reconcile`
   - `POST /api/v1/inventory/redis/rebuild`
   - 对账返回 `checked_count`、`diff_count` 和差异项；重建返回 `rebuild_count`
   - 复用管理员 RBAC 和操作审计中间件。

5. 新增前端入口：
   - 库存管理页新增“Redis 库存对账”和“重建 Redis 库存”按钮。

6. 同步文档：
   - 更新接口清单、权限设计、业务规则、Redis 设计、需求边界、可观测性说明、测试计划、测试结果和 REST Client 请求。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./internal/service ./internal/bizcache ./internal/handler ./router ./internal/app` | 通过 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |

## 仍未完成的后续进阶边界

- Redis/MySQL 定时自动对账；
- 线上部署和长期稳定性运行证据；
- 链路追踪；
- 真实压测、慢 SQL 分析和性能报告；
- 完整支付、退款、发货、售后等复杂订单链路。
