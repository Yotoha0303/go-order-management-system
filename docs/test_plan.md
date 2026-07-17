# 项目测试说明

## 1. 测试目的

本测试方法采用手动测试和自动测试两种不同的方法进行测试，首要的目的除了是验证商品、库存、订单模块在正常流程和错误流程下的数据一致性外，着重项目

核心代码的功能可靠性测试，并且有效缩短测试的流程。

其中，用 REST Client 手动测试进行接口测试，用自动化测试来围绕以订单创建和订单状态机状态流转为主的业务功能可靠性测试。

## 2. 测试类型

本项目当前包含四类测试方式：

1. REST Client 手动接口测试
2. service 层自动化测试
3. Redis 缓存集成测试
4. React 前端自动化测试


## 3. REST Client 接口测试

测试文件位置：

```text
docs/http/auth.http
docs/http/demo_flow.http
docs/http/products.http
docs/http/inventory.http
docs/http/stock_logs.http
docs/http/orders.http
docs/http/redis.http
```

执行方式：

1. 安装 VS Code REST Client 插件
2. 启动项目：`go run cmd/main.go`
3. 打开对应 `.http` 文件
4. 点击每个请求上方的 `Send Request`
5. 对比响应结果和数据库变化

### 3.1 用户与鉴权模块自测

- [x] 用户注册成功
- [x] 用户登录成功并返回 access_token
- [x] 携带 Bearer Token 可以查询当前用户
- [ ] 修改当前用户昵称
- [ ] 修改当前用户密码
- [ ] 未携带 Token 访问受保护接口返回 401
- [ ] 使用两个账号手动验证订单数据隔离

### 3.2 商品模块自测

- [x] 创建商品成功
- [x] 创建商品后 `status = 2`
- [x] `price_fen <= 0` 返回参数错误
- [x] `name` 为空返回参数错误
- [x] 查询商品列表成功
- [x] 按上架、下架和全部状态筛选商品
- [x] 商品列表分页及分页参数边界
- [x] 查询商品详情成功
- [x] 查询不存在商品返回错误
- [x] 商品上架成功
- [x] 商品下架成功

### 3.3 库存模块自测

- [x] 存在商品可以初始化库存
- [x] 不存在商品不能初始化库存
- [x] 重复初始化库存失败
- [x] `stock_quantity = 0` 可以初始化
- [x] 初始化库存后 `product_inventories` 有记录
- [x] 初始化库存后 `stock_logs` 有 `biz_type = 1` 记录
- [x] 已初始化库存的商品可以增加库存
- [x] 未初始化库存的商品不能增加库存
- [x] `quantity <= 0` 返回参数错误
- [x] 增加库存后 `stock_quantity` 正确变化
- [x] 增加库存后 `stock_logs` 有 `biz_type = 2` 记录

### 3.4 库存流水自测

- [x] 不传 `product_id` 可以查询全部流水
- [x] 传 `product_id` 可以查询指定商品流水
- [x] `product_id` 非法返回参数错误
- [x] 初始化库存后能查到 `biz_type = 1`
- [x] 增加库存后能查到 `biz_type = 2`
- [x] 创建订单后能查到 `biz_type = 3`
- [x] 取消订单后能查到 `biz_type = 4`
- [x] `before_quantity / change_quantity / after_quantity` 正确

### 3.5 订单状态机测试

创建订单

- [x] 正常创建订单成功
- [x] 商品不存在时创建订单失败
- [x] 商品下架时创建订单失败
- [x] 库存不存在时创建订单失败
- [x] 库存不足时创建订单失败
- [x] 创建订单成功后 `orders` 有记录
- [x] 创建订单成功后 `order_items` 有记录
- [x] 创建订单成功后 `product_inventories` 库存扣减
- [x] 创建订单成功后 `stock_logs` 有 `biz_type = 3` 记录
- [x] 相同 idempotency_key 和相同请求返回同一订单
- [x] 相同 idempotency_key 和不同请求返回冲突
- [x] 并发使用相同 idempotency_key 只创建一笔订单
- [x] 创建失败时幂等记录随事务回滚，允许重试

支付订单

- [x] 待支付订单可以支付
- [x] 已支付订单重复支付失败
- [x] 已取消订单支付失败
- [x] 已完成订单支付失败
- [x] 不存在订单支付失败

完成订单

- [x] 已支付订单可以完成
- [x] 未支付订单完成失败
- [x] 已取消订单完成失败
- [x] 已完成订单重复完成失败
- [x] 不存在订单完成失败

取消订单

