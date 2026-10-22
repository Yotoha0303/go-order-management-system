package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server             ServerConfig             `yaml:"server"`
	MySQL              MySQLConfig              `yaml:"mysql"`
	Redis              RedisConfig              `yaml:"redis"`
	RabbitMQ           RabbitMQConfig           `yaml:"rabbitmq"`
	InventoryReconcile InventoryReconcileConfig `yaml:"inventoryReconcile"`
	JWT                JWTConfig                `yaml:"jwt"`
	HttpServer         HttpServer               `yaml:"http"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type MySQLConfig struct {
	User     string `yaml:"user"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	TLS      string `yaml:"tls"`

	MaxOpenConns    int           `yaml:"maxOpenConns"`
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifeTime"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"`
	PingTimeout     time.Duration `yaml:"pingTimeout"`
	SlowThreshold   time.Duration `yaml:"slowThreshold"`
	LogLevel        string        `yaml:"logLevel"`
}

// RedisConfig describes how to reach Redis. Two shapes are supported:
//
//   - URL: a full connection string (redis:// or rediss://). When it is set,
//     applyEnvOverrides discards the discrete YAML fields so a stale addr in
//     config.yml cannot shadow a cloud rediss:// endpoint.
//   - discrete fields: addr / username / password / db / tls.
//
// Environment variables REDIS_URL / REDIS_ADDR / REDIS_USERNAME /
// REDIS_PASSWORD / REDIS_DB / REDIS_TLS are applied on top; a discrete
// environment variable overrides the field it names, including one derived
// from REDIS_URL.
type RedisConfig struct {
	URL      string `yaml:"url"`
	Addr     string `yaml:"addr"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	TLS      bool   `yaml:"tls"`
}

type InventoryReconcileConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	AutoFix  bool          `yaml:"autoFix"`
}

type JWTConfig struct {
	ExpireHours int `yaml:"expireHours"`
}

type HttpServer struct {
	Server HttpServerConfig `yaml:"server"`
}

type HttpServerConfig struct {
	ReadTimeOut       time.Duration `yaml:"readTimeout"`
	WriteTimeout      time.Duration `yaml:"writeTimeout"`
	IdleTimeout       time.Duration `yaml:"idleTimeout"`
	ReadHeaderTimeout time.Duration `yaml:"readHeaderTimeout"`
	MaxHeaderBytesKib int           `yaml:"maxHeaderBytesKib"`
	Timeout           time.Duration `yaml:"timeout"`
}

func LoadEnv() {
	_ = godotenv.Load()
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s failed: %w", path, err)
	}

	var cfg Config

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse config file %s failed: %w", path, err)
	}

	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, err
	}

	if err = cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) error {
	if v := os.Getenv("APP_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid APP_PORT: %w", err)
		}
		cfg.Server.Port = port
	}

	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.MySQL.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		cfg.MySQL.Port = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.MySQL.User = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.MySQL.Database = v
	}
	if v := os.Getenv("DB_TLS"); v != "" {
		cfg.MySQL.TLS = v
	}
	if v := os.Getenv("DB_SLOW_THRESHOLD"); v != "" {
		threshold, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid DB_SLOW_THRESHOLD: %w", err)
		}
		cfg.MySQL.SlowThreshold = threshold
	}
	if v := os.Getenv("DB_LOG_LEVEL"); v != "" {
		cfg.MySQL.LogLevel = v
	}

	if err := applyRedisEnvOverrides(&cfg.Redis); err != nil {
		return err
	}
	if err := applyRabbitMQEnvOverrides(&cfg.RabbitMQ); err != nil {
		return err
	}
	if err := applyInventoryReconcileEnvOverrides(&cfg.InventoryReconcile); err != nil {
		return err
	}

	if v := os.Getenv("JWT_EXPIRE_HOURS"); v != "" {
		hours, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid JWT_EXPIRE_HOURS: %w", err)
		}
		cfg.JWT.ExpireHours = hours
	}

	return nil
}

func applyRedisEnvOverrides(cfg *RedisConfig) error {
	if v := os.Getenv("REDIS_URL"); v != "" {
		// A connection URL is the single source of endpoint truth: discard every
		// discrete field from config.yml, otherwise the shipped default
		// `addr: 127.0.0.1:6379` would silently replace the cloud endpoint.
		// Discrete environment variables below may still re-populate a field.
		*cfg = RedisConfig{URL: v}
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("REDIS_USERNAME"); v != "" {
		cfg.Username = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		db, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid REDIS_DB: %w", err)
		}
		cfg.DB = db
	}
	if v := os.Getenv("REDIS_TLS"); v != "" {
		tlsEnabled, err := strconv.ParseBool(v)
		if err != nil {
			// Fail closed: a malformed flag must not degrade to plaintext.
			return fmt.Errorf("invalid REDIS_TLS: %w", err)
		}
		cfg.TLS = tlsEnabled
	}
	return nil
}

func applyInventoryReconcileEnvOverrides(cfg *InventoryReconcileConfig) error {
	if v := os.Getenv("INVENTORY_RECONCILE_ENABLED"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid INVENTORY_RECONCILE_ENABLED: %w", err)
		}
		cfg.Enabled = enabled
	}
	if v := os.Getenv("INVENTORY_RECONCILE_INTERVAL"); v != "" {
		interval, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid INVENTORY_RECONCILE_INTERVAL: %w", err)
		}
		cfg.Interval = interval
	}
	if v := os.Getenv("INVENTORY_RECONCILE_TIMEOUT"); v != "" {
		timeout, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid INVENTORY_RECONCILE_TIMEOUT: %w", err)
		}
		cfg.Timeout = timeout
	}
	if v := os.Getenv("INVENTORY_RECONCILE_AUTO_FIX"); v != "" {
		autoFix, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid INVENTORY_RECONCILE_AUTO_FIX: %w", err)
		}
		cfg.AutoFix = autoFix
	}
	return nil
}
