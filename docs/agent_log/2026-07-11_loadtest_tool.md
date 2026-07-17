# 2026-07-11 HTTP 压测工具执行日志

## 目标

继续执行 `docs/project_evolution.md` 中“真实压测、慢 SQL 分析和性能报告”的进阶边界，先补齐项目内可复现的 HTTP 压测工具和报告生成入口。

## 本轮完成内容

1. 新增 `internal/loadtest`：
   - 支持并发 HTTP 请求；
   - 统计成功数、非成功状态、请求错误、状态码分布；
   - 统计 RPS、平均延迟、p50、p95、p99、最小和最大延迟；
   - 可渲染 Markdown 报告。

2. 新增 `cmd/loadtest`：
   - 支持 `-url`、`-method`、`-header`、`-body`、`-requests`、`-concurrency`、`-timeout`、`-output`；
   - 可对运行中的服务生成压测报告。

3. 新增 Makefile 入口：
   - `make load-test`；
   - 默认目标 `http://127.0.0.1:8082/ping`；
   - 默认报告路径 `docs/evidence/loadtest_report.md`。

4. 新增文档：
   - `docs/performance.md` 记录压测执行流程、报告保存方式和慢 SQL + EXPLAIN 分析流程；
   - README、需求边界、项目演进、测试计划和测试结果同步更新。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./internal/loadtest ./cmd/loadtest` | 通过 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |
| `docker compose --env-file .env.example ps --all` | 未通过，Docker Desktop Linux Engine 未运行，无法连接 `npipe:////./pipe/dockerDesktopLinuxEngine` |

## 后续实测执行

Docker Desktop 恢复后继续完成运行证据：

1. 首次启动时，已有 MySQL 数据卷的 root 密码与当前示例配置不一致，迁移容器报 `Error 1045`。
2. 执行 `docker compose down --remove-orphans`，只移除容器和网络，保留原数据卷。
3. 使用 `COMPOSE_PROJECT_NAME=go-order-loadtest` 启动独立环境，创建新的 MySQL、Redis、RabbitMQ 数据卷；完整栈健康，Goose 版本为 14。
4. 注册普通压测用户并验证 JWT 商品列表查询。
5. 执行 `/ping` 5000 请求/50 并发，生成 `docs/evidence/loadtest_ping_2026-07-11.md`。
6. 向隔离数据库写入 1000 条商品夹具，执行商品列表 2000 请求/20 并发，生成 `docs/evidence/loadtest_products_2026-07-11.md`。
7. 核对 `/metrics`、应用日志和两条商品列表 SQL 的 EXPLAIN，生成 `docs/evidence/loadtest_summary_2026-07-11.md`。

实测结果：两组请求均 100% HTTP 200；`/ping` RPS 3645.19、p95 30ms；商品列表 RPS 1400.31、p95 23ms。应用日志无 error、panic 或超过 200ms 的慢 SQL。商品计数和分页查询均使用 `idx_products_status`。

最终回归：

| 命令 | 结果 |
|---|---|
| `gofmt -w cmd/loadtest/main.go internal/loadtest/*.go` | 通过 |
| `go test ./...` | 通过 |
| `go vet ./...` | 通过 |
| `git diff --check` | 通过，仅有工作区 LF/CRLF 提示 |
| `git status --short --untracked-files=all docs/evidence` | 仅列出三份 Markdown 报告，历史图片继续被忽略 |

## 当前边界

- 已提交本地 Docker 读链路真实报告和 EXPLAIN 文本证据；
- 原开发数据卷未删除，压测数据只存在于 `go-order-loadtest` 隔离卷；
- 本机数字只用于同机回归，不声明生产 QPS 或容量上限；
- 订单写链路、热点库存竞争、长期稳定性和资源峰值仍未覆盖。
