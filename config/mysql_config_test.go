package config

import (
	"errors"
	"testing"
	"time"
)

func TestMySQLSlowLogConfigValidate(t *testing.T) {
	cfg := validConfig()

	t.Run("allows empty log level with default", func(t *testing.T) {
		cfg := cfg
		cfg.MySQL.LogLevel = ""
		if err := cfg.Validate(); err != nil {
			t.Fatalf("empty log level should use default: %v", err)
		}
	})

	t.Run("rejects invalid slow threshold", func(t *testing.T) {
		cfg := cfg
		cfg.MySQL.SlowThreshold = -time.Millisecond
		if err := cfg.Validate(); !errors.Is(err, ErrMySQLInvalidSlowThreshold) {
			t.Fatalf("slow threshold error=%v", err)
		}
	})

	t.Run("rejects invalid log level", func(t *testing.T) {
		cfg := cfg
		cfg.MySQL.LogLevel = "debug"
		if err := cfg.Validate(); !errors.Is(err, ErrMySQLInvalidLogLevel) {
			t.Fatalf("log level error=%v", err)
		}
	})
}

func TestApplyEnvOverridesMySQLSlowLogConfig(t *testing.T) {
	t.Setenv("DB_SLOW_THRESHOLD", "350ms")
	t.Setenv("DB_LOG_LEVEL", "info")

	cfg := validConfig()
	if err := applyEnvOverrides(&cfg); err != nil {
		t.Fatalf("apply environment: %v", err)
	}
	if cfg.MySQL.SlowThreshold != 350*time.Millisecond {
		t.Fatalf("slow threshold=%s", cfg.MySQL.SlowThreshold)
	}
	if cfg.MySQL.LogLevel != "info" {
		t.Fatalf("log level=%q", cfg.MySQL.LogLevel)
	}
}

func TestInventoryReconcileConfigValidate(t *testing.T) {
	cfg := validConfig()

	t.Run("allows disabled empty config", func(t *testing.T) {
		cfg := cfg
		cfg.InventoryReconcile = InventoryReconcileConfig{Enabled: false}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("disabled reconcile should not require durations: %v", err)
		}
	})

	t.Run("rejects invalid interval", func(t *testing.T) {
		cfg := cfg
		cfg.InventoryReconcile.Interval = 0
		if err := cfg.Validate(); !errors.Is(err, ErrInvalidInventoryReconcileInterval) {
			t.Fatalf("interval error=%v", err)
		}
	})

	t.Run("rejects timeout greater than interval", func(t *testing.T) {
		cfg := cfg
		cfg.InventoryReconcile.Timeout = 6 * time.Minute
		if err := cfg.Validate(); !errors.Is(err, ErrInvalidInventoryReconcileTimeout) {
			t.Fatalf("timeout error=%v", err)
		}
	})
}

func TestApplyEnvOverridesInventoryReconcileConfig(t *testing.T) {
	t.Setenv("INVENTORY_RECONCILE_ENABLED", "false")
	t.Setenv("INVENTORY_RECONCILE_INTERVAL", "10m")
	t.Setenv("INVENTORY_RECONCILE_TIMEOUT", "5s")

	cfg := validConfig()
	if err := applyEnvOverrides(&cfg); err != nil {
		t.Fatalf("apply environment: %v", err)
	}
	if cfg.InventoryReconcile.Enabled {
		t.Fatal("reconcile should be disabled by environment override")
	}
	if cfg.InventoryReconcile.Interval != 10*time.Minute {
		t.Fatalf("interval=%s", cfg.InventoryReconcile.Interval)
	}
	if cfg.InventoryReconcile.Timeout != 5*time.Second {
		t.Fatalf("timeout=%s", cfg.InventoryReconcile.Timeout)
	}
}

func validConfig() Config {
	return Config{
		Server: ServerConfig{Port: 8082},
		MySQL: MySQLConfig{
			User:            "root",
			Host:            "127.0.0.1",
			Port:            "3306",
			Database:        "go_order_management_system",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
			PingTimeout:     3 * time.Second,
			SlowThreshold:   200 * time.Millisecond,
			LogLevel:        "warn",
		},
		RabbitMQ: validRabbitMQConfig(),
		InventoryReconcile: InventoryReconcileConfig{
			Enabled:  true,
			Interval: 5 * time.Minute,
			Timeout:  3 * time.Second,
		},
		JWT: JWTConfig{ExpireHours: 24},
		HttpServer: HttpServer{Server: HttpServerConfig{
			ReadTimeOut:       5 * time.Second,
			WriteTimeout:      20 * time.Second,
			IdleTimeout:       time.Minute,
			ReadHeaderTimeout: 2 * time.Second,
			MaxHeaderBytesKib: 128,
			Timeout:           18 * time.Second,
		}},
	}
}
