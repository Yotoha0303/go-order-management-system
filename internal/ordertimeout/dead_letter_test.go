package ordertimeout

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
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

type mockAcknowledger struct {
	mu       sync.Mutex
	acked    []uint64
	nacked   []mockNackCall
	rejected []mockRejectCall
}

type mockNackCall struct {
	tag      uint64
	multiple bool
	requeue  bool
}

type mockRejectCall struct {
	tag     uint64
	requeue bool
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acked = append(m.acked, tag)
	return nil
}

func (m *mockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nacked = append(m.nacked, mockNackCall{tag: tag, multiple: multiple, requeue: requeue})
	return nil
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rejected = append(m.rejected, mockRejectCall{tag: tag, requeue: requeue})
	return nil
}

type mockMetricsRecorder struct {
	mu      sync.Mutex
	records []struct {
		reason  string
		handled string
	}
}

func (m *mockMetricsRecorder) RecordOrderTimeoutDeadLetter(reason, handled string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, struct {
		reason  string
		handled string
	}{reason: reason, handled: handled})
}

type mockDLQDBConnector struct {
	shouldFail bool
	queries    []string
}

func (c *mockDLQDBConnector) Connect(context.Context) (driver.Conn, error) {
	return &mockDLQDBConn{cfg: c}, nil
}

func (c *mockDLQDBConnector) Driver() driver.Driver {
	return &mockDLQDBDriver{}
}

type mockDLQDBDriver struct{}

func (d *mockDLQDBDriver) Open(string) (driver.Conn, error) {
	return &mockDLQDBConn{}, nil
}

type mockDLQDBConn struct {
	cfg *mockDLQDBConnector
}

func (c *mockDLQDBConn) Prepare(string) (driver.Stmt, error) {
	return &mockDLQDBStmt{cfg: c.cfg}, nil
}
func (c *mockDLQDBConn) Close() error { return nil }
func (c *mockDLQDBConn) Begin() (driver.Tx, error) {
	return &mockDLQDBTx{}, nil
}
func (c *mockDLQDBConn) Ping(context.Context) error { return nil }

func (c *mockDLQDBConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.cfg != nil {
		c.cfg.queries = append(c.cfg.queries, query)
		if c.cfg.shouldFail {
			return nil, errors.New("simulated database disk full error")
		}
	}
	return mockResult{lastInsertID: 1, rowsAffected: 1}, nil
}

func (c *mockDLQDBConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return &mockDLQDBRows{}, nil
}

type mockDLQDBTx struct{}

func (t *mockDLQDBTx) Commit() error   { return nil }
func (t *mockDLQDBTx) Rollback() error { return nil }

type mockDLQDBStmt struct {
	cfg *mockDLQDBConnector
}

func (s *mockDLQDBStmt) Close() error  { return nil }
func (s *mockDLQDBStmt) NumInput() int { return -1 }
func (s *mockDLQDBStmt) Exec([]driver.Value) (driver.Result, error) {
	if s.cfg != nil && s.cfg.shouldFail {
		return nil, errors.New("simulated database disk full error")
	}
	return mockResult{lastInsertID: 1, rowsAffected: 1}, nil
}
func (s *mockDLQDBStmt) Query([]driver.Value) (driver.Rows, error) {
	return &mockDLQDBRows{}, nil
}

type mockDLQDBRows struct{}

func (r *mockDLQDBRows) Columns() []string { return []string{"id"} }
func (r *mockDLQDBRows) Close() error      { return nil }
func (r *mockDLQDBRows) Next([]driver.Value) error {
	return driver.ErrBadConn
}

type stubCanceller struct {
	err error
}

func (s *stubCanceller) CancelExpiredOrder(ctx context.Context, eventID string, orderID int64) error {
	return s.err
}

