# AGENTS.md — 订单库存一致性管理服务约束规范（唯一约束源）

> **项目定位**：Go 订单库存一致性管理服务（Go 1.25.7 + Gin + GORM + MySQL 8.4 + Redis 7.2 + RabbitMQ 4.1）。
> 本文件是本仓库协同研发的**唯一约束规范**。AI 总控与子代理在每次会话开始时必须读取本文件，并严格受其约束。
> **基线口径**：文内所有基线读数均由命令当场实测得出，**无一条为推断**（D1）。

---

## 0. 位置约定与双锚分工

### 0.0 位置约定
- **本文件位于项目仓根**：`$REPO = $(git rev-parse --show-toplevel)`。
- 本仓为**业务项目独立仓**，Git 历史与提交与外层框架完全隔离。
- 任何提交与门禁检查均在本仓作用域内执行。

### 0.1 执行前文档同步（双锚）
在进行任何代码修改或架构设计前，**必须先读完文档再动手**（零假设原则）：
1. **计划锚** (`docs/planning/`)：回答「去哪、为什么、架构规格是什么」——见 `docs/planning/roadmap.md`。
2. **足迹锚** (`docs/changes/`)：回答「这一步具体改了什么、为什么、怎么验证的」——先写后填。
3. **台账锚** (`docs/GAP台账.md`)：回答「还有什么没做、当前缺陷是什么」——**不在台账里的未竟事项等于不存在**。

---

## 1. 四大不可妥协纪律（Zero Tolerance）

### D1 · 读到才算，严禁推断
- 严禁从组件名、接口名推测运行时行为。所有结论必须依托实际读取的文件内容或命令的真实 stdout。
- 文件与统计口径强制使用 `git ls-files`，隔离构建产物与依赖。
- 不得把空输出、未验证代码当作已完成交付。

### D2 · 转段强阻断（Hard Stage Gates）
- 严格按门禁矩阵推进。前一环准出未满足，严禁生成下一环代码。
- 被阻断时输出明确的阻断原因与解除条件。

### D3 · 判据必须可证伪（Falsification）
- 编写任何验证用例、修复逻辑或门禁时，**必须同时提供负向验证用例**：注入故障或变异后必定触发失败，拒绝虚假全绿。

### D4 · 单一事实源（SSOT）与标识符防漂移
- 缺陷与未竟事项严格采用 `GAP-` 编号，架构决策采用 `ADR-` 编号。
- 登记新缺口前必须先检索 `docs/GAP台账.md`，严禁编号撞车。
- 产生结论必须立即回写对应设计文档，禁止两处不同成熟度的事实并存。

---

## 2. 门禁矩阵（Go 栈技术插槽）

| 环 | 性质 | 准出条件与实测命令 |
| :--- | :---: | :--- |
| **01 设计** | 🔒 硬门禁 | ADR 落盘（`docs/adr/`）+ 验收标准写成 Given-When-Then 可测形式 |
| **02 开发** | 🔒 硬门禁 | `docs/changes/NNN-*.md` **先写后填** + 统一代码风格 |
| **03 测试** | 🔒 硬门禁 | **代码必须通过**：`go vet ./... && go test -race ./...`（`-race` 为必选项） |
| **04 构建** | 🔒 硬门禁 | 静态检查：`golangci-lint run` + Goose 校验：`make migrate-validate` |
| **05 部署** | 🔒 硬门禁 | Docker Compose 编排可用：`docker compose config -q` 语法合法 |

---

## 3. 五段交互与输出协议

每次生成工程资产必须五段齐全：
1. **【准入审查】**：当前处于哪一环？前置硬资产是否满足？
2. **【可落地变更】**：符合现有 Go 技术栈的最小变更。
3. **【正向实测命令与预期输出】**：终端可执行的具体命令，明确判定依据。
4. **【负向证伪检验】**：证明判据不会静默的失败用例。
5. **【下游同步清单与未竟 GAP】**：引发的文档/配置联动，回写 `docs/GAP台账.md`。

