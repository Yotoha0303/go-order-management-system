package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"go-order-management-system/internal/middleware"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
	"go-order-management-system/internal/response"
	"go-order-management-system/internal/service"

	"github.com/gin-gonic/gin"
)

type stubOrderService struct {
	createResult *model.Order
	createErr    error
	createCalls  int
	userID       int64
	createReq    request.CreateOrderRequest
}

func (s *stubOrderService) CreateOrder(_ context.Context, userID int64, req request.CreateOrderRequest) (*model.Order, error) {
	s.createCalls++
	s.userID = userID
	s.createReq = req
	return s.createResult, s.createErr
}

func (*stubOrderService) ListOrders(context.Context, int64, int, int) ([]*model.Order, int64, error) {
	panic("unexpected ListOrders call")
}

func (*stubOrderService) ListAdminOrders(context.Context, request.ListAdminOrdersRequest) ([]*model.Order, int64, error) {
	panic("unexpected ListAdminOrders call")
}

func (*stubOrderService) GetOrderByID(context.Context, int64, int64) (*model.Order, []*model.OrderItem, error) {
	panic("unexpected GetOrderByID call")
}

func (*stubOrderService) PayOrder(context.Context, int64, int64) error {
	panic("unexpected PayOrder call")
}

func (*stubOrderService) FinishOrder(context.Context, int64, int64) error {
	panic("unexpected FinishOrder call")
}

func (*stubOrderService) CancelOrder(context.Context, int64, int64) error {
	panic("unexpected CancelOrder call")
}

func TestOrderHandlerCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("binds identity and request then returns order", func(t *testing.T) {
		stub := &stubOrderService{createResult: &model.Order{ID: 99, OrderNo: "ORDER-99"}}
		recorder := performCreateOrderRequest(t, stub, `{
			"idempotency_key":"request-1",
			"items":[{"product_id":7,"quantity":2}]
		}`)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", recorder.Code)
		}
		if stub.createCalls != 1 || stub.userID != 42 {
			t.Fatalf("unexpected service call: calls=%d userID=%d", stub.createCalls, stub.userID)
		}
		if stub.createReq.IdempotencyKey != "request-1" || len(stub.createReq.Items) != 1 || stub.createReq.Items[0].ProductID != 7 || stub.createReq.Items[0].Quantity != 2 {
			t.Fatalf("unexpected create request: %+v", stub.createReq)
		}

		var body struct {
			Code int `json:"code"`
			Data struct {
				ID      string `json:"id"`
				OrderNo string `json:"order_no"`
			} `json:"data"`
		}
		decodeHandlerResponse(t, recorder, &body)
		if body.Code != response.CodeSuccess || body.Data.ID != "99" || body.Data.OrderNo != "ORDER-99" {
			t.Fatalf("unexpected response: %+v", body)
		}
	})

	t.Run("rejects invalid body before calling service", func(t *testing.T) {
		stub := &stubOrderService{}
		recorder := performCreateOrderRequest(t, stub, `{
			"idempotency_key":"request-2",
			"items":[{"product_id":7,"quantity":0}]
		}`)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", recorder.Code)
		}
		if stub.createCalls != 0 {
			t.Fatalf("expected service not to be called, got %d calls", stub.createCalls)
		}
	})

	t.Run("maps business error to HTTP response", func(t *testing.T) {
		stub := &stubOrderService{createErr: service.ErrInsufficientStock}
		recorder := performCreateOrderRequest(t, stub, `{
			"idempotency_key":"request-3",
			"items":[{"product_id":7,"quantity":2}]
		}`)

		if recorder.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d", recorder.Code)
		}
		var body struct {
			Code int `json:"code"`
		}
		decodeHandlerResponse(t, recorder, &body)
		if body.Code != response.CodeInsufficientStock {
			t.Fatalf("expected code %d, got %d", response.CodeInsufficientStock, body.Code)
		}
	})
}