func TestConsumeFailedLoop_SuccessDeadLetter(t *testing.T) {
	connector := &mockDLQDBConnector{}
	sqlDB := sql.OpenDB(connector)
	defer sqlDB.Close()

	gormDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))
	metrics := &mockMetricsRecorder{}

	worker := &Worker{
		config: Config{
			PublishBatchSize: 10,
			OutboxRetryDelay: 5 * time.Second,
		},
		db:       gormDB,
		logger:   testLogger,
		now:      func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC) },
		topology: productionTopology,
		metrics:  metrics,
	}

	ack := &mockAcknowledger{}
	delivery := amqp.Delivery{
		Acknowledger: ack,
		Headers: amqp.Table{
			"x-death": []interface{}{
				amqp.Table{
					"count":  int64(3),
					"queue":  "order.timeout.cancel",
					"reason": "rejected",
				},
			},
		},
		DeliveryTag: 42,
		MessageId:   "msg-success-dlq",
		Body:        []byte(`{"event_id":"evt-dlq-101","order_id":5001,"user_id":12,"timeout_at":"2026-10-21T12:00:00Z"}`),
	}

	worker.handleFailedDelivery(context.Background(), delivery)

	// 1. 验证终态 Ack 语义
	ack.mu.Lock()
	if len(ack.acked) != 1 || ack.acked[0] != 42 {
		t.Errorf("expected delivery to be acknowledged with tag 42, got acked: %v", ack.acked)
	}
	if len(ack.nacked) > 0 {
		t.Errorf("expected zero Nacks, got: %v", ack.nacked)
	}
	ack.mu.Unlock()

	// 2. 验证数据库保存
	if len(connector.queries) == 0 {
		t.Fatal("expected database INSERT query to be executed")
	}

	// 3. 验证结构化日志
	logStr := logBuf.String()
	if !strings.Contains(logStr, "ORDER_TIMEOUT_DLQ_ALERT: dead letter received in failed queue") {
		t.Errorf("expected DLQ alert log, got: %s", logStr)
	}
	if !strings.Contains(logStr, "event_id=evt-dlq-101") || !strings.Contains(logStr, "order_id=5001") {
		t.Errorf("expected log to contain event_id and order_id, got: %s", logStr)
	}
	if !strings.Contains(logStr, "handled=stored") {
		t.Errorf("expected log to contain handled=stored, got: %s", logStr)
	}

	// 4. 验证指标递增
	metrics.mu.Lock()
	if len(metrics.records) != 1 || metrics.records[0].reason != "max_retries_exceeded" || metrics.records[0].handled != "stored" {
		t.Errorf("unexpected metrics records: %v", metrics.records)
	}
	metrics.mu.Unlock()
}

func TestConsumeFailedLoop_MalformedPayloadPoisonPill(t *testing.T) {
	connector := &mockDLQDBConnector{}
	sqlDB := sql.OpenDB(connector)
	defer sqlDB.Close()

	gormDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))
	metrics := &mockMetricsRecorder{}

	worker := &Worker{
		config: Config{
			PublishBatchSize: 10,
			OutboxRetryDelay: 5 * time.Second,
		},
		db:       gormDB,
		logger:   testLogger,
		now:      time.Now,
		topology: productionTopology,
		metrics:  metrics,
	}

	ack := &mockAcknowledger{}
	delivery := amqp.Delivery{
		Acknowledger: ack,
		Headers:      amqp.Table{"retry_count": 2},
		DeliveryTag:  99,
		MessageId:    "msg-poison-pill",
		Body:         []byte(`{malformed_non_json_payload_that_must_not_crash`),
	}

	// 毒丸执行宽容解析
	worker.handleFailedDelivery(context.Background(), delivery)

	// 必须无条件 Ack，杜绝无限死循环
	ack.mu.Lock()
	if len(ack.acked) != 1 || ack.acked[0] != 99 {
		t.Errorf("expected poison pill delivery to be Acked, got acked: %v", ack.acked)
	}
	if len(ack.nacked) > 0 {
		t.Errorf("poison pill must never be Nacked, got nacks: %v", ack.nacked)
	}
	ack.mu.Unlock()

	logStr := logBuf.String()
	if !strings.Contains(logStr, "reason=decode_error") {
		t.Errorf("expected reason=decode_error in log, got: %s", logStr)
	}

	metrics.mu.Lock()
	if len(metrics.records) != 1 || metrics.records[0].reason != "decode_error" || metrics.records[0].handled != "stored" {
		t.Errorf("unexpected metrics records for poison pill: %v", metrics.records)
	}
	metrics.mu.Unlock()
}

func TestConsumeFailedLoop_DBFailureEmergencyFallback(t *testing.T) {
	connector := &mockDLQDBConnector{shouldFail: true}
	sqlDB := sql.OpenDB(connector)
	defer sqlDB.Close()

	gormDB, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))
	metrics := &mockMetricsRecorder{}

	worker := &Worker{
		config: Config{
			PublishBatchSize: 10,
			OutboxRetryDelay: 5 * time.Second,
		},
		db:       gormDB,
		logger:   testLogger,
		now:      time.Now,
		topology: productionTopology,
		metrics:  metrics,
	}

	ack := &mockAcknowledger{}
	delivery := amqp.Delivery{
		Acknowledger: ack,
		Headers:      amqp.Table{"retry_count": 5},
		DeliveryTag:  105,
		MessageId:    "msg-db-fail",
		Body:         []byte(`{"event_id":"evt-db-fail","order_id":7001,"user_id":1}`),
	}

	worker.handleFailedDelivery(context.Background(), delivery)

	// 即使 DB 失败，仍然必须 Ack 释放队列，并通过日志与指标兜底
	ack.mu.Lock()
	if len(ack.acked) != 1 || ack.acked[0] != 105 {
		t.Errorf("expected delivery to be Acked despite DB failure, got: %v", ack.acked)
	}
	ack.mu.Unlock()

	logStr := logBuf.String()
	if !strings.Contains(logStr, "emergency fallback: failed to persist dead letter to database") {
		t.Errorf("expected emergency fallback log, got: %s", logStr)
	}
	if !strings.Contains(logStr, "handled=fallback_logged") {
		t.Errorf("expected handled=fallback_logged in alert log, got: %s", logStr)
	}

	metrics.mu.Lock()
	if len(metrics.records) != 1 || metrics.records[0].handled != "fallback_logged" {
		t.Errorf("expected metrics handled=fallback_logged, got: %v", metrics.records)
	}
	metrics.mu.Unlock()
}

