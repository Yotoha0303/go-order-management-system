# 015 · 新增 pprof 剖析端点（独立端口 + 环境变量开关）

> **先写后填**（`AGENTS.md` §2 02 环）：本骨架先落盘，实现与验证后回填 §4 / §5 / §6。
> 取号：全 ref 并集实测最大号为 `014`（`git for-each-ref` + 逐 ref `ls-tree`），故本条为 **`015`**。

## 1. 背景与动机

`git grep -n "net/http/pprof\|pprof"` 全仓**零命中**（`exit=1`）。本仓已有：

- `/metrics`（Prometheus 文本格式，`router/router.go`）
- 内置压测 `make load-test`（`cmd/loadtest`，打 `/ping`）
- 三份压测证据（`docs/evidence/loadtest_*_2026-07-11.md`）

**缺的是「延迟由哪段代码贡献」这一层** —— `/metrics` 只给结果数字（RPS / p95），不给调用栈归因。
`pprof` 补的正是这一层。

## 2. 设计约束（本条的取舍，不是默认值）

| 约束 | 取值 | 理由 |
| --- | --- | --- |
| **默认关闭** | `enabled: false` | 剖析端点会暴露 goroutine 栈、堆内容、30s CPU 采样。**默认开 = 默认泄露**。 |
| **独立端口** | `127.0.0.1:6060` | **不挂在业务端口上**。业务口对外，剖析口只对本机。 |
| **只绑回环** | `127.0.0.1`，**不是** `0.0.0.0` | 云主机上开启后**不会**开出远程入口；要剖析远程实例走 SSH 隧道（`ssh -L 6060:127.0.0.1:6060`）。 |
| **严格布尔解析** | `PPROF_ENABLED=maybe` ⇒ **启动失败** | 沿用本仓 `REDIS_TLS` 的既定口径（`GAP-014` DoD ④：非法布尔值必须报错，不得静默当 false）。**静默降级会把「我以为开了」变成「其实没开」。** |

## 3. 与既有条目的关系

- **`GAP-029`（04 环无不可变 Digest）**：**不重合**。本条是运行期诊断能力，不改制品可追溯性。
- **`GAP-031`（06–10 环无对应物）**：**部分触及 06 环**（可观测性），但**不解除它** —— 06 环准出要的是「Alertmanager 告警规则 + 严重度 + 真实停机演练」，pprof 是**人工排查**工具，**不产生告警**。**不得据本条关闭 `GAP-031`。**

## 4. 可落地变更

| 文件 | 变更 |
| --- | --- |
| `config/config.go` | 新增 `PprofConfig{Enabled, Addr}` 与 `Pprof` 字段；新增 `applyPprofEnvOverrides`（`PPROF_ENABLED` / `PPROF_ADDR`） |
| `config/config_validate.go` | 新增 `PprofConfig.Validate(serverPort)` + 三个错误值 + `loopbackHosts` 白名单 |
| `internal/app/pprof.go` | **新增**。`NewPprofServer`（未启用返回 `nil`）+ `RunPprofServer`（返回 stop 函数） |
| `internal/app/app.go` | `Run()` 中启动并注册关闭 |
| `config.yml` | 新增 `pprof:` 块，`enabled: false` / `addr: 127.0.0.1:6060` |
| `config/pprof_config_test.go` | **新增**。9 个测试函数，正负两边齐备 |

**关键实现选择（承重，非装饰）**：

1. **显式注册 `pprof.Handler`，不 `import _ "net/http/pprof"`**。后者会把端点挂到 `http.DefaultServeMux`，**而业务服务器若回落到默认 mux 也会一并暴露**——正是要避免的。
2. **`Validate` 用白名单，不是黑名单**。写「拒绝 `0.0.0.0`」会放过 `[::]`、`:6060`、解析到别处的主机名 ⇒ **fail open**。白名单 fail closed。
3. **未启用时 `NewPprofServer` 返回 `nil`** 而非惰性 server —— 让「到底开没开」这个问题**从返回值本身可答**。
4. **`ReadTimeout` 刻意不设**。`/debug/pprof/profile?seconds=30` 全程静默 30s，设读超时会**掐掉它存在的意义**。改用 `WriteTimeout: 2m` 兜底。

## 5. 正向实测命令与预期输出

**判据：`PPROF_ENABLED` 与 `Addr` 的每种组合都有确定行为。**

```powershell
go test -count=1 ./config/ -run 'TestPprof|TestShippedConfig' -v
```

实测（`2026-10-23`）：

