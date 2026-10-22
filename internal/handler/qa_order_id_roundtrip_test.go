package handler

// Independent (03-ring) verification of GAP-012. Written by qa-validator; it
// does not reuse the implementation's fixtures or stubs. Assertions are made on
// raw response bytes and on the int64 that actually reaches the service layer,
// because a passing unmarshal into int64 can still hide a truncated value.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"go-order-management-system/internal/middleware"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
	"go-order-management-system/pkg/idgen"

	"github.com/gin-gonic/gin"
)

// qaIDRecorder records every int64 that crosses into the service layer.
type qaIDRecorder struct {
	order      *model.Order
	items      []*model.OrderItem
	detailIDs  []int64
	payIDs     []int64
	finishIDs  []int64
	cancelIDs  []int64
	createCall int
}

func (s *qaIDRecorder) CreateOrder(_ context.Context, _ int64, _ request.CreateOrderRequest) (*model.Order, error) {
	s.createCall++
	return s.order, nil
}

func (s *qaIDRecorder) ListOrders(context.Context, int64, int, int) ([]*model.Order, int64, error) {
	return []*model.Order{s.order}, 1, nil
}

func (s *qaIDRecorder) ListAdminOrders(context.Context, request.ListAdminOrdersRequest) ([]*model.Order, int64, error) {
	return []*model.Order{s.order}, 1, nil
}

func (s *qaIDRecorder) GetOrderByID(_ context.Context, _ int64, id int64) (*model.Order, []*model.OrderItem, error) {
	s.detailIDs = append(s.detailIDs, id)
	return s.order, s.items, nil
}

func (s *qaIDRecorder) PayOrder(_ context.Context, _ int64, id int64) error {
	s.payIDs = append(s.payIDs, id)
	return nil
}

func (s *qaIDRecorder) FinishOrder(_ context.Context, _ int64, id int64) error {
	s.finishIDs = append(s.finishIDs, id)
	return nil
}

func (s *qaIDRecorder) CancelOrder(_ context.Context, _ int64, id int64) error {
	s.cancelIDs = append(s.cancelIDs, id)
	return nil
}

func qaRouter(rec *qaIDRecorder) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewOrderHandler(rec)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, int64(7))
		c.Next()
	})
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/:id", h.GetOrderByID)
	r.PATCH("/orders/:id/pay", h.PayOrder)
	r.PATCH("/orders/:id/finish", h.FinishOrder)
	r.PATCH("/orders/:id/cancel", h.CancelOrders)
	return r
}

type qaCreateEnvelope struct {
	Code int `json:"code"`
	Data struct {
		// Deliberately typed string: the wire contract under test is
		// "quoted digits", not "a number that happens to fit".
		ID      string `json:"id"`
		OrderNo string `json:"order_no"`
	} `json:"data"`
}

type qaDetailEnvelope struct {
	Code int `json:"code"`
	Data struct {
		Order struct {
			ID string `json:"id"`
		} `json:"order"`
		Items []struct {
			ID      int64  `json:"id"`
			OrderID string `json:"order_id"`
		} `json:"items"`
	} `json:"data"`
}

// qaSnowflakeIDs covers the value-range boundary, not just one fixture:
// 2^53-1 is the exact-integer ceiling, 2^53 the first lossy value, and the
// generator-produced value is what production actually emits today.
func qaSnowflakeIDs(t *testing.T) map[string]int64 {
	t.Helper()

	gen, err := idgen.NewSnowflakeWithEpoch(7, 1704067200000)
	if err != nil {
		t.Fatalf("build generator: %v", err)
	}
	// Fixed timestamp so the id is reproducible; 2026-10-22T00:00:00Z.
	gen.SetTimeFunc(func() int64 { return 1792627200000 })
	// Two ids in the same millisecond: the second one differs by +1 in the
	// sequence bits. That is the shape a real burst produces, and it is the one
	// that loses precision (the first id has trailing zero bits and happens to
	// be exactly representable as a float64, which would make a lossy-fixture
	// assertion silently vacuous).
	_ = gen.NextID()
	generated := gen.NextID()

	return map[string]int64{
		// Exact-integer boundary and its neighbours.
		"max_safe_2_53_minus_1":  int64(1)<<53 - 1,
		"exact_2_53":             int64(1) << 53,   // exactly representable
		"first_lossy_2_53_plus1": int64(1)<<53 + 1, // first integer that is not
		// Real-shaped ids: today's generator output and a fixed 19-digit id.
		"generator_18_digit":     generated,
		"nineteen_digit_qa_fix":  int64(9223372036854775807) / 8,
		"eighteen_digit_current": int64(371447562240004097),
	}
}

