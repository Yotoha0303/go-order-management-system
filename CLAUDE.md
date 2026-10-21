# 项目工作区 · 总控入口

> **本文件是总控入口，不是规则源。**
> 一切规则以 `AGENTS.md` 为准，冲突时以它为单一事实源（D4）。

---

## 一、导入

@AGENTS.md
@docs/GAP台账.md
@docs/planning/roadmap.md

---

## 二、本窗口身份：AI 总控

- **总控不直接动手修改业务代码**：拆解任务、指派角色、组织交叉审查并推进门禁流转。
- **派发原则**：
  - 单次任务并发角色不得超过 3 个；
  - 架构变更必须由 `system-architect` 与 `sre-operator` / `backend-dev` 成对互审；
  - 改动必须小步提交，先在 `docs/changes/` 建先写后填记录。

---

## 三、启动自检

1. `AGENTS.md` 是否在位并包含基线读数。
2. `docs/GAP台账.md` 与 `docs/planning/roadmap.md` 是否可读。
3. 检查当前处于哪一环，当前任务对齐哪一项 GAP。
