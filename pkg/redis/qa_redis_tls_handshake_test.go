package redis

// Independent (03-ring) verification of GAP-014: the code claims that rediss://
// performs a real TLS handshake and that failures are classified. The shipped
// tests only inspect the *options struct*; they never open a socket, so they
// cannot tell a working TLS path from a config that is silently ignored.
//
// These tests stand up a throwaway TLS/plaintext RESP server on loopback. They
// need no cloud credentials and no external service, so they run in CI.

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-order-management-system/config"

	goredis "github.com/redis/go-redis/v9"
)

type qaRedisServer struct {
	addr     string
	tls      bool
	username string
	password string
	cert     tls.Certificate
}

// startQARedisServer accepts one connection at a time and answers the minimal
// command set go-redis issues during connect: HELLO, AUTH, CLIENT, PING.
func startQARedisServer(t *testing.T, useTLS bool, username, password string) *qaRedisServer {
	t.Helper()

	var tlsCfg *tls.Config
	var cert tls.Certificate
	if useTLS {
		cert = qaSelfSignedCert(t)
		tlsCfg = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := &qaRedisServer{addr: ln.Addr().String(), tls: useTLS, username: username, password: password, cert: cert}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				if tlsCfg != nil {
					tc := tls.Server(c, tlsCfg)
					if err := tc.Handshake(); err != nil {
						return // client could not verify us: that is the test
					}
					c = tc
				}
				srv.serve(c)
			}(conn)
		}
	}()
	return srv
}

func (s *qaRedisServer) serve(c net.Conn) {
	r := bufio.NewReader(c)
	authed := s.password == ""
	for {
		args, err := qaReadCommand(r)
		if err != nil {
			return
		}
		if len(args) == 0 {
			continue
		}
		switch strings.ToUpper(args[0]) {
		case "HELLO":
			// Force the RESP2 fallback path so AUTH is exercised explicitly.
			fmt.Fprint(c, "-ERR unknown command 'HELLO'\r\n")
		case "AUTH":
			var user, pass string
			if len(args) == 3 {
				user, pass = args[1], args[2]
			} else {
				pass = args[1]
			}
			if (s.username == "" || user == s.username) && pass == s.password {
				authed = true
				fmt.Fprint(c, "+OK\r\n")
			} else {
				fmt.Fprint(c, "-WRONGPASS invalid username-password pair or user is disabled.\r\n")
			}
		case "PING":
			if authed {
				fmt.Fprint(c, "+PONG\r\n")
			} else {
				fmt.Fprint(c, "-NOAUTH Authentication required.\r\n")
			}
		default:
			fmt.Fprint(c, "+OK\r\n")
		}
	}
}

func qaReadCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		return nil, errors.New("unexpected RESP framing")
	}
	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		header = strings.TrimRight(header, "\r\n")
		if !strings.HasPrefix(header, "$") {
			return nil, errors.New("unexpected bulk header")
		}
		n, err := strconv.Atoi(header[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, n+2) // payload + CRLF
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:n]))
	}
	return args, nil
}

func qaSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "qa-loopback"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

// TestQAInitRedisRedissPerformsRealTLSHandshake is the core GAP-014 claim: a
// rediss:// URL must actually start a TLS handshake. Our self-signed cert is not
// in the trust store, so a *real* handshake must fail with a certificate error
// and be classified as such. A client that silently fell back to plaintext
// would instead get a protocol/reset error and fail this assertion.
func TestQAInitRedisRedissPerformsRealTLSHandshake(t *testing.T) {
	srv := startQARedisServer(t, true, "", "")
	cfg := &config.Config{Redis: config.RedisConfig{URL: "rediss://" + srv.addr + "/0"}}

	client, err := InitRedis(cfg)
	if err == nil {
		_ = client.Close()
		t.Fatal("rediss:// against an untrusted certificate must fail; a nil error means TLS is not being enforced")
	}
	if !errors.Is(err, ErrRedisTLSHandshake) {
		t.Fatalf("error must be classified as TLS handshake, got %v", err)
	}
	t.Logf("rediss handshake rejected as expected: %v", err)
}

