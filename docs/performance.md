# 性能压测与慢 SQL 分析说明

本文记录当前项目可复现的性能证据入口。项目已经提供压测工具、慢 SQL 日志，并提交一次本地 Docker 读链路实测；报告数字只作为同机基线，不外推为生产容量。

## 1. 压测工具

项目内置一个无第三方依赖的 HTTP 压测命令：

```bash
go run ./cmd/loadtest -url http://127.0.0.1:8082/ping -requests 200 -concurrency 20 -timeout 5s
```

也可以通过 Makefile 运行，并把 Markdown 报告写入 `docs/evidence/loadtest_report.md`：

```bash
make load-test
```

可覆盖参数：

```bash
make load-test LOADTEST_URL=http://127.0.0.1:8082/ping LOADTEST_REQUESTS=1000 LOADTEST_CONCURRENCY=50
```

需要认证的接口可以直接使用命令行参数：

```bash
go run ./cmd/loadtest \
  -url http://127.0.0.1:8082/api/v1/orders \
  -method POST \
  -header "Authorization: Bearer <token>" \
  -header "Content-Type: application/json" \
  -body "{\"idempotency_key\":\"bench-001\",\"items\":[{\"product_id\":1,\"quantity\":1}]}" \
  -requests 100 \
  -concurrency 10 \
  -output docs/evidence/loadtest_orders.md
```

输出指标包括：

- 请求总数、成功数、非 2xx/3xx 状态数、请求错误数；
- RPS；
- 平均延迟、p50、p95、p99、最小和最大延迟；
- HTTP 状态码分布。

## 2. 推荐执行流程

1. 启动完整依赖和应用：

```bash
make docker-up MYSQL_PASSWORD=replace-with-a-strong-database-password JWT_SECRET=replace-with-a-32-plus-character-secret
```

2. 准备测试数据：

- 注册测试用户；
- 按 `docs/permission_design.md` 将测试用户设置为 `admin`；
- 创建商品、初始化库存并上架；
- 对 Redis 可售库存执行一次重建，确保新格式 key 已生成。

3. 执行压测：

```bash
make load-test LOADTEST_URL=http://127.0.0.1:8082/ping LOADTEST_REQUESTS=1000 LOADTEST_CONCURRENCY=50
```

4. 保存结果：

- 把生成的 Markdown 报告保存在 `docs/evidence/`；
- 记录测试机器、数据库、Redis、RabbitMQ 是否与应用同机；
- 记录请求接口、并发数、总请求数和错误率。

## 3. 慢 SQL 分析流程

应用已接入 GORM 慢 SQL 日志，配置入口：

```yaml
mysql:
  slowThreshold: 200ms
  logLevel: warn
```

环境变量覆盖：

```env
DB_SLOW_THRESHOLD=100ms
DB_LOG_LEVEL=warn
```

分析步骤：

1. 压测期间收集 `gorm slow query` 日志；
2. 复制日志中的 SQL；
3. 在同一 MySQL 环境执行 `EXPLAIN`；
4. 重点检查 `type`、`key`、`rows` 和 `Extra`；
5. 只在查询模式稳定且有证据时增加索引。

## 4. 已提交的本地实测

2026-07-11 已在全新隔离 Docker Compose 数据卷上完成两组测试：

- `GET /ping`：5000 请求、50 并发；
- `GET /api/v1/products?page=1&page_size=20`：1000 条商品数据，2000 请求、20 并发；
- 两组请求均为 100% HTTP 200，应用日志无 error、panic 或超过 200ms 的慢 SQL；
- 商品列表计数与分页查询均使用 `idx_products_status`。

完整环境、结果、EXPLAIN 和限制见 [本地 Docker 压测与 SQL 分析报告](evidence/loadtest_summary_2026-07-11.md)。

## 5. 当前边界

- 当前已提供压测工具、读链路本地报告和 EXPLAIN 结果；
- 尚未覆盖订单写链路、热点库存竞争、长时间稳定性和生产网络；
- 不在简历或 README 中把本机 RPS 外推为生产 QPS，也不声明未经对照实验验证的性能提升百分比或生产级高可用能力。