- [x] 待支付订单可以取消
- [x] 取消订单后库存回滚
- [x] 取消订单后 `stock_logs` 有 `biz_type = 4` 记录
- [x] 已支付订单取消失败
- [x] 已完成订单取消失败
- [x] 不存在订单取消失败
- [x] 已取消订单再次取消直接成功
- [x] 已取消订单再次取消不会重复回滚库存

### 3.6 Redis 缓存接口自测

- [x] 商品详情缓存 key 是否正确
- [x] Redis 为空时，缓存函数不会影响主流程
- [x] SetProductDetail 后能 GetProductDetail
- [x] DeleteProductDetailCache 后再次 Get 应该 miss
- [x] 缓存 TTL 是否存在

## 4. service 层自动化测试（包含订单状态机测试）

测试文件位置：

```text
internal/service/*_test.go
```

执行方式：

```bash
make test-service
```

测试内容：

- [x] 商品创建、上下架和查询相关业务规则
- [x] 库存初始化、增加库存和库存异常场景
- [x] 创建订单时库存扣减、库存不足回滚
- [x] 订单支付、完成、取消状态流转
- [x] 已取消订单重复取消不会重复回滚库存
- [x] 关键异常链路返回预期业务错误
- [x] 注册用户和默认角色在同一事务内成功或回滚
- [x] 修改密码后旧密码失效、新密码可登录

## 5. Handler 业务接口测试

测试文件位置：

```text
internal/handler/*_test.go
```

执行方式：

```bash
go test -v ./internal/handler
```

最小测试内容：

- [x] 创建订单时正确传递当前用户、幂等键和商品明细
- [x] 非法请求在调用 service 前返回 400
- [x] service 业务错误正确映射为 HTTP 状态码和业务错误码

## 6. DAO MySQL 集成测试

测试文件位置：

```text
internal/dao/dao_integration_test.go
```

执行方式：

```bash
make test-dao
```

测试内容：

- [x] 用户角色变更后查询立即生效
- [x] 条件扣库存不会把库存扣成负数
- [x] 订单查询和状态修改强制校验用户归属

## 7. 数据库迁移集成测试

执行方式：

```bash
make test-migrations
```

测试内容：

- [x] 在隔离数据库执行全部迁移和回滚
- [x] 存量用户自动回填 `user` 角色
- [x] `user_roles` 外键完整创建
- [x] `order_timeout_outbox` 字段和订单外键完整创建
- [x] `operation_logs` 字段和索引完整创建

## 8. RabbitMQ 订单超时取消测试

执行方式：

```bash
make test-order-timeout
```

- [x] 创建订单与超时 Outbox 同事务提交，失败整体回滚
- [x] 超时截止时间为订单创建时间加 30 分钟
- [x] 超时取消回补库存并写入一条回滚流水
- [x] 重复超时消息不重复回补库存
- [x] 已支付订单收到超时消息保持已支付
- [x] MySQL Outbox 截止时间作为事实源，提前投递不能取消订单
- [x] RabbitMQ 消息 expiration 使用剩余毫秒，已过期事件最小为 1ms
- [x] 非法或带未知字段的消息进入失败队列

## 9. Redis 缓存集成测试


测试文件位置：

```text
internal/bizcache/product_cache_test.go
```

执行方式：

```bash
RUN_REDIS_TEST=1 go test -v ./internal/bizcache
```

测试内容：

- [x] key 正确  
- [x] Redis nil 不 panic  
- [x] Set/Get/Delete 正常  
- [x] TTL 正常  
- [x] 接口层手动验证上下架删除缓存

## 9.1 Redis 库存预扣测试

测试文件位置：

```text
internal/bizcache/inventory_stock_guard_test.go
internal/service/order_service_test.go
internal/service/inventory_service_test.go
```

测试内容：

- [x] Redis 可售库存 key 和 reservation key 命名正确
- [x] Redis 库存预扣 key 使用固定 hash tag，避免多 key Lua 跨 slot
- [x] Redis nil 时预扣、回补、确认和同步库存不影响主流程
- [x] Lua 预扣成功后可售库存减少并写 reservation
- [x] Redis 库存不足时不扣减并返回库存不足
- [x] Redis key 缺失时跳过预扣，降级走 MySQL
- [x] MySQL 事务失败后释放 Redis reservation
- [x] 支付成功后确认并删除 reservation
- [x] 取消订单成功后按 reservation 回补 Redis
- [x] 初始化库存和手动入库成功后同步 Redis 可售库存
- [x] 管理员可按 MySQL 当前库存重建 Redis 可售库存
- [x] 管理员可查看 Redis/MySQL 库存差异报告
- [x] 后台 worker 可定时触发 Redis/MySQL 库存对账
- [x] 自动对账发现差异只记录日志，不自动重建

