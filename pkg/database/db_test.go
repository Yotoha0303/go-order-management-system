package database

import (
	"go-order-management-system/config"
	"strings"
	"testing"
)

func TestBuildDSN_LocalWithoutTLS(t *testing.T) {
	cfg := &config.Config{
		MySQL: config.MySQLConfig{
			User:     "root",
			Host:     "127.0.0.1",
			Port:     "3306",
			Database: "order_db",
		},
	}

	dsn, err := buildDSN(cfg, "secret")
	if err != nil {
		t.Fatalf("buildDSN failed: %v", err)
	}

	expectedPrefix := "root:secret@tcp(127.0.0.1:3306)/order_db?charset=utf8mb4&parseTime=True&loc=Local"
	if dsn != expectedPrefix {
		t.Fatalf("expected %q, got %q", expectedPrefix, dsn)
	}
	if strings.Contains(dsn, "&tls=") {
		t.Fatalf("local DSN should not contain &tls=, got %q", dsn)
	}
}

func TestBuildDSN_ExplicitConfigTLS(t *testing.T) {
	cfg := &config.Config{
		MySQL: config.MySQLConfig{
			User:     "cloud_user",
			Host:     "mysql.internal",
			Port:     "3306",
			Database: "order_db",
			TLS:      "true",
		},
	}

	dsn, err := buildDSN(cfg, "cloud_pwd")
	if err != nil {
		t.Fatalf("buildDSN failed: %v", err)
	}

	wantSuffix := "&tls=true"
	if !strings.HasSuffix(dsn, wantSuffix) {
		t.Fatalf("expected DSN to end with %q, got %q", wantSuffix, dsn)
	}
}

func TestBuildDSN_EnvDBTLS(t *testing.T) {
	t.Setenv("DB_TLS", "skip-verify")

	cfg := &config.Config{
		MySQL: config.MySQLConfig{
			User:     "cloud_user",
			Host:     "10.0.0.5",
			Port:     "3306",
			Database: "order_db",
		},
	}

	dsn, err := buildDSN(cfg, "cloud_pwd")
	if err != nil {
		t.Fatalf("buildDSN failed: %v", err)
	}

	wantSuffix := "&tls=skip-verify"
	if !strings.HasSuffix(dsn, wantSuffix) {
		t.Fatalf("expected DSN to end with %q, got %q", wantSuffix, dsn)
	}
}

func TestBuildDSN_TiDBCloudAutoTLS(t *testing.T) {
	// 确保未设置环境变量
	t.Setenv("DB_TLS", "")

	cfg := &config.Config{
		MySQL: config.MySQLConfig{
			User:     "2ABC123.root",
			Host:     "gateway01.us-east-1.prod.aws.tidbcloud.com",
			Port:     "4000",
			Database: "test",
		},
	}

	dsn, err := buildDSN(cfg, "tidb_password")
	if err != nil {
		t.Fatalf("buildDSN failed: %v", err)
	}

	wantSuffix := "&tls=true"
	if !strings.HasSuffix(dsn, wantSuffix) {
		t.Fatalf("expected TiDB Cloud DSN to automatically end with %q, got %q", wantSuffix, dsn)
	}
}

func TestBuildDSN_MissingConfigReturnsError(t *testing.T) {
	cfg := &config.Config{
		MySQL: config.MySQLConfig{
			User: "root",
			// Host missing
			Port:     "3306",
			Database: "order_db",
		},
	}

	_, err := buildDSN(cfg, "pwd")
	if err == nil {
		t.Fatal("expected error when host is missing, got nil")
	}

	_, err = buildDSN(cfg, "")
	if err == nil {
		t.Fatal("expected error when password is empty, got nil")
	}
}

func TestBuildTestMySQLDSN_TLS(t *testing.T) {
	t.Setenv("MYSQL_TEST_DATABASE", "test_db")
	t.Setenv("DB_TLS", "true")

	cfg := &config.Config{
		MySQL: config.MySQLConfig{
			User: "root",
			Host: "127.0.0.1",
			Port: "3306",
		},
	}

	dsn, err := buildTestMySQLDSN(cfg, "pwd")
	if err != nil {
		t.Fatalf("buildTestMySQLDSN failed: %v", err)
	}

	wantSuffix := "&tls=true"
	if !strings.HasSuffix(dsn, wantSuffix) {
		t.Fatalf("expected test DSN to end with %q, got %q", wantSuffix, dsn)
	}
}