// TestQAInitRedisRedissSucceedsWithTrustedCertificate is the positive half of the
// same claim. Kicking off the handshake is not enough: after a *successful* TLS
// handshake the client must be able to PING. Production cannot use a custom root
// (BuildOptions exposes no RootCAs knob), but this is the only way to prove the
// TLS path carries application data end to end. It deliberately mutates a
// freshly built client, never the production config.
func TestQAInitRedisRedissSucceedsWithTrustedCertificate(t *testing.T) {
	srv := startQARedisServer(t, true, "", "")

	cfg := &config.Config{Redis: config.RedisConfig{URL: "rediss://" + srv.addr + "/0"}}
	opts, err := BuildOptions(&cfg.Redis)
	if err != nil {
		t.Fatalf("build options: %v", err)
	}
	if opts.TLSConfig == nil {
		t.Fatal("rediss:// must produce a TLS config")
	}
	// Trust the throwaway cert for this connection only.
	pool := x509.NewCertPool()
	leaf, err := x509.ParseCertificate(srv.cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse server cert: %v", err)
	}
	pool.AddCert(leaf)
	opts.TLSConfig.RootCAs = pool
	opts.TLSConfig.ServerName = "127.0.0.1"

	client := goredis.NewClient(opts)
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("PING over a verified TLS connection must succeed, got %v", err)
	}
	t.Log("rediss:// PING succeeded over a verified TLS handshake")
}

// TestQAInitRedisPlaintextAgainstTLSServerIsNotSuccess is the negative half: the
// same TLS server reached over redis:// must not connect. This proves the
// handshake above came from TLS, not from something the server always accepts.
func TestQAInitRedisPlaintextAgainstTLSServerIsNotSuccess(t *testing.T) {
	srv := startQARedisServer(t, true, "", "")
	cfg := &config.Config{Redis: config.RedisConfig{URL: "redis://" + srv.addr + "/0"}}

	client, err := InitRedis(cfg)
	if err == nil {
		_ = client.Close()
		t.Fatal("plaintext client must not complete a handshake against a TLS-only server")
	}
	t.Logf("plaintext against TLS server rejected: %v (classified %v)", err, err)
}

// TestQAInitRedisUsesCredentialsFromConfig is the credential-plumbing test: the
// password now comes from the config struct (no more os.Getenv inside pkg/redis).
// A correct credential must connect; a wrong one must be classified as auth.
func TestQAInitRedisUsesCredentialsFromConfig(t *testing.T) {
	srv := startQARedisServer(t, false, "order_app", "cfg-supplied-secret")

	t.Run("correct credentials from config connect", func(t *testing.T) {
		cfg := &config.Config{Redis: config.RedisConfig{
			Addr: srv.addr, Username: "order_app", Password: "cfg-supplied-secret",
		}}
		client, err := InitRedis(cfg)
		if err != nil {
			t.Fatalf("expected PING to succeed, got %v", err)
		}
		t.Cleanup(func() { _ = client.Close() })
	})

	t.Run("wrong password is classified as auth failure", func(t *testing.T) {
		cfg := &config.Config{Redis: config.RedisConfig{
			Addr: srv.addr, Username: "order_app", Password: "wrong-secret",
		}}
		client, err := InitRedis(cfg)
		if err == nil {
			_ = client.Close()
			t.Fatal("wrong password must not connect")
		}
		if !errors.Is(err, ErrRedisAuthFailed) {
			t.Fatalf("wrong password must classify as auth failure, got %v", err)
		}
		t.Logf("auth failure classified: %v", err)
	})
}

// TestQAInitRedisClosesClientWhenPingFails is a resource-leak guard: InitRedis
// must not hand back a live pool after a failed PING.
func TestQAInitRedisClosesClientWhenPingFails(t *testing.T) {
	// Point at a closed port: dial fails, InitRedis must return (nil, err).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	cfg := &config.Config{Redis: config.RedisConfig{Addr: addr}}
	client, err := InitRedis(cfg)
	if err == nil || client != nil {
		t.Fatalf("expected (nil, err) for an unreachable endpoint, got client=%v err=%v", client, err)
	}
	if !errors.Is(err, ErrRedisConnectFailed) {
		t.Fatalf("unreachable endpoint must classify as connect failure, got %v", err)
	}
}