// TestQAOrderIDRoundTripThroughRealHTTPPath is the end-to-end proof for the
// GAP-012 user-visible failure: take the id exactly as the browser would read it
// from the response body, paste it into the URL path, and assert the int64 that
// reaches the service layer is bit-identical to the created one.
func TestQAOrderIDRoundTripThroughRealHTTPPath(t *testing.T) {
	for name, id := range qaSnowflakeIDs(t) {
		t.Run(name, func(t *testing.T) {
			rec := &qaIDRecorder{
				order: &model.Order{ID: id, OrderNo: "QA-" + name},
				items: []*model.OrderItem{{ID: 11, OrderID: id, ProductID: 3, Quantity: 2}},
			}
			router := qaRouter(rec)

			// --- step 1: create, read the id the way a browser would ---
			createReq := httptest.NewRequest(http.MethodPost, "/orders",
				bytes.NewBufferString(`{"idempotency_key":"qa-key","items":[{"product_id":3,"quantity":2}]}`))
			createReq.Header.Set("Content-Type", "application/json")
			createRec := httptest.NewRecorder()
			router.ServeHTTP(createRec, createReq)

			if createRec.Code != http.StatusOK {
				t.Fatalf("create status=%d body=%s", createRec.Code, createRec.Body.String())
			}
			wantQuoted := fmt.Sprintf(`"id":"%d"`, id)
			if !strings.Contains(createRec.Body.String(), wantQuoted) {
				t.Fatalf("create body must quote the id: want %s in %s", wantQuoted, createRec.Body.String())
			}

			var created qaCreateEnvelope
			if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			browserVisibleID := created.Data.ID
			if browserVisibleID != strconv.FormatInt(id, 10) {
				t.Fatalf("browser-visible id %q != authoritative %q", browserVisibleID, strconv.FormatInt(id, 10))
			}

			// --- step 2: paste it into the four paths, verbatim ---
			for _, probe := range []struct {
				method string
				path   string
			}{
				{http.MethodGet, "/orders/" + browserVisibleID},
				{http.MethodPatch, "/orders/" + browserVisibleID + "/pay"},
				{http.MethodPatch, "/orders/" + browserVisibleID + "/finish"},
				{http.MethodPatch, "/orders/" + browserVisibleID + "/cancel"},
			} {
				req := httptest.NewRequest(probe.method, probe.path, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Fatalf("%s %s status=%d body=%s", probe.method, probe.path, w.Code, w.Body.String())
				}
			}

			// --- step 3: exact equality at the service boundary ---
			type arrival struct {
				endpoint string
				got      []int64
			}
			for _, a := range []arrival{
				{"detail", rec.detailIDs},
				{"pay", rec.payIDs},
				{"finish", rec.finishIDs},
				{"cancel", rec.cancelIDs},
			} {
				if len(a.got) != 1 {
					t.Fatalf("%s: expected exactly 1 arrival, got %v", a.endpoint, a.got)
				}
				if a.got[0] != id {
					t.Fatalf("%s: service received %d, want %d (delta=%d)",
						a.endpoint, a.got[0], id, a.got[0]-id)
				}
			}

			// --- step 4: the detail payload feeds the same paths ---
			detailRec := httptest.NewRecorder()
			router.ServeHTTP(detailRec, httptest.NewRequest(http.MethodGet, "/orders/"+browserVisibleID, nil))
			var detail qaDetailEnvelope
			if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
				t.Fatalf("decode detail: %v body=%s", err, detailRec.Body.String())
			}
			if detail.Data.Order.ID != browserVisibleID {
				t.Fatalf("detail order.id %q != created id %q", detail.Data.Order.ID, browserVisibleID)
			}
			if len(detail.Data.Items) != 1 || detail.Data.Items[0].OrderID != browserVisibleID {
				t.Fatalf("detail items[0].order_id %+v must equal %q", detail.Data.Items, browserVisibleID)
			}
			// items[].id is auto-increment and must stay a bare number.
			if !strings.Contains(detailRec.Body.String(), `"id":11`) {
				t.Fatalf("item id must stay unquoted: %s", detailRec.Body.String())
			}
		})
	}
}