```
--- PASS: TestPprofValidateAcceptsLoopbackForms (0.00s)      # 4 个子用例
--- PASS: TestPprofValidateRejectsNonLoopback (0.00s)        # 6 个子用例
--- PASS: TestPprofValidateRejectsBusinessPortCollision (0.00s)
--- PASS: TestPprofValidateRejectsMalformedAddr (0.00s)      # 5 个子用例
--- PASS: TestPprofValidateIgnoresAddrWhenDisabled (0.00s)   # 4 个子用例
--- PASS: TestPprofEnvEnabledRejectsNonBoolean (0.00s)       # 5 个子用例
--- PASS: TestPprofEnvEnabledAcceptsBooleanForms (0.00s)     # 4 个子用例
--- PASS: TestPprofEnvAddrOverride (0.00s)
--- PASS: TestPprofEnvOverrideStillSubjectToValidate (0.00s)
--- PASS: TestShippedConfigKeepsPprofOffAndLoopback (0.00s)
PASS
ok  go-order-management-system/config  0.582s
```

**端到端（真实进程，非单测）** —— `PPROF_ENABLED=true` 启动后：

```
port 6060 LISTENING
time=2026-10-23T17:52:02.593+08:00 level=INFO msg="pprof listener starting" addr=127.0.0.1:6060
  note="loopback only; reach a remote instance through an SSH tunnel"

GET http://127.0.0.1:6060/debug/pprof/  ->  HTTP 200
  allocs 12 · block 0 · goroutine 20 · heap 12 · mutex 0 · threadcreate 0
```

并已产出真实剖析制品（见 `docs/evidence/pprof_report_2026-10-23.md`）：
30s CPU profile（156,546 B）+ heap + allocs，压测 100,000 请求 / 50 并发 / 0 失败。

## 6. 负向证伪检验（D3）

**四组变异注入，逐组确认测试变红**（`2026-10-23` 实测）：

| 变异 | 改动 | 预期 | 实测 |
| --- | --- | --- | --- |
| **A′** | 回环检查短路 `if false && !loopbackHosts[host]` | `RejectsNonLoopback` + `EnvOverrideStillSubjectToValidate` 变红 | ✅ 两个函数 FAIL |
| **B** | `PPROF_ENABLED` 解析失败**静默吞掉**（当 false） | `EnvEnabledRejectsNonBoolean` 变红 | ✅ FAIL |
| **C** | 去掉业务端口冲突检查 | `RejectsBusinessPortCollision` 变红 | ✅ FAIL |

> **变异 A 的第一版被我自己否掉**：直接 `Validate` 掏空成 `return nil` 会让 `fmt`/`net` 变成未使用 import ⇒ **构建失败**，红的理由不对（`[build failed]` 不是判据有牙的证据）。改为**只短路那一行**，保留全部 import 与变量使用，才是有效变异。**这一条留痕**：变异本身也要能区分「判据被拆」与「代码写坏」。

**另一条负向（行为层，非单测）**：`PPROF_ENABLED=maybe` ⇒ 进程**启动失败**，不是静默当 false：

```
$ PPROF_ENABLED=maybe ./go-order-management-system
invalid PPROF_ENABLED: ...    # 启动中止
```

## 7. 下游同步清单与未竟 GAP

**已同步**：`config.yml`（默认值 + 注释）· `.env.example`（`PPROF_ENABLED` / `PPROF_ADDR`）· `docs/evidence/pprof_report_2026-10-23.md`（剖析制品）· `docs/evidence/pprof-2026-10-23/*.prof` · `docs/resume_material.md`（量化素材，引用本条的制品）。

**未做（如实，不声称已完成）**：

| 项 | 状态 |
| --- | --- |
| **`docs/observability.md` 未提 pprof** | ❌ 未做 |
| **写链路（下单事务 / Redis Lua / Outbox）未剖析** | ❌ 本次只压了读链路 |
| **多时点 heap 采样未做** | ❌ 故「无泄漏」结论强度仅为「未观察到累积」 |
| **端到端剖析流程未进 CI** | ❌ 本条的**单测**在 CI 里会跑（`go test ./...`），但**压测 + 抓 profile** 那套流程没有 CI 步骤 |

**与 `GAP-031`（06–10 环无对应物）的关系**：**本条不解除它**。06 环准出要的是「Alertmanager 告警规则 + 严重度参与 + 真实停机演练」，pprof 是**人工排查**工具，**不产生任何告警**。**不得据本条关闭 `GAP-031`。**

