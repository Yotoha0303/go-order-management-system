# 变更记录：012-fix-gap-013-order-form-reset-idempotency

- **变更编号**：012
- **创建时间**：2026-10-22T11:45:21Z (UTC)
- **完成时间**：2026-10-22T11:58:00Z (UTC)
- **作者**：`frontend-dev` (前端工程专家)
- **关联缺陷**：GAP-013 (P1: 创建订单表单 index key 致 Select 状态残留 / 幂等 Key 跨提交复用 / 缺重置入口) · GAP-012 前端一半 (P1: 订单 ID 精度截断)
- **关联 Issue**：#14
- **关联 ADR**：无（前端状态与契约适配，不引入架构决策）
- **状态**：已完成（4 道门禁全绿；负向证伪 5 项均变红后还原）

---

## 1. 变更动机与背景

Issue #14 的前端两项缺陷。**以下根因均由总控实测确认，本记录不重新推断**（D1）：

| # | 位置 | 现象 | 根因 |
| :-- | :-- | :-- | :-- |
| GAP-013-a | `fronted/src/features/order-inventory/orders.tsx:233-237` | 重置/删除行后，商品下拉框仍显示上一次的选择 | 行 `key={index}`：重置后 `key=0` 不变，React 不重新挂载该行；Radix `Select` 由受控（`value` 有值）切回非受控（`value=undefined`）时内部展示态残留 |
| GAP-013-b | `orders.tsx:114`、`orders.tsx:191`、`order-submission.ts:46` | 提交失败后重试，或想连续提交同一笔订单时，命中后端幂等重放/冲突 | `onError` 只弹 toast，**不清 `pendingSubmission`**；下一次提交把旧值传回 `prepareOrderSubmission`，指纹相同即**复用旧幂等 Key** |
| GAP-013-c | `orders.tsx` 左侧表单 | 无任何「清空/重置」入口 | 功能缺失 |
| GAP-012（前端） | `orders.tsx:69,107,448`、`types.ts:80` | 详情/支付/完成/取消 404 | 订单主键是 63 位 Snowflake `int64`，JSON 以原生数字输出，`JSON.parse` 在 2^53 处截断；前端用 `number` 承载后再拼进 URL，请求的是一个**不存在的 ID** |

后端由 `backend-dev` 并行修改 `model.Order.ID` 的 JSON 标签为 `json:"id,string"`（`internal/model/orders.go`）。
**本变更按「订单 ID 在 JSON 里是字符串」编码**，不改任何 `.go` 文件。

---

## 2. 拟变更清单（先写）

### 2.1 行键稳定化（GAP-013-a）

- `order-submission.ts`：导出 `createRowId()`（复用 `createIdempotencyKey` 的降级策略，不引入新依赖）。
- `orders.tsx`：`DraftOrderItem` 增加 `rowId: string`；新增/删除/重置/成功后均生成新 `rowId`；渲染 `key={item.rowId}`。
- 行集合的创建逻辑抽成**可测纯函数**（`createDraftOrderItem` / `replaceDraftItem` / `removeDraftItem` / `resetDraftItems`），
  以便对「重置后所有 rowId 变化」写单测，而不必测 React 内部状态。
- 另加一条**行为级**回归（`orders.test.tsx`），直接断言「重置后 Select 触发器回到 placeholder」——
  纯函数单测证明不了「Radix 真的重新挂载」，这一条才覆盖用户实际看到的症状。

### 2.2 幂等 Key 生命周期（GAP-013-b）

- `createOrderMutation.onError`：清空 `pendingSubmission`（下一次点击 = 新订单 = 新 Key）。
- `onSuccess`：保持清空（既有行为）。
- `prepareOrderSubmission` 的「指纹相同则复用」语义**保留但收窄**：只服务**同一次在途提交**；同步更新文档注释。

### 2.3 重置入口（GAP-013-c）

- 左侧表单新增「清空」按钮（`type='button'`）：`items` 重置为一行空行（新 `rowId`）、清空 `pendingSubmission`、清空 `selectedOrderId`。
- 按钮排布随之调整：第一行「添加商品 / 清空」，提交按钮独占第二行（`w-full`），避免三个按钮挤在一行。

### 2.4 订单 ID 字符串化（GAP-012 前端）

