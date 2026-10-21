package ordertimeout

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go-order-management-system/internal/model"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCalculateExponentialBackoff(t *testing.T) {
	base := 5 * time.Second
	maxDelay := 10 * time.Minute

	tests := []struct {
		name     string
		base     time.Duration
		attempts int
		maxDelay time.Duration
		want     time.Duration
	}{
		{
			name:     "zero or negative base delay returns 0",
			base:     0,
			attempts: 1,
			maxDelay: maxDelay,
			want:     0,
		},
		{
			name:     "negative base delay returns 0",
			base:     -time.Second,
			attempts: 2,
			maxDelay: maxDelay,
			want:     0,
		},
		{
			name:     "attempt 0 returns base delay",
			base:     base,
			attempts: 0,
			maxDelay: maxDelay,
			want:     5 * time.Second,
		},
		{
			name:     "attempt 1 returns base delay (5s)",
			base:     base,
			attempts: 1,
			maxDelay: maxDelay,
			want:     5 * time.Second,
		},
		{
			name:     "attempt 2 doubles base delay (10s)",
			base:     base,
			attempts: 2,
			maxDelay: maxDelay,
			want:     10 * time.Second,
		},
		{
			name:     "attempt 3 quadruples base delay (20s)",
			base:     base,
			attempts: 3,
			maxDelay: maxDelay,
			want:     20 * time.Second,
		},
		{
			name:     "attempt 4 is 8x base delay (40s)",
			base:     base,
			attempts: 4,
			maxDelay: maxDelay,
			want:     40 * time.Second,
		},
		{
			name:     "attempt 6 is 32x base delay (160s)",
			base:     base,
			attempts: 6,
			maxDelay: maxDelay,
			want:     160 * time.Second,
		},
		{
			name:     "attempt 8 exceeds maxDelay and is capped at 10m",
			base:     base,
			attempts: 8, // 5s * 128 = 640s > 600s
			maxDelay: maxDelay,
			want:     10 * time.Minute,
		},
		{
			name:     "attempt 10 capped at maxDelay",
			base:     base,
			attempts: 10,
			maxDelay: maxDelay,
			want:     10 * time.Minute,
		},
		{
			name:     "base delay greater than max delay is capped",
			base:     15 * time.Minute,
			attempts: 1,
			maxDelay: 10 * time.Minute,
			want:     10 * time.Minute,
		},
		{
			name:     "default maxDelay applied when maxDelay <= 0",
			base:     base,
			attempts: 10,
			maxDelay: 0,
			want:     maxOutboxBackoffDelay,
		},
		{
			name:     "large attempts shift >= 62 does not overflow int64",
			base:     base,
			attempts: 63,
			maxDelay: maxDelay,
			want:     10 * time.Minute,
		},
		{
			name:     "extreme attempts 1000 does not panic or overflow",
			base:     base,
			attempts: 1000,
			maxDelay: maxDelay,
			want:     10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateExponentialBackoff(tt.base, tt.attempts, tt.maxDelay)
			if got != tt.want {
				t.Errorf("calculateExponentialBackoff(%v, %v, %v) = %v, want %v",
					tt.base, tt.attempts, tt.maxDelay, got, tt.want)
			}
		})
	}
}

type mockWorkerOutboxConnector struct {
	events []model.OrderTimeoutOutbox
}

func (c *mockWorkerOutboxConnector) Connect(context.Context) (driver.Conn, error) {
	return &mockWorkerOutboxConn{cfg: c}, nil
}

func (c *mockWorkerOutboxConnector) Driver() driver.Driver {
	return &mockWorkerOutboxDriver{}
}

type mockWorkerOutboxDriver struct{}

func (d *mockWorkerOutboxDriver) Open(string) (driver.Conn, error) {
	return &mockWorkerOutboxConn{}, nil
}

type mockWorkerOutboxConn struct {
	cfg *mockWorkerOutboxConnector
}

func (c *mockWorkerOutboxConn) Prepare(string) (driver.Stmt, error) {
	return &mockWorkerOutboxStmt{cfg: c.cfg}, nil
}

