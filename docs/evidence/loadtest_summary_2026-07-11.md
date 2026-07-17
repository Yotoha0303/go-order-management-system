# 本地 Docker 压测与 SQL 分析报告

## 测试环境

- 时间：2026-07-11 19:55-19:57（Asia/Shanghai）
- 主机：Windows 10.0.19045，AMD64，16 个逻辑处理器
- Go：1.25.7 windows/amd64
- Docker Engine：29.3.1，Docker Desktop Linux Engine
- 服务：应用、MySQL 8.4、Redis 7.2、RabbitMQ 4.1 均运行在同一台主机的 Docker Compose 中
- 容器资源：未配置独立 CPU/内存限制，共享 Docker Desktop 3.825 GiB 内存上限
- 数据库：全新隔离数据卷，Goose migration 版本 14
- 测试数据：1000 条商品，其中上下架状态各 500 条
- 客户端：仓库内置 `go run ./cmd/loadtest`，从宿主机请求 `127.0.0.1:8082`

该环境用于本地可复现基线，不代表云服务器、生产网络或多机部署性能。

## 测试结果

| 场景 | 请求数 | 并发 | 成功率 | RPS | 平均延迟 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|
| `GET /ping` | 5000 | 50 | 100% | 3645.19 | 14ms | 30ms | 38ms |
| `GET /api/v1/products?page=1&page_size=20` | 2000 | 20 | 100% | 1400.31 | 14ms | 23ms | 28ms |

原始报告：

- [Ping 基线](loadtest_ping_2026-07-11.md)
- [商品列表](loadtest_products_2026-07-11.md)

商品列表经过 JWT 校验、Gin 中间件、Service、GORM，并对 MySQL 执行总数统计和分页查询。两组请求均无非 2xx 状态和客户端错误。

## 指标与日志核对

- `/metrics` 中 `/ping` 和商品列表的 `status="200"` 计数与压测请求及预检请求吻合。
- 最终应用日志抽样统计共 9071 行，`error`、`panic` 和 `gorm slow query` 均为 0。
- 慢 SQL 阈值为 200ms；“没有慢日志”只说明本次负载和数据规模下未超过阈值，不代表所有数据规模下均无慢查询。
- 压测结束后的资源快照：应用约 10.39 MiB、MySQL 451.9 MiB、Redis 3.41 MiB、RabbitMQ 88.45 MiB。该快照不是峰值监控数据，不用于容量结论。

## EXPLAIN 结果

商品列表默认查询 `status = 2`，对应两条 SQL：

```sql
SELECT COUNT(*) FROM products WHERE status = 2;
SELECT * FROM products WHERE status = 2 ORDER BY id DESC LIMIT 20;
```

| SQL | type | key | 估算 rows | Extra |
|---|---|---|---:|---|
| `COUNT(*) WHERE status = 2` | ref | `idx_products_status` | 500 | Using index |
| `WHERE status = 2 ORDER BY id DESC LIMIT 20` | ref | `idx_products_status` | 500 | Backward index scan |

当前查询使用 `idx_products_status`，没有全表扫描，也没有出现 filesort。现阶段无需为该查询新增索引；数据量和筛选维度扩大后应重新执行 `EXPLAIN ANALYZE`。

## 结论与边界

- 已形成一次可复现的本地 Docker 真实运行压测和 SQL 执行计划证据。
- 当前结果可用于项目复盘和同机版本回归，不应外推为生产 QPS、容量上限或高可用证明。
- 本次只覆盖健康接口和商品列表读链路；订单创建、热点库存竞争、Redis 预扣削峰及长时间稳定性仍需专门场景与数据隔离方案。
- 未接入持续采样的资源监控，因此不声明 CPU、内存峰值或性能提升百分比。
