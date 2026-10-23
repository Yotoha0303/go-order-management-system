package config

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// The pprof listener is the one endpoint in this service that must never be
// reachable from off-host. These tests pin the two properties that guarantee
// it: the address is loopback, and it is not the business port.
//
// Every "must reject" case below is the negative half of a falsifiable claim.
// A test that only asserted "127.0.0.1 passes" would stay green if Validate
// were gutted to `return nil`, so the rejects are the load-bearing ones.

func TestPprofValidateAcceptsLoopbackForms(t *testing.T) {
	const serverPort = 8082

	for _, addr := range []string{
		"127.0.0.1:6060",
		"localhost:6060",
		"[::1]:6060",
		"127.0.0.1:1",
	} {
		t.Run(addr, func(t *testing.T) {
			c := PprofConfig{Enabled: true, Addr: addr}
			if err := c.Validate(serverPort); err != nil {
				t.Fatalf("loopback addr %q must be accepted, got: %v", addr, err)
			}
		})
	}
}

func TestPprofValidateRejectsNonLoopback(t *testing.T) {
	const serverPort = 8082

	// The deny-list trap this pins: a rule written as "reject 0.0.0.0" would
	// accept every one of these, so the check has to be an allow-list.
	// "::" is the IPv6 unspecified address and is the direct analogue of
	// 0.0.0.0 - both mean "every interface".
	for _, addr := range []string{
		"0.0.0.0:6060",
		"[::]:6060",
		"192.168.1.10:6060",
		"10.0.0.1:6060",
		"example.com:6060",
		":6060", // empty host: binds every interface on most platforms
	} {
		t.Run(addr, func(t *testing.T) {
			c := PprofConfig{Enabled: true, Addr: addr}
			err := c.Validate(serverPort)
			if err == nil {
				t.Fatalf("non-loopback addr %q must be rejected, but Validate returned nil", addr)
			}
			if !errors.Is(err, ErrPprofAddrNotLoopback) && !errors.Is(err, ErrInvalidPprofAddr) {
				t.Fatalf("addr %q: want ErrPprofAddrNotLoopback or ErrInvalidPprofAddr, got: %v", addr, err)
			}
		})
	}
}

func TestPprofValidateRejectsBusinessPortCollision(t *testing.T) {
	// Sharing a port with the business listener would make exposure depend on
	// which handler wins, which is not a property anyone can reason about.
	c := PprofConfig{Enabled: true, Addr: "127.0.0.1:8082"}
	err := c.Validate(8082)
	if err == nil {
		t.Fatal("pprof addr equal to the business port must be rejected")
	}
	if !errors.Is(err, ErrPprofAddrCollidesWithServerPort) {
		t.Fatalf("want ErrPprofAddrCollidesWithServerPort, got: %v", err)
	}
}

func TestPprofValidateRejectsMalformedAddr(t *testing.T) {
	for _, addr := range []string{
		"127.0.0.1",     // no port
		"127.0.0.1:abc", // port is not a number
		"127.0.0.1:0",   // port 0 means "any free port", not a knowable endpoint
		"127.0.0.1:70000",
		"",
	} {
		t.Run(addr, func(t *testing.T) {
			c := PprofConfig{Enabled: true, Addr: addr}
			if err := c.Validate(8082); err == nil {
				t.Fatalf("malformed addr %q must be rejected", addr)
			}
		})
	}
}

// A disabled block is inert. Rejecting a stale or non-loopback addr that
// nothing will ever bind would break existing deployments that carry a leftover
// config value - the same reason a zero-value PprofConfig must stay valid.
func TestPprofValidateIgnoresAddrWhenDisabled(t *testing.T) {
	for _, addr := range []string{"", "0.0.0.0:6060", "127.0.0.1:8082", "garbage"} {
		t.Run(addr, func(t *testing.T) {
			c := PprofConfig{Enabled: false, Addr: addr}
			if err := c.Validate(8082); err != nil {
				t.Fatalf("disabled pprof must not validate its addr, got: %v", err)
			}
		})
	}
}

// Fail closed: a typo in the switch must stop startup rather than read as
// "off" while the operator believes profiling is running. This mirrors the
// REDIS_TLS rule from GAP-014 DoD 4.
func TestPprofEnvEnabledRejectsNonBoolean(t *testing.T) {
	for _, v := range []string{"maybe", "yes", "on", "2", "trueish"} {
		t.Run(v, func(t *testing.T) {
			t.Setenv("PPROF_ENABLED", v)
			cfg := &PprofConfig{}
			err := applyPprofEnvOverrides(cfg)
			if err == nil {
				t.Fatalf("PPROF_ENABLED=%q must fail closed, but it was accepted", v)
			}
			if !strings.Contains(err.Error(), "PPROF_ENABLED") {
				t.Fatalf("error must name the offending variable, got: %v", err)
			}
		})
	}
}

func TestPprofEnvEnabledAcceptsBooleanForms(t *testing.T) {
	for _, tc := range []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
	} {
		t.Run(tc.val, func(t *testing.T) {
			t.Setenv("PPROF_ENABLED", tc.val)
			cfg := &PprofConfig{}
			if err := applyPprofEnvOverrides(cfg); err != nil {
				t.Fatalf("PPROF_ENABLED=%q must be accepted, got: %v", tc.val, err)
			}
			if cfg.Enabled != tc.want {
				t.Fatalf("PPROF_ENABLED=%q: want Enabled=%v, got %v", tc.val, tc.want, cfg.Enabled)
			}
		})
	}
}

func TestPprofEnvAddrOverride(t *testing.T) {
	t.Setenv("PPROF_ADDR", "127.0.0.1:7070")
	cfg := &PprofConfig{}
	if err := applyPprofEnvOverrides(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != "127.0.0.1:7070" {
		t.Fatalf("want addr 127.0.0.1:7070, got %q", cfg.Addr)
	}
}

// The env layer and the validate layer must agree: an override that the
// operator believes turned profiling on must not be silently discarded, and a
// non-loopback override must still be caught by Validate.
func TestPprofEnvOverrideStillSubjectToValidate(t *testing.T) {
	t.Setenv("PPROF_ENABLED", "true")
	t.Setenv("PPROF_ADDR", "0.0.0.0:6060")

	cfg := &PprofConfig{}
	if err := applyPprofEnvOverrides(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := cfg.Validate(8082); err == nil {
		t.Fatal("a non-loopback PPROF_ADDR must be rejected by Validate, not silently bound")
	}
}

// Sanity: the shipped config.yml keeps pprof off and on a loopback address.
// This guards against someone flipping the default on while editing the file.
func TestShippedConfigKeepsPprofOffAndLoopback(t *testing.T) {
	data, err := os.ReadFile("../config.yml")
	if err != nil {
		t.Skipf("config.yml not readable from this working directory: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "pprof:") {
		t.Fatal("config.yml must carry a pprof block so the default is discoverable")
	}
	if !strings.Contains(body, "enabled: false") {
		t.Fatal("pprof must ship disabled; enabling it by default would expose profiling")
	}
	if !strings.Contains(body, "127.0.0.1:6060") {
		t.Fatal("pprof must ship bound to loopback")
	}
}
