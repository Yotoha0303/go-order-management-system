package app

import (
	"strings"
	"testing"

	"go-order-management-system/config"
)

// The runtime shape produced when only REDIS_URL is set: the config layer
// discards the config.yml addr, so logs must fall through to the URL branch.
// The password (both in the userinfo and in the path) must never reach a log.
func TestRedisEndpointForLogStripsCredentialsFromURLOnlyConfig(t *testing.T) {
	cfg := &config.Config{Redis: config.RedisConfig{URL: "rediss://default:sup3r-s3cret@redis.example.com:6380/0"}}
	got := redisEndpointForLog(cfg)
	if got != "rediss://redis.example.com:6380" {
		t.Fatalf("endpoint=%q", got)
	}
	if strings.Contains(got, "sup3r-s3cret") {
		t.Fatalf("password leaked into log endpoint: %q", got)
	}
	if strings.Contains(got, "default@") {
		t.Fatalf("username leaked into log endpoint: %q", got)
	}
}

func TestRedisEndpointForLogStripsCredentials(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "url form never echoes password",
			cfg:  &config.Config{Redis: config.RedisConfig{URL: "rediss://default:sup3r-s3cret@redis.example.com:6380/0"}},
			want: "rediss://redis.example.com:6380",
		},
		{
			name: "addr form wins when both present",
			cfg:  &config.Config{Redis: config.RedisConfig{Addr: "redis:6379", URL: "rediss://u:p@cloud:6380"}},
			want: "redis:6379",
		},
		{
			name: "unparseable url",
			cfg:  &config.Config{Redis: config.RedisConfig{URL: "://bad"}},
			want: "unconfigured",
		},
		{
			name: "nil config",
			cfg:  nil,
			want: "unconfigured",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redisEndpointForLog(tc.cfg)
			if got != tc.want {
				t.Fatalf("redisEndpointForLog()=%q, want %q", got, tc.want)
			}
			// Negative case (D3): credentials must never reach the log sink.
			if tc.cfg != nil && tc.cfg.Redis.Password != "" && got == tc.cfg.Redis.Password {
				t.Fatal("endpoint must not be the password")
			}
			if strings.Contains(got, "sup3r-s3cret") {
				t.Fatalf("password leaked into log endpoint: %q", got)
			}
		})
	}
}
