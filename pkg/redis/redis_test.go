package redis

import (
	"crypto/tls"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go-order-management-system/config"
)

// BuildOptions is pure: no connection is opened, so these tests need no Redis
// server and are safe in CI without the redis service.

// configYMLWithRedisAddr is the shipped config.yml shape: redis.addr is
// populated, so any deployment that only sets REDIS_URL inherits a localhost
// address unless the config layer discards it.
const configYMLWithRedisAddr = `
server:
  port: 8082
mysql:
  user: root
  host: 127.0.0.1
  port: "3306"
  database: go_order_management_system
  maxOpenConns: 10
  maxIdleConns: 5
  connMaxLifeTime: 30m
  connMaxIdleTime: 5m
  pingTimeout: 3s
  slowThreshold: 200ms
  logLevel: warn
redis:
  addr: 127.0.0.1:6379
  db: 0
inventoryReconcile:
  enabled: true
  interval: 5m
  timeout: 3s
  autoFix: true
rabbitmq:
  url: amqp://order_app:order_dev_password@127.0.0.1:5672/
  connectTimeout: 5s
  reconnectDelay: 5s
  orderTimeout:
    delay: 30m
    outboxPollInterval: 1s
    outboxRetryDelay: 5s
    publishBatchSize: 100
    consumerPrefetch: 10
jwt:
  expireHours: 24
http:
  server:
    readTimeout: 5s
    writeTimeout: 20s
    idleTimeout: 60s
    readHeaderTimeout: 2s
    maxHeaderBytesKib: 128
    timeout: 18s
`