## 10. React 前端自动化测试

测试文件位置：

```text
fronted/src/**/*.{test,spec}.{ts,tsx}
```

执行方式：

```bash
cd fronted
npm test
```

测试内容：

- [x] Token 过期后清除认证状态并触发登录跳转
- [x] 相同订单请求重试复用幂等 Key，请求内容改变后生成新 Key
- [x] 订单状态对应的支付、完成和取消操作权限
- [x] 管理员侧边栏展示操作日志入口，普通用户隐藏管理入口
- [x] 完成和取消订单要求二次确认
- [x] 商品元转分转换及非法小数位校验
- [x] 登录、注册和账号退出交互

## 11. 可观测性测试

测试文件位置：

```text
pkg/database/gorm_logger_test.go
config/mysql_config_test.go
internal/inventoryreconcile/worker_test.go
internal/loadtest/loadtest_test.go
internal/observability/*_test.go
internal/middleware/metrics_middleware_test.go
internal/middleware/trace_context_middleware_test.go
router/router_test.go
```

测试内容：

- [x] MySQL 慢 SQL 阈值和 GORM 日志等级配置校验
- [x] GORM 慢查询输出 `gorm slow query`
- [x] GORM 数据库错误输出 `gorm query error`
- [x] `record not found` 不按数据库错误日志输出
- [x] 合法 `traceparent` 复用 trace_id 并生成服务端 span_id
- [x] 非法 `traceparent` 自动重生成
- [x] Access Log 输出 trace_id 和 span_id
- [x] 超时响应保留 `traceparent`
- [x] Redis 自动对账 worker 记录差异和错误日志
- [x] HTTP 压测工具统计状态码、错误、RPS 和延迟分位
- [x] HTTP 压测工具可渲染 Markdown 报告
- [x] HTTP metrics 按 method、route、status 聚合请求数
- [x] HTTP metrics 输出 Prometheus 文本格式
- [x] metrics 中间件使用 Gin 路由模板，避免真实 ID 进入指标 label
- [x] `/metrics` 不要求业务 JWT
- [x] 订单创建结果指标
- [x] 订单状态流转结果指标
- [x] Redis 预扣、reservation 和库存同步指标
- [x] 可观测性文档记录指标含义和告警建议

## 12. 第十阶段后续验收清单

本节只列尚未完成或部分完成的第十阶段事项。勾选前必须同时具备实现、自动化测试和运行证据。

### 12.1 订单写链路与热点库存压测（未完成，P0）

- [ ] 压测请求可为每次创建订单生成唯一幂等键
- [ ] 单商品热点库存并发下单覆盖成功、库存不足和系统错误分布
- [ ] 多商品并发下单验证固定加锁顺序，记录死锁和事务重试情况
- [ ] 压测后核对 MySQL 库存、Redis 可售库存、reservation 和库存流水一致性
- [ ] 记录应用、MySQL、Redis 和 RabbitMQ 的 CPU/内存峰值
- [ ] 提交原始压测报告、日志摘要、测试数据规模和环境说明

### 12.2 OpenTelemetry 完整追踪（未完成，P1）

- [ ] 确定 OpenTelemetry SDK、Collector 和 Jaeger/Tempo 方案
- [ ] HTTP 服务端请求创建并导出 span
- [ ] DB、Redis 和 RabbitMQ 关键调用可关联到同一 trace
- [ ] 采样或导出失败不影响订单主链路
- [ ] 追踪后端可按 trace_id 查询完整链路
- [ ] 提交配置、自动化测试和追踪界面证据

### 12.3 线上部署与稳定性（未完成，P1）

- [ ] 云主机完成应用、MySQL、Redis 和 RabbitMQ 部署
- [ ] 配置域名、TLS、密钥和最小网络访问边界
- [ ] 健康检查和指标被外部监控系统持续采集
- [ ] 完成容器异常退出后的自动恢复验证
- [ ] 连续运行至少 24 小时并保存错误率、资源和日志证据
- [ ] 形成可复现部署、回滚和故障处理说明

### 12.4 支付、退款、发货和售后（未完成，P2）

- [ ] 确认模拟支付或真实支付沙箱方案
- [ ] 评审支付、退款、发货、收货和售后状态机
- [ ] 评审表结构、公开 API、权限、审计和迁移影响
- [ ] 支付回调和退款请求具备幂等与签名校验
- [ ] 状态变更、金额和库存处理具备事务一致性
- [ ] 补齐服务、Handler、DAO 集成测试和前端操作流程
