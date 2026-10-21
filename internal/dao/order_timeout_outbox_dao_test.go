package dao_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"go-order-management-system/internal/dao"
	"go-order-management-system/internal/model"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockClaimOutboxConnector struct {
	events []model.OrderTimeoutOutbox
}

func (c *mockClaimOutboxConnector) Connect(context.Context) (driver.Conn, error) {
	return &mockClaimOutboxConn{cfg: c}, nil
}

func (c *mockClaimOutboxConnector) Driver() driver.Driver {
	return &mockClaimOutboxDriver{}
}

type mockClaimOutboxDriver struct{}

func (d *mockClaimOutboxDriver) Open(string) (driver.Conn, error) {
	return &mockClaimOutboxConn{}, nil
}

type mockClaimOutboxConn struct {
	cfg *mockClaimOutboxConnector
}

func (c *mockClaimOutboxConn) Prepare(string) (driver.Stmt, error) {
	return &mockClaimOutboxStmt{cfg: c.cfg}, nil
}

func (c *mockClaimOutboxConn) Close() error { return nil }
func (c *mockClaimOutboxConn) Begin() (driver.Tx, error) {
	return &mockClaimOutboxTx{}, nil
}
func (c *mockClaimOutboxConn) Ping(context.Context) error { return nil }

func (c *mockClaimOutboxConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(int64(len(c.cfg.events))), nil
}

func (c *mockClaimOutboxConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(strings.ToLower(query), "order_timeout_outbox") && c.cfg != nil {
		return &mockClaimOutboxRows{events: c.cfg.events}, nil
	}
	return &mockClaimOutboxRows{}, nil
}

type mockClaimOutboxTx struct{}

func (t *mockClaimOutboxTx) Commit() error   { return nil }
func (t *mockClaimOutboxTx) Rollback() error { return nil }

type mockClaimOutboxStmt struct {
	cfg *mockClaimOutboxConnector
}

func (s *mockClaimOutboxStmt) Close() error  { return nil }
func (s *mockClaimOutboxStmt) NumInput() int { return -1 }
func (s *mockClaimOutboxStmt) Exec([]driver.Value) (driver.Result, error) {
	if s.cfg != nil {
		return driver.RowsAffected(int64(len(s.cfg.events))), nil
	}
	return driver.RowsAffected(1), nil
}
func (s *mockClaimOutboxStmt) Query([]driver.Value) (driver.Rows, error) {
	if s.cfg != nil {
		return &mockClaimOutboxRows{events: s.cfg.events}, nil
	}
	return &mockClaimOutboxRows{}, nil
}

type mockClaimOutboxRows struct {
	events []model.OrderTimeoutOutbox
	idx    int
}

func (r *mockClaimOutboxRows) Columns() []string {
	return []string{
		"id", "event_id", "order_id", "user_id", "timeout_at",
		"published_at", "attempts", "next_attempt_at", "last_error",
		"created_at", "updated_at",
	}
}

func (r *mockClaimOutboxRows) Close() error { return nil }

