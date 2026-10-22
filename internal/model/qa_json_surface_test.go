package model

// Independent (03-ring) surface sweep for GAP-012 and its same-class follow-up
// GAP-015: every int64 that can reach an HTTP response body is listed here with
// its wire shape. The point is to answer "was any Snowflake carrier missed?" with
// evidence instead of a grep argument.
//
// There are THREE Snowflake carriers on the external surface, and all three are
// serialized as JSON strings:
//   - Order.ID          -> POST /api/v1/orders, GET /api/v1/orders[/:id]
//   - OrderItem.OrderID -> the items[] array inside GET /api/v1/orders/:id
//   - StockLog.BizID    -> GET /api/v1/stock-logs (admin), nullable
// Every other int64 that reaches a response is an auto-increment id (or a
// reference to one) or an amount. Two further order_id carriers exist on models
// that no handler returns today; they are pinned separately by
// TestQAInternalModelWireShapes as latent risk, not live defects.

import (
	"encoding/json"
	"strings"
	"testing"
)

// qaWireShape reports, for each json field of the marshalled value, whether the
// value is emitted as a JSON string or as anything else (number/null/object).
func qaWireShape(t *testing.T, v any) map[string]string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal into field map: %v (%s)", err, raw)
	}
	shape := make(map[string]string, len(fields))
	for name, value := range fields {
		if len(value) > 0 && value[0] == '"' {
			shape[name] = "string"
		} else {
			shape[name] = "non-string"
		}
	}
	return shape
}

func TestQAOrderWireSurfaceSweep(t *testing.T) {
	order := &Order{ID: 371447562240004097, UserID: 42, OrderNo: "QA-SWEEP", TotalAmountFen: 1999, Status: OrderStatusPending}
	shape := qaWireShape(t, order)

	want := map[string]string{
		"id":               "string", // Snowflake: the GAP-012 fix
		"user_id":          "non-string",
		"total_amount_fen": "non-string",
		"status":           "non-string",
	}
	for field, expected := range want {
		got, ok := shape[field]
		if !ok {
			t.Fatalf("order payload lost field %q: %+v", field, shape)
		}
		if got != expected {
			t.Fatalf("order.%s wire shape=%s, want %s", field, got, expected)
		}
	}
	t.Logf("Order surface: %+v", shape)
}

func TestQAOrderItemWireSurfaceSweep(t *testing.T) {
	item := &OrderItem{
		ID: 5, OrderID: 371447562240004097, ProductID: 7,
		ProductName: "QA", ProductPriceFen: 1000, Quantity: 2, SubtotalFen: 2000,
	}
	shape := qaWireShape(t, item)

	want := map[string]string{
		"order_id":          "string", // same Snowflake as Order.ID: fixed together
		"id":                "non-string",
		"product_id":        "non-string",
		"product_price_fen": "non-string",
		"subtotal_fen":      "non-string",
		"quantity":          "non-string",
	}
	for field, expected := range want {
		got, ok := shape[field]
		if !ok {
			t.Fatalf("order item payload lost field %q: %+v", field, shape)
		}
		if got != expected {
			t.Fatalf("order item %s wire shape=%s, want %s", field, got, expected)
		}
	}
	t.Logf("OrderItem surface: %+v", shape)
}

// TestQAStockLogBizIDShapePinsTheOpenContract pins the CURRENT wire contract for
// StockLog.BizID (the third Snowflake carrier, GAP-015). It carries the order
// Snowflake (order_service.go:188,557) and is exposed by GET /api/v1/stock-logs,
// so it must be a JSON string for the same reason as Order.ID: a bare number is
// truncated past 2^53 by JSON.parse.
//
// This test is a tripwire, not a bug report. If it fails, someone changed the
// shape — either back to a bare number (the GAP-012 defect class) or to some
// other encoding — and must simultaneously re-check the frontend type in
// fronted/src/features/order-inventory/types.ts (`biz_id?: string | null`) and
// the renderer in stock-logs.tsx, which prints the value verbatim.
func TestQAStockLogBizIDShapePinsTheOpenContract(t *testing.T) {
	log := &StockLog{ID: 1, ProductID: 7, BizType: StockBizOrderDeduct, BizID: qaPtr(int64(371447562240004097))}
	shape := qaWireShape(t, log)

	if shape["biz_id"] != "string" {
		t.Fatalf("StockLog.biz_id must be a JSON string (order Snowflake), got %s; frontend type says `string | null`", shape["biz_id"])
	}
	// `,string` must apply to the pointed-to value only: a nil BizID still has
	// to marshal as JSON null (not "", not "0"). The frontend relies on null to
	// render '-' for init/manual-add logs, which carry no aggregate id.
	nilShape := qaWireShape(t, &StockLog{ID: 1, ProductID: 7, BizType: StockBizInit})
	if nilShape["biz_id"] != "non-string" {
		t.Fatalf("nil BizID must marshal as JSON null, got shape %s", nilShape["biz_id"])
	}
	rawNil, err := json.Marshal(&StockLog{ID: 1, ProductID: 7, BizType: StockBizInit})
	if err != nil {
		t.Fatalf("marshal nil-BizID log: %v", err)
	}
	if !strings.Contains(string(rawNil), `"biz_id":null`) {
		t.Fatalf("nil BizID must emit \"biz_id\":null, got %s", rawNil)
	}
	t.Logf("StockLog.biz_id is %s for a set id, and null when unset (order Snowflake rendered in the admin stock-log table)", shape["biz_id"])
}

func qaPtr[T any](v T) *T { return &v }

// TestQAInternalModelWireShapes pins how the models that carry the order
// Snowflake would serialize *if* someone ever attached them to a response.
// None of them is reachable from an HTTP handler today (grep for the type names
// across internal/handler and internal/response returns only the stock-log
// path), so they are latent risk, not live defects.
func TestQAInternalModelWireShapes(t *testing.T) {
	cases := []struct {
		name        string
		v           any
		wantOrderID string // "" means "marshals under the Go field name"
	}{
		{"OrderTimeoutOutbox", &OrderTimeoutOutbox{ID: 1, OrderID: 371447562240004097}, ""},
		{"OrderIdempotencyKey", &OrderIdempotencyKey{ID: 1, OrderID: qaPtr(int64(371447562240004097))}, "non-string"},
		{"OrderTimeoutDeadLetter", &OrderTimeoutDeadLetter{ID: 1, OrderID: 371447562240004097}, "non-string"},
	}
	for _, tc := range cases {
		shape := qaWireShape(t, tc.v)
		if tc.wantOrderID == "" {
			if _, tagged := shape["order_id"]; tagged {
				t.Fatalf("%s gained a json \"order_id\" tag: it is now part of the external surface and must be re-evaluated", tc.name)
			}
			if _, goName := shape["OrderID"]; !goName {
				t.Fatalf("%s no longer marshals OrderID under its Go name: %+v", tc.name, shape)
			}
			t.Logf("%s: persistence-only (Go field name, no json tag)", tc.name)
			continue
		}
		if shape["order_id"] != tc.wantOrderID {
			t.Fatalf("%s.order_id wire shape=%s, want %s", tc.name, shape["order_id"], tc.wantOrderID)
		}
		t.Logf("%s: json tag present, order_id is %s and NOT exposed by any handler today", tc.name, shape["order_id"])
	}
}