func (c *mockWorkerOutboxConn) Close() error { return nil }
func (c *mockWorkerOutboxConn) Begin() (driver.Tx, error) {
	return &mockWorkerOutboxTx{}, nil
}
func (c *mockWorkerOutboxConn) Ping(context.Context) error { return nil }

func (c *mockWorkerOutboxConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (c *mockWorkerOutboxConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(strings.ToLower(query), "order_timeout_outbox") && c.cfg != nil {
		return &mockWorkerOutboxRows{events: c.cfg.events}, nil
	}
	return &mockWorkerOutboxRows{}, nil
}

type mockWorkerOutboxTx struct{}

func (t *mockWorkerOutboxTx) Commit() error   { return nil }
func (t *mockWorkerOutboxTx) Rollback() error { return nil }

type mockWorkerOutboxStmt struct {
	cfg *mockWorkerOutboxConnector
}

func (s *mockWorkerOutboxStmt) Close() error  { return nil }
func (s *mockWorkerOutboxStmt) NumInput() int { return -1 }
func (s *mockWorkerOutboxStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}
func (s *mockWorkerOutboxStmt) Query([]driver.Value) (driver.Rows, error) {
	if s.cfg != nil {
		return &mockWorkerOutboxRows{events: s.cfg.events}, nil
	}
	return &mockWorkerOutboxRows{}, nil
}

type mockWorkerOutboxRows struct {
	events []model.OrderTimeoutOutbox
	idx    int
}

func (r *mockWorkerOutboxRows) Columns() []string {
	return []string{
		"id", "event_id", "order_id", "user_id", "timeout_at",
		"published_at", "attempts", "next_attempt_at", "last_error",
		"created_at", "updated_at",
	}
}

func (r *mockWorkerOutboxRows) Close() error { return nil }

func (r *mockWorkerOutboxRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.events) {
		return io.EOF
	}
	ev := r.events[r.idx]
	r.idx++
	dest[0] = ev.ID
	dest[1] = ev.EventID
	dest[2] = ev.OrderID
	dest[3] = ev.UserID
	dest[4] = ev.TimeoutAt
	if ev.PublishedAt == nil {
		dest[5] = nil
	} else {
		dest[5] = *ev.PublishedAt
	}
	dest[6] = int64(ev.Attempts)
	dest[7] = ev.NextAttemptAt
	dest[8] = ev.LastError
	dest[9] = ev.CreatedAt
	dest[10] = ev.UpdatedAt
	return nil
}

func TestWorker_PublishBatch_HandlesFailureAndCapping(t *testing.T) {
	now := time.Now().UTC()
	events := []model.OrderTimeoutOutbox{
		{
			ID:            101,
			EventID:       "evt-fail-cap-101",
			OrderID:       2001,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			Attempts:      9, // 认领后会变为 10，达到 defaultMaxOutboxAttempts
			NextAttemptAt: now.Add(-time.Minute),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	connector := &mockWorkerOutboxConnector{events: events}
	sqlDB := sql.OpenDB(connector)
	defer sqlDB.Close()

	mockDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open mock gorm db failed: %v", err)
	}

	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))

	worker := &Worker{
		config: Config{
			PublishBatchSize: 10,
			OutboxRetryDelay: 5 * time.Second,
		},
		db:       mockDB,
		now:      func() time.Time { return now },
		logger:   testLogger,
		topology: productionTopology,
	}

	// 传入 nil channel 模拟发布网络 IO 失败
	err = worker.publishBatch(context.Background(), nil)
	if err == nil {
		t.Fatal("expected publishBatch with nil channel to return error")
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "order timeout outbox reached max attempts") {
		t.Errorf("expected log output to contain %q, got: %s", "order timeout outbox reached max attempts", logOutput)
	}
	if !strings.Contains(logOutput, "attempts=10") {
		t.Errorf("expected log output to contain %q, got: %s", "attempts=10", logOutput)
	}
	if !strings.Contains(logOutput, "event_id=evt-fail-cap-101") {
		t.Errorf("expected log output to contain %q, got: %s", "event_id=evt-fail-cap-101", logOutput)
	}
}
