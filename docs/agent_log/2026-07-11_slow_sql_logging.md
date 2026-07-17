# 2026-07-11 慢 SQL 日志入口执行日志

## 目标

继续执行 `docs/project_evolution.md` 中“真实压测、慢 SQL 分析和性能报告”的前置能力建设，先补齐本地可验证的慢 SQL 日志入口。

## 本轮完成内容

1. 新增 MySQL 日志配置：
   - `mysql.slowThreshold`：慢查询阈值；
   - `mysql.logLevel`：GORM 日志等级；
   - `DB_SLOW_THRESHOLD` 和 `DB_LOG_LEVEL` 环境变量覆盖。

2. 新增 GORM `slog` 适配器：
   - 慢查询输出 `gorm slow query`；
   - 数据库错误输出 `gorm query error`；
   - 默认忽略常规 `record not found` 错误日志；
   - 日志字段包含 `elapsed_ms`、`threshold_ms`、`rows` 和 `sql`。

3. 接入应用启动链路：
   - `InitDBWithLogger` 接收应用 `slog.Logger`；
   - `internal/app.InitDeps` 使用同一个 logger 初始化 MySQL；
   - 保留 `InitDB` 兼容旧调用。

4. 同步文档：
   - 更新 README、需求边界、项目演进、可观测性说明、测试计划和测试结果；
   - 明确该能力是慢 SQL 分析入口，不等同于已经完成真实压测报告。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./config ./pkg/database ./internal/app` | 通过 |
| `go test ./...` | 通过 |

## 仍未完成的后续进阶边界

- 线上部署和长期稳定性运行证据；
- OpenTelemetry 或其他链路追踪；
- Redis/MySQL 定时自动对账；
- 真实压测、EXPLAIN 截图和慢 SQL 优化报告；
- 完整支付、退款、发货、售后等复杂订单链路。