// TestQANumericIDPathLosesTheOrder is the negative half (D3): if the id ever
// travelled as a JS number, the path the browser builds is a *different* id and
// the service layer demonstrably receives that different id. It fails loudly if
// a fixture stops being lossy instead of silently asserting nothing.
func TestQANumericIDPathLosesTheOrder(t *testing.T) {
	lossy := 0
	for name, id := range qaSnowflakeIDs(t) {
		asJSNumber := int64(float64(id))
		if asJSNumber == id {
			// Only values <= 2^53 are allowed to survive; anything above it that
			// survives means the negative control itself stopped working.
			if id > int64(1)<<53 {
				t.Fatalf("%s: %d > 2^53 yet survived float64; negative control is dead", name, id)
			}
			t.Logf("%s: id=%d is exactly representable (control boundary, not a defect)", name, id)
			continue
		}
		lossy++

		truncatedPath := strconv.FormatInt(asJSNumber, 10)
		rec := &qaIDRecorder{order: &model.Order{ID: id}, items: []*model.OrderItem{{ID: 1, OrderID: id}}}
		router := qaRouter(rec)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+truncatedPath, nil))

		if len(rec.detailIDs) != 1 {
			t.Fatalf("%s: expected 1 arrival, got %v", name, rec.detailIDs)
		}
		if rec.detailIDs[0] != asJSNumber {
			t.Fatalf("%s: service received %d, want truncated %d", name, rec.detailIDs[0], asJSNumber)
		}
		if rec.detailIDs[0] == id {
			t.Fatalf("%s: truncation produced the same id; negative control is dead", name)
		}
		t.Logf("%s: id=%d js-number=%d delta=%d (service receives a different order id)",
			name, id, asJSNumber, asJSNumber-id)
	}
	if lossy == 0 {
		t.Fatal("no fixture lost precision; the GAP-012 negative control no longer proves anything")
	}
}

// TestQAOrderIDListAndAdminListAlsoQuoted closes the surface question: list
// payloads carry the same id and the UI copies it from there, so a contract that
// only held on create/detail would still 404 on the very first click.
func TestQAOrderIDListAndAdminListAlsoQuoted(t *testing.T) {
	id := int64(371447562240004097)
	rec := &qaIDRecorder{order: &model.Order{ID: id, OrderNo: "QA-LIST"}}
	gin.SetMode(gin.TestMode)
	h := NewOrderHandler(rec)
	r := gin.New()
	// ListOrders reads the identity from the context, so it needs the same
	// middleware the protected group installs.
	r.Use(func(c *gin.Context) {
		c.Set(middleware.UserIDKey, int64(7))
		c.Next()
	})
	r.GET("/orders", h.ListOrders)
	r.GET("/admin/orders", h.ListAdminOrders)

	for _, path := range []string{"/orders?page=1&page_size=10", "/admin/orders?page=1&page_size=10"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), fmt.Sprintf(`"id":"%d"`, id)) {
			t.Fatalf("%s must quote the order id: %s", path, w.Body.String())
		}
		if strings.Contains(w.Body.String(), fmt.Sprintf(`"id":%d`, id)) {
			t.Fatalf("%s must not emit a bare numeric order id: %s", path, w.Body.String())
		}
	}
}