func (r *mockClaimOutboxRows) Next(dest []driver.Value) error {
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

func TestClaimPendingOrderTimeoutOutbox_UnitMock(t *testing.T) {
	now := time.Now().UTC()
	leaseDuration := 45 * time.Second
	events := []model.OrderTimeoutOutbox{
		{
			ID:            1,
			EventID:       "evt-claim-1",
			OrderID:       1001,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			Attempts:      0,
			NextAttemptAt: now.Add(-time.Minute),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            2,
			EventID:       "evt-claim-2",
			OrderID:       1002,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			Attempts:      2,
			NextAttemptAt: now.Add(-time.Minute),
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	connector := &mockClaimOutboxConnector{events: events}
	sqlDB := sql.OpenDB(connector)
	defer sqlDB.Close()

	mockDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("open mock db failed: %v", err)
	}

	claimed, err := dao.ClaimPendingOrderTimeoutOutbox(
		context.Background(),
		mockDB,
		now,
		10,
		leaseDuration,
		10,
	)
	if err != nil {
		t.Fatalf("ClaimPendingOrderTimeoutOutbox failed: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed events, got %d", len(claimed))
	}

	// 验证租期更新与 attempts 原子递增
	if claimed[0].Attempts != 1 {
		t.Fatalf("expected event 1 attempts incremented to 1, got %d", claimed[0].Attempts)
	}
	if !claimed[0].NextAttemptAt.Equal(now.Add(leaseDuration)) {
		t.Fatalf("expected next_attempt_at %v, got %v", now.Add(leaseDuration), claimed[0].NextAttemptAt)
	}
	if claimed[1].Attempts != 3 {
		t.Fatalf("expected event 2 attempts incremented to 3, got %d", claimed[1].Attempts)
	}
}

func TestClaimPendingOrderTimeoutOutbox_Validation(t *testing.T) {
	// 验证 limit <= 0 时直接返回 nil, nil
	claimed, err := dao.ClaimPendingOrderTimeoutOutbox(
		context.Background(),
		nil,
		time.Now(),
		0,
		time.Minute,
		10,
	)
	if err != nil || claimed != nil {
		t.Fatalf("expected nil, nil for non-positive limit, got %v, %v", claimed, err)
	}
}

func TestClaimPendingOrderTimeoutOutbox_MySQLIntegration(t *testing.T) {
	if os.Getenv("RUN_MYSQL_TEST") != "1" {
		t.Skip("skip MySQL integration test; set RUN_MYSQL_TEST=1 to run")
	}
	db := setupIsolatedDAOTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	publishedTime := now.Add(-time.Hour)
	testEvents := []*model.OrderTimeoutOutbox{
		{
			EventID:       "mysql-claim-1",
			OrderID:       5001,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			Attempts:      0,
			NextAttemptAt: now.Add(-time.Minute), // 可认领
		},
		{
			EventID:       "mysql-claim-2",
			OrderID:       5002,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			Attempts:      1,
			NextAttemptAt: now.Add(-10 * time.Second), // 可认领
		},
		{
			EventID:       "mysql-claim-3",
			OrderID:       5003,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			PublishedAt:   &publishedTime, // 已发布，不可认领
			Attempts:      0,
			NextAttemptAt: now.Add(-time.Minute),
		},
		{
			EventID:       "mysql-claim-4",
			OrderID:       5004,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			Attempts:      0,
			NextAttemptAt: now.Add(5 * time.Minute), // 未到期，不可认领
		},
		{
			EventID:       "mysql-claim-5",
			OrderID:       5005,
			UserID:        1,
			TimeoutAt:     now.Add(30 * time.Minute),
			Attempts:      10, // 达到最大重试，不可认领
			NextAttemptAt: now.Add(-time.Minute),
		},
	}
	for _, e := range testEvents {
		if err := db.Create(e).Error; err != nil {
			t.Fatalf("seed outbox event failed: %v", err)
		}
	}

	// 第一次认领：应精准命中 5001 与 5002
	lease := 2 * time.Minute
	claimed, err := dao.ClaimPendingOrderTimeoutOutbox(ctx, db, now, 10, lease, 10)
	if err != nil {
		t.Fatalf("ClaimPendingOrderTimeoutOutbox failed: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed events, got %d", len(claimed))
	}
	if claimed[0].OrderID != 5001 || claimed[1].OrderID != 5002 {
		t.Fatalf("unexpected claimed order IDs: %v, %v", claimed[0].OrderID, claimed[1].OrderID)
	}
	if claimed[0].Attempts != 1 || claimed[1].Attempts != 2 {
		t.Fatalf("expected attempts [1, 2], got [%d, %d]", claimed[0].Attempts, claimed[1].Attempts)
	}

	// 第二次并发/连续认领：由于租期已被推进至 now + 2m，此时应返回 0 条（租期隔离防重认领）
	secondClaim, err := dao.ClaimPendingOrderTimeoutOutbox(ctx, db, now, 10, lease, 10)
	if err != nil {
		t.Fatalf("second ClaimPendingOrderTimeoutOutbox failed: %v", err)
	}
	if len(secondClaim) != 0 {
		t.Fatalf("expected 0 events in lease period, got %d", len(secondClaim))
	}
}