---

## 4. 实测基线与当前状态

* **采集时点**：`2026-10-22T06:27:12Z`（UTC）
* **Git 分支**：`fix/issue-14-order-module`（upstream = `origin/fix/issue-14-order-module`，**领先 0 个提交**，即已全部推送）
* **当前 HEAD（全 40 位）**：`a6439d24683cc4472a999bedaa53f3e6e157426d`（短号 `a6439d2`）
* **分支基底**：`4fa68f9`（即当时 `main` 的 HEAD）；**本分支已在其上提交 8 个提交**
  > ✅ **issue #14 的修复已提交并已推送**——不再是「未提交」状态。改动分 **8** 个提交落盘，
  > 已推到 `origin/fix/issue-14-order-module`，并开了 PR [#17](https://github.com/Yotoha0303/go-order-management-system/pull/17)。
  > **PR #17 的 CI 已连续两轮全绿**（run `39732018029` @ `6fa11f8`、run `39733467897` @ `a6439d2`），
  > 两轮均为 `Frontend` + `Test, Vet, Validate, Build` 双作业 pass。
  > ⚠️ **但 PR 尚未合并**——故 `main` 上的 CI **仍然一次未绿**（见台账 `GAP-020`）。
  > **合并后必须重新采集本节**：届时分支会回到 `main`、提交数与受管文件数会变，且 GAP-020 的
  > 「`main` 无绿」前提将被 PR 的绿 run 打破——**这三项都会同时过期**。
* **历史提交数**：`56`
* **代码与受管文件数 (`git ls-files`)**：`439`
* **当前活跃 GAP 数量**：`13` 项（`GAP-012`–`GAP-020` 与 `GAP-022`–`GAP-025`，均登记于 `docs/GAP台账.md`）
  * **「活跃」定义**：台账状态**非 `CLOSED`** 的条目（即「未解除」）。
  * **台账总数**：`24` 项；其中 `GAP-001`–`GAP-011` 共 **11** 项为 `CLOSED`，其余 **13** 项为 `IN_PROGRESS`。
  * **`GAP-021` 不在表内**：该编号已撤销（经复核为误判），**编号保留、不复用**，台账中仅留撤销说明。
  * **关联 Issue 分布**：`GAP-012`–`GAP-015` 关联 `#14`；`GAP-016` 关联 `#15`；`GAP-017` 关联 `#16`；`GAP-018`–`GAP-025` 尚无 Issue。

**本节读数的复核命令**（在工作区根执行，实测口径 D1）：

| 读数 | 命令 |
| :--- | :--- |
| 分支 / 提交 | `git rev-parse --abbrev-ref HEAD` · `git rev-parse --short HEAD` |
| 自基底起的提交数 | `git rev-list --count 4fa68f9..HEAD` |
| 是否已推送 | `git rev-list --count 'origin/fix/issue-14-order-module..HEAD'`（0 = 已全部推送） |
| 历史提交数 | `git rev-list --count HEAD` |
| 受管文件数 | `(git ls-files \| Measure-Object -Line).Lines`（**注意**：`Measure-Object -Line` 计的是**非空行**；数文件请用 `(git ls-files).Count`） |
| 未提交改动数 | `git status --porcelain`（行数 = 已改 + 未跟踪；`^ ?M` 计已改，`^\?\?` 计未跟踪，`^[AMD]` 计暂存） |
| 活跃 GAP 数 | 取 `docs/GAP台账.md`「质量缺口登记表」中**状态列非 `CLOSED`** 的行数 |
| 台账总数 | 取同一张表的**全部数据行数**（撤销的编号不占行） |
| CI 是否绿 | `gh run list --branch <branch> --limit 5 --json databaseId,conclusion,headSha` |
| 采集时点 | `[DateTime]::UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ")` |

---

## 5. 运行态隔离策略

* 运行态文件（`memory.md`、`record.md`、`project-progress.html`）属 AI 运行态私有产物，**不得提交入 Git 仓库**，由 `.gitignore` 机械阻断。
