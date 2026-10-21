# 变更记录：004-support-mysql-tls-connection

- **变更编号**：004 (云端数据库 TLS 适配专项)
- **创建时间**：2026-10-21T18:05:00Z (UTC)
- **完成时间**：2026-10-21T18:10:00Z (UTC)
- **作者**：`backend-dev` (Go/后端工程专家)
- **关联背景**：Render 云端环境连接 TiDB Cloud Serverless / 云端 MySQL 实例连接握手被拒 (Error 1105: Connections using insecure transport are prohibited)
- **关联 ADR**：ADR-001 (高并发与云原生数据基础设施连接安全规范)
- **状态**：已完成 (全项通过)

---

## 1. 变更动机与背景

在将订单系统服务部署到 Render 云端并连接 TiDB Cloud Serverless 数据库时，应用启动失败并抛出异常：
`Error 1105 (HY000): Connections using insecure transport are prohibited. See https://docs.pingcap.com/tidbcloud/secure-connections-to-serverless-tier-clusters`

经排查根因：
1. `pkg/database/db.go` 与 `pkg/database/testdb.go` 中的 `buildDSN` 方法强制拼装了不带 TLS 参数的 MySQL DSN：
   `%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local`
2. 当前配置模块 `config.MySQLConfig` 缺乏 `TLS` 配置项，且未支持通过 `DB_TLS` 环境变量进行动态注入覆盖；
3. TiDB Cloud Serverless 及众多现代云托管 MySQL 强制要求客户端握手启用安全传输协议（TLS），否则直接拒绝 TCP 连接。

为解决云端连接问题并保持本地开发体验，方案如下：
- 在 `config.MySQLConfig` 增加 `TLS string yaml:"tls"`，并在 `applyEnvOverrides` 中支持 `DB_TLS` 环境变量覆盖；
- 在 `pkg/database/db.go` 的 `buildDSN` 中优先读取配置的 `TLS`，若为空则读取 `os.Getenv("DB_TLS")`；
- 增加智能默认机制：若未显式指定 `DB_TLS`，但检测到 `cfg.MySQL.Host` 包含 `tidbcloud.com`，自动追加 `&tls=true`；
- 在 `pkg/database/db_test.go` 中编写完备单元测试，覆盖本地无 TLS、显式配置 TLS、环境变量覆盖 TLS、以及针对 `tidbcloud.com` 的智能 TLS 默认行为。

---

## 2. 变更实施清单

1. `config/config.go`:
   - `MySQLConfig` 增加 `TLS string yaml:"tls"` 字段；
   - `applyEnvOverrides` 增加 `DB_TLS` 环境变量解析。
2. `config/mysql_config_test.go`:
   - 新增 `TestApplyEnvOverridesMySQLTLSConfig` 单元测试。
3. `pkg/database/db.go`:
   - 修改 `buildDSN` 支持 `&tls=` 参数追加与 `tidbcloud.com` 智能默认检测。
4. `pkg/database/testdb.go`:
   - 修改 `buildTestMySQLDSN`，同步支持 TLS 配置与 `tidbcloud.com` 智能识别。
5. `pkg/database/db_test.go`:
   - 新增针对 `buildDSN` 与 `buildTestMySQLDSN` 的单元测试套件，全面覆盖 6 大场景。

---

## 3. 实测验证日志 (纪律 D1: 真实 stdout 回填)

### 3.1 数据库与配置模块单元测试（带 `-race`）

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go test -v -race ./pkg/database/... ./config/...
=== RUN   TestBuildDSN_LocalWithoutTLS
--- PASS: TestBuildDSN_LocalWithoutTLS (0.00s)
=== RUN   TestBuildDSN_ExplicitConfigTLS
--- PASS: TestBuildDSN_ExplicitConfigTLS (0.00s)
=== RUN   TestBuildDSN_EnvDBTLS
--- PASS: TestBuildDSN_EnvDBTLS (0.00s)
=== RUN   TestBuildDSN_TiDBCloudAutoTLS
--- PASS: TestBuildDSN_TiDBCloudAutoTLS (0.00s)
=== RUN   TestBuildDSN_MissingConfigReturnsError
--- PASS: TestBuildDSN_MissingConfigReturnsError (0.00s)
=== RUN   TestBuildTestMySQLDSN_TLS
--- PASS: TestBuildTestMySQLDSN_TLS (0.00s)
=== RUN   TestGORMLoggerTraceSlowQuery
--- PASS: TestGORMLoggerTraceSlowQuery (0.01s)
=== RUN   TestGORMLoggerTraceError
--- PASS: TestGORMLoggerTraceError (0.00s)
=== RUN   TestGORMLoggerIgnoresRecordNotFoundAtErrorLevel
--- PASS: TestGORMLoggerIgnoresRecordNotFoundAtErrorLevel (0.00s)
=== RUN   TestParseGORMLogLevelDefaultsToWarn
--- PASS: TestParseGORMLogLevelDefaultsToWarn (0.00s)
PASS
ok  	go-order-management-system/pkg/database	1.861s
=== RUN   TestMySQLSlowLogConfigValidate
=== RUN   TestMySQLSlowLogConfigValidate/allows_empty_log_level_with_default
=== RUN   TestMySQLSlowLogConfigValidate/rejects_invalid_slow_threshold
=== RUN   TestMySQLSlowLogConfigValidate/rejects_invalid_log_level
--- PASS: TestMySQLSlowLogConfigValidate (0.00s)
    --- PASS: TestMySQLSlowLogConfigValidate/allows_empty_log_level_with_default (0.00s)
    --- PASS: TestMySQLSlowLogConfigValidate/rejects_invalid_slow_threshold (0.00s)
    --- PASS: TestMySQLSlowLogConfigValidate/rejects_invalid_log_level (0.00s)
