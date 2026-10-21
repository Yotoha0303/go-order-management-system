# ADR-002: RabbitMQ 分级阶梯延时队列消除队头阻塞架构设计

- **Title**: ADR-002: RabbitMQ 分级阶梯延时队列消除队头阻塞架构设计
- **Status**: Accepted
- **Date**: 2026-10-21
- **Deciders**: 分布式系统架构师 (`system-architect`), SRE 稳定性专家 (`sre-operator`), 后端开发专家 (`backend-dev`)
- **Target System**: 订单超时取消与消息拓扑 (`go-order-management-system`)
- **Related GAPs**: GAP-004 (P2), GAP-008 (P1), GAP-003 (P2)
- **Supersedes/Extends**: 扩展 ADR-001 决策中的异步消息解耦链路

---

## 1. 上下文与问题陈述 (Context & Problem Statement)

### 1.1 当前实现现状
在现有的 `internal/ordertimeout/worker.go` 中，订单超时关单机制基于 RabbitMQ 的单消息 TTL 与 Dead Letter Exchange (DLX) 实现：
1. 声明单一延迟队列 `order.timeout.delay`，配置死信路由：
   ```go
   "x-dead-letter-exchange":    "order.timeout"
   "x-dead-letter-routing-key": "cancel"
   ```
2. 生产者（`publishEvent`）在发布每条超时消息时，动态计算并设置单消息属性：
   ```go
   Expiration: expirationMilliseconds(w.now(), event.TimeoutAt)
   ```

### 1.2 队头阻塞（Head-of-Line Blocking, HoLB）严重隐患 (GAP-004)
RabbitMQ 官方规范对单消息设置 TTL 的底层处理机制有明确限制：
> *"When setting per-message TTL, expired messages can queue up behind non-expired ones until the latter are consumed or expired. Hence proper dead-lettering of such messages would not occur until all preceding messages expire."*

**根本物理缺陷**：RabbitMQ 原生队列仅定期检查**队头（Head of Line）**消息是否到期。如果队头消息设置了较长的 TTL（如普通订单的 30 分钟），而后续进入队列的消息设置了较短的 TTL（如大促抢购订单的 5 分钟，或某些极速订单 10 分钟）：
- 尽管短 TTL 消息在物理时间上早已到达超时时刻（已过去 5 分钟），但由于排在队头后面的位置，RabbitMQ 根本不会扫描并移动它；
- 短消息被迫停滞整整 25 分钟，直到排在它前面的 30 分钟长消息到期出队，短消息才得以被检查和死信路由至 `order.timeout.cancel`；
- **业务灾难**：在秒杀与热点促销场景下，短超时订单无法及时取消并释放 Redis/MySQL 预占库存，导致库存锁定时间拉长数倍，有效库存在黄金销售期被恶意占用，严重损害高并发吞吐与业务可用性。

---

## 2. 方案比选与权衡裁定 (Decision & Options Comparison)

为了根除 GAP-004 队头阻塞隐患，针对三种主流演进方案进行全方位技术评审：

### 方案对比矩阵

| 评估维度 | Option A: `rabbitmq_delayed_message_exchange` 插件 | Option B: 分级固定 TTL 阶梯队列 (Ladder Queues) 🌟 **推荐** | Option C: 统一单队列固定 TTL + Outbox 轮询兜底 |
| :--- | :--- | :--- | :--- |
| **底层实现机制** | Exchange 级时间轮调度，内部基于 Mnesia 存储 | 队列级 `x-message-ttl` 原生特性，严格单队列 FIFO 到期 | 单个固定延时队列 + 数据库 Outbox 定时扫描 |
| **队头阻塞消除** | 完全消除（支持任意微秒级延迟） | **完全消除**（同一队列内 TTL 恒等，队头即最早到期） | 消除（但牺牲了多种订单超时等级支持） |
| **环境与依赖成本** | **极高**。官方 Alpine/标准镜像默认**无插件**；云厂商托管 MQ（如 AWS MQ / 阿里云）多不支持第三方插件 | **零依赖**。纯原生 RabbitMQ 4.1 标准能力，100% 开箱即用 | **零依赖**。依赖已有 Outbox 表 |
| **高并发与内存稳定性** | **较差**。Mnesia 数据库在高并发大量堆积时易出现磁盘 I/O 抖动与 OOM 风险，且不可分布式横向扩展 | **极优**。基于 Erlang 进程轻量级 Timer，吞吐量可达数万 QPS，纯顺序写入，性能极强 | **良好**。但高频轮询 DB 会增加数据库锁与连接压力 |
| **业务契合度** | 过于灵活（实际电商超时均为离散固定枚举） | **完美契合**。电商订单超时天然为离散阶梯（如 30m / 15m / 5m） | 较差。无法区分秒杀单与普通单的差异化超时 |

