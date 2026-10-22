package handler

// GAP-015 (03-ring follow-up): StockLog.BizID carries the order Snowflake for
// biz_type 3/4 and is exposed to admins through GET /api/v1/stock-logs
// (router/router.go:138). This suite asserts the *raw response bytes* — an
// unmarshal into int64 can still hide a truncated value, so decoding is not
// accepted as evidence.
//
// Placement rationale: the defect is the wire contract of an HTTP endpoint, not
// the model in isolation. internal/model would only prove what `json.Marshal`
// does to one struct; here the Gin response envelope, the handler's error path
// and the nil branch are all exercised together. The service is stubbed, exactly
// like stubOrderService in order_handler_test.go, so no DB is required.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-order-management-system/internal/model"

	"github.com/gin-gonic/gin"
)

// stockLogBizIDFixture is a real-shaped 63-bit Snowflake id (the same fixture
// family as snowflake19 in order_handler_test.go). It is > maxSafeJSONInteger,
// so a float64 round trip is guaranteed to lose precision.
const stockLogBizIDFixture int64 = 1058897343283204097

type stubStockLogService struct {
	logs    []*model.StockLog
	err     error
	gotProd *int64
	calls   int
}

func (s *stubStockLogService) ListStockLogsByProductID(_ context.Context, productID *int64) ([]*model.StockLog, error) {
	s.calls++
	s.gotProd = productID
	return s.logs, s.err
}

func performListStockLogs(t *testing.T, svc StockLogService, target string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stock-logs", NewStockLogHandler(svc).ListStockLogs)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	router.ServeHTTP(recorder, req)
	return recorder
}

// TestStockLogBizIDSerializedAsString is the positive judgement: over the wire,
// an order-backed biz_id must be a quoted string containing every digit.
func TestStockLogBizIDSerializedAsString(t *testing.T) {
	t.Run("order-backed biz_id is a quoted 19-digit string", func(t *testing.T) {
		stub := &stubStockLogService{logs: []*model.StockLog{
			{ID: 11, ProductID: 7, ChangeQuantity: -2, BizType: model.StockBizOrderDeduct, BizID: ptrInt64(stockLogBizIDFixture)},
			{ID: 12, ProductID: 7, ChangeQuantity: 2, BizType: model.StockBizOrderRollback, BizID: ptrInt64(stockLogBizIDFixture)},
		}}

		recorder := performListStockLogs(t, stub, "/stock-logs")
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d; body=%s", recorder.Code, recorder.Body.String())
		}

		raw := recorder.Body.Bytes()
		t.Logf("raw response bytes: %s", raw)
		assertStockLogBizIDQuoted(t, raw, stockLogBizIDFixture)

		// The frontend renders `#${log.biz_id}` verbatim; the digits it receives
		// must be the digits the backend stored.
		var body struct {
			Code int               `json:"code"`
			Data []*model.StockLog `json:"data"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decode response failed: %v; body=%s", err, raw)
		}
		if len(body.Data) != 2 {
			t.Fatalf("expected 2 stock logs, got %d", len(body.Data))
		}
		for i, log := range body.Data {
			if log.BizID == nil || *log.BizID != stockLogBizIDFixture {
				t.Fatalf("data[%d].biz_id lost precision: %v", i, log.BizID)
			}
		}
	})

	t.Run("nil biz_id still marshals as null", func(t *testing.T) {
		// biz_type 1/2 (init / manual add) have no aggregate id. The `,string`
		// option must not turn nil into "0" or "<nil>".
		stub := &stubStockLogService{logs: []*model.StockLog{
			{ID: 21, ProductID: 7, ChangeQuantity: 10, BizType: model.StockBizManualAdd},
		}}

		recorder := performListStockLogs(t, stub, "/stock-logs")
		raw := recorder.Body.Bytes()
		if !bytes.Contains(raw, []byte(`"biz_id":null`)) {
			t.Fatalf(`nil biz_id must marshal as "biz_id":null, got: %s`, raw)
		}
		if bytes.Contains(raw, []byte(`"biz_id":"0"`)) {
			t.Fatalf("nil biz_id must not be coerced to a string value: %s", raw)
		}
	})

	t.Run("product_id filter is forwarded unchanged", func(t *testing.T) {
		stub := &stubStockLogService{logs: []*model.StockLog{}}
		recorder := performListStockLogs(t, stub, "/stock-logs?product_id=7")
		if recorder.Code != http.StatusOK || stub.calls != 1 {
			t.Fatalf("expected 200 and 1 service call, got %d / %d", recorder.Code, stub.calls)
		}
		if stub.gotProd == nil || *stub.gotProd != 7 {
			t.Fatalf("expected product_id=7 forwarded, got %v", stub.gotProd)
		}
	})
}

// TestStockLogBizIDPrecisionNegativeControl is the falsification half (D3): it
// proves that carrying this very value as a JSON number loses precision, so the
// "must be a string" judgement is testing something real. It fails loudly if the
// fixture ever stops being lossy instead of silently asserting nothing.
func TestStockLogBizIDPrecisionNegativeControl(t *testing.T) {
	assertPrecisionLostByFloat64(t, "stock log biz_id", stockLogBizIDFixture)

	// Simulate the pre-fix wire shape: the same id emitted as a bare number.
	unquoted := []byte(fmt.Sprintf(`{"biz_id":%d}`, stockLogBizIDFixture))
	var decoded struct {
		BizID float64 `json:"biz_id"`
	}
	if err := json.Unmarshal(unquoted, &decoded); err != nil {
		t.Fatalf("decode bare-number payload: %v", err)
	}
	truncated := int64(decoded.BizID)
	if truncated == stockLogBizIDFixture {
		t.Fatalf("bare-number payload round-tripped losslessly; negative control is dead: %s", unquoted)
	}
	t.Logf("bare-number payload: original=%d javascript-parsed=%d delta=%d",
		stockLogBizIDFixture, truncated, truncated-stockLogBizIDFixture)
}

func assertStockLogBizIDQuoted(t *testing.T, raw []byte, id int64) {
	t.Helper()
	quoted := fmt.Sprintf(`"biz_id":"%d"`, id)
	if !bytes.Contains(raw, []byte(quoted)) {
		t.Fatalf("response must contain %s (quoted string), got: %s", quoted, raw)
	}
	unquoted := fmt.Sprintf(`"biz_id":%d`, id)
	if bytes.Contains(raw, []byte(unquoted)) {
		t.Fatalf("response must not contain unquoted numeric biz_id==%d: %s", id, raw)
	}
}

func ptrInt64(v int64) *int64 { return &v }

var _ StockLogService = (*stubStockLogService)(nil)