=== RUN   TestApplyEnvOverridesMySQLSlowLogConfig
--- PASS: TestApplyEnvOverridesMySQLSlowLogConfig (0.00s)
=== RUN   TestApplyEnvOverridesMySQLTLSConfig
--- PASS: TestApplyEnvOverridesMySQLTLSConfig (0.00s)
=== RUN   TestInventoryReconcileConfigValidate
--- PASS: TestInventoryReconcileConfigValidate (0.00s)
=== RUN   TestApplyEnvOverridesInventoryReconcileConfig
--- PASS: TestApplyEnvOverridesInventoryReconcileConfig (0.00s)
=== RUN   TestRabbitMQAndOrderTimeoutConfigValidate
--- PASS: TestRabbitMQAndOrderTimeoutConfigValidate (0.00s)
=== RUN   TestOrderTimeoutConfigAllowsShortTestDelay
--- PASS: TestOrderTimeoutConfigAllowsShortTestDelay (0.00s)
=== RUN   TestOrderTimeoutConfigRejectsNonPositiveDelay
--- PASS: TestOrderTimeoutConfigRejectsNonPositiveDelay (0.00s)
=== RUN   TestApplyRabbitMQEnvOverridesUsesNestedOrderTimeout
--- PASS: TestApplyRabbitMQEnvOverridesUsesNestedOrderTimeout (0.00s)
=== RUN   TestRabbitMQOrderTimeoutYAMLLayering
--- PASS: TestRabbitMQOrderTimeoutYAMLLayering (0.00s)
=== RUN   TestLoadProjectConfigUsesNestedRabbitMQOrderTimeout
--- PASS: TestLoadProjectConfigUsesNestedRabbitMQOrderTimeout (0.00s)
PASS
ok  	go-order-management-system/config	(cached)
```

### 3.2 静态代码检查

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go vet ./...
# Exit Code 0，全仓零告警
```

### 3.3 全仓单元测试与竞态检测

```powershell
PS D:\MyDocuments\ai-collaboration-framework\go-order-management-system> go test -race ./...
ok  	go-order-management-system/config	2.130s
ok  	go-order-management-system/internal/app	3.590s
ok  	go-order-management-system/internal/auth	(cached)
ok  	go-order-management-system/internal/bizcache	2.017s
ok  	go-order-management-system/internal/dao	2.204s
ok  	go-order-management-system/internal/handler	(cached)
ok  	go-order-management-system/internal/inventoryreconcile	(cached)
ok  	go-order-management-system/internal/loadtest	(cached)
ok  	go-order-management-system/internal/middleware	(cached)
ok  	go-order-management-system/internal/observability	(cached)
ok  	go-order-management-system/internal/ordertimeout	3.562s
ok  	go-order-management-system/internal/service	3.558s
ok  	go-order-management-system/migrations/integration	1.800s
ok  	go-order-management-system/pkg/database	1.770s
ok  	go-order-management-system/pkg/idgen	(cached)
ok  	go-order-management-system/router	(cached)
```

---

## 4. 负向证伪检验 (纪律 D3)

1. **缺省无 TLS 隔离证伪**：
   在 `TestBuildDSN_LocalWithoutTLS` 中，断言普通本地 host（127.0.0.1）且未设置任何 TLS 参数时，生成的 DSN 绝不含 `&tls=`，保证本地无 TLS 开发环境零回归。
2. **必填配置缺失报错证伪**：
   在 `TestBuildDSN_MissingConfigReturnsError` 中故意省略 Host 与 Password，断言 `buildDSN` 立即阻断并返回 `database config missing`，避免生成残缺无效 DSN。
3. **TiDB Cloud 自动防护证伪**：
   在 `TestBuildDSN_TiDBCloudAutoTLS` 中，即使不提供 `DB_TLS`，只要 Host 包含 `tidbcloud.com`，DSN 必须强制自动包含 `&tls=true`，从源头杜绝云端握手被拒事故。