type stubAdminOrderService struct {
	stubOrderService
	listAdminResult []*model.Order
	listAdminTotal  int64
	listAdminErr    error
	listAdminCalls  int
	listAdminReq    request.ListAdminOrdersRequest
}

func (s *stubAdminOrderService) ListAdminOrders(_ context.Context, req request.ListAdminOrdersRequest) ([]*model.Order, int64, error) {
	s.listAdminCalls++
	s.listAdminReq = req
	return s.listAdminResult, s.listAdminTotal, s.listAdminErr
}

func TestOrderHandlerListAdminOrders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("binds query params and returns order list", func(t *testing.T) {
		expectedOrders := []*model.Order{
			{ID: 1, OrderNo: "ORD-1", UserID: 10, TotalAmountFen: 100, Status: model.OrderStatusPending},
			{ID: 2, OrderNo: "ORD-2", UserID: 20, TotalAmountFen: 200, Status: model.OrderStatusPaid},
		}
		stub := &stubAdminOrderService{
			listAdminResult: expectedOrders,
			listAdminTotal:  2,
		}

		router := gin.New()
		router.GET("/admin/orders", NewOrderHandler(stub).ListAdminOrders)

		req := httptest.NewRequest(http.MethodGet, "/admin/orders?page=2&page_size=20&user_id=10&status=1", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", recorder.Code)
		}
		if stub.listAdminCalls != 1 {
			t.Fatalf("expected 1 service call, got %d", stub.listAdminCalls)
		}
		if stub.listAdminReq.Page != 2 || stub.listAdminReq.PageSize != 20 {
			t.Fatalf("expected page=2 pageSize=20, got page=%d pageSize=%d", stub.listAdminReq.Page, stub.listAdminReq.PageSize)
		}
		if stub.listAdminReq.UserID == nil || *stub.listAdminReq.UserID != 10 {
			t.Fatalf("expected user_id=10, got %v", stub.listAdminReq.UserID)
		}
		if stub.listAdminReq.Status == nil || *stub.listAdminReq.Status != 1 {
			t.Fatalf("expected status=1, got %v", stub.listAdminReq.Status)
		}

		var body struct {
			Code int                        `json:"code"`
			Data response.OrderListResponse `json:"data"`
		}
		decodeHandlerResponse(t, recorder, &body)
		if body.Code != response.CodeSuccess || body.Data.Total != 2 || len(body.Data.Orders) != 2 {
			t.Fatalf("unexpected response body: %+v", body)
		}
	})

	t.Run("normalizes pagination defaults when query params omitted", func(t *testing.T) {
		stub := &stubAdminOrderService{
			listAdminResult: []*model.Order{},
			listAdminTotal:  0,
		}

		router := gin.New()
		router.GET("/admin/orders", NewOrderHandler(stub).ListAdminOrders)

		req := httptest.NewRequest(http.MethodGet, "/admin/orders", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", recorder.Code)
		}
		if stub.listAdminReq.Page != 1 || stub.listAdminReq.PageSize != 10 {
			t.Fatalf("expected default page=1 pageSize=10, got page=%d pageSize=%d", stub.listAdminReq.Page, stub.listAdminReq.PageSize)
		}
	})

	t.Run("maps service error to HTTP response", func(t *testing.T) {
		stub := &stubAdminOrderService{
			listAdminErr: service.ErrInvalidUserID,
		}

		router := gin.New()
		router.GET("/admin/orders", NewOrderHandler(stub).ListAdminOrders)

		req := httptest.NewRequest(http.MethodGet, "/admin/orders?user_id=-1", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", recorder.Code)
		}
	})
}

type stubDetailOrderService struct {
	stubOrderService
	detailOrder *model.Order
	detailItems []*model.OrderItem
	detailErr   error
}

func (s *stubDetailOrderService) GetOrderByID(context.Context, int64, int64) (*model.Order, []*model.OrderItem, error) {
	return s.detailOrder, s.detailItems, s.detailErr
}

// maxSafeJSONInteger is IEEE-754's exact-integer ceiling (Number.MAX_SAFE_INTEGER).
// Anything above it may be silently rounded by JSON.parse on the consumer side.
const maxSafeJSONInteger = int64(1)<<53 - 1

