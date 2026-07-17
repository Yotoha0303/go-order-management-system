# 项目需求边界

## 已实现

- JWT 注册、登录、个人资料和密码修改；
- 最小 RBAC，限制商品、库存、库存流水和操作日志管理能力；
- 商品创建、分页查询、详情、上下架；
- 库存初始化、增加、查询和完整库存流水；
- 用户级幂等创建订单、订单详情和分页列表；
- 支付、完成、主动取消与库存回补；
- RabbitMQ + Outbox 实现待支付订单超时取消；
- Redis 商品详情 cache-aside 与故障降级；
- Redis Lua 库存预扣软保护；
- Redis 库存预扣 key 固定 hash tag，降低 Cluster 跨 slot 风险；
- Redis/MySQL 库存差异报告；
- Redis/MySQL 定时自动对账日志；
- 管理员触发 Redis 可售库存重建；
- 管理员后台操作审计；
- Prometheus 文本格式 HTTP metrics 和订单/Redis 业务指标；
- W3C `traceparent` 透传和 access log 链路关联字段；
- GORM 慢 SQL 日志入口，支持配置慢查询阈值和日志等级；
- HTTP 压测命令、本地 Docker 读链路压测报告和 SQL 执行计划分析；
- 多商品订单按 product_id 升序处理，降低库存行锁顺序不一致导致的死锁风险；
- React 管理台、Docker Compose、Goose migration 和自动化测试。

## 明确边界

- 不包含真实支付网关、退款、发货和售后；
- 不提供公开角色管理 API；
- Redis 参与库存预扣，但库存事实源仍为 MySQL；
- 本地压测结果只作为同机回归基线，不承诺生产吞吐量或高可用；
- 线上告警落地证据、OpenTelemetry/追踪后端、订单写链路压测和长期稳定性证据仍是后续增强项。

## 后续跟进（未完成）

| 优先级 | 状态 | 内容 | 当前边界 |
|---|---|---|---|
| P0 | 未完成 | 订单写链路和热点库存并发压测 | 当前只有健康接口与商品列表的本地读链路基线 |
| P1 | 未完成 | OpenTelemetry 和追踪后端 | 当前只有 `traceparent` 透传、request_id/trace_id 日志关联 |
| P1 | 未完成 | 云主机部署和长期稳定性证据 | 当前只有 Docker/Compose/健康检查等部署前置能力 |
| P2 | 未完成 | 支付网关、退款、发货和售后 | 当前“支付/完成”仅为本地订单状态流转，不代表真实支付和履约 |

详细完成判定与前置条件见 [project_evolution.md](project_evolution.md#第十阶段后续进阶边界)。

接口范围见 [api_list.md](api_list.md)，业务规则见 [business_rules.md](business_rules.md)。
