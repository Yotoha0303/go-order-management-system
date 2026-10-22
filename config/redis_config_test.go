package config

import (
	"strings"
	"testing"
)

// configYMLRedisShape mirrors the shipped config.yml: a non-empty addr and an
// implicit db=0. Any deployment that sets only REDIS_URL inherits these values.
func configYMLRedisShape() RedisConfig {
	return RedisConfig{Addr: "127.0.0.1:6379", DB: 0}
}

func clearRedisEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"REDIS_URL", "REDIS_ADDR", "REDIS_USERNAME",
		"REDIS_PASSWORD", "REDIS_DB", "REDIS_TLS",
	} {
		t.Setenv(key, "")
	}
}

// Regression for the shadowing defect: a stale addr from config.yml must not
// survive when REDIS_URL is supplied. Before the fix, cfg.Redis.Addr stayed
// "127.0.0.1:6379" and pkg/redis overwrote the cloud endpoint with it.
func TestApplyRedisEnvOverridesURLDiscardsConfigFileFields(t *testing.T) {
	clearRedisEnv(t)
	t.Setenv("REDIS_URL", "rediss://user:pass@cloud-host:6380/5")

	cfg := validConfig()
	cfg.Redis = configYMLRedisShape()
	if err := applyEnvOverrides(&cfg); err != nil {
		t.Fatalf("apply environment: %v", err)
	}

	if cfg.Redis.URL != "rediss://user:pass@cloud-host:6380/5" {
		t.Fatalf("url=%q", cfg.Redis.URL)
	}
	if cfg.Redis.Addr != "" {
		t.Fatalf("config.yml addr must be discarded when REDIS_URL is set, got %q", cfg.Redis.Addr)
	}
	if cfg.Redis.DB != 0 {
		t.Fatalf("config.yml db must be discarded when REDIS_URL is set, got %d", cfg.Redis.DB)
	}
	if cfg.Redis.Username != "" || cfg.Redis.Password != "" || cfg.Redis.TLS {
		t.Fatalf("discrete config fields must be discarded: %+v", cfg.Redis)
	}
}

// The discrete environment variables must still win over REDIS_URL.
func TestApplyRedisEnvOverridesDiscreteEnvBeatsURL(t *testing.T) {
	clearRedisEnv(t)
	t.Setenv("REDIS_URL", "rediss://user:pass@cloud-host:6380/5")
	t.Setenv("REDIS_ADDR", "other-host:6379")
	t.Setenv("REDIS_TLS", "true")

	cfg := validConfig()
	cfg.Redis = configYMLRedisShape()
	if err := applyEnvOverrides(&cfg); err != nil {
		t.Fatalf("apply environment: %v", err)
	}

	if cfg.Redis.Addr != "other-host:6379" {
		t.Fatalf("REDIS_ADDR must override the URL, got %q", cfg.Redis.Addr)
	}
	if !cfg.Redis.TLS {
		t.Fatal("REDIS_TLS=true must survive")
	}
}

func TestApplyRedisEnvOverridesURL(t *testing.T) {
	t.Setenv("REDIS_URL", "rediss://user:pass@redis.example.com:6380/0")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_USERNAME", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("REDIS_TLS", "")

	cfg := validConfig()
	cfg.Redis = configYMLRedisShape()
	if err := applyEnvOverrides(&cfg); err != nil {
		t.Fatalf("apply environment: %v", err)
	}
	if cfg.Redis.URL != "rediss://user:pass@redis.example.com:6380/0" {
		t.Fatalf("REDIS_URL not applied, got %q", cfg.Redis.URL)
	}
}

func TestApplyRedisEnvOverridesDiscreteFields(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_ADDR", "redis.cloud:6380")
	t.Setenv("REDIS_USERNAME", "order_app")
	t.Setenv("REDIS_PASSWORD", "s3cret")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("REDIS_TLS", "true")

	cfg := validConfig()
	if err := applyEnvOverrides(&cfg); err != nil {
		t.Fatalf("apply environment: %v", err)
	}
	if cfg.Redis.Addr != "redis.cloud:6380" {
		t.Fatalf("addr=%q", cfg.Redis.Addr)
	}
	if cfg.Redis.Username != "order_app" {
		t.Fatalf("username=%q", cfg.Redis.Username)
	}
	if cfg.Redis.Password != "s3cret" {
		t.Fatalf("password=%q", cfg.Redis.Password)
	}
	if cfg.Redis.DB != 3 {
		t.Fatalf("db=%d", cfg.Redis.DB)
	}
	if !cfg.Redis.TLS {
		t.Fatal("REDIS_TLS=true must enable TLS")
	}
}

// Negative case (D3): a malformed boolean must fail the load, never degrade to
// plaintext silently. If this test starts passing with a nil error, the
// fail-closed judgement is dead.
func TestApplyRedisEnvOverridesRejectsInvalidTLS(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_TLS", "not-a-bool")

	cfg := validConfig()
	err := applyEnvOverrides(&cfg)
	if err == nil {
		t.Fatalf("expected error for REDIS_TLS=not-a-bool, got nil (cfg.Redis.TLS=%v)", cfg.Redis.TLS)
	}
	if !strings.Contains(err.Error(), "invalid REDIS_TLS") {
		t.Fatalf("unexpected error text: %v", err)
	}
	if cfg.Redis.TLS {
		t.Fatal("invalid REDIS_TLS must not silently enable TLS either")
	}
}

func TestApplyRedisEnvOverridesRejectsInvalidDB(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_DB", "one")

	cfg := validConfig()
	if err := applyEnvOverrides(&cfg); err == nil || !strings.Contains(err.Error(), "invalid REDIS_DB") {
		t.Fatalf("expected invalid REDIS_DB error, got %v", err)
	}
}
