# 变更记录：014-fix-gap-015-stock-log-biz-id-precision

- **变更编号**：014
- **创建时间**：2026-10-22T05:03:42Z (UTC)
- **完成时间**：2026-10-22T05:03:42Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-015（**待 `ssot-scribe` 登记**，本角色不改 `docs/GAP台账.md`）；P2，GAP-012 同类缺陷的收口
- **关联 Issue**：#14
- **关联 ADR**：无（JSON 序列化契约修复，不改变架构决策）
- **状态**：代码与测试完成；**该项自身不再有红点**——原本被
  `internal/model/qa_json_surface_test.go:102` 阻断的问题，已由 `qa-validator` 翻转断言解除，
  实测见 §7.7（`--- PASS: TestQAStockLogBizIDShapePinsTheOpenContract`）。
  ⚠️ 但这**不等于**「CI 已绿」：PR #17 的 `Test, Vet, Validate, Build` 仍 `fail`，
  失败点为 `TestRebuildRedisInventoryStock_Success`（与本变更无关），系统性归因见台账 `GAP-020`。

---

## 1. 变更动机与背景

GAP-012 把 `Order.ID` 与 `OrderItem.OrderID` 字符串化时，**同类载体的清单只扫了订单侧**。
`StockLog.BizID` 是第三个承载同一个 63 位 Snowflake 的字段，且**直接暴露在管理端**：

```text
router/router.go:138   admin.GET("/stock-logs", stockLogHandler.ListStockLogs)   // AdminMiddleware 下
internal/model/stock_log.go:20   BizID *int64 ... json:"biz_id"
```

前端 `stock-logs.tsx:120` 渲染 `` {log.biz_id ? `#${log.biz_id}` : '-'} ``，而 `types.ts:52` 当时声明为
`biz_id?: number | null` ⇒ 管理员看到的是**被 IEEE-754 截断后的订单号**（末位 -1 或更多）。
与前两处同族：**不会报错，只显示一个错的号**（D3「失败即静默」）。

## 2. 根因（实测定位）

| 位置 | 实测内容 |
| --- | --- |
| `internal/model/stock_log.go:20` | `BizID *int64 \`gorm:"type:bigint;index:idx_stock_logs_biz_id" json:"biz_id"\`` |
| `internal/service/order_service.go:188` | 下单扣减：`BizID: &order.ID`（`StockBizOrderDeduct`） |
| `internal/service/order_service.go:557` | 取消回滚：`BizID: &order.ID`（`StockBizOrderRollback`） |
| `router/router.go:138` | 管理端路由，响应经 `response.Success(c, stockLogs)` 直接内嵌 `[]*model.StockLog` |
| `fronted/.../types.ts:52` | `biz_id?: number | null`（与运行时一致，故**声明本身没错，错的是后端契约**） |

## 3. `BizID` 全部写入点判断清单（要求项 ②）

`git grep -n "StockLog{"` 与 `grep BizID` 的全仓并集如下（**每一处都读了源码，不是按名字推断**）：

| # | 写入点 | `BizType` | `BizID` 赋值 | 值域 | 是否 Snowflake | 前端是否消费 | 处置 |
| - | --- | --- | --- | --- | --- | --- | --- |
| 1 | `internal/service/order_service.go:182-190`（`CreateOrder`） | `StockBizOrderDeduct (3)` | `&order.ID` | 63 位 Snowflake | **是** | 是（`#${biz_id}`） | **本次修复的目标** |
| 2 | `internal/service/order_service.go:555-563`（取消链路） | `StockBizOrderRollback (4)` | `&order.ID` | 同上 | **是** | 是 | 同批修复（与 #1 是同一个值） |
| 3 | `internal/service/inventory_service.go:53-60`（`InitInventory`） | `StockBizInit (1)` | **未赋值**（`nil`） | 无 | 否 | 渲染为 `-` | **不动**；测试钉住 `"biz_id":null` |
| 4 | `internal/service/inventory_service.go:116-123`（`AddInventory`） | `StockBizManualAdd (2)` | **未赋值**（`nil`） | 无 | 否 | 渲染为 `-` | **不动**；同上 |
| 5 | `internal/service/stock_log_service.go:9-11`（`CreateStockLog`） | — | 透传调用方传入的 `*model.StockLog` | 取决于调用方 | — | — | 非独立写入点，仅转发 |

