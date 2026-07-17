# 2026-07-11 业务指标执行日志

## 目标

继续执行 `docs/project_evolution.md` 中“完善项目可观测性”的业务指标和告警建议部分。

## 本轮完成内容

1. 新增 `BusinessMetrics`：
   - `app_order_create_total`
   - `app_order_state_transition_total`
   - `app_redis_inventory_prededuct_total`
   - `app_redis_inventory_reservation_total`
   - `app_redis_inventory_sync_total`

2. 扩展 `/metrics`：
   - 将 HTTP metrics 和 business metrics 合并输出；
   - 保持 Prometheus 文本格式；
   - 继续避免 user_id、order_id、product_id 等高基数 label。

3. 接入订单服务：
   - 记录创建订单成功、幂等重放和关键失败原因；
   - 记录支付、完成、取消和超时取消结果。

4. 接入 Redis 库存预扣：
   - 记录预扣 applied、insufficient、skipped_missing_key、error、disabled；
   - 记录 reservation release / confirm；
   - 记录库存同步 Redis 的结果。

5. 新增 `docs/observability.md`：
   - 记录指标名、标签和含义；
   - 给出 5xx、下单错误率、Redis 预扣失败、Redis key 缺失和 reservation 回补失败的告警建议。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./internal/observability ./internal/bizcache ./internal/service ./router ./internal/app` | 通过 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |

## 仍未完成的后续进阶边界

- Prometheus 线上配置文件和告警截图证据；
- OpenTelemetry 或其他链路追踪；
- Redis/MySQL 库存对账任务；
- 真实压测、慢 SQL 分析和性能报告；
- 完整支付、退款、发货、售后等复杂订单链路。
