# 2026-07-11 Trace Context 链路关联执行日志

## 目标

继续执行 `docs/project_evolution.md` 中“完善项目可观测性”的链路追踪相关边界，先实现不引入新依赖的 trace context 透传和日志关联。

## 本轮完成内容

1. 新增全局 Trace Context 中间件：
   - 兼容 W3C `traceparent` 请求头；
   - 合法输入复用 `trace_id`，并生成当前服务端 `span_id`；
   - 缺失或非法输入自动生成新的 `trace_id` 和 `span_id`；
   - 响应头写回新的 `traceparent`。

2. 接入 access log：
   - 每条请求日志增加 `trace_id` 和 `span_id`；
   - 保留原有 `request_id`，用于和操作审计、错误日志继续关联。

3. 补齐 timeout 边界：
   - 外层 `TimeoutHandler` 在提前返回超时响应时也写入 `traceparent`；
   - 超时响应仍保持原统一响应结构。

4. 同步文档：
   - 更新 README、需求边界、业务规则、项目演进、可观测性说明、测试计划和测试结果。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./internal/middleware ./router ./internal/app` | 通过 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |

## 当前边界

- 当前只实现 trace context 透传和日志关联；
- 暂未接入 OpenTelemetry SDK、Collector、Jaeger 或 Tempo；
- 暂未采集跨服务 span、数据库 span 或消息队列 span；
- 后续若需要完整链路追踪，应在已有 `trace_id` 基础上接入标准 tracing 后端。
