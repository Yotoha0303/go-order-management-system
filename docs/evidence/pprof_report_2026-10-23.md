# 读链路压测与 pprof 剖析报告

> **采集时点**：`2026-10-23T09:56Z`（`17:56` Asia/Shanghai）
> **制品**：`docs/evidence/pprof-2026-10-23/{cpu,heap,allocs}.prof` · `docs/evidence/loadtest_read_2026-10-23.md`
> **本报告是证据，不是基线** —— 数字只对本机本次有效，引用须带时点。

## 1. 测试环境（实测）

| 项 | 值 |
| --- | --- |
| OS | Windows 10 Pro 10.0.19045 |
| CPU | 16 逻辑核 |
| 内存 | 15.4 GiB |
| Go | `go1.25.7 windows/amd64` |
| MySQL | 8.4，**临时实例，端口 3307**，仅绑 `127.0.0.1` |
| Redis | 7.2，端口 6380（本机既有服务） |
| RabbitMQ | 4.3.4（本机 scoop 安装） |
| 数据 | 125 商品 / 125 库存 / 250 流水（`docs/sql/data/001_seed_non_user_business_data_500.sql`） |
| 客户端 | 仓库内置 `cmd/loadtest`，与**服务同机** |

> ⚠️ **同机压测**：客户端与服务共享 16 个逻辑核，客户端自身也要消耗 CPU。**RPS 因此偏保守，不是上限**。
> 与 2026-07-11 那份报告的口径差异见 §5。

## 2. 压测结果

| 指标 | 值 |
| --- | ---: |
| 请求数 / 并发 | 100,000 / 50 |
| 成功率 | **100%**（0 失败 / 0 错误状态） |
| 墙钟 | 32.5 s |
| **RPS** | **3073** |
| 平均 / P50 | 16 ms / 16 ms |
| **P95 / P99** | **19 ms / 23 ms** |
| Min / Max | 1 ms / 43 ms |

对照 `/metrics`（同轮实测）：`app_http_requests_total{route="/api/v1/products",status="200"}` = **100201**。
两次读数的差值来自压测前的一次 smoke 与手工探测。

## 3. pprof 结论（这才是 `/metrics` 看不见的那一层）

CPU profile：`Duration 30.12s, Total samples 122.15s (405.57%)` —— 即压测期间**约 4 个核在忙**。

### 3.1 应用自身代码：`AccessLog` 中间件占 22.2% CPU

`pprof -peek` 实测的调用方归因（**唯一调用方**）：

```
                                            27.12s   100% |   go-order-management-system/router.SetupRouters.AccessLog.func5
     0.02s 0.016% 0.016%     27.12s 22.20%                | log/slog.(*Logger).LogAttrs
```

**⇒ 每 5 个 CPU 周期里有 1 个花在写访问日志上。**

原因可读代码定位（`internal/middleware/access_log_middleware.go`）：`AccessLog` 对**每一个请求**（含 2xx 成功）构造 **10 个 `slog.Attr`** 并以 `LevelInfo` 输出到 stdout。

**这不是缺陷，是一个没有被写下来的取舍**：
- 对**排障**，全量访问日志是资产（`request_id` / `trace_id` / `span_id` 三件套可关联链路）；
- 对**吞吐**，它是当前读链路最大的单项成本。

**两条可选路径（本报告不预判，需你裁定）**：
1. **保持全量，接受成本** —— 理由：读链路 3073 RPS 对目标规模够用；
2. **降噪** —— 成功请求改为采样 / 降到 `Debug` 级 / 只在 `status >= 400` 时输出。**代价**：丢失成功请求的链路证据，而这正是 07 环排障要的东西。

### 3.2 平台开销：`runtime.cgocall` 39.3%（Windows 特有，非应用问题）

| 函数 | flat% |
| --- | ---: |
| `runtime.cgocall` | **39.28%** |
| `runtime.stdcall2` / `stdcall1` / `stdcall6` / `stdcall0` | 12.6% 合计 |
| `internal/poll.execIO` | 40.29%（cum） |

**归因（实测）**：`FD.Write` 的调用方里 `os.(*File).write` 占 55.5%、`net.(*netFD).Write` 占 44.5%。

- 前者 = **写访问日志到 stdout**（与 §3.1 同源）；
- 后者 = 写 HTTP 响应体。

