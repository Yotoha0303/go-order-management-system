package dao_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"go-order-management-system/internal/dao"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type mockInventoryResult struct {
	rowsAffected int64
}

func (r mockInventoryResult) LastInsertId() (int64, error) { return 1, nil }
func (r mockInventoryResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

type mockInventoryConnector struct {
	lastQuery    string
	rowsAffected int64
	execErr      error
}

func (c *mockInventoryConnector) Connect(context.Context) (driver.Conn, error) {
	return &mockInventoryConn{cfg: c}, nil
}
func (c *mockInventoryConnector) Driver() driver.Driver { return &mockInventoryDriver{} }

type mockInventoryDriver struct{}

func (d *mockInventoryDriver) Open(string) (driver.Conn, error) { return &mockInventoryConn{}, nil }

type mockInventoryConn struct {
	cfg *mockInventoryConnector
}

func (c *mockInventoryConn) Prepare(string) (driver.Stmt, error) {
	return &mockInventoryStmt{cfg: c.cfg}, nil
}
func (c *mockInventoryConn) Close() error { return nil }
func (c *mockInventoryConn) Begin() (driver.Tx, error) {
	return &mockInventoryTx{}, nil
}
func (c *mockInventoryConn) Ping(context.Context) error { return nil }

func (c *mockInventoryConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.cfg != nil {
		c.cfg.lastQuery = query
		if c.cfg.execErr != nil {
			return nil, c.cfg.execErr
		}
		return mockInventoryResult{rowsAffected: c.cfg.rowsAffected}, nil
	}
	return mockInventoryResult{rowsAffected: 1}, nil
}

func (c *mockInventoryConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.cfg != nil {
		c.cfg.lastQuery = query
	}
	return &mockInventoryRows{}, nil
}

type mockInventoryStmt struct {
	cfg *mockInventoryConnector
}

func (s *mockInventoryStmt) Close() error  { return nil }
func (s *mockInventoryStmt) NumInput() int { return -1 }
func (s *mockInventoryStmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.cfg != nil {
		if s.cfg.execErr != nil {
			return nil, s.cfg.execErr
		}
		return mockInventoryResult{rowsAffected: s.cfg.rowsAffected}, nil
	}
	return mockInventoryResult{rowsAffected: 1}, nil
}
func (s *mockInventoryStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockInventoryRows{}, nil
}

type mockInventoryTx struct{}

func (t *mockInventoryTx) Commit() error   { return nil }
func (t *mockInventoryTx) Rollback() error { return nil }

type mockInventoryRows struct{}

func (r *mockInventoryRows) Columns() []string            { return []string{"id", "order_id", "product_id"} }
func (r *mockInventoryRows) Close() error                 { return nil }
func (r *mockInventoryRows) Next(dest []driver.Value) error { return driver.ErrBadConn }

func newMockInventoryDB(connector *mockInventoryConnector) (*gorm.DB, error) {
	db := sql.OpenDB(connector)
	return gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func TestReserveInventory_Unit(t *testing.T) {
	t.Run("successful reservation updates stock and locked quantity", func(t *testing.T) {
		connector := &mockInventoryConnector{rowsAffected: 1}
		db, err := newMockInventoryDB(connector)
		if err != nil {
			t.Fatalf("newMockInventoryDB failed: %v", err)
		}

		err = dao.ReserveInventory(context.Background(), db, 101, 5)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}

		q := strings.ToLower(connector.lastQuery)
		if !strings.Contains(q, "update") || !strings.Contains(q, "product_inventories") {
			t.Errorf("expected update product_inventories, got: %s", connector.lastQuery)
		}
		if !strings.Contains(q, "stock_quantity") || !strings.Contains(q, "locked_quantity") {
			t.Errorf("expected stock_quantity and locked_quantity in query, got: %s", connector.lastQuery)
		}
		if !strings.Contains(q, "stock_quantity >=") {
			t.Errorf("expected stock_quantity >= condition, got: %s", connector.lastQuery)
		}
	})

	t.Run("insufficient stock returns ErrReserveInventoryFailed", func(t *testing.T) {
		connector := &mockInventoryConnector{rowsAffected: 0}
		db, err := newMockInventoryDB(connector)
		if err != nil {
			t.Fatalf("newMockInventoryDB failed: %v", err)
		}

		err = dao.ReserveInventory(context.Background(), db, 101, 50)
		if !errors.Is(err, dao.ErrReserveInventoryFailed) {
			t.Fatalf("expected ErrReserveInventoryFailed, got: %v", err)
		}
	})
}

func TestReleaseLockedInventory_Unit(t *testing.T) {
	t.Run("successful release restores stock and decrements locked quantity", func(t *testing.T) {
		connector := &mockInventoryConnector{rowsAffected: 1}
		db, err := newMockInventoryDB(connector)
		if err != nil {
			t.Fatalf("newMockInventoryDB failed: %v", err)
		}

		err = dao.ReleaseLockedInventory(context.Background(), db, 101, 3)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}

		q := strings.ToLower(connector.lastQuery)
		if !strings.Contains(q, "locked_quantity >=") {
			t.Errorf("expected locked_quantity >= condition, got: %s", connector.lastQuery)
		}
	})

	t.Run("insufficient locked stock returns ErrReleaseLockedInventoryFailed", func(t *testing.T) {
		connector := &mockInventoryConnector{rowsAffected: 0}
		db, err := newMockInventoryDB(connector)
		if err != nil {
			t.Fatalf("newMockInventoryDB failed: %v", err)
		}

		err = dao.ReleaseLockedInventory(context.Background(), db, 101, 10)
		if !errors.Is(err, dao.ErrReleaseLockedInventoryFailed) {
			t.Fatalf("expected ErrReleaseLockedInventoryFailed, got: %v", err)
		}
	})
}