**结论**：`BizID` 的取值域是 **{nil, 订单 Snowflake}**——**不存在自增 id 的写入点**，
因此不存在「自增 id 被字符串化后前端依赖数字」的破坏风险。
唯一需要判断的点是 **nil 分支**（`StockBizInit` / `StockBizManualAdd`）：`,string` 对 `*int64` 的 nil
必须仍是 `null` 而不是 `"0"`，已用用例钉死（见 §6.1 第 2 条）。

> 若将来出现「`biz_id` 承载自增 id」的新 `BizType`，**不能沿用 `,string`**——
> 那时应改为按 `biz_type` 分流，或新增独立字段。本次不预留空分支（避免无人维护的开关）。

## 4. 变更实施清单

1. `internal/model/stock_log.go`：`BizID` 的 JSON tag 改为 `json:"biz_id,string"`，并加注释说明
   值域（nil / 订单 Snowflake）与 nil 行为。**GORM 列类型、索引、可空性一律不变**——无迁移。
2. `fronted/src/features/order-inventory/types.ts`：`biz_id?: number | null` → `biz_id?: string | null`，
   并补注释（与 `Order.id` 的注释同口径）。`stock-logs.tsx:120` 的渲染模板**不需要改**（模板字符串对 string 天然成立）。
3. `internal/handler/stock_log_handler_test.go`（**新增**）：见 §5 的落点理由。
4. `.github/workflows/ci.yml`：另两项独立修复（本记录 §8 摘要，详见各自小节）。

## 5. 测试落点选择：`internal/handler/`（而非 `internal/model/`）

**理由（二选一，给出判据）**：缺陷的本体是**一个 HTTP 端点的线上字节**，不是单个结构体的 `json.Marshal` 行为。

| 维度 | `internal/model/`（未选） | `internal/handler/`（选中） |
| --- | --- | --- |
| 能证明什么 | 只有 `json.Marshal(StockLog)` 的输出 | 加上 Gin `response.Success` 包封、路由挂载、`product_id` 查询参数解析 |
| 能否发现「响应被别的层改形」 | 不能 | 能 |
| 是否依赖 DB | 否 | 否（`stubStockLogService`，模式同 `order_handler_test.go` 的 `stubOrderService`） |
| 与缺陷报告的一致性 | 间接 | 直接对应「`GET /api/v1/stock-logs` 返回什么字节」 |

断言全部打在 **`recorder.Body.Bytes()` 原始字节**上：`"biz_id":"<19位>"` 必须存在，
`"biz_id":<19位数字>` 必须不存在。解码成 `int64` 再比较**不作为证据**——它能同时接受被截断的值。

## 6. 实测验证日志（D1：真实 stdout）

采集时间：2026-10-22T05:00Z 前后（UTC），工作树 `fix/issue-14-order-module`。

### 6.1 新增用例（`internal/handler`）

```text
=== RUN   TestStockLogBizIDSerializedAsString
=== RUN   TestStockLogBizIDSerializedAsString/order-backed_biz_id_is_a_quoted_19-digit_string
    stock_log_handler_test.go:74: raw response bytes: {"code":0,"message":"success","data":[{"id":11,"product_id":7,"change_quantity":-2,"before_quantity":0,"after_quantity":0,"biz_type":3,"biz_id":"1058897343283204097","remark":"","created_at":"0001-01-01T00:00:00Z"},{"id":12,"product_id":7,"change_quantity":2,"before_quantity":0,"after_quantity":0,"biz_type":4,"biz_id":"1058897343283204097","remark":"","created_at":"0001-01-01T00:00:00Z"}]}
=== RUN   TestStockLogBizIDSerializedAsString/nil_biz_id_still_marshals_as_null
=== RUN   TestStockLogBizIDSerializedAsString/product_id_filter_is_forwarded_unchanged
=== RUN   TestStockLogBizIDPrecisionNegativeControl
    stock_log_handler_test.go:129: stock log biz_id: original=1058897343283204097 javascript-round-tripped=1058897343283204096 delta=-1
    stock_log_handler_test.go:143: bare-number payload: original=1058897343283204097 javascript-parsed=1058897343283204096 delta=-1
--- PASS: TestStockLogBizIDSerializedAsString (0.00s)
    --- PASS: .../order-backed_biz_id_is_a_quoted_19-digit_string (0.00s)
    --- PASS: .../nil_biz_id_still_marshals_as_null (0.00s)
    --- PASS: .../product_id_filter_is_forwarded_unchanged (0.00s)
--- PASS: TestStockLogBizIDPrecisionNegativeControl (0.00s)
PASS
ok  	go-order-management-system/internal/handler	3.258s
```

