# 变更记录：013-fix-gap-014-redis-cloud-tls-config

- **变更编号**：013
- **创建时间**：2026-10-22T00:00:00Z (UTC)
- **完成时间**：2026-10-22T03:50:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联缺陷**：GAP-014 (P1: 云端 Redis 凭据与 TLS 支持缺失，Issue #14)
- **关联 ADR**：ADR-001（Redis 前置预扣防线不可旁路）
- **状态**：已完成（门禁 5/5 通过，负向证伪通过）

---

## 1. 变更动机与背景

云托管 Redis 通常以 `rediss://`（TLS）暴露并强制凭据。当前实现：

- `config.RedisConfig` 只有 `Addr` / `DB`，**没有** `Username` / `Password` / `TLS` / `URL`；
- `applyEnvOverrides` 只解析 `REDIS_ADDR` / `REDIS_DB`；
- `pkg/redis/redis.go:24` **绕过配置结构**直接读 `os.Getenv("REDIS_PASSWORD")`（违反 D4 单一事实源）；
- `openRedisClient` 无 TLS、无 Username，仅明文 TCP；
- `internal/app/deps.go:51-54` 连接失败仅 `Warn` 后继续，`inventoryStockGuard` / `productCache`
  以 nil 客户端降级（`internal/bizcache/*` 各方法判 nil 后静默返回）。

结果是：**服务能启动、接口能响应，但首道预扣防线与商品缓存在云端被静默旁路**（D3「失败即静默」）。

## 2. 根因（实测定位）

| 位置 | 实测内容 |
| --- | --- |
| `config/config.go:43-46`（改前） | `RedisConfig{Addr string; DB int}`，字段缺失 |
| `config/config.go:135-144`（改前） | 仅 `REDIS_ADDR` / `REDIS_DB` 覆盖 |
| `pkg/redis/redis.go:24`（改前） | `password := os.Getenv("REDIS_PASSWORD")` |
| `pkg/redis/redis.go:42-48`（改前） | `redis.Options{Addr, Password, DB}`，无 `Username` / `TLSConfig` |
| `internal/app/deps.go:51-54`（改前） | `logger.Warn("redis unavailable, product cache disabled")` 后继续 |

**依赖侧实测**（`go-redis v9.19.0` `options.go` 的 `setupTCPConn`）：`rediss://` 由 `ParseURL`
自动产出 `&tls.Config{ServerName: h, MinVersion: tls.VersionTLS12}`；`redis://` 不设 TLS。
本实现沿用该行为，并额外为裸 `addr` + `tls: true` 的组合自建同等配置。

## 3. 变更实施清单

1. `config/config.go`
   - `RedisConfig` 增加 `URL`、`Username`、`Password`、`TLS`（yaml：`url`/`username`/`password`/`tls`）；
   - 新增 `applyRedisEnvOverrides`：`REDIS_URL` / `REDIS_ADDR` / `REDIS_USERNAME` / `REDIS_PASSWORD` /
     `REDIS_DB` / `REDIS_TLS`；`REDIS_TLS` 解析失败**返回错误**（fail-closed）。
   - **优先级（复核后修正）**：`REDIS_URL` 存在时**整体重置** `RedisConfig`（`*cfg = RedisConfig{URL: v}`），
     即 `config.yml` 里的 `addr/username/password/db/tls` **全部作废**；随后再依次应用离散**环境变量**覆盖。
     详见 §4。
2. `pkg/redis/redis.go`（整文件重写）
   - `BuildOptions(*config.RedisConfig)`：`redis.ParseURL` 处理 `rediss://` / `redis://`；
     离散字段覆盖；`TLS=true` 时补 `TLSConfig`（`ServerName` 取地址主机、`MinVersion=TLS1.2`）；
   - 移除 `os.Getenv("REDIS_PASSWORD")`（D4：配置成为唯一事实源）；
   - 签名保持 `InitRedis(cfg *config.Config) (*redis.Client, error)`；
   - 哨兵错误分类：`ErrRedisConfigMissing` / `ErrRedisAddrNotFound` / `ErrRedisURLInvalid` /
     `ErrRedisTLSHandshake` / `ErrRedisAuthFailed` / `ErrRedisConnectFailed`，
     `InitRedis` 以 `%w: %w` 同时保留分类与原错误文本；
   - PING 失败时 **关闭客户端**（改前泄漏连接池）。
3. `internal/app/deps.go`
   - 成功：`logger.Info("redis connected", "addr", ...)`；
   - 失败：`logger.Error(... "inventory pre-deduction (Lua) and product cache are degraded to pass-through bypass")`，
     日志级别从 `Warn` 提升为 `Error`（GAP-014 DoD-3 要求「失败必须显式可见」）；
   - 新增未导出 `redisEndpointForLog`：只输出 scheme + host，**剥离凭据**。
4. 配置样例
   - `.env.example`：`REDIS_URL` / `REDIS_ADDR` / `REDIS_USERNAME` / `REDIS_PASSWORD` / `REDIS_TLS` 及注释；
   - `config.yml`：`redis:` 段补 `url` / `username` / `password` / `tls`（全为空/默认，**无真实凭据**）；
   - `README.md`：配置示例段同步（保持文档与样例同成熟度）。
5. 单测
   - `config/redis_config_test.go`（新建，`package config` 内测未导出函数），含 §4 的劫持回归用例；
   - `pkg/redis/redis_test.go`（新建，只测 `BuildOptions` / `TLSConfig` / `classifyConnectionError`，**零网络**），
     含**端到端**回归用例（真实临时 `config.yml` + `REDIS_URL` → `LoadConfig` → `BuildOptions`）；
   - `internal/app/deps_test.go`（新建，覆盖日志脱敏，含 URL-only 形态）。

## 4. 复核发现的缺陷：`config.yml` 的 `addr` 会劫持 `REDIS_URL`（已修）

### 4.1 缺陷描述

首版实现里 `applyRedisEnvOverrides` 只做 `cfg.URL = v`，而 `pkg/redis/BuildOptions` 随后无条件执行
`if cfg.Addr != "" { opts.Addr = cfg.Addr }`。仓内 `config.yml` **自带** `addr: 127.0.0.1:6379`（非空），
于是真实云端部署（只设 `REDIS_URL`、不改 `config.yml`）会得到：

```
opts.Addr = "127.0.0.1:6379"      // 被 config.yml 覆盖
opts.TLSConfig = rediss 的 TLS 配置 // 仍在，但已指向 localhost
```

**后果**：云端连不上、日志显示在连 `127.0.0.1`、TLS 与主机不匹配。**与待修的缺陷同形**——静默地错，
且首版注释还声称「stale addr 不会 shadow 云端 URL」，**文档与行为不一致**（D4）。

### 4.2 修法（方案 A：在配置层解决）

```go
if v := os.Getenv("REDIS_URL"); v != "" {
    // 连接串是端点的唯一事实源：丢弃 config.yml 的全部离散字段，
    // 否则仓内默认 `addr: 127.0.0.1:6379` 会静默替换云端端点。
    // 其后的离散环境变量仍可重新填充单个字段。
    *cfg = RedisConfig{URL: v}
}
```

这样实现的语义与三处文档（`config.yml` 注释、`.env.example`、`RedisConfig` 注释）**逐字一致**：
URL 为基线且**忽略配置文件离散字段**；离散**环境变量**可覆盖 URL 的对应字段。

未选方案 B（用来源标记区分「值来自 yaml / 环境变量」）：需要新增字段与整套来源追踪，改动面大且更容易漂移。

### 4.3 RED 输出（修复前，真实失败）

```
=== RED (pre-fix): regression cases against current code ===
--- FAIL: TestApplyRedisEnvOverridesURLDiscardsConfigFileFields (0.00s)
    redis_config_test.go:41: config.yml addr must be discarded when REDIS_URL is set, got "127.0.0.1:6379"
FAIL
FAIL	go-order-management-system/config	0.583s
--- FAIL: TestLoadConfigThenBuildOptionsKeepsCloudURL (0.01s)
    redis_test.go:100: addr="127.0.0.1:6379", want cloud-host:6380 (config.yml addr must not hijack REDIS_URL)
FAIL
FAIL	go-order-management-system/pkg/redis	0.772s
red exit=1
```

第二条是**端到端**复现：真实临时 `config.yml`（与仓内同形）+ `REDIS_URL=rediss://user:pass@cloud-host:6380/5`
→ `config.LoadConfig` → `BuildOptions`，得到的 `addr` 就是 `127.0.0.1:6379`。

### 4.4 GREEN 输出（修复后，真实通过）

```
=== RUN   TestApplyRedisEnvOverridesURLDiscardsConfigFileFields
--- PASS: TestApplyRedisEnvOverridesURLDiscardsConfigFileFields (0.00s)
=== RUN   TestApplyRedisEnvOverridesDiscreteEnvBeatsURL
--- PASS: TestApplyRedisEnvOverridesDiscreteEnvBeatsURL (0.00s)
PASS
ok  	go-order-management-system/config	0.607s
=== RUN   TestLoadConfigThenBuildOptionsKeepsCloudURL
--- PASS: TestLoadConfigThenBuildOptionsKeepsCloudURL (0.01s)
=== RUN   TestLoadConfigThenBuildOptionsDiscreteEnvBeatsURL
--- PASS: TestLoadConfigThenBuildOptionsDiscreteEnvBeatsURL (0.00s)
PASS
ok  	go-order-management-system/pkg/redis	0.837s
green exit=0
```

三条用例的具体断言：
1. **劫持回归**：`Addr="127.0.0.1:6379"` + `DB=0`（config.yml 形态）+ `REDIS_URL=rediss://user:pass@cloud-host:6380/5`
   → `opts.Addr == "cloud-host:6380"`、`opts.DB == 5`、`TLSConfig != nil`、`Username/Password` 保留；
   配置层另断言 `cfg.Redis.Addr == ""` 且 `cfg.Redis.DB == 0`（yaml 值已被丢弃）。
2. **离散环境变量仍覆盖 URL**：URL + `REDIS_ADDR=other-host:6379` → `Addr == "other-host:6379"`（配置层与端到端各一条）。
3. **既有用例保持通过**：URL / 离散 / 非法布尔 / 非法 DB / 纯构造 / 错误分类全部 PASS。

### 4.5 日志脱敏复核

`redisEndpointForLog` 优先读 `cfg.Redis.Addr`；修完方案 A 后 URL 场景下该字段为**空**，逻辑自然回落到 URL 分支。
新增 `TestRedisEndpointForLogStripsCredentialsFromURLOnlyConfig` 固化该形态：
`rediss://default:sup3r-s3cret@redis.example.com:6380/0` → 输出 `rediss://redis.example.com:6380`，
断言不含密码子串、也不含 `default@`。

## 5. 兼容性说明

- `compose.yml` / `.github/workflows/ci.yml` 未改：它们只设 `REDIS_ADDR`（及可选 `REDIS_PASSWORD`），
  新逻辑下仍走离散字段分支，行为不变。这一点由 `go test ./...` 中依赖 Redis 的
  `internal/bizcache`、`internal/dao` 等包（在 `RUN_REDIS_TEST=1` 的 CI 中）继续通过来保证。
- **唯一的语义变更**：`REDIS_PASSWORD` 现在经由 `config` 结构传递，而不是在 `pkg/redis` 直接读环境变量。
  对调用方**不可见**（`InitRedis` 签名未变），但依赖 `config.LoadConfig` 被调用过——仓内两个调用点
  （`internal/app/deps.go:41`、`internal/bizcache/test_helper_test.go:19`）都满足。

## 6. 实测验证日志（D1：真实 stdout 回填）

> 本节与 §7 均为**修复 §4 缺陷之后**的最终复跑结果。

采集时间：2026-10-22T04:10Z 前后（UTC），工作树 `fix/issue-14-order-module`。

### 6.1 针对性用例（含 `-race`）

```text
PS> go test -count=1 -v -run "TestApplyRedis|TestBuildOptions|TestLoadConfigThenBuildOptions|TestClassify|TestTLSConfig|TestRedisEndpointForLog" ./config/... ./pkg/redis/... ./internal/app/...
=== RUN   TestApplyRedisEnvOverridesURLDiscardsConfigFileFields
--- PASS: TestApplyRedisEnvOverridesURLDiscardsConfigFileFields (0.00s)
=== RUN   TestApplyRedisEnvOverridesDiscreteEnvBeatsURL
--- PASS: TestApplyRedisEnvOverridesDiscreteEnvBeatsURL (0.00s)
=== RUN   TestApplyRedisEnvOverridesURL
--- PASS: TestApplyRedisEnvOverridesURL (0.00s)
=== RUN   TestApplyRedisEnvOverridesDiscreteFields
--- PASS: TestApplyRedisEnvOverridesDiscreteFields (0.00s)
=== RUN   TestApplyRedisEnvOverridesRejectsInvalidTLS
--- PASS: TestApplyRedisEnvOverridesRejectsInvalidTLS (0.00s)
=== RUN   TestApplyRedisEnvOverridesRejectsInvalidDB
--- PASS: TestApplyRedisEnvOverridesRejectsInvalidDB (0.00s)
PASS
ok  	go-order-management-system/config	0.607s
=== RUN   TestLoadConfigThenBuildOptionsKeepsCloudURL
--- PASS: TestLoadConfigThenBuildOptionsKeepsCloudURL (0.01s)
=== RUN   TestLoadConfigThenBuildOptionsDiscreteEnvBeatsURL
--- PASS: TestLoadConfigThenBuildOptionsDiscreteEnvBeatsURL (0.00s)
=== RUN   TestBuildOptionsRedissURL
--- PASS: TestBuildOptionsRedissURL (0.00s)
=== RUN   TestBuildOptionsPlainURL
--- PASS: TestBuildOptionsPlainURL (0.00s)
=== RUN   TestBuildOptionsDiscreteTLS
--- PASS: TestBuildOptionsDiscreteTLS (0.00s)
=== RUN   TestBuildOptionsDiscreteOverridesURL
--- PASS: TestBuildOptionsDiscreteOverridesURL (0.00s)
=== RUN   TestBuildOptionsRequiresEndpoint
--- PASS: TestBuildOptionsRequiresEndpoint (0.00s)
=== RUN   TestBuildOptionsRejectsUnsupportedScheme
--- PASS: TestBuildOptionsRejectsUnsupportedScheme (0.00s)
=== RUN   TestClassifyConnectionError
    --- PASS: TestClassifyConnectionError/tls_handshake (0.00s)
    --- PASS: TestClassifyConnectionError/tls_record (0.00s)
    --- PASS: TestClassifyConnectionError/auth_wrongpass (0.00s)
    --- PASS: TestClassifyConnectionError/auth_noauth (0.00s)
    --- PASS: TestClassifyConnectionError/network (0.00s)
--- PASS: TestClassifyConnectionError (0.00s)
=== RUN   TestClassifyConnectionErrorKeepsCategoriesDistinct
--- PASS: TestClassifyConnectionErrorKeepsCategoriesDistinct (0.00s)
=== RUN   TestTLSConfigWithoutPort
--- PASS: TestTLSConfigWithoutPort (0.00s)
PASS
ok  	go-order-management-system/pkg/redis	0.837s
=== RUN   TestRedisEndpointForLogStripsCredentialsFromURLOnlyConfig
--- PASS: TestRedisEndpointForLogStripsCredentialsFromURLOnlyConfig (0.00s)
=== RUN   TestRedisEndpointForLogStripsCredentials
    --- PASS: TestRedisEndpointForLogStripsCredentials/url_form_never_echoes_password (0.00s)
    --- PASS: TestRedisEndpointForLogStripsCredentials/addr_form_wins_when_both_present (0.00s)
    --- PASS: TestRedisEndpointForLogStripsCredentials/unparseable_url (0.00s)
    --- PASS: TestRedisEndpointForLogStripsCredentials/nil_config (0.00s)
--- PASS: TestRedisEndpointForLogStripsCredentials (0.00s)
PASS
ok  	go-order-management-system/internal/app	1.328s
green exit=0
```

`TestBuildOptionsRedissURL` 断言的具体值：`TLSConfig != nil`、`MinVersion == tls.VersionTLS12`、
`ServerName == "redis.example.com"`、`Username == "user"` / `Password == "pass"`、
`Addr == "redis.example.com:6380"`、`DB == 2`。

### 6.2 五道门禁

```text
PS> go vet ./...
vet exit=0

PS> go test -count=1 ./...
ok  	go-order-management-system/config	1.175s
ok  	go-order-management-system/internal/app	2.665s
ok  	go-order-management-system/internal/handler	3.793s
ok  	go-order-management-system/pkg/redis	1.021s
...（全部 ok，无 FAIL）
test exit=0

PS> go test -count=1 -race ./...
ok  	go-order-management-system/config	1.681s
ok  	go-order-management-system/internal/app	4.073s
ok  	go-order-management-system/internal/handler	4.386s
ok  	go-order-management-system/pkg/redis	1.811s
...（全部 ok，无 DATA RACE）
race exit=0

PS> golangci-lint run ./...
0 issues.
lint exit=0

PS> goose -dir migrations validate
goose exit=0

PS> go build ./...
build exit=0
```

> 过程记录：首轮 lint 报 1 条 `errorlint`（测试里用 `==` 比较哨兵错误）。
> 已改为 `errors.Is` 后复跑，输出 `0 issues.`（真实输出见上）。

## 7. 负向证伪检验（D3 实际输出）

**变异 2：同时做两处退化** —— ① `REDIS_TLS` 解析错误被吞掉（`cfg.TLS, _ = strconv.ParseBool(v)`）；
② `BuildOptions` 的 TLS 分支被禁用（`if cfg.TLS && false`）。

```text
=== MUTATION 2: REDIS_TLS parse error ignored + TLS branch disabled ===
--- FAIL: TestApplyRedisEnvOverridesRejectsInvalidTLS (0.00s)
    redis_config_test.go:64: expected error for REDIS_TLS=not-a-bool, got nil (cfg.Redis.TLS=false)
FAIL
FAIL	go-order-management-system/config	0.578s
--- FAIL: TestBuildOptionsDiscreteTLS (0.00s)
    redis_test.go:61: tls: true must produce a TLS config
FAIL
FAIL	go-order-management-system/pkg/redis	0.662s
mutation2 exit=1
```

两条判据各有独立的「牙」：非法布尔**不是**被默默当作 `false`，TLS 分支被摘除时**必定**红灯。
变异已还原，还原后五道门禁复跑全绿（§6.2）。

**常驻负向用例**：
- `TestClassifyConnectionErrorKeepsCategoriesDistinct`：TLS / 认证 / 网络三类必须落到**不同**哨兵；
  若分类塌缩成一类，云端排障就分辨不出「换证书」还是「换防火墙」。
- `TestRedisEndpointForLogStripsCredentials`：`rediss://default:sup3r-s3cret@host:6380/0`
  必须输出 `rediss://redis.example.com:6380`，且不得包含密码子串。

## 8. 下游影响与未竟项

- **`sre-operator` 需知会**：云端部署现在应设 `REDIS_URL=rediss://…`（或 `REDIS_ADDR` + `REDIS_TLS=true`
  + `REDIS_USERNAME`/`REDIS_PASSWORD`）。启动日志会出现 `redis connected`（Info）或
  `redis connection failed … degraded to pass-through bypass`（Error）。
- **未做（超出本次授权 / 需要外部环境）**：
  1. **未对真实云 Redis 做 TLS 握手实测**——无云端凭据。已用纯构造/解析单测覆盖 TLS 分支；
     真实握手需 `sre-operator` 或用户在云端验证。这是一条**未闭环**项，不得当作已验证。
  2. **`pkg/redis` 无集成测试文件**——只在 `internal/bizcache/test_helper_test.go` 有 Redis 集成入口，
     本次未新增（避免引入 `RUN_REDIS_TEST` 依赖导致 CI 行为变化）。
  3. **降级仍是「继续启动」**——本次只把失败日志提升为 `Error` 并写明旁路后果，**未**改为 fail-fast。
     是否要 fail-fast 属架构决策（会改变「Redis 不可用时服务是否可启动」），需 `system-architect` /
     `sre-operator` 裁定，本角色不擅自决定。
     > **后续裁定（`2026-10-22`）**：**用户裁定「维持降级启动」，不引入 fail-fast**（经总控征询）。
     > 故本条**已不是未决项**；该裁定已落痕于 `docs/GAP台账.md` 的 `GAP-014` 状态段（含依据与残留义务）。
     > **残留义务不变**：既然不 fail-fast，则「Redis 连接失败必须显式可见」是这条裁定的**前提**——
     > 即上文 §8 第 1 条的两条日志（`redis connected` Info / `redis connection failed … degraded to
     > pass-through bypass` Error）**不可删减**，`GAP-014` 的 DoD ③ 因此继续有效。
  4. `compose.yml` / `.env.example` 之外的样例文件（`docs/old/README.old.md`）未同步——`docs/old/` 是历史快照，
     按纪律不改写。