- `types.ts`：`Order.id: string`；`OrderItem.order_id: string`。
  `OrderItem.id` / `product_id` 后端**当前仍是数字**（实测 `internal/model/order_items.go:6,8` 为 `json:"id"` / `json:"product_id"`），保持 `number`，**不猜**。
- `api.ts`：`orderApi.detail/pay/finish/cancel(id: string)`；`queryKeys.order(id: string)`。
- `orders.tsx`：`selectedOrderId: string | null`；查询键 `queryKeys.order(selectedOrderId ?? '')`、`enabled: selectedOrderId !== null`。
- **禁止** `Number()` / `parseInt` 回转订单 ID。

---

## 3. 验证计划（先写）

| # | 命令 / 用例 | 预期信号 |
| :-- | :-- | :-- |
| V1 | `npx tsc -b --force`（**订正**：原写 `npx tsc --noEmit`，该命令在本仓检查 0 个文件，见 §4.2 小注与 §5.4(b)） | exit 0 |
| V2 | `npx vitest run` | exit 0，全部用例 PASS |
| V3 | `npx eslint .` | exit 0，0 problems |
| V4 | `npm run build`（`tsc -b && vite build`） | exit 0 |
| N1 | `order-submission.test.ts`：`prepareOrderSubmission(payload, null)` 在上一次已清空时生成**新** Key | 断言 `createKey` 调用 2 次、两 Key 不等 |
| N2 | 同一 `pendingSubmission`（在途）重提 → 复用同一 Key（**保留的原有正向用例**） | 断言 `toBe(first)` 且 `createKey` 仅 1 次 |
| N3 | JS 精度截断回归：19 位字符串 ID，`String(Number(id)) !== id` | 断言成立，并打印截断前后两个不同字符串 |
| N4 | 纯函数：重置后所有 `rowId` 均变化 | 旧/新 rowId 交集为空，且长度为 1 |
| N5 | 行为级：选定商品后点「清空」，Select 触发器文本回到「请选择商品」 | 断言成立。**负向预期已订正**：原写「`key={index}` 时该断言失败」，实测**不成立**（见 §4.5.1）；`key` 回退改由 §4.5.2 的 DOM 节点身份用例捕获 |
| N6 | 行为级：首次创建失败后重提，两次 `orderApi.create` 的 `idempotencyKey` 不同 | 断言成立（负向：`onError` 不清 `pendingSubmission` 时两 Key 相同） |

---

## 4. 验证结果（完成后回填）

### 4.1 前置

- `fronted/node_modules` 原本不存在，先执行 `npm install`（exit 0）。
- `npx playwright install chromium` → exit 0（下载 Chrome for Testing 147 + headless shell）。
- **本机环境阻塞**：vitest browser API 默认端口 `63315` 落在 Windows 保留端口段
  （`netsh int ipv4 show excludedportrange protocol=tcp` 实测 `63221-63420`），
  `npx vitest run` 报 `listen EACCES 127.0.0.1:63315`，**与代码无关**。
  CLI 的 `--browser.api.port` 在本机 4.1.8 上未生效，故验证时使用**一次性**的
  `fronted/vitest.local.config.ts`（仅覆盖 `test.browser.api.port=8899`），**验证后已删除**，
  未污染项目配置。CI 无此保留端口，不受影响。

### 4.2 门禁命令与实测输出

| # | 命令 | exit | 实测尾部 |
| :-- | :-- | :--: | :-- |
| V1 | `npx tsc -b --force`（**订正**，原为 `npx tsc --noEmit`） | **0** | 无诊断输出 |
| V2 | `npx vitest run --browser.headless --config vitest.local.config.ts` | **0** | `Test Files 20 passed (20)` / `Tests 118 passed (118)` |
| V3 | `npx eslint .` | **0** | 无输出（0 problems） |
| V4 | `npm run build` | **0** | `✓ built in 765ms`（`tsc -b` 已含在脚本内） |