**要求项 ② 的直接证据（`,string` 对 `*int64` 确实生效、且 nil 保持 null）**：
上表 §6.1 第 1 组的原始字节里 `"biz_id":"1058897343283204097"` 带引号，
而同一响应里 `"product_id":7` / `"id":11` 仍是裸数字——**同一个结构体、同一次 `json.Marshal`**，
说明 `,string` 只作用于 `BizID`，不是全局行为。第 2 组用 `StockBizManualAdd`（`BizID == nil`）
断言 `"biz_id":null` 且**不含** `"biz_id":"0"`。

### 6.2 负向证伪（D3 实际输出）

**变异：把 `BizID` 的 `json:"biz_id,string"` 改回 `json:"biz_id"`**

```text
=== MUTATION: StockLog.BizID json tag reverted to bare number ===
--- FAIL: TestStockLogBizIDSerializedAsString (0.00s)
    --- FAIL: TestStockLogBizIDSerializedAsString/order-backed_biz_id_is_a_quoted_19-digit_string (0.00s)
        stock_log_handler_test.go:75: response must contain "biz_id":"1058897343283204097" (quoted string), got: {...,"biz_type":3,"biz_id":1058897343283204097,...}
FAIL
FAIL	go-order-management-system/internal/handler	2.908s
exit=1
```

变异已还原并复跑通过。**常驻负向对照**（`TestStockLogBizIDPrecisionNegativeControl`）不依赖变异：
它每次都打印 `original=1058897343283204097` vs `javascript-parsed=1058897343283204096 delta=-1`，
若某天二者相等则用例主动 `t.Fatalf`（负向对照失效会自曝）。

## 7. 门禁实测（要求项 ④）与**未做项**

| # | 命令 | exit | 实测结论 |
| :- | :-- | :--: | :-- |
| G1 | `go vet ./...` | **0** | 无输出 |
| G2 | `go test -count=1 ./...` | **1** | **除 `internal/model` 外全绿**；唯一失败见下（**该红点已于 §7.7 解除**） |
| G3 | `go test -count=1 -race ./...` | **1** | 同上，**无 DATA RACE** |
| G4 | `golangci-lint run ./...` | **0** | `0 issues.` |
| G5 | `goose -dir migrations validate` | **0** | 无输出 |
| F1 | `npx tsc -b` | **0** | 无诊断（**订正**：原记 `npx tsc --noEmit` 为空转命令，见 §7.2） |
| F2 | `npx eslint .` | **0** | 0 problems |
| F3 | `npx vitest run --browser.headless`（临时 config，见下） | **0** | `Test Files 22 passed (22)` / `Tests 137 passed (137)` |
| F4 | `npm run build` | **0** | `✓ built in 1.31s` |

```text
--- FAIL: TestQAStockLogBizIDShapePinsTheOpenContract (0.00s)
    qa_json_surface_test.go:102: StockLog.biz_id wire shape changed to string; frontend type says number
FAIL	go-order-management-system/internal/model
```

### 7.2 订正：F1 原读数不携带信息（`tsc --noEmit` 在本仓空转）

原 F1 记「`npx tsc --noEmit` exit 0 → 无诊断」。该读数**不能作为门禁证据**：
`fronted/tsconfig.json` 是 solution 风格（`"files": []` + `references`），按根配置执行时 `tsc`
**检查 0 个文件**，因此永远 exit 0，与代码对错无关。

实测对照（本机 `tsc 6.0.3` / `node v22.23.2`，`fronted/` 下）：

```text
npx tsc --noEmit --listFilesOnly             → 0 files,   exit 0
npx tsc -p tsconfig.app.json --listFilesOnly → 893 files, exit 0
npx tsc -b --force                           → exit 0（按 references 真编译）
```