### 架构决策裁定
**正式决定采纳 Option B（分级固定 TTL 阶梯队列拓扑）为主干方案，辅以 Transactional Outbox 调度与防重机制（结合 GAP-003 与 GAP-008 治理）。明确拒绝 Option A。**

#### 决策理由（Trade-offs 权衡依据）：
1. **零外部假设与云原生可移植性**：
   本项目要求在 `compose.yml`（`rabbitmq:4.1-management-alpine`）、本地轻量环境及任何公有云原生环境中零摩擦运行。Option A 要求编译或挂载额外的 `.ez` 插件并执行启用命令，严重违反环境幂等与零假设原则；
2. **队头阻塞的数学破解**：
   队头阻塞发生的充要条件是“**同一物理队列内存在不同 TTL 的消息**”。只要建立若干个固定延时阶梯队列，并将队列属性设为 `x-message-ttl`（统一生效），则同一队列内所有消息的物理过期时间必然严格单调递增。**队头消息必然是最先到期的消息**，从数学上保证绝对不会阻塞队尾消息；
3. **性能与可靠性上限**：
   阶梯队列完全复用 RabbitMQ 高性能内核与 DLX 机制，没有插件版 Mnesia 锁表和内存崩溃的单点隐患。

---

## 3. 拓扑设计与协议规范 (Topology & Protocol Specification)

### 3.1 阶梯队列拓扑全景图

```
                          +-------------------------------------------------------+
                          |                 Exchange: order.timeout                |
                          |                 Type: direct, Durable                 |
                          +-------------------------------------------------------+
                                    |                 |                 |
         routing: "delay.30m"       |                 |                 |  routing: "delay.5m"
                                    |                 | routing:        |
                                    |                 | "delay.15m"     |
                                    v                 v                 v
            +---------------------------+ +---------------------------+ +---------------------------+
            | Queue:                    | | Queue:                    | | Queue:                    |
            | order.timeout.delay.30m   | | order.timeout.delay.15m   | | order.timeout.delay.5m    |
            | x-message-ttl: 1800000 ms | | x-message-ttl: 900000 ms  | | x-message-ttl: 300000 ms  |
            | (普通日常订单)            | | (标准大促订单)            | | (极速抢购/秒杀订单)       |
            +---------------------------+ +---------------------------+ +---------------------------+
                        |                             |                             |
                        +-----------------------------+-----------------------------+
                                                      |
                                                      | 死信触发 (DLX: order.timeout, Key: "cancel")
                                                      v
                                        +-------------------------------------------+
                                        | Queue: order.timeout.cancel               |
                                        | (统一消费队列: 执行幂等关单与释放库存)    |
                                        +-------------------------------------------+
                                                      | (消费失败达到重试上限)
                                                      v 死信转发 (DLX: order.timeout, Key: "failed")
                                        +-------------------------------------------+
                                        | Queue: order.timeout.failed (GAP-008治理) |
                                        | (死信告警/人工排障归档，消除黑洞)         |
                                        +-------------------------------------------+
```

### 3.2 队列与参数详细规范表

| 逻辑实体 | 物理命名 | 类型 / 路由键 | 关键配置参数 (`amqp.Table`) | 说明 |
| :--- | :--- | :--- | :--- | :--- |
| **Exchange** | `order.timeout` | `direct` | `durable: true` | 超时核心交换机（复用主路由） |
| **阶梯队列 1** | `order.timeout.delay.30m` | bind `delay.30m` | `x-message-ttl`: `1800000`<br>`x-dead-letter-exchange`: `order.timeout`<br>`x-dead-letter-routing-key`: `cancel` | 30分钟阶梯（普通订单） |
| **阶梯队列 2** | `order.timeout.delay.15m` | bind `delay.15m` | `x-message-ttl`: `900000`<br>`x-dead-letter-exchange`: `order.timeout`<br>`x-dead-letter-routing-key`: `cancel` | 15分钟阶梯（推荐基线） |
| **阶梯队列 3** | `order.timeout.delay.5m` | bind `delay.5m` | `x-message-ttl`: `300000`<br>`x-dead-letter-exchange`: `order.timeout`<br>`x-dead-letter-routing-key`: `cancel` | 5分钟阶梯（秒杀抢购） |
| **消费队列** | `order.timeout.cancel` | bind `cancel` | `x-dead-letter-exchange`: `order.timeout`<br>`x-dead-letter-routing-key`: `failed` | 统一取消消费者监听 |
| **失败队列** | `order.timeout.failed` | bind `failed` | `durable: true` | GAP-008 治理：死信报警监控 |

