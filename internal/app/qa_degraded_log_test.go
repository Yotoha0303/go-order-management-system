package app

// Independent (03-ring) verification of GAP-014's degraded-mode visibility.
//
// Why this file looks unusual: the failure branch lives inline inside InitDeps,
// which returns as soon as MySQL is unreachable, so the "Redis failed" line
// cannot be produced without a live database. Rather than re-typing the message
// in a test (which would assert against a copy of the production string, i.e. a
// tautology), this file parses the production source with go/ast and asserts the
// branch that actually ships.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"go-order-management-system/config"
)

// qaErrorCallInInitDeps finds the `logger.Error(...)` call in InitDeps.
func qaErrorCallInInitDeps(t *testing.T) *ast.CallExpr {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "deps.go", nil, 0)
	if err != nil {
		t.Fatalf("parse deps.go: %v", err)
	}

	var found *ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "InitDeps" {
			return true
		}
		ast.Inspect(fn, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Error" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "logger" {
				found = call
			}
			return true
		})
		return false
	})
	if found == nil {
		t.Fatal("InitDeps no longer has a logger.Error call; GAP-014 DoD-3 requires the failure to be visible")
	}
	return found
}

// TestQARedisFailureIsLoggedAtErrorWithBypassSemantics is the positive half of
// GAP-014 DoD-3: the failure path must be ERROR-level and must name the bypass.
func TestQARedisFailureIsLoggedAtErrorWithBypassSemantics(t *testing.T) {
	call := qaErrorCallInInitDeps(t)
	if len(call.Args) == 0 {
		t.Fatal("logger.Error call has no arguments")
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		t.Fatalf("first argument must be a string literal, got %T", call.Args[0])
	}
	msg, err := strconv.Unquote(lit.Value)
	if err != nil {
		t.Fatalf("unquote message: %v", err)
	}
	if !strings.Contains(msg, "degraded to pass-through bypass") {
		t.Fatalf("message must state the bypass, got %q", msg)
	}
	if !strings.Contains(msg, "redis connection failed") {
		t.Fatalf("message must state the failure, got %q", msg)
	}
	t.Logf("degraded-mode log message: %q", msg)
}

// TestQARedisFailureLogDoesNotAcceptPassword is the leak half: the arguments to
// the log call must never reference cfg.Redis.Password.
func TestQARedisFailureLogDoesNotAcceptPassword(t *testing.T) {
	call := qaErrorCallInInitDeps(t)
	for _, arg := range call.Args {
		var rendered strings.Builder
		ast.Inspect(arg, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				rendered.WriteString(sel.Sel.Name + " ")
			}
			return true
		})
		if strings.Contains(rendered.String(), "Password") {
			t.Fatalf("degraded-mode log arguments reference a Password field: %s", rendered.String())
		}
	}
}

// TestQARedisEndpointForLogNeverEchoesCredentials exercises the real production
// helper (not a copy) across the three configuration shapes that can reach it.
func TestQARedisEndpointForLogNeverEchoesCredentials(t *testing.T) {
	cases := []struct {
		name   string
		cfg    *config.Config
		want   string
		secret string
	}{
		{
			name:   "url only (config.yml fields discarded)",
			cfg:    &config.Config{Redis: config.RedisConfig{URL: "rediss://default:sup3r-s3cret@redis.example.com:6380/0"}},
			want:   "rediss://redis.example.com:6380",
			secret: "sup3r-s3cret",
		},
		{
			name:   "discrete addr + password",
			cfg:    &config.Config{Redis: config.RedisConfig{Addr: "redis.cloud.internal:6379", Username: "order_app", Password: "s3cret-from-env", TLS: true}},
			want:   "redis.cloud.internal:6379",
			secret: "s3cret-from-env",
		},
		{
			name:   "plaintext url with password",
			cfg:    &config.Config{Redis: config.RedisConfig{URL: "redis://user:plaintext-pw@127.0.0.1:6379/0"}},
			want:   "redis://127.0.0.1:6379",
			secret: "plaintext-pw",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redisEndpointForLog(tc.cfg)
			if got != tc.want {
				t.Fatalf("endpoint=%q, want %q", got, tc.want)
			}
			if strings.Contains(got, tc.secret) {
				t.Fatalf("credential leaked into log endpoint: %q", got)
			}
			if !strings.Contains(got, "://") && !strings.Contains(got, ":") {
				t.Fatalf("endpoint %q does not identify a host:port", got)
			}
		})
	}
}

// TestQARedisEndpointForLogRejectsNilAndGarbage is the D3 negative: the helper
// must not panic or echo the raw URL when it cannot parse it.
func TestQARedisEndpointForLogRejectsNilAndGarbage(t *testing.T) {
	if got := redisEndpointForLog(nil); got != "unconfigured" {
		t.Fatalf("nil config endpoint=%q", got)
	}
	garbage := &config.Config{Redis: config.RedisConfig{URL: "rediss://user:secret@[::1"}}
	got := redisEndpointForLog(garbage)
	if strings.Contains(got, "secret") {
		t.Fatalf("unparseable URL leaked its credential: %q", got)
	}
}
