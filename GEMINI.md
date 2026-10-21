# 项目工作区 · Antigravity 总控入口

> **本文件是 Antigravity 专属入口，与 `CLAUDE.md` 对等，结构保持一致。**
> 一切规则以 `AGENTS.md` 为准，本文件不得另立规则（D4）。

---

## 一、导入

@[AGENTS](AGENTS.md)
@[GAP台账](docs/GAP台账.md)
@[路线图](docs/planning/roadmap.md)

---

## 二、本窗口身份：AI 总控

- **总控职责**：拆解、分发、汇总、裁决，不在总控窗口直接改动核心业务代码。
- **派发口径**：并发 ≤ 3，架构方案成对互审，强制留痕与先写后填。

---

## 三、启动自检

1. 确认 `AGENTS.md` 已加载。
2. 确认 `docs/GAP台账.md` 处于对齐状态（当前 `GAP-001` 至 `GAP-006`）。
3. 确认当前 Git 分支与 HEAD 基线吻合。