> **开发与测试环境动态映射**：
> 在本地测试或集成验证环境中，通过配置允许将阶梯映射为秒级（如 `delay.30s` -> 30000ms, `delay.15s` -> 15000ms, `delay.5s` -> 5000ms），以便在 CI 矩阵中无需真实等待数十分钟即可快速验证。

### 3.3 生产者路由判定协议
在 `publishEvent` 中，**严禁向消息体注入 `amqp.Publishing.Expiration` 单消息属性**。生产者根据订单计算的超时时长，路由到对应的阶梯 Routing Key：
```go
// 路由匹配伪代码
func resolveLadderRoutingKey(duration time.Duration) string {
    switch {
    case duration <= 5*time.Minute:
        return "delay.5m"
    case duration <= 15*time.Minute:
        return "delay.15m"
    default:
        return "delay.30m"
    }
}
```

---

## 4. 收益、代价与降级方案 (Consequences & Trade-offs)

### 4.1 核心收益
1. **彻底根除队头阻塞 (HoLB)**：短超时订单准时在自身时间窗口被消费，库存及时释放，杜绝秒杀活动中的库存长时间锁定；
2. **零插件、高可用**：开箱即用支持官方标准 RabbitMQ 镜像与任意云服务商，消除了环境依赖；
3. **架构正交性**：生产者无需关心复杂的延时机制，仅按业务阶梯投递；消费者依然只监听单一的 `order.timeout.cancel` 队列，消费端业务逻辑完全解耦无侵入。

### 4.2 代价与限制
- **延时颗粒度为离散阶梯**：不支持任意实数秒级的自由连续延时。在电商业务场景下，订单支付时限本身就是由产品或运营规则离散定义的（如 5m/15m/30m），因此完全满足业务诉求。

### 4.3 降级与兜底策略
- **Outbox 扫描保底**：当 RabbitMQ 发生不可抗力宕机或网络中断时，后台的 Transactional Outbox 扫描器（使用 `SELECT ... FOR UPDATE SKIP LOCKED`，GAP-003）将检测数据库中超期未取消的订单，执行兜底取消与库存回补，保证系统在消息队列不可用时依旧具备分布式最终一致性。

---

## 5. 验收标准 (Acceptance Criteria - Given-When-Then)

### 5.1 场景一：交错投递防队头阻塞验证 (GAP-004 核心验收)
```gherkin
Scenario: 先发送 30 秒长延时消息，后发送 5 秒短延时消息，验证短消息准时到期不受队头阻塞
  Given 系统配置了 30s 阶梯队列（delay.30s）与 5s 阶梯队列（delay.5s）
  And 统一消费队列 order.timeout.cancel 准备就绪
  When 在 t=0s 向 exchange 投递订单 A（RoutingKey="delay.30s"）
  And 紧接着在 t=1s 向 exchange 投递订单 B（RoutingKey="delay.5s"）
  Then 在 t=6s (1s + 5s) 左右，订单 B 准时被消费队列 order.timeout.cancel 接收并处理
  And 此时订单 A 依旧静默驻留在 30s 阶梯队列中，未对订单 B 产生任何阻塞
  And 直至 t=30s，订单 A 才被消费队列接收并处理
```

### 5.2 场景二：同阶梯批量投递严格 FIFO 顺序验证
```gherkin
Scenario: 同一阶梯队列内多条消息并发进入，严格按先进先出顺序到期
  Given 阶梯队列 delay.15s 配置完毕
  When 连续投递 10 条超时消息（Msg-1 至 Msg-10）进入 delay.15s
  Then 15 秒后，所有消息按原发送顺序依次死信进入 order.timeout.cancel
  And 无任何乱序与消息积压异常
```

### 5.3 场景三：坏消息死信至 failed 队列与告警就绪 (GAP-008 联动)
```gherkin
Scenario: 订单取消处理连续发生重试失败，最终落入 failed 队列
  Given 订单取消消费者正在运行
  When 消费端遇到不可恢复的异常（如数据库损坏）导致重试次数超过 defaultMaxCancelRetries (3次)
  Then 消息被 NACK 并不再入队，自动由 DLX 转发至 order.timeout.failed 队列
  And 产生相应的 Error 级别日志与指标告警
```
