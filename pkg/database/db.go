package database

import (
	"context"
	"fmt"
	"go-order-management-system/config"
	"log/slog"
	"os"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var openMySQL = func(cfg *config.Config, dsn string, logger *slog.Logger) (*gorm.DB, error) {
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         newGORMLogger(cfg.MySQL, logger),
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect mysql failed: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get mysql db failed: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(cfg.MySQL.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(cfg.MySQL.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		cfg.MySQL.PingTimeout,
	)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return gormDB, nil
}

func buildDSN(cfg *config.Config, dbPassword string) (string, error) {
	if cfg.MySQL.User == "" || dbPassword == "" || cfg.MySQL.Host == "" || cfg.MySQL.Port == "" || cfg.MySQL.Database == "" {
		return "", fmt.Errorf("database config missing")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User,
		dbPassword,
		cfg.MySQL.Host,
		cfg.MySQL.Port,
		cfg.MySQL.Database,
	)

	tlsParam := cfg.MySQL.TLS
	if tlsParam == "" {
		tlsParam = os.Getenv("DB_TLS")
	}
	// 智能默认：如果是 tidbcloud 主机，默认启用 tls=true
	if tlsParam == "" && strings.Contains(cfg.MySQL.Host, "tidbcloud.com") {
		tlsParam = "true"
	}

	if tlsParam != "" {
		dsn += "&tls=" + tlsParam
	}

	return dsn, nil
}

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	return InitDBWithLogger(cfg, nil)
}

func InitDBWithLogger(cfg *config.Config, logger *slog.Logger) (*gorm.DB, error) {
	dsn, err := buildDSN(cfg, os.Getenv("MYSQL_PASSWORD"))
	if err != nil {
		return nil, err
	}

	if dsn == "" {
		return nil, fmt.Errorf("database dsn not found")
	}
	return openMySQL(cfg, dsn, logger)
}