> ⚠️ **V1 订正（2026-10-22）**：原表写 `npx tsc --noEmit` 得 exit 0，**该读数不携带信息**——
> 本仓 `fronted/tsconfig.json` 是 solution 风格（`"files": []` + `references`，无 `include`），
> 以它为准时 `tsc --noEmit` **编译 0 个文件**、恒 exit 0。
> 实测：`npx tsc --noEmit --listFilesOnly` → **0** 行；`npx tsc -p tsconfig.app.json --listFilesOnly` → **893**（其中 `src/` 下 **150** 个，其余为 `node_modules` 里的声明文件）；
> 注入 `TS2322` 探针（总控与 `backend-dev` 各自独立复现）→ `tsc --noEmit` exit 0（**漏检**）、`tsc -b` exit 2（报出 `TS2322`）。
> 故 V1 改为 `npx tsc -b --force`（`--force` 绕开 `.tsbuildinfo` 增量跳过）。
> 订正后实测 exit 0。**原因与完整证据见 §5.4(b)，本节不重复论证**。

清理临时文件后**复跑**：`npx tsc -b --force` exit 0（**订正**，原写 `npx tsc --noEmit`，理由同上）；
`npx vitest run ... src/features/order-inventory` → `Test Files 5 passed (5)` / `Tests 31 passed (31)`，exit 0。

**Radix 受控性告警的机械判据（追加，见 4.6）**：全量 20 个测试文件、118 条用例的输出中
`Select is changing from uncontrolled to controlled` **0 条**（`Select-String -Pattern "warn|Warn|Select is"`
命中的全部是 Node 的 `UNDICI-EHPA` 环境告警，与本组件无关）。

### 4.3 本变更新增/修改用例（逐条）

```
✓ order-submission.test.ts > createRowId > never repeats, even within the same millisecond
✓ order-submission.test.ts > createRowId > does not derive from the row index (0 and 1 stay distinct identities)
✓ order-submission.test.ts > prepareOrderSubmission > reuses the key when an unchanged request is retried while in flight   ← 保留的在途复用
✓ order-submission.test.ts > prepareOrderSubmission > creates a new key when the same payload follows a cleared submission ← 新增(GAP-013)
✓ order-submission.test.ts > order id precision (GAP-012) > truncates a 19-digit id when coerced to number               ← 新增(GAP-012)
✓ order-draft.test.ts > order draft rows > reset changes every rowId so the row subtree remounts                          ← 新增(GAP-013)
✓ order-draft.test.ts > order draft rows > reset called twice never returns the same identity
✓ orders.test.tsx > clears the selected product after reset and still accepts a new choice                               ← 新增(GAP-013 行为级)
✓ orders.test.tsx > uses a fresh idempotency key after a failed submission                                                ← 新增(GAP-013 行为级)
✓ orders.test.tsx > passes the created order id through to the detail query verbatim                                      ← 新增(GAP-012 行为级，见 §5.4(a))
✓ orders.test.tsx > replaces the draft row subtree with a new DOM node on reset                                           ← 新增(GAP-013-a 行为级，见 §4.5.2)
```

`orders.test.tsx` 用 `vi.mock('./api')` 提供**受控的接口替身**，断言的是「组件在重置/失败后发给 API 的入参」
与「Select 触发器展示文本」，**不是**用假数据冒充真实读数——真实网络路径不在该用例的判据范围内。

### 4.4 精度截断负向用例的实际输出（D3）

测试内断言（`order-submission.test.ts`）：

```
const id = '1900000000000000001'
const truncated = String(Number(id))     // '1900000000000000000'
expect(truncated).not.toBe(id)           // PASS
expect(truncated).toBe('1900000000000000000')
expect(Number.isSafeInteger(Number(id))).toBe(false)
```

命令实测（node，同口径）：

```
1900000000000000001 1900000000000000000 false false
7300000000000000001 7300000000000000000 false false
1900000000000000128 1900000000000000000 false false
```

> 三者都是 19 位字符串 id 与 `String(Number(id))` 的对照，第三列是 `相等?`，第四列是 `Number.isSafeInteger?`。
> **截断后的字符串与原始 id 不同**，这就是「前端用 number 承载订单 id 后请求 404」的机械证据。

### 4.5 负向证伪（注入回归 → 必红 → 已还原）

