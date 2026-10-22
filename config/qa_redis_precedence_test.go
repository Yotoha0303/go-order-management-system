package config

// Independent (03-ring) verification of GAP-014 precedence, through the real
// startup entry point LoadConfig (file + environment + Validate), not through the
// internal helper alone.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// qaConfigYML is a valid configuration whose redis block deliberately carries a
// stale localhost endpoint plus credentials, so the "URL must win" rule has
// something concrete to discard.
const qaConfigYML = `
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
  url: ""
  addr: 127.0.0.1:6379
  username: yaml-user
  password: yaml-password
  db: 7
  tls: false
inventoryReconcile:
  enabled: false
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

func qaLoadConfig(t *testing.T, env map[string]string) (*Config, error) {
	t.Helper()
	for _, key := range []string{
		"REDIS_URL", "REDIS_ADDR", "REDIS_USERNAME",
		"REDIS_PASSWORD", "REDIS_DB", "REDIS_TLS",
	} {
		t.Setenv(key, "")
	}
	for key, value := range env {
		t.Setenv(key, value)
	}

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(qaConfigYML), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return LoadConfig(path)
}

// TestQARedisURLDiscardsEveryConfigFileField: URL is the endpoint's single
// source of truth. Every discrete YAML value must be gone, including the
// credentials and the non-zero db, or a cloud deployment would inherit
// localhost credentials without any error.
func TestQARedisURLDiscardsEveryConfigFileField(t *testing.T) {
	cfg, err := qaLoadConfig(t, map[string]string{
		"REDIS_URL": "rediss://cloud-user:cloud-pass@cloud-host:6380/3",
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Redis.URL != "rediss://cloud-user:cloud-pass@cloud-host:6380/3" {
		t.Fatalf("url=%q", cfg.Redis.URL)
	}
	for name, got := range map[string]string{
		"addr":     cfg.Redis.Addr,
		"username": cfg.Redis.Username,
		"password": cfg.Redis.Password,
	} {
		if got != "" {
			t.Fatalf("config.yml %s must be discarded when REDIS_URL is set, got %q", name, got)
		}
	}
	if cfg.Redis.DB != 0 || cfg.Redis.TLS {
		t.Fatalf("config.yml db/tls must be discarded, got db=%d tls=%v", cfg.Redis.DB, cfg.Redis.TLS)
	}
}

// TestQARedisDiscreteEnvBeatsURLFieldByField checks each override separately:
// one URL field silently winning over an explicit environment variable would be
// the same class of defect as the original shadowing bug.
func TestQARedisDiscreteEnvBeatsURLFieldByField(t *testing.T) {
	cases := []struct {
		env   map[string]string
		check func(t *testing.T, cfg *Config)
	}{
		{
			env: map[string]string{"REDIS_ADDR": "env-host:6390"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Redis.Addr != "env-host:6390" {
					t.Fatalf("REDIS_ADDR lost to URL: %q", cfg.Redis.Addr)
				}
			},
		},
		{
			env: map[string]string{"REDIS_USERNAME": "env-user"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Redis.Username != "env-user" {
					t.Fatalf("REDIS_USERNAME lost to URL: %q", cfg.Redis.Username)
				}
			},
		},
		{
			env: map[string]string{"REDIS_PASSWORD": "env-pass"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Redis.Password != "env-pass" {
					t.Fatalf("REDIS_PASSWORD lost to URL: %q", cfg.Redis.Password)
				}
			},
		},
		{
			env: map[string]string{"REDIS_DB": "9"},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Redis.DB != 9 {
					t.Fatalf("REDIS_DB lost to URL: %d", cfg.Redis.DB)
				}
			},
		},
		{
			env: map[string]string{"REDIS_TLS": "true"},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Redis.TLS {
					t.Fatal("REDIS_TLS lost to URL")
				}
			},
		},
	}
	for _, tc := range cases {
		env := map[string]string{"REDIS_URL": "rediss://url-user:url-pass@url-host:6380/3"}
		for k, v := range tc.env {
			env[k] = v
		}
		cfg, err := qaLoadConfig(t, env)
		if err != nil {
			t.Fatalf("%v: load config: %v", tc.env, err)
		}
		tc.check(t, cfg)
	}
}

// TestQARedisTLSInvalidFailsStartup is the fail-closed contract from GAP-014
// DoD-4: a malformed boolean must abort LoadConfig. The negative half is that it
// must not be treated as false (silently plaintext).
func TestQARedisTLSInvalidFailsStartup(t *testing.T) {
	for _, bad := range []string{"maybe", "yes-please", "2", "TRUE "} {
		_, err := qaLoadConfig(t, map[string]string{"REDIS_TLS": bad})
		if err == nil {
			t.Fatalf("REDIS_TLS=%q must fail startup, got nil error", bad)
		}
		if !strings.Contains(err.Error(), "invalid REDIS_TLS") {
			t.Fatalf("REDIS_TLS=%q error must name the variable, got %v", bad, err)
		}
		t.Logf("REDIS_TLS=%q -> %v", bad, err)
	}
}

// TestQARedisTLSInvalidStillFailsWithURLSet covers the ordering trap: URL resets
// the struct, so a malformed flag applied afterwards must still surface.
func TestQARedisTLSInvalidStillFailsWithURLSet(t *testing.T) {
	_, err := qaLoadConfig(t, map[string]string{
		"REDIS_URL": "rediss://u:p@host:6380/0",
		"REDIS_TLS": "not-a-bool",
	})
	if err == nil {
		t.Fatal("REDIS_TLS must be validated after the URL reset, got nil error")
	}
	if !strings.Contains(err.Error(), "invalid REDIS_TLS") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestQARedisNoURLKeepsConfigFileValues is the complement: without REDIS_URL the
// YAML block must be honoured untouched (the fix must not have broken the
// plain local setup).
func TestQARedisNoURLKeepsConfigFileValues(t *testing.T) {
	cfg, err := qaLoadConfig(t, nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Redis.Addr != "127.0.0.1:6379" || cfg.Redis.Username != "yaml-user" ||
		cfg.Redis.Password != "yaml-password" || cfg.Redis.DB != 7 {
		t.Fatalf("config.yml redis block must survive without REDIS_URL: %+v", cfg.Redis)
	}
}
