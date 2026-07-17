package database

import (
	"context"
	"errors"
	"fmt"
	"go-order-management-system/config"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const defaultMySQLSlowThreshold = 200 * time.Millisecond

type slogGORMLogger struct {
	logger        *slog.Logger
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

func newGORMLogger(cfg config.MySQLConfig, logger *slog.Logger) gormlogger.Interface {
	if logger == nil {
		logger = slog.Default()
	}

	slowThreshold := cfg.SlowThreshold
	if slowThreshold <= 0 {
		slowThreshold = defaultMySQLSlowThreshold
	}

	return &slogGORMLogger{
		logger:        logger,
		level:         parseGORMLogLevel(cfg.LogLevel),
		slowThreshold: slowThreshold,
	}
}

func parseGORMLogLevel(level string) gormlogger.LogLevel {
	switch level {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "info":
		return gormlogger.Info
	case "", "warn":
		return gormlogger.Warn
	default:
		return gormlogger.Warn
	}
}

func (l *slogGORMLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	copied := *l
	copied.level = level
	return &copied
}

func (l *slogGORMLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Info {
		return
	}
	l.logger.InfoContext(ctx, fmt.Sprintf(msg, data...))
}

func (l *slogGORMLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Warn {
		return
	}
	l.logger.WarnContext(ctx, fmt.Sprintf(msg, data...))
}

func (l *slogGORMLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Error {
		return
	}
	l.logger.ErrorContext(ctx, fmt.Sprintf(msg, data...))
}

func (l *slogGORMLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.level >= gormlogger.Error:
		sql, rows := fc()
		l.logger.ErrorContext(ctx, "gorm query error",
			"error", err,
			"elapsed_ms", elapsed.Milliseconds(),
			"rows", rows,
			"sql", sql,
		)
	case l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= gormlogger.Warn:
		sql, rows := fc()
		l.logger.WarnContext(ctx, "gorm slow query",
			"elapsed_ms", elapsed.Milliseconds(),
			"threshold_ms", l.slowThreshold.Milliseconds(),
			"rows", rows,
			"sql", sql,
		)
	case l.level >= gormlogger.Info:
		sql, rows := fc()
		l.logger.InfoContext(ctx, "gorm query",
			"elapsed_ms", elapsed.Milliseconds(),
			"rows", rows,
			"sql", sql,
		)
	}
}