// snowflake19 is a real-shaped 63-bit Snowflake ID (epoch 2024-01-01, timestamp
// 2032-01-01, worker 1, sequence 1). 19 digits > maxSafeJSONInteger, so a
// double-precision round trip is guaranteed to lose precision.
//
// The 18-digit IDs generated today lose precision in the same way (e.g.
// 371447562240004097 -> 371447562240004096); the 19-digit fixture is used so
// the regression proof does not expire, and it additionally pins the
// "quoted string" contract against the digit-count assumption.
const snowflake19 = int64(1058897343283204097)

func TestOrderIDSerializedAsString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if snowflake19 <= maxSafeJSONInteger {
		t.Fatalf("fixture %d must exceed MAX_SAFE_INTEGER %d", snowflake19, maxSafeJSONInteger)
	}

	t.Run("create response emits quoted 19-digit id in raw bytes", func(t *testing.T) {
		stub := &stubOrderService{createResult: &model.Order{ID: snowflake19, OrderNo: "ORDER-SNOWFLAKE"}}
		recorder := performCreateOrderRequest(t, stub, `{
			"idempotency_key":"precision-1",
			"items":[{"product_id":7,"quantity":1}]
		}`)
		assertQuotedInt64InRawJSON(t, recorder.Body.Bytes(), "id", snowflake19)

		// The consumer-side contract: the JSON decoder must be able to
		// round-trip the ID exactly (this is what the frontend relies on).
		var body struct {
			Data struct {
				ID int64 `json:"id,string"`
			} `json:"data"`
		}
		decodeHandlerResponse(t, recorder, &body)
		if body.Data.ID != snowflake19 {
			t.Fatalf("round-tripped id %d != %d", body.Data.ID, snowflake19)
		}
	})

	t.Run("detail response emits quoted 19-digit ids in raw bytes", func(t *testing.T) {
		order := &model.Order{ID: snowflake19, OrderNo: "ORDER-SNOWFLAKE"}
		stub := &stubDetailOrderService{
			detailOrder: order,
			detailItems: []*model.OrderItem{{ID: 5, OrderID: snowflake19, ProductID: 7, Quantity: 2}},
		}

		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.UserIDKey, int64(42))
			c.Next()
		})
		router.GET("/orders/:id", NewOrderHandler(stub).GetOrderByID)

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/orders/"+strconv.FormatInt(snowflake19, 10), nil)
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d; body=%s", recorder.Code, recorder.Body.String())
		}
		raw := recorder.Body.Bytes()
		assertQuotedInt64InRawJSON(t, raw, "id", snowflake19)
		// The item carries the very same Snowflake order id, so it must be
		// quoted too; a bare number there would reintroduce the defect through
		// the detail payload.
		assertQuotedInt64InRawJSON(t, raw, "order_id", snowflake19)
		// OrderItem.ID is auto-increment (not Snowflake) and intentionally
		// stays a JSON number; pin that so nobody "fixes" it by accident.
		if !bytes.Contains(raw, []byte(`"id":5`)) {
			t.Fatalf("item id must stay an unquoted number, got: %s", raw)
		}

		var body struct {
			Data struct {
				Items []struct {
					ID      int64 `json:"id"`
					OrderID int64 `json:"order_id,string"`
				} `json:"items"`
			} `json:"data"`
		}
		decodeHandlerResponse(t, recorder, &body)
		if len(body.Data.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(body.Data.Items))
		}
		if body.Data.Items[0].OrderID != snowflake19 {
			t.Fatalf("item order_id round-trip %d != %d", body.Data.Items[0].OrderID, snowflake19)
		}
		if body.Data.Items[0].ID != 5 {
			t.Fatalf("item id %d != 5", body.Data.Items[0].ID)
		}
	})

	t.Run("javascript float64 round trip diverges from original id", func(t *testing.T) {
		assertPrecisionLostByFloat64(t, "order id", snowflake19)
	})

	t.Run("javascript float64 round trip diverges from original order_id", func(t *testing.T) {
		// Negative case (D3) for the item payload: the same Snowflake value
		// travels as items[].order_id, so it needs the same proof.
		assertPrecisionLostByFloat64(t, "order item order_id", snowflake19)
	})

	t.Run("unquoted json number is truncated by float64 decode", func(t *testing.T) {
		// Negative case (D3), end to end at the wire level: decode the very
		// payload the model would have produced WITHOUT the `,string` option.
		// The result must differ from the authoritative ID. Asserting on a
		// float64 (not int64) keeps the failure independent of how Go rounds
		// out-of-range floats to integers.
		unquoted := []byte(fmt.Sprintf(`{"code":0,"data":{"id":%d,"order_no":"ORDER-SNOWFLAKE"}}`, snowflake19))
		var body struct {
			Data struct {
				ID float64 `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(unquoted, &body); err != nil {
			t.Fatalf("decode unquoted payload: %v", err)
		}
		jsRoundTripped := int64(body.Data.ID)
		if jsRoundTripped == snowflake19 {
			t.Fatalf("numeric ID survived the float64 path (%d); negative control is dead", snowflake19)
		}
		t.Logf("unquoted payload: original=%d truncated=%d delta=%d",
			snowflake19, jsRoundTripped, jsRoundTripped-snowflake19)

		// And the string form is the one that survives the same decode.
		quoted := []byte(fmt.Sprintf(`{"code":0,"data":{"id":"%d","order_no":"ORDER-SNOWFLAKE"}}`, snowflake19))
		var quotedBody struct {
			Data struct {
				ID int64 `json:"id,string"`
			} `json:"data"`
		}
		if err := json.Unmarshal(quoted, &quotedBody); err != nil {
			t.Fatalf("decode quoted payload: %v", err)
		}
		if quotedBody.Data.ID != snowflake19 {
			t.Fatalf("quoted id %d != %d", quotedBody.Data.ID, snowflake19)
		}
	})
}

// assertQuotedIDInRawJSON inspects the raw response bytes so that a future
// regression to a JSON number cannot slip through an unmarshal-based assertion
// (unmarshalling into an int64 would succeed for many truncated-looking values).
func assertQuotedInt64InRawJSON(t *testing.T, raw []byte, field string, id int64) {
	t.Helper()
	quoted := fmt.Sprintf(`"%s":"%d"`, field, id)
	if !bytes.Contains(raw, []byte(quoted)) {
		t.Fatalf("response must contain %s (quoted string), got: %s", quoted, raw)
	}
	unquoted := fmt.Sprintf(`"%s":%d`, field, id)
	if bytes.Contains(raw, []byte(unquoted)) {
		t.Fatalf("response must not contain unquoted numeric %s==%d: %s", field, id, raw)
	}
	if bytes.Contains(raw, []byte(fmt.Sprintf(`"%s":null`, field))) {
		t.Fatalf("%s must never be null: %s", field, raw)
	}
}

// assertPrecisionLostByFloat64 is the shared negative control (D3): it proves
// that carrying this value as a JSON number loses precision, so the "must be a
// string" judgement still has teeth. It fails loudly if the fixture ever stops
// being lossy instead of silently asserting nothing.
func assertPrecisionLostByFloat64(t *testing.T, label string, id int64) {
	t.Helper()
	jsRoundTripped := int64(float64(id))
	if jsRoundTripped == id {
		t.Fatalf("float64 round trip of %s %d is lossless; the negative control no longer proves anything", label, id)
	}
	t.Logf("%s: original=%d javascript-round-tripped=%d delta=%d",
		label, id, jsRoundTripped, jsRoundTripped-id)
}

func performCreateOrderRequest(t *testing.T, service OrderService, payload string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, int64(42))
		c.Next()
	})
	router.POST("/orders", NewOrderHandler(service).CreateOrder)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeHandlerResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response failed: %v; body=%s", err, recorder.Body.String())
	}
}