注入探针证伪（临时文件 `src/__tsc_gate_probe.ts`，内容 `export const tscGateProbe: number = 'str'`，测后已删除）：

```text
[A] npx tsc --noEmit   → A exit=0        （完全没有报出这个错误）
[B] npx tsc -b         → src/__tsc_gate_probe.ts(3,14): error TS2322: Type 'string' is not assignable to type 'number'.
                         B exit=2
```

删除探针后 `npx tsc -b` 复跑 exit 0。故 F1 **改为以 `npx tsc -b` 为准**（CI 的 Type check 步骤已同步改为该命令）。

### 7.6 该失败是**预期的、且必须由测试方处置**（未擅自修改）

`internal/model/qa_json_surface_test.go` 是 `qa-validator` 在 03 环新增的**独立测试产物**，
其函数名与注释都写明它**钉住的是修复前的形态**：

```go
// TestQAStockLogBizIDShapePinsTheOpenContract documents a *contract mismatch* ...
// This test pins the current behaviour so the mismatch cannot drift silently.
if shape["biz_id"] != "non-string" {
    t.Fatalf("StockLog.biz_id wire shape changed to %s; frontend type says number", shape["biz_id"])
}
```

它在修复**之前**是绿的（**实测**：改动前 `go test -count=1 ./internal/model/...` → `ok ... 0.613s`），
修复后必然红。**这不是回归，是「钉住缺陷」的用例随缺陷一起到期**。

按派发边界「不要改 `qa-validator` 新增的 `qa_*.go` / `qa-*.test.*`，若需要更新则报告」，
**本角色未改动它**。**解除条件**：由 `qa-validator` 把断言翻转为
`shape["biz_id"] == "string"`（并保留 nil 分支的 `null` 断言），或由总控裁定删除。
**在那之前 `go test ./...` / `-race` 不可能全绿**——这是三项修复之外唯一的红点。

> **后续（`2026-10-22`）**：上述阻塞**已解除**——`qa-validator` 已翻转该断言，复跑 PASS，见 §7.7。
> §7.6 保留原文是因为它记录的是**当时**的事实（历史快照不改写），不是当前状态。

### 7.2 本机 vitest 的临时 config（已删除）

`fronted/vitest.local.config.ts` 在**接手时已存在且未被 git 跟踪**（`git status` 显示 `??`），
非本角色创建。本次 vitest 沿用并随后**删除**（`Test-Path` 复核 = `False`）。
它只覆盖 `test.browser.api.port=8899`，因为本机 Windows 保留端口段
（`netsh int ipv4 show excludedportrange protocol=tcp` 实测 `63221-63320` 与 `63321-63420`）
覆盖了 vitest 默认的 63315。**CI（ubuntu）无此保留段，故 ci.yml 不引用该文件。**

### 7.7 后续：断言已由 `qa-validator` 翻转（§7.6 的门禁红点已解除）

§7.6 记录的是**当时**的阻塞：钉住用例断言 `shape["biz_id"] == "non-string"`，修复后必然红。
`qa-validator` **随后已翻转** `internal/model/qa_json_surface_test.go`，现断言：

- `shape["biz_id"] == "string"`（订单 Snowflake 必须为 JSON 字符串）；
- nil 分支必须序列化为 `"biz_id":null`（**并**用 `strings.Contains(rawNil, "\"biz_id\":null")` 做**原始字节**断言，不以 `interface{}` 比较代替）。

**复跑实测（本机，`2026-10-22`）**：

```text
go test -count=1 ./internal/model/ -run TestQAStockLogBizIDShapePinsTheOpenContract -v
=== RUN   TestQAStockLogBizIDShapePinsTheOpenContract
    qa_json_surface_test.go:120: StockLog.biz_id is string for a set id, and null when unset (order Snowflake rendered in the admin stock-log table)
--- PASS: TestQAStockLogBizIDShapePinsTheOpenContract (0.00s)
PASS
ok  	go-order-management-system/internal/model	0.689s
```