| 注入的回归 | 结果 |
| :-- | :-- |
| `createDraftOrderItem` 恒定返回 `rowId: 'row-fixed'`（等价于复用 index 身份） | `order-draft.test.ts` **5 条变红**（`expected 'row-fixed' not to be 'row-fixed'` 等），`Tests 6 failed \| 7 passed`，exit 1 |
| `prepareOrderSubmission` 改为 `if (previous !== null) return previous`（无条件复用） | 「内容变化应换 Key」**变红**（`expected 'first-key' to be 'second-key'`），exit 1 |
| `orders.tsx` 行 `key` 改回 `key={index}`（**最终代码**） | **既有行为用例全部无法捕获**：`src/features/order-inventory` 全量 `Tests 1 failed \| 49 passed (50)`，唯一失败项是 `qa-orders-form.test.tsx > keys draft rows by rowId, never by the array index`（源码级钉住）。原文「该用例变红」的结论**不成立**，原因见 §4.5.1；**其后已补行为级判据**，见 §4.5.2 |
| `onError` 移除 `setPendingSubmission(null)` | `orders.test.tsx > uses a fresh idempotency key after a failed submission` **变红**：`expected '5f616467-…-c04c3228516c' not to be '5f616467-…-c04c3228516c'`，exit 1 |
| `<Select value={item.product_id \|\| undefined}>`（回到非恒受控） | 同一条用例 **变红**：`AssertionError: expected [ …(2) ] to deeply equal []`，并打出两条 `Select is changing from uncontrolled to controlled…`，exit 1 |

五处注入**全部已还原**，还原后 4 道门禁重新全绿（见 4.2）。

#### 4.5.1 订正：`key` 回退在当前代码上**不会被既有行为用例捕获**（2026-10-22）

**原声明（已作废）**：「`key` 改回 `index` → `orders.test.tsx > clears the selected product after reset…` 变红」。

**实测（`qa-validator` 首测，我在本文件所称的「最终代码」上复核）**：注入 `key={index}` 后，
`src/features/order-inventory` 全量结果是 **`Tests 1 failed | 49 passed (50)`**——
唯一失败的只有 `qa-orders-form.test.tsx > keys draft rows by rowId, never by the array index`
（**源码文本**断言）。两条行为用例（`clears the selected product…`、`uses a fresh idempotency key…`）**全绿**。

**为什么不成立（根因是 §4.6 改动了判据的捕获面）**：原结论是在 §4.6 **之前**的代码上得到的。
那一版 `value={item.product_id || undefined}` 在空串档位是**非受控**，换 key 触发重挂载时
会走「非受控 → 受控」，Radix 发出 `uncontrolled to controlled` 告警，用例里的告警断言因此变红。
§4.6 把 Select 改为**恒受控**（`value={item.product_id}`，空串也是受控值）之后，
Radix 不再发出该告警 —— 于是这条告警断言**只能拦住 `value` 回退（§4.5 第 5 行，仍成立），拦不住 `key` 回退**。
**同一条断言在两个代码版本上判的不是同一件事**，这正是原声明失效的机制。

**补强现状**：`qa-validator` 已加源码级钉住用例 `keys draft rows by rowId, never by the array index`
（`qa-orders-form.test.tsx:92-95`），断言源码含 `key={item.rowId}` 且不含 `key={index}`。
按上方实测，它当时是**唯一**能拦住 `key` 回退的用例；**行为级判据已于 §4.5.2 补上**。

> ⚠️ **本小节原文的一段判断已被证伪，保留可见（D4）**：我原先写「不为它再加一层 DOM 判据」，
> 理由是 DOM 身份比对「在 `vitest-browser-react` 的重渲染语义下易随版本漂移」。
> **该理由不成立**：`Locator.element(): HTMLElement | SVGElement` 是**公开长期 API**
> （`node_modules/@vitest/browser/context.d.ts:676` 实测），返回的就是 DOM 节点对象本身，
> 与框架匹配器版本无关。`qa-validator` 用该 API 实测出「有牙、不假绿」，§4.5.2 据此补了行为用例。
> **教训**：以「易漂移」为由拒绝一条判据，须先读它的 API 契约；否则只是把**没验证**说成了**不可行**。

### 4.5.2 补强：`key` 的行为级判据（DOM 节点身份，2026-10-22）

**用例**：`orders.test.tsx > replaces the draft row subtree with a new DOM node on reset`

**判据形态**：重置**前**取 `screen.getByRole('combobox').first().element()` 的节点引用，
点「清空」后**重新查询**同一个 locator，断言新取到的 `.element()` 与旧引用**不是同一对象**（`not.toBe`）。
选这个形态的理由：

