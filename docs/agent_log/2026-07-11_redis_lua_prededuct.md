# 2026-07-11 Redis Lua 库存预扣执行日志

## 目标

继续执行 `docs/project_evolution.md` 中“Redis 进阶使用”和评估重点“库存扣减加一层 Redis Lua 做预扣”。

## 本轮完成内容

1. 新增 Redis 库存预扣 guard：
   - 新增 `InventoryStockGuard`；
   - 新增 `inventory:available:{product_id}` 可售库存 key；
   - 新增 `inventory:reservation:{order_id}` 预扣 reservation key；
   - 使用 Lua 原子检查多商品库存并批量 `DECRBY`。

2. 接入订单创建：
   - 只有幂等首次创建订单时执行 Redis 预扣；
   - 幂等重放不重复预扣；
   - Redis key 缺失、Redis nil 或 Redis 异常时降级走 MySQL；
   - Redis 判断库存不足时提前返回库存不足；
   - MySQL 事务失败时按 reservation 回补 Redis。

3. 接入订单状态流转：
   - 支付成功后删除 reservation，不回补 Redis；
   - 主动取消和超时取消成功后按 reservation 回补 Redis；
   - reservation 不存在时不凭空创建 Redis 库存 key。

4. 接入库存管理：
   - 初始化库存成功后同步 Redis 可售库存；
   - 手动入库成功后同步 Redis 可售库存为 MySQL 事务提交后的库存。

5. 补充测试：
   - Redis nil 降级；
   - key 命名；
   - Lua 预扣成功、库存不足、key 缺失跳过；
   - 事务失败释放 reservation；
   - 支付确认清理 reservation；
   - 取消订单回补 reservation；
   - 库存初始化和手动入库同步 Redis。

6. 同步文档：
   - 更新 Redis 设计、订单流程、业务规则、需求边界、README、面试讲解、测试计划、测试结果和项目演进文档。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./internal/service ./internal/bizcache ./internal/app` | 通过 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |

## 仍未完成的后续进阶边界

- Redis/MySQL 库存对账和重建任务；
- Redis 预扣相关指标；
- 线上部署和长期稳定性运行证据；
- 真实压测、慢 SQL 分析和性能报告；
- 完整支付、退款、发货、售后等更复杂订单链路。

## 决策说明

Redis 预扣只作为软保护层，不改变 MySQL 作为库存事实源的设计。这样可以展示 Redis Lua 在热点库存场景下的削峰能力，同时保留 MySQL 行锁和条件扣减作为最终一致性保障。
