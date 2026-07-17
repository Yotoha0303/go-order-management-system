# 2026-07-11 Redis/MySQL 定时自动对账执行日志

## 目标

继续执行 `docs/project_evolution.md` 中 Redis/MySQL 库存对账相关边界，在已有手动差异报告和重建接口基础上补齐周期自动对账。

## 本轮完成内容

1. 新增自动对账配置：
   - `inventoryReconcile.enabled`；
   - `inventoryReconcile.interval`；
   - `inventoryReconcile.timeout`；
   - 支持 `INVENTORY_RECONCILE_ENABLED`、`INVENTORY_RECONCILE_INTERVAL`、`INVENTORY_RECONCILE_TIMEOUT` 环境变量覆盖。

2. 新增后台 worker：
   - `internal/inventoryreconcile.Worker` 周期调用 `ReconcileRedisInventoryStock`；
   - `RunOnce` 便于单元测试和后续手动触发；
   - 对账失败只记录日志，不影响 HTTP 服务和下单主流程。

3. 差异处理策略：
   - 无差异记录 `inventory Redis reconcile finished`；
   - 有差异记录 `inventory Redis reconcile found differences`；
   - 不自动重建 Redis，避免隐藏库存数据问题；
   - 修复差异仍由管理员查看差异报告后手动触发重建。

4. 接入应用生命周期：
   - `InitDeps` 根据配置创建 worker；
   - `app.Run` 使用同一个 context cancel 和 WaitGroup 管理订单超时 worker 与库存对账 worker；
   - 优雅退出时统一等待后台任务结束。

5. 同步文档：
   - 更新 README、需求边界、业务规则、缓存设计、可观测性说明、项目演进、测试计划和测试结果。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./config ./internal/inventoryreconcile ./internal/app ./internal/service` | 通过 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |

## 仍未完成的后续进阶边界

- 线上部署和长期稳定性运行证据；
- OpenTelemetry 或其他链路追踪；
- Prometheus 线上配置文件和告警截图证据；
- 真实压测、EXPLAIN 截图和慢 SQL 优化报告；
- Redis Cluster hash tag 和跨 slot Lua 适配验证；
- 完整支付、退款、发货、售后等复杂订单链路。
