package dao_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"go-order-management-system/internal/dao"
	"go-order-management-system/internal/model"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r mockResult) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r mockResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type mockDeadLetterConnector struct {
	lastInsertID int64
	lastQuery    string
}

func (c *mockDeadLetterConnector) Connect(context.Context) (driver.Conn, error) {
	return &mockDeadLetterConn{cfg: c}, nil
}

func (c *mockDeadLetterConnector) Driver() driver.Driver {
	return &mockDeadLetterDriver{}
}

type mockDeadLetterDriver struct{}

func (d *mockDeadLetterDriver) Open(string) (driver.Conn, error) {
	return &mockDeadLetterConn{}, nil
}

type mockDeadLetterConn struct {
	cfg *mockDeadLetterConnector
}

func (c *mockDeadLetterConn) Prepare(string) (driver.Stmt, error) {
	return &mockDeadLetterStmt{cfg: c.cfg}, nil
}
func (c *mockDeadLetterConn) Close() error { return nil }
func (c *mockDeadLetterConn) Begin() (driver.Tx, error) {
	return &mockDeadLetterTx{}, nil
}
func (c *mockDeadLetterConn) Ping(context.Context) error { return nil }

func (c *mockDeadLetterConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.cfg != nil {
		c.cfg.lastQuery = query
		c.cfg.lastInsertID++
		return mockResult{lastInsertID: c.cfg.lastInsertID, rowsAffected: 1}, nil
	}
	return mockResult{lastInsertID: 1, rowsAffected: 1}, nil
}

func (c *mockDeadLetterConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return &mockDeadLetterRows{}, nil
}

type mockDeadLetterTx struct{}

func (t *mockDeadLetterTx) Commit() error   { return nil }
func (t *mockDeadLetterTx) Rollback() error { return nil }

type mockDeadLetterStmt struct {
	cfg *mockDeadLetterConnector
}

func (s *mockDeadLetterStmt) Close() error  { return nil }
func (s *mockDeadLetterStmt) NumInput() int { return -1 }
func (s *mockDeadLetterStmt) Exec([]driver.Value) (driver.Result, error) {
	if c := s.cfg; c != nil {
		c.lastInsertID++
		return mockResult{lastInsertID: c.lastInsertID, rowsAffected: 1}, nil
	}
	return mockResult{lastInsertID: 1, rowsAffected: 1}, nil
}
func (s *mockDeadLetterStmt) Query([]driver.Value) (driver.Rows, error) {
	return &mockDeadLetterRows{}, nil
}

type mockDeadLetterRows struct{}

func (r *mockDeadLetterRows) Columns() []string { return []string{"id"} }
func (r *mockDeadLetterRows) Close() error      { return nil }
func (r *mockDeadLetterRows) Next([]driver.Value) error {
	return driver.ErrBadConn
}

func TestCreateOrderTimeoutDeadLetter(t *testing.T) {
	t.Run("nil dead letter record returns error", func(t *testing.T) {
		err := dao.CreateOrderTimeoutDeadLetter(context.Background(), &gorm.DB{}, nil)
		if err == nil {
			t.Fatal("expected error for nil dead letter record, got nil")
		}
	})

	t.Run("nil database returns error", func(t *testing.T) {
		err := dao.CreateOrderTimeoutDeadLetter(context.Background(), nil, &model.OrderTimeoutDeadLetter{})
		if err == nil {
			t.Fatal("expected error for nil database connection, got nil")
		}
	})

	t.Run("successful persistence via mock db", func(t *testing.T) {
		connector := &mockDeadLetterConnector{}
		sqlDB := sql.OpenDB(connector)
		defer sqlDB.Close()

		gormDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
			Conn:                      sqlDB,
			SkipInitializeWithVersion: true,
		}), &gorm.Config{
			Logger: logger.Discard,
		})
		if err != nil {
			t.Fatalf("failed to open mock gorm db: %v", err)
		}

		now := time.Now().UTC()
		dl := &model.OrderTimeoutDeadLetter{
			MessageID:   "msg-test-1",
			EventID:     "evt-test-1",
			OrderID:     8888,
			UserID:      99,
			RetryCount:  3,
			Reason:      "max_retries_exceeded",
			Payload:     `{"event_id":"evt-test-1","order_id":8888}`,
			ErrorDetail: "timeout reached",
			Status:      "PENDING",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := dao.CreateOrderTimeoutDeadLetter(context.Background(), gormDB, dl); err != nil {
			t.Fatalf("CreateOrderTimeoutDeadLetter failed: %v", err)
		}

		if !strings.Contains(strings.ToLower(connector.lastQuery), "order_timeout_dead_letters") {
			t.Errorf("expected query to insert into order_timeout_dead_letters, got: %s", connector.lastQuery)
		}
	})
}
