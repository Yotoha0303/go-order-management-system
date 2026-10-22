# docs/adr/ — 架构决策记录 (Architecture Decision Records)

> **决策锚（AGENTS.md §0.2）**：回答**「为什么这样定」**。
> 所有突破既有模式的架构与设计变更，必须在此落盘。

---

## 模板要求

每个 ADR 文档建议遵循以下结构：
- **Title**: `ADR-NNN: 决策标题`
- **Status**: `Proposed` | `Accepted` | `Deprecated` | `Superseded`
- **Context**: 遇到了什么问题，当前的瓶颈与约束是什么？
- **Decision**: 做出了什么架构决策，技术选型是什么？
- **Consequences**: 该决策带来的收益、代价与权衡（Trade-offs）。

---

## 决策记录索引

| 编号 | 标题 | 状态 | 关联 GAP | 责任角色 |
| :--- | :--- | :---: | :--- | :--- |
| [ADR-001](001-high-concurrency-distributed-consistency-inventory-architecture.md) | 高并发低延迟与分布式最终一致性库存架构设计 | `Accepted` | GAP-001, GAP-002, GAP-005, GAP-007, **[GAP-032](../GAP台账.md)（决策三 TTL / 决策四未落地）** | `system-architect` |
| [ADR-002](002-rabbitmq-ladder-delay-queue-topology.md) | RabbitMQ 分级阶梯延时队列消除队头阻塞架构设计 | `Accepted` | GAP-004, GAP-008, GAP-003 | `system-architect` |
| [ADR-003](003-inventory-locked-quantity-and-reconcile-autofix.md) | 健全库存三态数据模型、全局防死锁偏序加锁与缓存对账自愈架构设计 | `Accepted` | GAP-006 | `data-engineer` |


