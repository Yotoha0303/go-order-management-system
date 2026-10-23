package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

var (
	ErrInvalidServerPort                  = errors.New("invalid server port")
	ErrInvalidMySQLPort                   = errors.New("invalid mysql port")
	ErrMySQLDatabaseNotFound              = errors.New("MySQL database name not found")
	ErrMySQLUserNotFound                  = errors.New("MySQL user not found")
	ErrMySQLHostNotFound                  = errors.New("MySQL host not found")
	ErrInvalidHttpServerReadTimeout       = errors.New("invalid server read time out")
	ErrInvalidHttpServerWriteTimeout      = errors.New("invalid server write time out")
	ErrInvalidHttpServerIdleTimeout       = errors.New("invalid server idle time out")
	ErrInvalidHttpServerReadHeaderTimeout = errors.New("invalid server read header time out")
	ErrInvalidHttpServerMaxHeaderBytes    = errors.New("invalid server max header bytes")
	ErrInvalidHttpServerTimeout           = errors.New("invalid http server time out")
	ErrMySQLMaxOpenConnsFailed            = errors.New("MySQL max open conns failed")
	ErrMySQLMaxIdleConnsFailed            = errors.New("MySQL mysql max idle conns failed")
	ErrMySQLInvalidConnMaxIdleTime        = errors.New("invalid mysql conn max idle time")
	ErrMySQLInvalidConnMaxLifetime        = errors.New("invalid mysql conn max life time")
	ErrMySQLInvalidPingTimeout            = errors.New("invalid mysql conn ping time out")
	ErrMySQLInvalidSlowThreshold          = errors.New("invalid mysql slow threshold")
	ErrMySQLInvalidLogLevel               = errors.New("invalid mysql log level")
	ErrInvalidInventoryReconcileInterval  = errors.New("invalid inventory reconcile interval")
	ErrInvalidInventoryReconcileTimeout   = errors.New("invalid inventory reconcile timeout")
	ErrInvalidJWTExpireHours              = errors.New("invalid jwt expire hours")
	ErrInvalidPprofAddr                   = errors.New("invalid pprof addr")
	ErrPprofAddrNotLoopback               = errors.New("pprof addr must be a loopback address")
	ErrPprofAddrCollidesWithServerPort    = errors.New("pprof addr must not reuse the business server port")
)

// loopbackHosts are the host forms that resolve to the local machine only.
// "localhost" is included because it is the spelling operators actually type;
// it is resolved by the OS and never reaches a network interface other than
// the loopback one on the platforms this service targets.
var loopbackHosts = map[string]bool{
	"127.0.0.1": true,
	"::1":       true,
	"localhost": true,
}

func (c Config) Validate() error {
	server := c.Server
	mysql := c.MySQL
	http := c.HttpServer.Server
	jwt := c.JWT

	if server.Port <= 0 {
		return ErrInvalidServerPort
	}

	if mysql.Host == "" {
		return ErrMySQLHostNotFound
	}

	mysqlPort, err := strconv.Atoi(mysql.Port)
	if err != nil || mysqlPort <= 0 || mysqlPort > 65535 {
		return ErrInvalidMySQLPort
	}

	if mysql.Database == "" {
		return ErrMySQLDatabaseNotFound
	}

	if mysql.User == "" {
		return ErrMySQLUserNotFound
	}

	if http.ReadTimeOut <= 0 {
		return ErrInvalidHttpServerReadTimeout
	}

	if http.WriteTimeout <= 0 {
		return ErrInvalidHttpServerWriteTimeout
	}

	if http.IdleTimeout <= 0 {
		return ErrInvalidHttpServerIdleTimeout
	}

	if http.ReadHeaderTimeout <= 0 {
		return ErrInvalidHttpServerReadHeaderTimeout
	}

	if http.MaxHeaderBytesKib <= 0 {
		return ErrInvalidHttpServerMaxHeaderBytes
	}

	if http.Timeout <= 0 {
		return ErrInvalidHttpServerTimeout
	}

	if mysql.MaxOpenConns <= 0 {
		return ErrMySQLMaxOpenConnsFailed
	}

	if mysql.MaxIdleConns < 0 || mysql.MaxIdleConns > mysql.MaxOpenConns {
		return ErrMySQLMaxIdleConnsFailed
	}

	if mysql.ConnMaxIdleTime <= 0 {
		return ErrMySQLInvalidConnMaxIdleTime
	}

	if mysql.ConnMaxLifetime <= 0 {
		return ErrMySQLInvalidConnMaxLifetime
	}

	if mysql.PingTimeout <= 0 {
		return ErrMySQLInvalidPingTimeout
	}

	if mysql.SlowThreshold < 0 {
		return ErrMySQLInvalidSlowThreshold
	}

	switch mysql.LogLevel {
	case "", "silent", "error", "warn", "info":
	default:
		return ErrMySQLInvalidLogLevel
	}

	if jwt.ExpireHours <= 0 {
		return ErrInvalidJWTExpireHours
	}

	if err := c.RabbitMQ.Validate(); err != nil {
		return err
	}

	if err := c.InventoryReconcile.Validate(); err != nil {
		return err
	}

	if err := c.Pprof.Validate(server.Port); err != nil {
		return err
	}

	return nil
}

// Validate rejects a profiling listener that is reachable from off-host.
//
// The check is deliberately an allow-list of loopback host forms rather than a
// deny of "0.0.0.0": a deny-list would accept "::" , "0:0:0:0:0:0:0:0" and a
// hostname that resolves elsewhere, so it would fail open. Rejecting anything
// not on the allow-list fails closed.
//
// It also rejects reusing the business port. The two listeners carry very
// different exposure - one is meant to face the network, the other must not -
// so sharing a port would make the exposure a function of which handler wins.
func (c PprofConfig) Validate(serverPort int) error {
	if !c.Enabled {
		// A disabled block is inert; do not reject a stale addr that nothing
		// will ever bind.
		return nil
	}

	host, portStr, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrInvalidPprofAddr, c.Addr, err)
	}

	if !loopbackHosts[host] {
		return fmt.Errorf("%w: %q (got host %q)", ErrPprofAddrNotLoopback, c.Addr, host)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("%w: %q", ErrInvalidPprofAddr, c.Addr)
	}

	if port == serverPort {
		return fmt.Errorf("%w: %d", ErrPprofAddrCollidesWithServerPort, port)
	}

	return nil
}

func (c InventoryReconcileConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Interval <= 0 || c.Interval > 24*time.Hour {
		return ErrInvalidInventoryReconcileInterval
	}
	if c.Timeout <= 0 || c.Timeout > c.Interval {
		return ErrInvalidInventoryReconcileTimeout
	}
	return nil
}
