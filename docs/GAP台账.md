# GAP台账 — 质量缺口与未竟事项唯一台账

> **单一事实源（D4）**：所有已知架构隐患、并发缺陷与待修复项**必须在此登记并分配唯一编号**。
> 不在台账里的“还没做”等于不存在。
> 关联 GitHub 仓库：[Yotoha0303/go-order-management-system](https://github.com/Yotoha0303/go-order-management-system)

---

## 质量缺口登记表

| 编号 | 严重度 | 模块 | 核心问题简述 | 对应 Issue | 状态 | 责任角色 |
| :--- | :---: | :--- | :--- | :---: | :---: | :--- |
| **GAP-001** | P1 | 订单/库存 | Redis Lua 原子预扣位于 MySQL 事务内部，未能前置拦截热点且引发长事务与双重扣减 | [#2](https://github.com/Yotoha0303/go-order-management-system/issues/2) | OPEN | `backend-dev` / `system-architect` |
| **GAP-002** | P1 | 订单/幂等 | 订单幂等校验在 MySQL RR 隔离级别下的并发同 Key 查询异常与失败回滚状态丢失 | [#3](https://github.com/Yotoha0303/go-order-management-system/issues/3) | OPEN | `backend-dev` / `data-engineer` |
| **GAP-003** | P2 | 延迟投递 | Transactional Outbox 轮询缺乏并发锁，多实例部署下消息重复发布 | [#4](https://github.com/Yotoha0303/go-order-management-system/issues/4) | OPEN | `sre-operator` / `backend-dev` |
| **GAP-004** | P2 | 队列拓扑 | RabbitMQ 单消息 TTL 存在队头阻塞（Head-of-Line Blocking）隐患 | [#5](https://github.com/Yotoha0303/go-order-management-system/issues/5) | OPEN | `sre-operator` / `system-architect` |
| **GAP-005** | P1 | 最终一致性 | 订单超时取消在异常重启或网络故障时导致 Redis 库存永久丢失回补 | [#6](https://github.com/Yotoha0303/go-order-management-system/issues/6) | OPEN | `backend-dev` / `qa-validator` |
| **GAP-006** | P2 | 数据模型 | 库存模型缺少锁定库存（`locked_quantity`）设计，且定时对账 Worker 缺乏自愈机制 | [#7](https://github.com/Yotoha0303/go-order-management-system/issues/7) | OPEN | `data-engineer` / `backend-dev` |

---

## 缺口细则与完成准则 (DoD)

### GAP-001 · Redis 预扣前置与解耦
- **现状**：`order_service.go` 的 `CreateOrder` 在开启 MySQL 事务且写入订单表后，才调用 Redis 预扣；紧接着又在同一事务内使用 `SELECT ... FOR UPDATE` 扣减 MySQL。
- **风险**：事务内网络 IO 拉长连接持有；热点请求即便库存不足也经历了 MySQL 事务开启与回滚。
- **DoD**：
  1. 将 Redis 预扣移至 MySQL 开启事务之前，使用全局预生成的业务编号（UUID / 分布式 ID）作为预扣标识；
  2. 若 Redis 预扣返回库存不足，直接响应客户端错误，绝对不触碰 MySQL；
  3. 完善扣减失败时的释放补偿链路。

### GAP-002 · 幂等校验在 RR 隔离级别下的并发防御
- **现状**：`TryCreateOrderIdempotencyKey` 在 MySQL 事务中使用 `ON CONFLICT DO NOTHING`，并发同 Key 时后到达的线程查询记录，但在 RR 隔离级别下读不到未提交快照，报 `RecordNotFound`。
- **风险**：客户端收到内部服务器错误，而非预期的 `ErrOrderBeingCreated`。
- **DoD**：
  1. 幂等 Key 获取解耦或优化查询方式（采用当前读或前置分布式锁）；
  2. 补齐并发同 Key 的自动化测试用例，断言并发下稳定返回“订单处理中”。

### GAP-003 · Transactional Outbox 多实例防重发布
- **现状**：`ListPendingOrderTimeoutOutbox` 使用普通 `SELECT` 捞取待发布记录，无锁保护。
- **风险**：多 Pod / 多副本部署时同时捞取并重复发布超时消息。
- **DoD**：
  1. 查询 SQL 增加 `FOR UPDATE SKIP LOCKED`，或引入轻量分布式锁；
  2. 增加指数退避重试与最大重试次数归档。

### GAP-004 · 延迟消息队头阻塞防范
- **现状**：采用单消息设置 `Expiration` 属性投递至延迟队列。
- **风险**：RabbitMQ 仅检查队头消息 TTL，长消息阻塞后续短消息。
- **DoD**：
  1. 引入延迟消息插件（`rabbitmq-delayed-message-exchange`）或固定延迟阶梯队列设计；
  2. 验证多批次交错超时的消息能否准时进入死信取消链路。

### GAP-005 · 订单超时取消最终一致性加固
- **现状**：`CancelExpiredOrder` 在 MySQL 事务提交后调用 `restorePreDeductedInventory`。若此时服务崩溃，重试消费时因订单状态为 Cancelled 导致直接退出，Redis 库存永远少加。
- **风险**：分布式系统崩溃恢复时产生永久性可用库存泄漏。
- **DoD**：
  1. 消费端逻辑优化：即便订单已是 `Cancelled`，依然校验并清理该订单关联的 Redis reservation，确保回补幂等执行；
  2. 编写模拟事务提交后注入故障的异常测试用例。

### GAP-006 · 健全库存数据模型与对账自愈
- **现状**：`product_inventories` 仅有 `stock_quantity`，无 `locked_quantity`；取消回补按 `id ASC` 加锁而非 `product_id ASC`；对账 Worker 仅 Warn 日志。
- **风险**：库存状态语义不清；取消订单存在边缘死锁风险；对账无法自愈。
- **DoD**：
  1. 数据表增加 `locked_quantity` 字段并设计状态迁移；
  2. 取消订单回补库存显式按 `product_id ASC` 加锁；
  3. 对账 Worker 增加可配置的自动修正或告警推送。