func TestConsumeLockedInventory_Unit(t *testing.T) {
	t.Run("successful consume decrements locked quantity", func(t *testing.T) {
		connector := &mockInventoryConnector{rowsAffected: 1}
		db, err := newMockInventoryDB(connector)
		if err != nil {
			t.Fatalf("newMockInventoryDB failed: %v", err)
		}

		err = dao.ConsumeLockedInventory(context.Background(), db, 101, 2)
		if err != nil {
			t.Fatalf("expected nil error, got: %v", err)
		}

		q := strings.ToLower(connector.lastQuery)
		if !strings.Contains(q, "locked_quantity") || !strings.Contains(q, "locked_quantity >=") {
			t.Errorf("expected locked_quantity decrement with condition, got: %s", connector.lastQuery)
		}
	})

	t.Run("insufficient locked quantity returns ErrConsumeLockedInventoryFailed", func(t *testing.T) {
		connector := &mockInventoryConnector{rowsAffected: 0}
		db, err := newMockInventoryDB(connector)
		if err != nil {
			t.Fatalf("newMockInventoryDB failed: %v", err)
		}

		err = dao.ConsumeLockedInventory(context.Background(), db, 101, 5)
		if !errors.Is(err, dao.ErrConsumeLockedInventoryFailed) {
			t.Fatalf("expected ErrConsumeLockedInventoryFailed, got: %v", err)
		}
	})
}

func TestListOrderItemsByOrderIDForInventory_OrdersByProductIDASC(t *testing.T) {
	connector := &mockInventoryConnector{}
	db, err := newMockInventoryDB(connector)
	if err != nil {
		t.Fatalf("newMockInventoryDB failed: %v", err)
	}

	_, _ = dao.ListOrderItemsByOrderIDForInventory(context.Background(), db, 12345)
	q := strings.ToLower(connector.lastQuery)
	if !strings.Contains(q, "order by product_id asc") {
		t.Errorf("expected 'order by product_id asc' in query, got: %s", connector.lastQuery)
	}
}
