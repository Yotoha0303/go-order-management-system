# 2026-07-11 Redis Cluster hash tag 执行日志

## 目标

继续执行 `docs/project_evolution.md` 中 Redis 进阶边界，处理多商品 Redis Lua 预扣在 Redis Cluster 下可能出现的多 key 跨 slot 风险。

## 本轮完成内容

1. 调整库存预扣 key：
   - 可售库存 key：`inventory:{stock}:available:{product_id}`；
   - reservation key：`inventory:{stock}:reservation:{order_id}`；
   - 两类 key 使用固定 hash tag `{stock}`，让 Lua 脚本涉及的库存 key 和 reservation key 落在同一 slot。

2. 同步补偿逻辑：
   - reservation 回补时使用新格式库存 key 前缀；
   - 预扣、确认、释放、重建、对账统一走 key 生成函数。

3. 升级影响说明：
   - MySQL 表结构不变，不需要数据库迁移；
   - Redis 旧格式 key 不会被新代码读取；
   - 部署后应调用管理员 Redis 可售库存重建接口，按 MySQL 当前库存生成新格式 key。

4. 同步文档：
   - 更新 README、需求边界、缓存设计、订单流程、项目演进、测试计划和测试结果。

## 验证结果

| 命令 | 结果 |
|---|---|
| `gofmt` | 已执行 |
| `go test ./internal/bizcache ./internal/service ./internal/app` | 通过 |
| `go test ./...` | 通过 |
| `cd fronted && npm run build` | 通过 |
| `cd fronted && npm test` | 通过，18 个测试文件、105 个测试用例 |

## 当前边界

- 固定 hash tag 会把库存预扣相关 key 放到同一 Redis Cluster slot，便于 Lua 多 key 原子操作；
- 这不等同于已经完成 Redis Cluster 实机部署验证；
- 真实热点库存削峰效果仍需要压测数据证明。
