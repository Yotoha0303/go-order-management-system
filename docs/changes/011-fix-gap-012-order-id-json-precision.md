# 变更记录：011-fix-gap-012-order-id-json-precision

- **变更编号**：011
- **创建时间**：2026-10-22T00:00:00Z (UTC)
- **完成时间**：2026-10-22T04:35:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-012 (P1: 订单主键大整数 JSON 精度截断，Issue #14)
- **关联 ADR**：无（序列化契约修复，不改变架构决策）
- **状态**：已完成（`Order.ID` 与 `OrderItem.OrderID` 双双字符串化；门禁 5/5 通过，负向证伪通过）

---

## 1. 变更动机与背景

订单主键由 `pkg/idgen` 的 Snowflake 生成（41 位毫秒时间戳 + 10 位 worker + 12 位序列，共 63 位有效位）。
`internal/model/orders.go` 以 `json:"id"` 原生数字输出，超出 IEEE-754 双精度安全整数上限
`Number.MAX_SAFE_INTEGER = 9007199254740991`（2^53-1）。

前端 `JSON.parse` 会把该数字截断到可表示的双精度值，再用截断后的 ID 拼装
`GET /orders/:id`、`POST /orders/:id/pay`、`/finish`、`/cancel` 路径，后端查无此单 ⇒ 四条链路口径一致返回
`404 CodeOrderNotFound`。

**关键性质**：这不是类型错误，是**静默的数值精度丢失**——不会抛异常，只表现为业务 404（D3「失败即静默」同族）。

**该缺陷的载体不止一处**：订单详情响应里 `items[].order_id` 承载的**也是同一个 63 位订单主键**。
只改 `Order.id` 会留下一个同类的半开口子，故本次两者一并字符串化（见 §4）。

## 2. 根因（实测定位）

| 位置 | 实测内容 |
| --- | --- |
| `internal/model/orders.go:13` | `ID int64 \`gorm:"primaryKey;autoIncrement;type:bigint" json:"id"\`` |
| `internal/model/order_items.go:7` | `OrderID int64 \`gorm:"type:bigint;...;index:idx_order_items_order_id" json:"order_id"\``（同一个 Snowflake 主键） |
| `pkg/idgen/idgen.go:118` | `id := ((now - epoch) << 22) \| (workerID << 12) \| sequence`（63 位） |
| `internal/handler/product_handler.go:128` | `parsePositiveID` 用 `strconv.ParseInt` 解析路径参数，**字符串形态的 ID 不受影响** |
| `internal/response/order_response.go:5-14` | 直接内嵌 `*model.Order` / `[]*model.OrderItem`，序列化契约由 model tag 决定 |

## 3. 变更实施清单

1. `internal/model/orders.go`：`Order.ID` 的 JSON tag 改为 `json:"id,string"`（仅序列化侧；GORM 列类型不变）。
2. `internal/model/order_items.go`：`OrderItem.OrderID` 改为 `json:"order_id,string"`；
   `ID` / `ProductID` **不动**，并在字段注释里写明「auto-increment, not Snowflake」的区分。
3. `internal/handler/order_handler_test.go`：
   - 既有用例 `Data struct{ ID int64 }` 改为 `string` 断言（该用例断言的正是被修复的行为）；
   - `assertQuotedIDInRawJSON` 泛化为 `assertQuotedInt64InRawJSON(t, raw, field, id)`，供 `id` / `order_id` 共用；
   - 新增 `assertPrecisionLostByFloat64(t, label, id)` 作为共用负向对照，避免重复造轮子；
   - `TestOrderIDSerializedAsString` 扩为 5 组子用例：创建响应、详情响应（`id` 与 `order_id` 双断言）、
     两个 `float64` 负向对照（`order id` / `order item order_id`）、裸数字载荷失配；
   - 详情用例额外钉住 `OrderItem.ID` 仍是**裸数字**（`"id":5`），防止后人把它「顺手改错」。
4. 负向夹具选择：`snowflake19 = 1058897343283204097`（epoch 2024-01-01 + 时间戳 2032-01-01 + worker 1 + seq 1）。

## 4. 影响面判定（哪些字段该改、哪些不该改）

| 字段 | 类型 | 是否 Snowflake | 前端是否消费 | 本次处置 |
| --- | --- | --- | --- | --- |
| `Order.ID` | int64 | 是 | 是（详情/支付/完成/取消） | **改为 string** |
| `OrderItem.OrderID` | int64 | **是**（就是订单主键本身） | 前端已声明为 `string` | **改为 string** |
| `OrderItem.ID` | int64 自增 bigint | 否（auto-increment，短期不溢出） | 否 | **不改**（注释写明区分） |
| `OrderItem.ProductID` | int64 自增 bigint | 否（auto-increment） | 否 | **不改**（注释写明区分） |

**判据是「值域是否可能超过 2^53」，不是「字段名里有没有 id」**：
`OrderItem.OrderID` 与 `Order.ID` 是**同一个值**（订单主键），必须同一处理；
`OrderItem.ID` / `ProductID` 是自增列，当前不可能超过 `MAX_SAFE_INTEGER`，改动它们只会扩大契约面而不消除风险。

> **范围说明（补记）**：本表仅覆盖 **GAP-012 当时的范围**。同类载体还有 `StockLog.BizID`
> （承载订单 Snowflake 主键，`internal/model/stock_log.go`），**已**在 **GAP-015** /
> `docs/changes/014-fix-gap-015-stock-log-biz-id-precision.md` 中处置，不属本表遗漏。

- 前端 `fronted/src/features/order-inventory/` 对订单 ID 的使用全部走 `Order.id`（详情路由 / 动作 API
  `api.ts:130-136` 的 `detail/pay/finish/cancel` 均接收 `id: string`）。
- `items[].order_id` 在 `fronted/src/features/order-inventory/types.ts:98` 已声明为 `string`；
  后端补齐后，**声明与运行时终于一致**。
- `internal/request/order_request.go:13` 的 `CancelOrderRequest{OrderID int64 json:"order_id"}`
  在仓内**无任何引用**（`git grep CancelOrderRequest` 仅命中定义处），取消走路径参数，不在本次影响面内。

## 5. 契约冲突的处置（已由总控裁决，后端补齐）

首版实现只改了 `Order.ID`，同时发现前端已把 `OrderItem.order_id` 声明为 `string`（`types.ts:98`），
而后端仍输出数字——**声明与运行时不一致**。该冲突已上报，总控裁决为：**由后端补齐**
（`OrderItem.OrderID` 改为字符串），前端声明保持不变。

本记录 §4 的表格与结论已按裁决更新：不再是「只改 `Order.ID`」，而是「`Order.ID` 与
`OrderItem.OrderID` 双双字符串化」。前端文件本角色**未改动**（其声明本就正确）。

## 6. 额外发现（仅报告，未改动）

`GET /api/v1/products/:id` 的响应体若包含商品主键，仍是 JSON 数字。商品 ID 为**自增 bigint**，
远未到 2^53，当前**不是**缺陷；但它与订单 ID 的序列化约定不一致，若将来商品 ID 改为 Snowflake 会重演 GAP-012。
建议由 `ssot-scribe` 决定是否登记为独立条目，**本次不动**（避免范围扩散）。

> **后续处置（`2026-10-22`，`ssot-scribe`）**：**已登记为 `GAP-018`**（`docs/GAP台账.md`）。
> 核实依据：`internal/model/product.go:11` 为 `autoIncrement` + `json:"id"`，
> `migrations/00001_create_product.sql:3` 为 `AUTO_INCREMENT`，且 `internal/handler/product_handler.go:49/83/147`
> 直接内联返回该 model（无 DTO 层）——故「暴露到响应体」属实。
> **与 `GAP-015` 不重叠**：GAP-015 的载体是 `StockLog.BizID`（取值为**订单** Snowflake），
> 本项载体是 `Product.ID`（取值为**商品**自增主键），**不得互相替代**。
> 本记录**不**改写原结论（「本次不动」仍是当时的事实），仅追加去向。

## 7. 实测验证日志（D1：真实 stdout 回填）

采集时间：2026-10-22T04:30Z 前后（UTC），工作树 `fix/issue-14-order-module`。

### 7.1 针对性用例

```text
=== RUN   TestOrderHandlerCreateOrder
    --- PASS: TestOrderHandlerCreateOrder/binds_identity_and_request_then_returns_order (0.00s)
    --- PASS: TestOrderHandlerCreateOrder/rejects_invalid_body_before_calling_service (0.00s)
    --- PASS: TestOrderHandlerCreateOrder/maps_business_error_to_HTTP_response (0.00s)
--- PASS: TestOrderHandlerCreateOrder (0.00s)
=== RUN   TestOrderIDSerializedAsString
=== RUN   TestOrderIDSerializedAsString/create_response_emits_quoted_19-digit_id_in_raw_bytes
=== RUN   TestOrderIDSerializedAsString/detail_response_emits_quoted_19-digit_ids_in_raw_bytes
=== RUN   TestOrderIDSerializedAsString/javascript_float64_round_trip_diverges_from_original_id
    order_handler_test.go:336: order id: original=1058897343283204097 javascript-round-tripped=1058897343283204096 delta=-1
=== RUN   TestOrderIDSerializedAsString/javascript_float64_round_trip_diverges_from_original_order_id
    order_handler_test.go:342: order item order_id: original=1058897343283204097 javascript-round-tripped=1058897343283204096 delta=-1
=== RUN   TestOrderIDSerializedAsString/unquoted_json_number_is_truncated_by_float64_decode
    order_handler_test.go:364: unquoted payload: original=1058897343283204097 truncated=1058897343283204096 delta=-1
--- PASS: TestOrderIDSerializedAsString (0.00s)
    --- PASS: TestOrderIDSerializedAsString/create_response_emits_quoted_19-digit_id_in_raw_bytes (0.00s)
    --- PASS: TestOrderIDSerializedAsString/detail_response_emits_quoted_19-digit_ids_in_raw_bytes (0.00s)
    --- PASS: TestOrderIDSerializedAsString/javascript_float64_round_trip_diverges_from_original_id (0.00s)
    --- PASS: TestOrderIDSerializedAsString/javascript_float64_round_trip_diverges_from_original_order_id (0.00s)
    --- PASS: TestOrderIDSerializedAsString/unquoted_json_number_is_truncated_by_float64_decode (0.00s)
PASS
ok  	go-order-management-system/internal/handler	2.984s
```

详情响应用例同时断言四件事（原始字节层面）：

1. 含 `"id":"1058897343283204097"`，且**不含** `"id":1058897343283204097`；
2. 含 `"order_id":"1058897343283204097"`，且**不含** `"order_id":1058897343283204097`；
3. `OrderItem.ID` 仍为**裸数字** `"id":5`（auto-increment，刻意不动）；
4. 解码后 `items[0].order_id == 1058897343283204097`（消费侧可无损往返）。

### 7.2 五道门禁

```text
PS> go vet ./...
vet exit=0

PS> go test -count=1 ./...
ok  	go-order-management-system/config	1.492s
ok  	go-order-management-system/internal/app	3.279s
ok  	go-order-management-system/internal/handler	4.134s
ok  	go-order-management-system/pkg/redis	1.170s
...（全部 ok，无 FAIL）
test exit=0

PS> go test -count=1 -race ./...
ok  	go-order-management-system/config	1.906s
ok  	go-order-management-system/internal/app	3.734s
ok  	go-order-management-system/internal/handler	4.301s
ok  	go-order-management-system/pkg/redis	1.615s
...（全部 ok，无 DATA RACE）
race exit=0

PS> golangci-lint run ./...
0 issues.
lint exit=0

PS> goose -dir migrations validate
goose exit=0
```

### 7.3 观测事实（D1）

实测 Snowflake ID 的位数随时间增长，**不是**固定 19 位：

```text
2026-10-22: id=371447562240004097 digits=18 roundtrip=371447562240004096 equal=false
2032-01-01: id=1058897343283204097 digits=19 roundtrip=1058897343283204096 equal=false
```

两者都 > `MAX_SAFE_INTEGER`，都必然失真。测试夹具取 19 位是为了让回归证明**不随年份到期**，
同时把「带引号」契约钉死在位数假设之外。

## 8. 负向证伪检验（D3 实际输出）

**变异 1a：把 `Order.ID` 的 `json:"id,string"` 改回 `json:"id"`**

```text
--- FAIL: TestOrderIDSerializedAsString (0.00s)
    --- FAIL: TestOrderIDSerializedAsString/create_response_emits_quoted_19-digit_id_in_raw_bytes (0.00s)
        order_handler_test.go:268: response must contain "id":"1058897343283204097" (quoted string), got: {"code":0,...,"data":{"id":1058897343283204097,...}}
    --- FAIL: TestOrderIDSerializedAsString/detail_response_emits_quoted_19-digit_ids_in_raw_bytes (0.00s)
        order_handler_test.go:304: response must contain "id":"1058897343283204097" (quoted string), got: {...,"data":{"order":{"id":1058897343283204097,...},"items":[{"id":5,"order_id":1058897343283204097,...}]}}
FAIL
FAIL	go-order-management-system/internal/handler	2.318s
mutation-test exit=1
```

**变异 1b：把 `OrderItem.OrderID` 的 `json:"order_id,string"` 改回 `json:"order_id"`**

```text
=== MUTATION: OrderItem.OrderID json:"order_id" (string option removed) ===
--- FAIL: TestOrderIDSerializedAsString (0.00s)
    --- FAIL: TestOrderIDSerializedAsString/detail_response_emits_quoted_19-digit_ids_in_raw_bytes (0.00s)
        order_handler_test.go:308: response must contain "order_id":"1058897343283204097" (quoted string), got: {"code":0,"message":"success","data":{"order":{"id":"1058897343283204097",...},"items":[{"id":5,"order_id":1058897343283204097,...}]}}
FAIL
FAIL	go-order-management-system/internal/handler	2.436s
mutation exit=1
```

注意这份输出里 `order.id` 仍是**带引号**的（`"id":"1058897343283204097"`），只有 `order_id` 变回裸数字
——说明两条判据**彼此独立**，不是一条断言顺带覆盖另一条。

**常驻负向数值断言（见 §7.1）**：`order id` 与 `order item order_id` 两个对照各打印
`original=1058897343283204097` vs `truncated=1058897343283204096`，差值 -1。
用例内写明：若某天二者相等，说明负向对照已失效，用例主动 `t.Fatalf`。

## 9. 下游影响与未竟项

- **前端契约（GAP-013，`frontend-dev` 负责）**：`Order.id` 与 `OrderItem.order_id` 现在**都是** JSON 字符串。
  前端 `types.ts` 的声明（两者均为 `string`）与运行时**已一致**，无需改动。
  ⚠️ `types.ts:95-96` 的注释仍写「auto-increment ids, still emitted as JSON numbers… only the order id moved to string」，
  该注释现已**过时**（`order_id` 也已是字符串）——属 `frontend-dev` 的文件，本角色未改，请其同步。
- **`response.OrderDetailResponse.Items`**：`items[].order_id` 已字符串化；
  `items[].id` / `product_id` 仍为数字，且**不应**改动（auto-increment）。
- **无数据库迁移**：仅 JSON tag 变化，`orders.id` / `order_items.order_id` 列类型与索引不变。
- **风险未登记项**：商品 ID 的序列化一致性（§6）需 `ssot-scribe` 决定是否入台账；
  本角色不改 `docs/GAP台账.md`。
  > **后续（`2026-10-22`）**：已登记为 **`GAP-018`**，详见 §6 末尾的处置说明。
- **无数据库迁移**：仅 JSON tag 变化，`orders.id` 列类型与索引不变。