**⇒ 这 39% 在 Linux 上不会是这个形态**。Go 在 Windows 上把 socket / 文件写走 `cgocall` 进 Win32，Linux 上是直接 syscall。
**结论**：**不要把本报告的 CPU 分布当作跨平台结论**；它描述的是 Windows 本机。

### 3.3 内存：无泄漏迹象

| 指标 | 值 |
| --- | ---: |
| heap `inuse_space` 总量 | **5.66 MB** |
| 最大单项 | `runtime.allocm` 3.08 MB（54%）—— goroutine 栈，非业务数据 |
| goroutine 总数（压测后） | **23** |

**判据**：压测 10 万请求后 heap 仍在 6 MB 量级、goroutine 23 个（其中 5 个是 MySQL 连接的 watcher）。
**⇒ 读链路无内存累积、无 goroutine 泄漏。**

> ⚠️ **边界（诚实）**：heap profile 是**单次快照**，不是增长曲线。要证明「无泄漏」严格说需要**多时点采样对比**（如压测中每 30s 取一次）。
> 本次**未做**多时点采样，故结论强度是「**未观察到累积**」，不是「**已证明无泄漏**」。

## 4. 未覆盖的路径（必须说清，否则本报告会被读成"整个服务已剖析"）

| 路径 | 状态 |
| --- | --- |
| **读链路** `GET /api/v1/products` | ✅ 已剖析（本报告全部结论仅对此成立） |
| **写链路** `POST /api/v1/orders`（下单事务 + Redis Lua 预扣 + Outbox） | ❌ **未压** |
| **Redis 命中路径** | ❌ **未压**——本次读链路全程 200，未构造缓存穿透/击穿场景 |
| **RabbitMQ 超时取消链路** | ❌ 未压 |
| **多时点 heap 对比** | ❌ 未做（见 §3.3 边界） |

**⇒ 本报告回答的是「读接口在高并发下的 CPU 花在哪」，不回答「整个服务性能如何」。**

## 5. 与 2026-07-11 那份基线的差异（口径不同，不可直接比）

| | 2026-07-11 | 2026-10-23（本报告） |
| --- | --- | --- |
| 部署 | Docker Compose（容器化） | **原生进程** |
| 客户端 | 宿主机请求容器 | **与服务工作进程同机** |
| 商品数 | 1000 | 125（100 上架） |
| RPS | 1400.31 | **3073** |
| P95 | 23 ms | 19 ms |

**⇒ RPS 翻倍不能读成「优化了 2.2 倍」** —— 部署形态、客户端位置、数据量三项都不同。
**要让两个数字可比，必须固定其中两项**。本次未做该对照实验。

## 6. 复现命令

```bash
# 1. 起临时 MySQL（仅绑回环，独立 datadir）
mysqld --defaults-file=<tmp>/my.ini --initialize-insecure
mysqld --defaults-file=<tmp>/my.ini

# 2. 建库 + 迁移 + 种子
mysql -h127.0.0.1 -P3307 -uroot -e "CREATE DATABASE go_order_management_system"
goose -dir ./migrations mysql "root:<pw>@tcp(127.0.0.1:3307)/go_order_management_system?parseTime=true" up
mysql -h127.0.0.1 -P3307 -uroot go_order_management_system -e "source docs/sql/data/001_seed_non_user_business_data_500.sql"

# 3. 起服务（pprof 开关 + 独立端口）
PPROF_ENABLED=true PPROF_ADDR=127.0.0.1:6060 \
  MYSQL_PASSWORD=<pw> DB_PORT=3307 REDIS_ADDR=127.0.0.1:6380 JWT_SECRET=<32+ chars> \
  ./go-order-management-system

# 4. 压测 + 并发抓 30s CPU profile
go run ./cmd/loadtest -url "http://127.0.0.1:8082/api/v1/products?page=1&page_size=20" \
  -header "Authorization: Bearer <token>" -requests 100000 -concurrency 50 -output report.md &
curl -o cpu.prof "http://127.0.0.1:6060/debug/pprof/profile?seconds=30"

# 5. 分析
go tool pprof -top -cum cpu.prof
go tool pprof -peek='log/slog.(*Logger).LogAttrs' cpu.prof
```
