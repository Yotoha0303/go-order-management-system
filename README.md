<div align="right">
  <strong>Languages:</strong>
  <a href="README.md">简体中文</a> |
  <a href="README.en.md">English</a>
</div>

# Go 订单库存一致性管理服务

[![CI](https://github.com/Yotoha0303/go-order-management-system/actions/workflows/ci.yml/badge.svg)](https://github.com/Yotoha0303/go-order-management-system/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25.7-00ADD8?logo=go&logoColor=white)](go.mod)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)](fronted/package.json)
[![MySQL](https://img.shields.io/badge/MySQL-8.4-4479A1?logo=mysql&logoColor=white)](compose.yml)
[![Redis](https://img.shields.io/badge/Redis-7.2-DC382D?logo=redis&logoColor=white)](compose.yml)
[![RabbitMQ](https://img.shields.io/badge/RabbitMQ-4.1-FF6600?logo=rabbitmq&logoColor=white)](compose.yml)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](compose.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

> 一个订单库存一致性后端服务，覆盖事务一致性、并发库存扣减、用户级幂等、订单状态机、RabbitMQ 延迟取消、用户数据隔离、Redis 缓存与 CI 门禁。

本项目不是完整商城系统，不刻意堆购物车、支付、物流、优惠券等外围功能；项目重点是把订单库存主链路做扎实：下单幂等、并发扣减不超卖、跨表写入一致、订单状态机、超时自动取消与库存回补、用户数据隔离、Redis 缓存与降级、可观测性与 CI 门禁。

> 📖 **相关入口**
> - 📘 [核心业务设计](#核心业务设计)：订单创建、幂等、缓存与业务规则的设计说明
> - 🧪 [测试与验证](#测试与验证)：本地门禁命令与测试覆盖范围
> - 🚀 [快速启动](#快速启动)：本地、Docker 与云主机三种跑法
> - 📚 [项目文档](#项目文档)：全部设计文档与运行证据索引

---

## 一眼看懂这个项目

| 维度 | 说明 |
| --- | --- |
| 项目定位 | Go 订单库存一致性管理服务 |
| 适用场景 | 中小规模订单/库存系统的后端基座；并发扣减、幂等与最终一致性的参考实现 |
| 核心主链路 | 登录用户创建订单 → 校验幂等 Key → 事务扣减库存 → 写入订单明细与库存流水 → 状态流转 |
| 主要后端能力 | Gin、GORM、MySQL 事务、行锁、条件更新、RabbitMQ、Outbox、JWT、Redis cache-aside、Docker、CI |
| 工程取舍 | 覆盖下单幂等、防超卖、跨表一致性、状态机、超时取消回补、数据隔离与 CI 门禁，不铺外围电商功能 |

### 在线演示

部署实例由云端托管，可直接访问；演示账号请联系仓库维护者获取（**凭据不写入仓库**）。

| 入口 | 地址 | 说明 |
| --- | --- | --- |
| 前端管理台 | `https://go-order-management-system.vercel.app` | React SPA，覆盖商品、库存、订单、操作日志 |
| 后端就绪探针 | `https://go-order-management-system.onrender.com/readyz` | 返回 `{"code":0,"data":{"status":"ready"}}` 表示依赖就绪 |

> 云端实例运行在免费档，长时间无访问后会冷启动，首次请求可能等待数十秒。

## 核心业务链路

**同步主链路**（创建订单到状态流转）：

```mermaid
flowchart LR
    A[用户注册 / 登录] --> B[JWT 鉴权]
    B --> C[商品与库存准备]
    C --> D[提交订单]
    D --> E[用户级幂等校验]
    E --> F[开启事务]
    F --> G[锁定库存并条件扣减]
    G --> H[创建订单与订单明细]
    H --> I[写入库存流水]
    I --> J{订单状态机}
    J -->|支付| K[已支付]
    K -->|完成| L[已完成]
    J -->|取消| M[已取消并回滚库存]
```

**异步超时链路**（待支付订单自动取消）：

```mermaid
flowchart LR
    N[创建订单] -->|同事务| O[order_timeout_outbox]
    O -->|轮询认领| P[发布到延迟队列]
    P --> Q[阶梯队列 5m / 15m / 30m]
    Q -->|TTL 到期经 DLX| R[超时消费]
    R -->|仍为待支付| S[取消订单并回补库存]
    R -->|已支付 / 已取消| T[幂等跳过]
    R -->|多次失败| U[失败队列 order.timeout.failed]
```

## 项目重点解决的问题

| 问题 | 解决方式 | 项目价值 |
| --- | --- | --- |
| 重复提交导致重复订单 | `(user_id, idempotency_key)` 唯一索引 + SHA-256 请求摘要 | 避免用户重复点击、客户端重试造成重复下单和重复扣库存 |
| 热点库存直接打 DB | Redis Lua 原子预扣 + reservation 补偿 | 在 MySQL 事务前增加软保护，降低热点请求直接进入数据库扣减 |
| 并发下单导致超卖 | MySQL 事务 + `SELECT ... FOR UPDATE` + `stock_quantity >= quantity` 条件扣减 | 保证库存不会被扣成负数 |
| 多表写入不一致 | 幂等记录、订单、订单项、库存、库存流水放在同一事务 | 任一步失败时整体回滚 |
| 订单越权访问 | 订单查询和状态变更同时匹配当前登录用户 `user_id` | 防止用户访问或操作他人订单 |
| 非法订单状态流转 | service 层状态机 + DAO 条件状态更新 | 限制待支付、已支付、已完成、已取消之间的合法流转 |
| 多商品下单死锁风险 | 下单事务内按 `product_id` 升序处理商品与库存行锁 | 降低不同请求商品顺序不一致造成的循环等待 |
| 商品详情重复查询 | Redis cache-aside 缓存 + 商品上下架删除缓存 | 减少热点商品详情对 MySQL 的重复访问 |
| 待支付订单长期占用库存 | 事务 Outbox + RabbitMQ TTL/DLX + 幂等消费者 | 订单创建 30 分钟后仍未支付则自动取消并回补库存 |
| 管理操作不可追踪 | 管理员接口写入 `operation_logs`，并用 Request ID 关联 access log | 支持后台审计和排障 |
| 请求和业务状态不可见 | `/metrics` 输出 HTTP、订单创建、状态流转和 Redis 预扣指标 | 支持本地和容器环境采集基础运行指标 |
| 请求链路难关联 | 兼容 W3C `traceparent`，access log 输出 `trace_id/span_id` | 为后续接入 OpenTelemetry 或追踪后端做准备 |
| 慢 SQL 难定位 | GORM 接入 `slog` 慢查询日志，可配置阈值和日志等级 | 为后续 EXPLAIN 分析和性能报告提供入口 |
| Redis 库存 key 丢失 | 管理员可查看 Redis/MySQL 差异报告，并按 MySQL 当前库存重建 Redis 可售库存 | Redis 重启或 key 缺失后可快速恢复预扣能力 |
| Redis/MySQL 库存不一致不可见 | 后台 worker 定时对账并记录差异日志 | 及时发现 Redis 软保护层与 MySQL 事实源的偏差 |
| Redis Cluster 多 key Lua 跨 slot | 库存预扣 key 使用固定 hash tag `{stock}` | 多商品预扣和 reservation 操作可落在同一 slot |
| 队列不同 TTL 消息互相阻塞 | 分级固定 TTL 阶梯队列，替代单消息 `Expiration` | 消除 RabbitMQ 队头阻塞（详见 [ADR-002](docs/adr/002-rabbitmq-ladder-delay-queue-topology.md)） |

## 技术选型检索表

> 图形化流程无法被搜索引擎与代码检索工具索引，故以表格补充关键技术的**英文检索关键词**，便于按技术栈定位实现位置。

| 核心组件 / 服务模块 | 英文检索关键词 (Keywords) | 部署层级与形态 | 通信协议与端口 | 职责与技术特性 |
| :--- | :--- | :--- | :--- | :--- |
| **分布式预扣缓存** | `Redis Lua Pre-deduct`, `Atomic DECRBY / INCRBY`, `Hash Tag {stock}` | 缓存层 `internal/bizcache` | RESP `6379` | 事务前置预扣，Lua 原子比对库存并写入 reservation 凭证，消除热点请求直接击穿 MySQL |
| **事务一致性扣减** | `MySQL Transaction`, `SELECT FOR UPDATE`, `Row-Level Lock`, `Conditional Update` | 存储层 `internal/dao` | MySQL `3306` | 排他行锁 + `stock_quantity >= quantity` 条件更新，杜绝并发超卖，保障 ACID |
| **分布式唯一 ID** | `Snowflake ID Generator`, `Distributed ID`, `64-bit Int` | 底层库 `pkg/idgen` | 进程内 | 41 位时间戳 + 10 位 worker + 12 位序列，订单入库前完成全局 ID 预分配 |
| **高并发幂等防重** | `Idempotency Key`, `SHA-256 Request Hash`, `MySQL RR Isolation` | 领域层 `internal/service` | 内存摘要 + 唯一索引 | `(user_id, idempotency_key)` 复合唯一索引 + 请求哈希校验，拦截重复点击与网络重放 |
| **可靠延迟取消** | `Transactional Outbox`, `RabbitMQ TTL + DLX`, `Dead Letter Queue`, `Lease-based Claim` | 消息层 `internal/ordertimeout` | AMQP `5672` | 订单与 Outbox 同事务持久化；多实例用 `FOR UPDATE SKIP LOCKED` + 租期认领防重复发布 |
| **阶梯延时队列** | `Ladder Delay Queue`, `Queue-level x-message-ttl`, `Head-of-Line Blocking` | 消息层 `internal/ordertimeout` | AMQP `5672` | 5m / 15m / 30m 三级固定 TTL 物理队列，消除单消息 TTL 的队头阻塞 |
| **库存三态与防死锁** | `Locked Quantity`, `Three-State Model`, `Deadlock Prevention`, `Ordered Locking` | 存储与模型层 `migrations/` | SQL DDL / ACID | 拆分可售与锁定库存，下单事务按 `product_id ASC` 偏序遍历，破除哲学家就餐环路 |
| **全链路可观测性** | `Prometheus Metrics`, `W3C Traceparent`, `Structured Slog`, `Slow Query Log` | 观测层 `/metrics` `/readyz` | HTTP `8082` | 暴露黄金业务指标，关联 request_id / trace_id，慢查询可配置阈值告警 |
| **缓存防穿透** | `Null Object Cache`, `TTL Jitter`, `Cache Penetration` | 缓存层 `internal/bizcache` | RESP `6379` | 空值哨兵 + 短 TTL 随机抖动，非法商品 ID 不穿透到 MySQL |
| **对账自愈** | `Reconcile Worker`, `Auto-Fix`, `Redis/MySQL Drift` | 后台 `internal/inventoryreconcile` | 定时任务 | 周期性比对 Redis 与 MySQL 库存，差异可自动纠偏并暴露 Prometheus 指标 |

## 核心功能

### 用户与鉴权

- 用户注册、登录、查询当前用户信息
- 修改昵称、修改密码
- HS256 JWT 签发、过期校验和 Bearer 鉴权
- 除健康检查、注册和登录外，业务接口统一要求 JWT
- 订单读写接口基于当前登录用户做数据隔离

### 商品模块

- 创建商品
- 查询商品列表和商品详情
- 商品上架、下架
- 商品价格使用 `price_fen`，单位为分，避免浮点精度问题
- 商品创建后默认下架，避免无库存商品直接进入下单链路

### 库存模块

- 初始化商品库存
- 增加商品库存
- 查询商品库存
- 记录库存变更流水
- 通过库存流水追踪初始化、入库、下单扣减、取消回滚等业务来源

### 订单模块

- 使用 `idempotency_key` 幂等创建订单
- 使用 Redis Lua 预扣库存；Redis 不可用或 key 缺失时降级走 MySQL
- 库存预扣 key 使用 `inventory:{stock}:...` 固定 hash tag，降低 Redis Cluster 跨 slot 风险
- 创建订单时扣减库存
- 库存不足时事务回滚
- 查询订单列表和订单详情
- 支付、完成、取消订单
- 取消待支付订单时回滚库存
- 订单状态机限制非法状态流转
- 待支付超过 30 分钟后由 RabbitMQ 触发自动取消和库存回补
- 创建订单与超时 Outbox 同事务提交，发布使用 Publisher Confirm，消费使用手动 ACK
- Redis 预扣成功后写 reservation；事务失败或取消订单时回补 Redis，支付成功时清理 reservation
- 管理员可触发 Redis/MySQL 库存差异报告和 Redis 可售库存重建，重建数据来源为 MySQL 当前库存
- 后台 worker 定时执行 Redis/MySQL 库存对账，发现差异可配置自动纠偏

### 前端管理台

- React 管理台已接入商品、库存、库存流水、操作日志、订单和用户资料接口
- 首页作为项目展示页，突出后端健康状态、订单摘要、库存流水和工程能力点
- 可完整演示商品准备、库存初始化、用户下单、支付、取消、库存回滚等业务流程

## 技术栈

### 后端

- Go 1.25.7
- Gin + GORM
- MySQL 8.4
- Redis 7.2
- RabbitMQ 4.1
- JWT
- Goose 数据库迁移
- YAML + godotenv 配置

### 前端管理台

- React 19 + TypeScript + Vite
- TanStack Router / Query
- Axios + Zustand
- Tailwind CSS + Shadcn UI

### 工程化

- Docker + Docker Compose
- Makefile
- GitHub Actions
- golangci-lint
- go test / go test -race / go vet / go build

## 项目结构

```text
.github/workflows/ci.yml  持续集成配置
cmd/                      项目启动入口
config/                   YAML 加载、环境变量覆盖和配置校验
docs/                     设计文档、REST Client 请求和验证证据
internal/apperror/        业务错误定义与错误码映射
internal/app/             依赖装配、HTTP Server 和优雅退出
internal/auth/            JWT 签发、校验和认证上下文
internal/bizcache/        Redis 业务缓存
internal/dao/             数据库访问层
internal/handler/         HTTP 接口层
internal/middleware/      请求 ID、日志、超时和恢复中间件
internal/model/           GORM 数据模型
internal/ordertimeout/    RabbitMQ 延迟拓扑、Outbox 发布和超时消费
internal/request/         请求参数和校验规则
internal/response/        统一响应结构
internal/service/         业务规则、状态机和事务
fronted/                  React 管理台、认证状态和后端 API 适配层
migrations/               Goose SQL 迁移
pkg/database/             MySQL 初始化与连接池
pkg/redis/                Redis 客户端初始化
router/                   路由注册
compose.yml               应用、MySQL、Redis、RabbitMQ 编排
Dockerfile                应用镜像多阶段构建（本地与云部署共用）
Makefile                  开发、测试、Docker 和迁移命令入口
```

## 分层说明

项目采用常见业务后端分层方式：

| 层 | 职责 |
| --- | --- |
| handler | HTTP 请求处理、参数绑定、错误映射和统一响应 |
| service | 业务规则、状态流转、事务控制和跨表操作 |
| dao | 数据库 CRUD、条件查询和条件更新 |
| model | 数据库表结构映射 |
| request / response | 入参校验与响应结构 |
| bizcache | 业务缓存读写、缓存 key 规则和缓存失效 |
| apperror | 业务错误、错误码和错误信息封装 |

核心原则：**handler 不写业务规则，service 不直接拼 HTTP 响应，dao 不处理业务状态。**

## 核心表设计

| 表 | 作用 | 关键设计 |
| --- | --- | --- |
| `users` | 用户表 | `username` 唯一、密码哈希、用户状态、软删除 |
| `roles` | 角色表 | `role_name` 唯一，内置 `admin` 与 `user` |
| `user_roles` | 用户角色关联表 | `user_id` 唯一，当前每个用户只绑定一个角色 |
| `products` | 商品表 | `price_fen` 金额分、商品状态、默认下架 |
| `product_inventories` | 商品库存表 | `product_id` 唯一；`stock_quantity` 可售 + `locked_quantity` 锁定（三态模型） |
| `stock_logs` | 库存流水表 | 记录变更前、变更量、变更后、业务类型和业务 ID |
| `orders` | 订单主表 | `user_id` 归属隔离、`order_no` 唯一、订单状态机 |
| `order_items` | 订单明细表 | 保存商品名称、价格快照和数量 |
| `order_idempotency_keys` | 订单幂等表 | `(user_id, idempotency_key)` 复合唯一索引 + `request_hash` |
| `order_timeout_outbox` | 订单超时 Outbox | 与订单同事务写入，记录发布时间、重试次数和超时截止点 |
| `order_timeout_dead_letters` | 超时死信表 | 持久化进入失败队列的消息，便于人工排障 |
| `operation_logs` | 后台操作日志表 | 记录管理员动作、路由、HTTP 状态、请求 ID 和来源信息 |

详细表结构见：[docs/table_design.md](docs/table_design.md)

## 核心业务设计

| 主题 | 文档 | 说明 |
| --- | --- | --- |
| 订单流程 | [docs/order_flow.md](docs/order_flow.md) | 订单创建、取消和状态流转 |
| 幂等设计 | [docs/idempotency.md](docs/idempotency.md) | 幂等 Key、请求哈希与并发冲突处理 |
| 缓存设计 | [docs/cache_design.md](docs/cache_design.md) | Redis 商品详情缓存与防穿透 |
| 业务规则 | [docs/business_rules.md](docs/business_rules.md) | 完整业务规则约束 |
| 接口清单 | [docs/api_list.md](docs/api_list.md) | REST 接口列表 |
| 架构决策 | [docs/adr](docs/adr) | ADR-001 库存架构 / ADR-002 阶梯延时队列 / ADR-003 库存三态与对账自愈 |

## 工程化能力

- HTTP Server 设置 ReadTimeout、WriteTimeout、IdleTimeout、ReadHeaderTimeout 和 MaxHeaderBytes
- MySQL 初始化时配置连接池：MaxOpenConns、MaxIdleConns、ConnMaxLifetime、ConnMaxIdleTime
- 启动时使用 PingContext 检查 MySQL 连通性
- GORM 慢 SQL 日志默认记录超过 `mysql.slowThreshold` 的 SQL，日志等级可通过 `mysql.logLevel` 或环境变量覆盖
- 请求层使用 Request ID、Trace Context、Access Log、Recovery 和超时中间件
- Access Log 输出 request_id、trace_id 和 span_id；超时响应也会补充 `traceparent`
- Redis 不可用时商品详情缓存自动降级，不影响主流程；**降级会以 ERROR 级日志显式告警**，不静默
- Redis 库存 key 采用固定 hash tag；旧格式 Redis key 升级后可通过管理员重建接口刷新
- RabbitMQ 使用 TTL + DLX 延迟投递；发布端 Confirm、消费端手动 ACK，重复消息由订单状态条件更新消解
- `/metrics` 暴露 Prometheus 文本格式指标，覆盖 HTTP 请求、订单创建、订单状态流转和 Redis 预扣结果
- 单一 Dockerfile 多阶段构建（本地 Compose 与云部署共用），非 root 运行；BuildKit 缓存与可配置 `GOPROXY`/`GOSUMDB`/`APK_MIRROR`
- Docker Compose 编排应用、MySQL、Redis 和 RabbitMQ，并通过健康检查控制依赖启动顺序
- Goose 管理数据库版本，Makefile 统一封装开发、测试、Docker 和迁移命令
- 内置 HTTP 压测命令，可生成 Markdown 性能报告；已提交 [本地 Docker 读链路压测与 SQL 分析](docs/evidence/loadtest_summary_2026-07-11.md)，数字仅作为同机基线
- CI 覆盖 go test、go test -race、go vet、golangci-lint、goose validate 和 go build；**前端另有独立作业**覆盖 `tsc -b`、eslint、vitest（浏览器模式）与构建
- CI 在集成测试开关打开时**断言零跳过**：任何被 `t.Skip` 绕过的集成用例都会让流水线变红

## 快速启动

### 前置依赖

- Go 1.25.7
- GNU Make
- Docker 与 Docker Compose
- Goose v3.27.1

```bash
go mod download
go install github.com/pressly/goose/v3/cmd/goose@v3.27.1
```

### 配置环境变量

应用启动时先加载 `.env`，再读取 [config.yml](config.yml)。环境变量会覆盖 YAML 中适合按环境变化的连接配置。

```env
MYSQL_PASSWORD=your-password
DB_SLOW_THRESHOLD=200ms
DB_LOG_LEVEL=warn
JWT_SECRET=replace-with-at-least-32-random-characters
JWT_EXPIRE_HOURS=24
# Redis: use REDIS_URL for managed/cloud endpoints (rediss:// enables TLS),
# or the discrete REDIS_ADDR/REDIS_USERNAME/REDIS_PASSWORD/REDIS_TLS set.
REDIS_URL=
REDIS_ADDR=
REDIS_USERNAME=
REDIS_PASSWORD=
REDIS_TLS=
RABBITMQ_URL=amqp://order_app:order_dev_password@127.0.0.1:5672/
ORDER_TIMEOUT_DELAY=30m
INVENTORY_RECONCILE_ENABLED=true
INVENTORY_RECONCILE_INTERVAL=5m
INVENTORY_RECONCILE_TIMEOUT=3s
```

不要提交真实 `.env`，可从 [.env.example](.env.example) 复制后修改。

> `RABBITMQ_URL` 中的 `order_dev_password` 是**本地开发默认值**（Compose 的兜底值），不是真实凭据；云端部署请通过环境变量覆盖。

### 本地运行后端

PowerShell 示例：

```powershell
Copy-Item .env.example .env
$env:MYSQL_PASSWORD = "your-password"
$env:JWT_SECRET = "replace-with-at-least-32-random-characters"

make infra-up
make migrate-up
make run
```

默认访问地址：`http://localhost:8082`

健康检查：

```bash
curl http://localhost:8082/ping
curl http://localhost:8082/live
curl http://localhost:8082/readyz
curl http://localhost:8082/metrics
```

### Docker 运行完整服务

```powershell
$env:MYSQL_PASSWORD = "your-password"
$env:JWT_SECRET = "replace-with-at-least-32-random-characters"

make docker-up
```

`make docker-up` 会构建应用镜像，启动 MySQL、Redis、RabbitMQ 和应用，并执行数据库迁移。

应用镜像构建默认面向中国大陆网络：`GOPROXY=goproxy.cn`（含 goproxy.io 兜底）、`GOSUMDB=sum.golang.google.cn`、Alpine 使用阿里云源；本地与云部署共用同一 Dockerfile。海外可覆盖：

```bash
GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org APK_MIRROR=dl-cdn.alpinelinux.org make docker-up
```

基础镜像拉不动时，可在 `.env` 设置 `GO_IMAGE` / `RUNTIME_IMAGE` 为镜像站地址，详见 [docs/deploy.md](docs/deploy.md)。

### 云主机一键部署

Ubuntu/Debian 云主机可使用仓库根目录脚本，自动安装 Docker、启动完整后端栈、构建前端并由 Nginx 同源反代：

```bash
chmod +x deploy.sh
sudo ./deploy.sh
```

说明见 [docs/deploy.md](docs/deploy.md)。脚本会生成或补齐 `.env`，部署后提供前端页面、`/api`、`/readyz` 与 `/metrics`。

### 本地运行前端

```powershell
Set-Location fronted
npm install
npm run dev
```

前端默认运行在 `http://127.0.0.1:8880`，Vite 会把 `/api` 和 `/ping` 代理到 `http://localhost:8082`。

## 常用命令

| 命令 | 作用 |
| --- | --- |
| `make infra-up` | 仅启动 MySQL、Redis 和 RabbitMQ，并等待健康 |
| `make infra-down` | 停止 Compose 项目 |
| `make migrate-validate` | 静态校验迁移文件 |
| `make migrate-status` | 查看数据库迁移状态 |
| `make migrate-up` | 执行全部待处理迁移 |
| `make migrate-down` | 回滚最近一条迁移 |
| `make docker-build` | 构建应用镜像 |
| `make docker-up` | 构建并启动应用、MySQL、Redis、RabbitMQ |
| `make docker-down` | 停止并移除容器，保留数据卷 |
| `make load-test` | 对 `LOADTEST_URL` 执行 HTTP 压测并输出 Markdown 报告 |
| `make test` | 运行全部 Go 测试 |
| `make test-service` | 运行 MySQL service 集成测试 |
| `make test-dao` | 运行关键 DAO MySQL 集成测试 |
| `make test-migrations` | 在隔离数据库实跑迁移和回滚 |
| `make test-redis` | 运行 Redis 集成测试 |
| `make test-order-timeout` | 运行 RabbitMQ + MySQL 订单超时端到端测试 |
| `make test-all` | 运行普通测试、MySQL service/DAO/迁移测试和 Redis 集成测试 |
| `make test-race` | 使用 race detector 运行测试 |
| `make check` | 执行格式化、模块校验、vet 和测试 |

## 测试与验证

service 测试会清理所连接数据库中的业务表。必须使用独立测试库，禁止将 `DB_NAME` 指向含有开发数据或生产数据的数据库。

当前测试重点：

- 商品、库存、订单创建、状态机和关键异常分支
- 并发下单防超卖
- 多商品事务回滚
- 多商品订单按 `product_id` 升序处理，降低锁顺序风险
- 创建订单幂等：同 Key 重放、不同请求冲突、并发同 Key、失败回滚后重试
- 订单状态并发：并发支付、并发取消、支付与取消竞争
- Redis 商品详情缓存命中、删除和降级
- Redis Lua 库存预扣、库存不足拒绝、reservation 回补和支付确认清理
- 订单 ID 序列化契约：Snowflake 主键以 JSON 字符串输出，防止前端精度截断
- Redis 配置优先级：`REDIS_URL` 与离散环境变量的覆盖关系，以及 TLS 开关的失败关闭

手动接口测试文件位于 [docs/http](docs/http)，完整业务链路见 [docs/http/demo_flow.http](docs/http/demo_flow.http)。测试计划见 [docs/test_plan.md](docs/test_plan.md)。

## 项目文档

| 文档 | 内容 |
| --- | --- |
| [docs/api_list.md](docs/api_list.md) | 接口清单 |
| [docs/business_rules.md](docs/business_rules.md) | 业务规则 |
| [docs/table_design.md](docs/table_design.md) | 数据表设计 |
| [docs/order_flow.md](docs/order_flow.md) | 订单创建、取消和状态流转说明 |
| [docs/idempotency.md](docs/idempotency.md) | 订单创建幂等设计说明 |
| [docs/cache_design.md](docs/cache_design.md) | Redis 商品详情缓存设计说明 |
| [docs/adr](docs/adr) | 架构决策记录（ADR-001 / 002 / 003） |
| [docs/GAP台账.md](docs/GAP台账.md) | 质量缺口与未竟事项唯一台账 |
| [docs/test_plan.md](docs/test_plan.md) | 测试计划 |
| [docs/test_result.md](docs/test_result.md) | 测试结果记录 |
| [docs/observability.md](docs/observability.md) | 可观测性指标和告警建议 |
| [docs/performance.md](docs/performance.md) | 压测工具和慢 SQL 分析流程 |
| [docs/deploy.md](docs/deploy.md) | 云主机一键部署脚本说明 |
| [docs/project_evolution.md](docs/project_evolution.md) | 后续演进 |
| [docs/interview_guide.md](docs/interview_guide.md) | 简历描述、项目讲解和面试追问 |
| [docs/evidence](docs/evidence) | 项目运行、测试与关键业务证据 |

## 当前边界与后续演进

当前项目重点是后端业务基本功，不把系统包装成完整电商平台。

后续可演进方向：

- 增加订单列表状态筛选和时间范围筛选
- 扩展 handler 边界测试和 DAO 集成测试覆盖
- 增加线上告警配置证据、结构化日志字段规范和链路追踪
- 为幂等记录增加过期清理策略
- 增加操作日志高级筛选、导出和留存策略
- 补齐覆盖率阈值门禁（当前仅产出覆盖率报告，未设地板值）

## 开源许可证

本项目采用 [Apache License 2.0](LICENSE)。`fronted/` 子目录基于上游模板，遵循其自身的 [MIT 许可](fronted/LICENSE)。