- 用 `Locator.element()` 公开 API，**不依赖** `rowId` 命名、Radix 告警、DOM 属性名或测试框架匹配器版本；
- `key` 的语义本来就是「这个位置上的子树还是不是原来那个实例」，节点身份是它的直接观测面；
- 与 §4.5.1 的源码钉住**分工**：源码判据失败信息直接指向「是 key 回退了」，行为判据证明**用户可见的后果**真的发生了。两条都保留。

**两个刻意的写法**：① 重置后**重新查询 locator**，不复用重置前的 locator（旧节点可能已 detach）；
② 只断言节点身份，**不**同时断言「无 console 告警」——那是 `clears the selected product…` 那条用例的职责，
混进来会让失败信息分不清是 `key` 回退还是 `value` 回退。

**负向实测（红 → 还原 → 绿）**：

```
# 注入 key={index}
× replaces the draft row subtree with a new DOM node on reset
AssertionError: expected <button type="button" …(9)>…(2)</button> not to be <button type="button" …(9)>…(2)</button>
Test Files  1 failed (1) / Tests  1 failed | 3 passed (4) / EXIT=1

# 还原为 key={item.rowId}
Test Files  1 passed (1) / Tests  4 passed (4) / EXIT=0
```

### 4.6 补充修复：Select 恒受控（GAP-013-a 的真根因）

**问题**：第一版只换了 `key`。`value={item.product_id || undefined}` 在「未选中」时传 `undefined`，
Radix 的 `useControllableState` 以 `prop !== undefined` 判定受控性 —— 于是空串档位是**非受控**（内部 state 自持），
选中后切**受控**。上一条用例能过，只是因为换 key 触发重挂载、恰好把内部 state 丢了；
「选择值到底由 `items` 还是由 Radix 内部 state 决定」这一点**并未被消除**。

**修法**：`value={item.product_id}`，**恒受控**。依据（实测读源码，非推断）：
- `node_modules/@radix-ui/react-use-controllable-state/dist/index.mjs:16` → `isControlled = prop !== void 0`；
  空串满足受控条件。
- `node_modules/@radix-ui/react-select/dist/index.mjs:1139` → `shouldShowPlaceholder(value) => value === '' || value === undefined`；
  空串即「未选中」，配合 `<SelectValue placeholder='请选择商品' />` 正常显示 placeholder。
- 同文件 `:843` 的限制是「`<Select.Item>` 的 **value** 不得为空串」——本组件的 Item 值是商品 id，不受影响。

**验收判据（可证伪）**：`orders.test.tsx` 内用 `vi.spyOn(console, 'warn')` 捕获所有告警，
断言其中**不含** `uncontrolled to controlled`；同时该用例还断言「重置后显示 placeholder」，
并且「**再点同一商品仍能正常选中**」（`toHaveTextContent(/测试商品/)`）——证明没有把正常交互改坏。

---

## 5. 下游同步与未竟项

### 5.1 与后端契约的对齐（实测）

- `internal/model/orders.go:16` 实测为 `json:"id,string"`（`backend-dev` 的改动已落盘），
  前端 `Order.id: string` 与之对齐。
- `internal/model/order_items.go:7,11` 实测为 `OrderID … json:"order_id,string"`、
  `ID` / `ProductID` 仍为 `json:"id"` / `json:"product_id"`（数字）。
  故 `OrderItem.id` / `product_id` **保持 `number`**，`order_id` 为 `string` —— 类型与运行时一致。
  第一版 `types.ts` 注释曾写「Only the order id moved to string」，已被后端改动证伪（D4：两处不同成熟度），**已改**。

### 5.2 风险点（未在本轮验证，交由总控/`backend-dev` 确认）

1. **`OrderItem.order_id` 的 JSON 形态**：已随 `backend-dev` 的改动落盘为 `json:"order_id,string"`（`order_items.go:11` 实测），
   与本文件的 `order_id: string` 一致，**风险已消除**。前端目前没有任何代码消费该字段（`git grep` 实测仅 `types.ts` 声明）。
2. **`OrderDetail.order.id` 与 URL 拼接**：前端把 `string` 直接插入模板字符串，不做任何数值转换，符合「禁止 `Number()` 回转」的要求。
3. **后端是否对 `/orders/:id` 的路径参数改动**：不需要——id 在 URL 里本来就是字符串形态。

### 5.3 已知未做 / 未验证

