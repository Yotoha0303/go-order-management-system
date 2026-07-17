# 2026-07-11 可观测性执行日志

## 目标

继续推进 `docs/project_evolution.md` 中“完善项目可观测性”的后续进阶边界。

## 本轮完成内容

1. 新增 `internal/observability.HTTPMetrics`：
   - 按 method、route、status 聚合 HTTP 请求数；
   - 记录 HTTP 请求耗时汇总和样本数；
   - 输出 Prometheus 文本格式。

2. 新增 metrics 中间件：
   - 注册顺序为 RequestID -> HTTPMetrics -> AccessLog -> Recovery；
   - 使用 Gin 路由模板作为 route label，避免真实资源 ID 造成高基数；
   - 未匹配路由统一记录为 `unmatched`。

3. 新增 `/metrics` 端点：
   - 不要求 Bearer Token；
   - 输出 `text/plain; version=0.0.4`；
   - 可被本地 curl 或 Prometheus 类采集器读取。

4. 同步文档：
   - 更新接口清单、业务规则、需求边界、测试计划、README 和项目演进文档；
   - 将最小 HTTP metrics 标记为已完成；
   - 保留业务指标、告警和链路追踪为后续增强。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |

## 仍未完成的后续进阶边界

- Redis Lua 库存预扣；
- 线上部署和长期稳定性运行证据；
- 业务指标、告警规则和链路追踪；
- 真实压测、慢 SQL 分析和性能报告；
- 完整支付、退款、发货、售后等更复杂订单链路。