**结论**：§7.6 所说的「`go test ./...` / `-race` 不可能全绿」**已不再成立**——该红点已消失。
但其**性质**不变：本节只证明「该用例不再阻塞」，**不等于** `GAP-015` 可判 `CLOSED`——
按台账状态枚举，`CLOSED` 还需**门禁裁决**放行，而本轮 `gate-auditor` 裁决为阻断。
仍有效的若干净证据：§6.1（新用例全 PASS）、§6.2（负向证伪）、§7.2（`tsc --noEmit` 空转订正）。

### 7.8 后续：PR #17 的 CI 实跑结果（首次端到端执行本批改动）

PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17) 的 CI run 为 `39729484909`
（`event=pull_request`，`headSha=3584d1f`，与本批提交的 HEAD 一致），结果：

| 作业 | 结论 | 耗时 |
| --- | --- | --- |
| `Frontend (tsc, eslint, vitest browser, build)` | **pass** | 1m0s |
| `Test, Vet, Validate, Build` | **fail** | 2m59s |

**Go 作业的逐包结果（实测 `gh run view 39729484909 --log-failed`）**：18 个包中 **17 个 `ok`**，
仅 `FAIL go-order-management-system/internal/service`。失败点：

```text
--- FAIL: TestRebuildRedisInventoryStock_Success (0.11s)
    inventory_service_test.go:203: expected inventories ordered by product_id asc, got [0xc0003accd0 0xc0003acd20]
```

**三点必须说清（否则会被误读成本变更的回归）**：

1. **`internal/model` 为 `ok`**——即本变更与 `qa-validator` 的钉住用例**在 CI 上也已通过**，不只是本机；
2. **失败测试与本变更无关**：`TestRebuildRedisInventoryStock_Success` 属库存重建排序断言，本变更只动了 `StockLog.BizID` 的 JSON tag；
3. **`GOFLAGS=-mod=readonly` 不拦新增测试文件**：该标志约束的是 `go.mod` / `go.sum`，与是否有未跟踪的 `_test.go` 无关——故「新用例在 CI 里被静默忽略」这一担心**在本次不成立**（`internal/model` 的 `ok` 即为证据）。

**结论**：本变更在**本机与 CI 两侧**均无红点；但**整条 CI 仍未绿**，故本项**不判 `CLOSED`**，
系统性归因见台账 `GAP-020`（`main` 连红 15 次，`CLOSED` 结论从未经 CI 验证）。

## 8. 同批的两项 CI 修复（P1）

### 8.1 `MYSQL_TEST_DATABASE` 含连字符 ⇒ CI 必然红

- **现状**：`.github/workflows/ci.yml:22` 原为 `go-order-management-system_test`；
  `internal/service/test_helper_test.go:95` 的 `safeDatabaseName = ^[a-zA-Z0-9_]+$` 在 `:101-103` 不匹配即 `t.Fatalf`。
- **改法**：改为下划线形态 `go_order_management_system_test`，与 `.env.example:50` 的约定一致。
  **未放宽正则**——那是把外部环境变量插进 `CREATE DATABASE` 的安全围栏，不是障碍。
- **实测（正向，修复后）**：

  ```text
  RUN_MYSQL_TEST=1 MYSQL_TEST_DATABASE='go_order_management_system_test' \
    go test -count=1 -run TestCreateOrder_Success ./internal/service/...
  --- FAIL: TestCreateOrder_Success (0.02s)
      order_service_test.go:433: create test database failed: Error 1045 (28000):
      Access denied for user 'root'@'localhost' (using password: YES)
  exit=1
  ```

  **如实报告**：本机**没有** CI 那套 MySQL。`ensureTestDatabase` 用
  `cfg.MySQL.User`（`config.yml` = `root`）+ `MYSQL_TEST_PASSWORD` 连 `127.0.0.1:3306`，
  本机该口令不对 ⇒ 1045。**关键点是失败位置前移了**：不再是 `invalid MYSQL_TEST_DATABASE`，
  而是真的走到了连接阶段。
- **实测（负向对照，修复前）**：

  ```text
  RUN_MYSQL_TEST=1 MYSQL_TEST_DATABASE='go-order-management-system_test' ...
  --- FAIL: TestCreateOrder_Success (0.00s)
      order_service_test.go:433: invalid MYSQL_TEST_DATABASE "go-order-management-system_test"
  exit=1
  ```

  两条命令**只差环境变量的形态**，失败原因从「库名非法」变为「连不上库」——
  证明修复点精确命中，且不是靠放宽围栏换来的。