func TestConsumeLoop_CancelQueueRetries_ExceededRoutesToDLX(t *testing.T) {
	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))

	worker := &Worker{
		config: Config{
			OutboxRetryDelay: 5 * time.Millisecond,
		},
		canceller: &stubCanceller{err: errors.New("simulated order cancel error")},
		logger:    testLogger,
	}

	ack := &mockAcknowledger{}
	delivery := amqp.Delivery{
		Acknowledger: ack,
		Headers: amqp.Table{
			"x-death": []interface{}{
				amqp.Table{
					"count": int64(3), // 达到 defaultMaxCancelRetries
					"queue": "order.timeout.cancel",
				},
			},
		},
		DeliveryTag: 1,
		Body:        []byte(`{"event_id":"evt-exceed","order_id":3001,"user_id":1,"timeout_at":"2026-10-21T12:00:00Z"}`),
	}

	deliveries := make(chan amqp.Delivery, 1)
	deliveries <- delivery
	close(deliveries)

	err := worker.consumeLoop(context.Background(), deliveries)
	if err != nil && !strings.Contains(err.Error(), "closed") {
		t.Fatalf("unexpected error: %v", err)
	}

	// 达到最大重试上限，必须 Nack(false, false) 让 DLX 路由到 failedQueue
	ack.mu.Lock()
	defer ack.mu.Unlock()
	if len(ack.nacked) != 1 {
		t.Fatalf("expected 1 Nack call, got: %v", ack.nacked)
	}
	if ack.nacked[0].requeue != false {
		t.Errorf("expected requeue=false to trigger DLX, got requeue: %v", ack.nacked[0].requeue)
	}

	logStr := logBuf.String()
	if !strings.Contains(logStr, "order timeout cancel reached max retries, dead-lettering to failed queue") {
		t.Errorf("expected dead-lettering log, got: %s", logStr)
	}
}

func TestConsumeLoop_CancelQueueRetries_BelowThresholdRequeues(t *testing.T) {
	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))

	worker := &Worker{
		config: Config{
			OutboxRetryDelay: 1 * time.Millisecond,
		},
		canceller: &stubCanceller{err: errors.New("temporary lock conflict")},
		logger:    testLogger,
	}

	ack := &mockAcknowledger{}
	delivery := amqp.Delivery{
		Acknowledger: ack,
		Headers: amqp.Table{
			"x-death": []interface{}{
				amqp.Table{
					"count": int64(1), // 未达到上限 3
					"queue": "order.timeout.cancel",
				},
			},
		},
		DeliveryTag: 2,
		Body:        []byte(`{"event_id":"evt-retry","order_id":3002,"user_id":1,"timeout_at":"2026-10-21T12:00:00Z"}`),
	}

	deliveries := make(chan amqp.Delivery, 1)
	deliveries <- delivery
	close(deliveries)

	_ = worker.consumeLoop(context.Background(), deliveries)

	ack.mu.Lock()
	defer ack.mu.Unlock()
	if len(ack.nacked) != 1 {
		t.Fatalf("expected 1 Nack call, got: %v", ack.nacked)
	}
	if ack.nacked[0].requeue != true {
		t.Errorf("expected requeue=true for retries below threshold, got: %v", ack.nacked[0].requeue)
	}
}

func TestExtractDeathCount(t *testing.T) {
	t.Run("nil headers returns 0", func(t *testing.T) {
		if got := extractDeathCount(nil); got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("fallback to retry_count header", func(t *testing.T) {
		headers := amqp.Table{"retry_count": 4}
		if got := extractDeathCount(headers); got != 4 {
			t.Errorf("expected 4, got %d", got)
		}
	})

	t.Run("interface slice with amqp.Table", func(t *testing.T) {
		headers := amqp.Table{
			"x-death": []interface{}{
				amqp.Table{"count": int64(2)},
				amqp.Table{"count": int64(3)},
			},
		}
		if got := extractDeathCount(headers); got != 5 {
			t.Errorf("expected 5, got %d", got)
		}
	})

	t.Run("interface slice with map[string]interface{}", func(t *testing.T) {
		headers := amqp.Table{
			"x-death": []interface{}{
				map[string]interface{}{"count": int64(4)},
			},
		}
		if got := extractDeathCount(headers); got != 4 {
			t.Errorf("expected 4, got %d", got)
		}
	})

	t.Run("amqp.Table slice", func(t *testing.T) {
		headers := amqp.Table{
			"x-death": []amqp.Table{
				{"count": int64(7)},
			},
		}
		if got := extractDeathCount(headers); got != 7 {
			t.Errorf("expected 7, got %d", got)
		}
	})
}