func loadConfigWithRedisURL(t *testing.T, redisURL string, extraEnv ...string) *config.Config {
	t.Helper()
	for _, key := range []string{
		"REDIS_URL", "REDIS_ADDR", "REDIS_USERNAME",
		"REDIS_PASSWORD", "REDIS_DB", "REDIS_TLS",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("REDIS_URL", redisURL)
	for i := 0; i+1 < len(extraEnv); i += 2 {
		t.Setenv(extraEnv[i], extraEnv[i+1])
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(configYMLWithRedisAddr), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	cfg, err := config.LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

// Regression for the shadowing defect, exercised through the real runtime path
// (config.yml + REDIS_URL -> LoadConfig -> BuildOptions). Before the fix the
// stale `addr: 127.0.0.1:6379` survived and replaced the cloud endpoint, so the
// service dialled localhost in plaintext while believing it had TLS.
func TestLoadConfigThenBuildOptionsKeepsCloudURL(t *testing.T) {
	cfg := loadConfigWithRedisURL(t, "rediss://user:pass@cloud-host:6380/5")

	opts, err := BuildOptions(&cfg.Redis)
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.Addr != "cloud-host:6380" {
		t.Fatalf("addr=%q, want cloud-host:6380 (config.yml addr must not hijack REDIS_URL)", opts.Addr)
	}
	if opts.DB != 5 {
		t.Fatalf("db=%d, want 5 (config.yml db must not hijack REDIS_URL)", opts.DB)
	}
	if opts.TLSConfig == nil {
		t.Fatal("rediss:// must keep TLS after the config layer ran")
	}
	if opts.Username != "user" || opts.Password != "pass" {
		t.Fatalf("credentials from URL lost: username=%q", opts.Username)
	}
}

// Discrete environment variables must still override the URL end to end.
func TestLoadConfigThenBuildOptionsDiscreteEnvBeatsURL(t *testing.T) {
	cfg := loadConfigWithRedisURL(t,
		"rediss://user:pass@cloud-host:6380/5",
		"REDIS_ADDR", "other-host:6379",
	)

	opts, err := BuildOptions(&cfg.Redis)
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.Addr != "other-host:6379" {
		t.Fatalf("addr=%q, want other-host:6379", opts.Addr)
	}
}

func TestBuildOptionsRedissURL(t *testing.T) {
	opts, err := BuildOptions(&config.RedisConfig{URL: "rediss://user:pass@redis.example.com:6380/2"})
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.TLSConfig == nil {
		t.Fatal("rediss:// must produce a TLS config")
	}
	if opts.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion=%d, want TLS 1.2", opts.TLSConfig.MinVersion)
	}
	if opts.TLSConfig.ServerName != "redis.example.com" {
		t.Fatalf("ServerName=%q", opts.TLSConfig.ServerName)
	}
	if opts.Username != "user" || opts.Password != "pass" {
		t.Fatalf("credentials not parsed: username=%q", opts.Username)
	}
	if opts.Addr != "redis.example.com:6380" {
		t.Fatalf("addr=%q", opts.Addr)
	}
	if opts.DB != 2 {
		t.Fatalf("db=%d", opts.DB)
	}
}

func TestBuildOptionsPlainURL(t *testing.T) {
	opts, err := BuildOptions(&config.RedisConfig{URL: "redis://127.0.0.1:6379/0"})
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.TLSConfig != nil {
		t.Fatal("redis:// must not enable TLS")
	}
}

func TestBuildOptionsDiscreteTLS(t *testing.T) {
	opts, err := BuildOptions(&config.RedisConfig{
		Addr:     "redis.cloud.internal:6379",
		Username: "default",
		Password: "s3cret",
		DB:       1,
		TLS:      true,
	})
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.TLSConfig == nil {
		t.Fatal("tls: true must produce a TLS config")
	}
	if opts.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion=%d, want TLS 1.2", opts.TLSConfig.MinVersion)
	}
	if opts.TLSConfig.ServerName != "redis.cloud.internal" {
		t.Fatalf("ServerName=%q", opts.TLSConfig.ServerName)
	}
	if opts.Username != "default" || opts.Password != "s3cret" || opts.DB != 1 {
		t.Fatalf("discrete fields not applied: username=%q db=%d", opts.Username, opts.DB)
	}
}

func TestBuildOptionsDiscreteOverridesURL(t *testing.T) {
	opts, err := BuildOptions(&config.RedisConfig{
		URL:      "redis://base:base@baseline:6379/0",
		Addr:     "override:6380",
		Username: "override-user",
		Password: "override-pass",
		DB:       4,
	})
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.Addr != "override:6380" || opts.Username != "override-user" ||
		opts.Password != "override-pass" || opts.DB != 4 {
		t.Fatalf("discrete overrides not applied: %+v", opts)
	}
}

func TestBuildOptionsRequiresEndpoint(t *testing.T) {
	if _, err := BuildOptions(&config.RedisConfig{}); !errors.Is(err, ErrRedisAddrNotFound) {
		t.Fatalf("expected ErrRedisAddrNotFound, got %v", err)
	}
	if _, err := BuildOptions(nil); !errors.Is(err, ErrRedisConfigMissing) {
		t.Fatalf("expected ErrRedisConfigMissing, got %v", err)
	}
}

func TestBuildOptionsRejectsUnsupportedScheme(t *testing.T) {
	if _, err := BuildOptions(&config.RedisConfig{URL: "http://redis:6379"}); !errors.Is(err, ErrRedisURLInvalid) {
		t.Fatalf("expected ErrRedisURLInvalid, got %v", err)
	}
}

func TestClassifyConnectionError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"tls handshake", errors.New("tls: failed to verify certificate: x509: certificate signed by unknown authority"), ErrRedisTLSHandshake},
		{"tls record", errors.New("remote error: tls: handshake failure"), ErrRedisTLSHandshake},
		{"auth wrongpass", errors.New("WRONGPASS invalid username-password pair or user is disabled."), ErrRedisAuthFailed},
		{"auth noauth", errors.New("NOAUTH Authentication required."), ErrRedisAuthFailed},
		{"network", errors.New("dial tcp 10.0.0.5:6379: i/o timeout"), ErrRedisConnectFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyConnectionError(tc.err); !errors.Is(got, tc.want) {
				t.Fatalf("classify(%v)=%v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// Negative case (D3): the classifier must not collapse every failure into one
// bucket. If a TLS error were reported as a generic connect error, an operator
// debugging a cloud deployment could not tell "fix the certificate" from
// "fix the firewall".
func TestClassifyConnectionErrorKeepsCategoriesDistinct(t *testing.T) {
	tlsErr := classifyConnectionError(errors.New("tls: handshake failure"))
	authErr := classifyConnectionError(errors.New("WRONGPASS invalid password"))
	connErr := classifyConnectionError(errors.New("dial tcp: connection refused"))
	if errors.Is(tlsErr, authErr) || errors.Is(tlsErr, connErr) || errors.Is(authErr, connErr) {
		t.Fatalf("categories collapsed: tls=%v auth=%v conn=%v", tlsErr, authErr, connErr)
	}
	if classifyConnectionError(nil) != nil {
		t.Fatal("nil error must classify to nil")
	}
}

func TestTLSConfigWithoutPort(t *testing.T) {
	tlsCfg := TLSConfig("redis.example.com")
	if tlsCfg.ServerName != "redis.example.com" {
		t.Fatalf("ServerName=%q", tlsCfg.ServerName)
	}
}