- **CI 上为何会通过**：`ci.yml` 的 `mysql` service（`mysql:8.4`）设了
  `MYSQL_ROOT_PASSWORD: ci-password`，与 `env.MYSQL_TEST_PASSWORD: ci-password` 一致；
  `test_helper_test.go` 的 `ensureTestDatabase` 用该口令自行 `CREATE DATABASE IF NOT EXISTS`，
  故测试库无需预先存在。库名现在是合法标识符 ⇒ 围栏不再拦截。

### 8.2 CI 无前端门禁

- **现状（实测）**：原 `ci.yml` 对 `node|npm|fronted|vite` **零命中**；GAP-013 的全部前端用例
  （含 `qa-validator` 新增的 18 条）在 PR 上不执行，违反 03 环「用例 100% 纳入 CI」。
- **改法**：新增**独立 job `frontend`**（不并入 `quality`，避免每次 Go 迭代都等 `npm ci` + Playwright），
  `defaults.run.working-directory: fronted`，步骤：

  ```text
  npm ci
  npx playwright install --with-deps chromium     # vitest 是 browser mode，先装 Chromium
  npx tsc -b                                      # 不是 --noEmit，见 §7.2
  npx eslint .
  npx vitest run --browser.headless
  npm run build
  ```

- **Node 版本推断依据（锁文件 `engines`，非猜测）**：

  | 包 | 版本 | engines.node |
  | --- | --- | --- |
  | `vite` | 8.0.16 | `^20.19.0 \|\| >=22.12.0` |
  | `eslint` | 10.4.1 | `^20.19.0 \|\| ^22.13.0 \|\| >=24` |
  | `vitest` | 4.1.8 | `^20.0.0 \|\| ^22.0.0 \|\| >=24.0.0` |

  交集里的大版本只有 **22** ⇒ `actions/setup-node@v4` 用 `node-version: '22'`，
  `cache: npm` + `cache-dependency-path: fronted/package-lock.json`（锁文件存在，实测 250291 字节）。
- **未加 `vitest.local.config.ts`**：那是本机 Windows 保留端口的绕过，CI 无此问题（见 §7.2）。
- **YAML 合法性实测**：

  ```text
  yaml OK
  jobs: ['quality', 'frontend']
  env.MYSQL_TEST_DATABASE = go_order_management_system_test
  frontend steps: ['Checkout code', 'Setup Node', 'Install dependencies', 'Install Playwright Chromium', 'Type check', 'Lint', 'Unit and browser tests', 'Build']
  quality steps: 13
  ```

  ⚠️ **未做项**：本机无 `act`/Docker-in-CI 条件，**`frontend` job 无法在本机端到端跑一遍**。
  已做的等价验证是 §7 的 F1–F4 逐条在本机执行（Node v22.23.2，与 job 同大版本）。
  首次 PR 推送时请以 Actions 的实际日志为准。

## 9. 下游同步清单与未竟项

| 项 | 归属 | 说明 |
| --- | --- | --- |
| `docs/GAP台账.md` 登记 **GAP-015** | `ssot-scribe` | P2：`StockLog.BizID` 裸数字精度截断（GAP-012 同类收口）。本角色**未改台账** |
| `qa_json_surface_test.go:97-105` 断言翻转 | `qa-validator` | **已解除**：断言已翻转为 `string` + nil→`null`，实测 PASS，见 §7.7。不得由实现方代改 |
| `docs/changes/011` 的字段影响表 | 已完成记录的**作者** | 011 §4 的影响面表只列了 `Order.ID` / `OrderItem.OrderID`，未列 `StockLog.BizID`；011 记录的结论本身没错（当时范围如此），但读者可能误以为「Snowflake 载体只有两处」。建议由 011 的作者补一句指针，**本角色不改 011** |
| 前端 `stock-logs.tsx:120` | 无需改动 | 模板字符串对 `string` 天然成立；`types.ts` 类型收紧后 `npx tsc -b` 实测 exit 0（原记的 `--noEmit` 为空转，已订正见 §7.2） |
| 数据库迁移 | 无 | 仅 JSON tag 变化，列类型 / 索引 / 可空性不变 |
