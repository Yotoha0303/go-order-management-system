package redis

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"go-order-management-system/config"

	"github.com/redis/go-redis/v9"
)

// Sentinel errors are the machine-readable classification of connection
// failures. They exist so a cloud deployment can tell "we never reached the
// endpoint" apart from "TLS did not complete" and "the server rejected the
// credentials" instead of staring at one opaque dial error.
var (
	ErrRedisConfigMissing = errors.New("redis config missing")
	ErrRedisAddrNotFound  = errors.New("redis addr missing")
	ErrRedisURLInvalid    = errors.New("redis url invalid")
	ErrRedisTLSHandshake  = errors.New("redis tls handshake failed")
	ErrRedisAuthFailed    = errors.New("redis authentication failed")
	ErrRedisConnectFailed = errors.New("redis connect failed")
)

// InitRedis builds a client from configuration and verifies connectivity with
// a single PING. The client is only returned when the server answered, so a
// caller can rely on a non-nil client meaning "Redis is actually usable".
func InitRedis(cfg *config.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, ErrRedisConfigMissing
	}

	opts, err := BuildOptions(&cfg.Redis)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("%w: %w", classifyConnectionError(err), err)
	}

	return client, nil
}

// BuildOptions turns configuration into go-redis options without opening a
// connection. URL (redis:// or rediss://) is the baseline; non-zero discrete
// fields then override it. The config layer guarantees that, when REDIS_URL is
// set, the discrete fields carry only environment overrides — never values
// inherited from config.yml.
func BuildOptions(cfg *config.RedisConfig) (*redis.Options, error) {
	if cfg == nil {
		return nil, ErrRedisConfigMissing
	}

	var opts *redis.Options
	if cfg.URL != "" {
		parsed, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrRedisURLInvalid, err)
		}
		opts = parsed
	} else {
		if cfg.Addr == "" {
			return nil, ErrRedisAddrNotFound
		}
		opts = &redis.Options{Addr: cfg.Addr}
	}

	if cfg.Addr != "" {
		opts.Addr = cfg.Addr
	}
	if cfg.Username != "" {
		opts.Username = cfg.Username
	}
	if cfg.Password != "" {
		opts.Password = cfg.Password
	}
	if cfg.DB != 0 {
		opts.DB = cfg.DB
	}
	// ParseURL already installs a TLS config for rediss://; an explicit
	// tls: true forces TLS for redis:// or bare addr: forms as well.
	if cfg.TLS {
		opts.TLSConfig = TLSConfig(opts.Addr)
	}

	return opts, nil
}

// TLSConfig returns the TLS settings used for Redis connections. ServerName is
// derived from the address so SNI and certificate verification work against
// managed providers; TLS 1.2 is the floor.
func TLSConfig(addr string) *tls.Config {
	host := addr
	if h, _, err := net.SplitHostPort(addr); err == nil {
		host = h
	}
	return &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
}

// classifyConnectionError maps a go-redis PING failure onto the sentinel that
// describes the layer that failed. Unknown failures fall through untouched so
// their original text is never lost.
func classifyConnectionError(err error) error {
	if err == nil {
		return nil
	}

	var certErr *tls.CertificateVerificationError
	lower := strings.ToLower(err.Error())
	if errors.As(err, &certErr) ||
		strings.Contains(lower, "tls:") || strings.Contains(lower, "x509:") {
		return ErrRedisTLSHandshake
	}

	if strings.Contains(lower, "wrongpass") || strings.Contains(lower, "noauth") ||
		strings.Contains(lower, "invalid password") || strings.Contains(lower, "wrong password") {
		return ErrRedisAuthFailed
	}

	return ErrRedisConnectFailed
}