- **未提交**：按硬边界，本轮不 `git add` / `commit` / `push`，由总控统一提交。
- **真实后端联调未跑**：本任务范围内无可用后端实例，GAP-012 的端到端 404 消除只做了类型与精度层面的机械证明；
  端到端需在集成环境复核（建议由 `qa-validator` 在 03 环覆盖）。
- **本机端口占用问题**：`63315` 在 Windows 保留段属**环境问题**，非仓库缺陷；
  若 CI 或他人机器出现同样报错，处置方式见 4.1（临时覆盖 `test.browser.api.port`）。
- `fronted/src/routeTree.gen.ts` 在本轮开始前就处于 `M` 状态（`git diff` 无内容差异，为行尾/生成时间戳所致），
  **非本变更产生**，未做处理。

### 5.4 补记（2026-10-22）：onSuccess 从未被观测 + `tsc --noEmit` 在本仓空转

**(a) `onSuccess` 此前没有判据覆盖（`falsifier` 判 DEAD，属实）**。
本文件 §4.3 列的三条 `orders.test.tsx` 用例中，两条让 `orderApi.create` 走 reject，
第三条（`clears the selected product…`）虽未 reject 但也**从未让它成功 resolve**。
因此 `orders.tsx` 的 `createOrderMutation.onSuccess` **一次都没被执行过**，
其中 `setSelectedOrderId(order.id)` 无任何断言覆盖 —— 把它改成
`setSelectedOrderId(String(Number(order.id)))`（**正是 GAP-012 的原始缺陷形态**）**测试全绿**。

**补强**：`orders.test.tsx` 新增 `passes the created order id through to the detail query verbatim`：
迫使 `create()` 以 19 位 Snowflake `'1058897343283204097'` 成功 resolve，用
`expect(detail.mock.calls[0][0]).toBe('1058897343283204097')` 做**严格相等**断言，
并在同一用例内钉住前提 `String(Number(id)) === '1058897343283204100' ≠ id`。

**负向实测（红 → 还原 → 绿）**：注入上述 `Number()` 回转后
`AssertionError: expected '1058897343283204100' to be '1058897343283204097'`，exit 1；
还原后同命令 `Tests 3 passed (3)`，exit 0。

**(b) `npx tsc --noEmit` 在本仓是空转的（实测；已由 `docs/GAP台账.md` 的 `GAP-017` DoD ① 覆盖，不另立编号）**。
`tsconfig.json` 是 solution 风格：`"files": []` + `references`，**没有 `include`**。
故 `npx tsc --noEmit` 以它为准时**编译 0 个文件**：

```
npx tsc --noEmit --listFilesOnly                  → 0 行
npx tsc -p tsconfig.app.json --listFilesOnly      → 893（src/ 下 150，其余为 node_modules 声明文件）
```

**后果**：§4.2 的 V1「`npx tsc --noEmit` exit 0 → 无诊断输出」**不携带信息** ——
它绿是因为它什么都没检查。本轮实测撞到：新用例里一处 `TS2493`
（元组 `[]` 无下标 0）被 `npm run build`（`tsc -b`，会按 references 真正编译）抓出并 exit 2，
而 `npx tsc --noEmit` 仍 exit 0。
**订正后的 V1**：以 `npx tsc -b --force`（或 `npx tsc -p tsconfig.app.json --noEmit`）为准。
**该订正已回写到 §3 的 V1 行与 §4.2 的读数表**（读者先看到的是那两处，本小节保留为论证）。
**未改仓库配置**（是否把 `tsc --noEmit` 换成 `tsc -b` 属门禁定义变更，交总控裁定），此处只登记实测。

**独立复现一致性**：总控与 `backend-dev` 各自独立复现了同一组读数
（`tsc --noEmit` → 0 个文件、exit 0；注入 `TS2322` 探针后 `tsc --noEmit` 漏检、`tsc -b` exit 2）。
本条与 `docs/GAP台账.md` 的 `GAP-017`（CI 缺前端作业，其 DoD ① 明确要求类型步骤用 `tsc -b`，并已把本组读数作为判据依据）
指向同一事实，**故不另分配新编号**——避免同一缺口出现两个编号（D4）。
两处不构成「不同成熟度的副本」：台账是缺陷登记与 DoD，本文件是本次变更的验证足迹。
