package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
	"go-order-management-system/internal/response"
	"go-order-management-system/internal/service"

	"github.com/gin-gonic/gin"
)

type stubInventoryService struct {
	rebuildCount    int
	rebuildErr      error
	rebuildCalls    int
	reconcileReport service.InventoryRedisReconcileReport
	reconcileErr    error
	reconcileCalls  int
}

func (*stubInventoryService) InitInventory(context.Context, *request.InitInventoryRequest) error {
	panic("unexpected InitInventory call")
}

func (*stubInventoryService) AddInventory(context.Context, request.AddInventoryRequest) error {
	panic("unexpected AddInventory call")
}

func (*stubInventoryService) GetInventoryByProductID(context.Context, int64) (*model.Inventory, error) {
	panic("unexpected GetInventoryByProductID call")
}

func (s *stubInventoryService) RebuildRedisInventoryStock(context.Context) (int, error) {
	s.rebuildCalls++
	return s.rebuildCount, s.rebuildErr
}

func (s *stubInventoryService) ReconcileRedisInventoryStock(context.Context) (service.InventoryRedisReconcileReport, error) {
	s.reconcileCalls++
	return s.reconcileReport, s.reconcileErr
}

func TestInventoryHandlerRebuildRedisInventoryStock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns rebuild count", func(t *testing.T) {
		stub := &stubInventoryService{rebuildCount: 3}
		recorder := performRebuildRedisInventoryRequest(stub)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", recorder.Code)
		}
		if stub.rebuildCalls != 1 {
			t.Fatalf("expected one rebuild call, got %d", stub.rebuildCalls)
		}
		var body struct {
			Code int `json:"code"`
			Data struct {
				RebuildCount int `json:"rebuild_count"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}
		if body.Code != response.CodeSuccess || body.Data.RebuildCount != 3 {
			t.Fatalf("unexpected response: %+v", body)
		}
	})

	t.Run("maps service error", func(t *testing.T) {
		stub := &stubInventoryService{rebuildErr: service.ErrInventoryStockRebuildUnavailable}
		recorder := performRebuildRedisInventoryRequest(stub)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", recorder.Code)
		}
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response failed: %v", err)
		}
		if body.Code != response.CodeRebuildInventoryCacheFailed {
			t.Fatalf("expected code %d, got %d", response.CodeRebuildInventoryCacheFailed, body.Code)
		}
	})
}

func TestInventoryHandlerReconcileRedisInventoryStock(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redisQuantity := int64(8)
	stub := &stubInventoryService{
		reconcileReport: service.InventoryRedisReconcileReport{
			CheckedCount: 2,
			DiffCount:    1,
			Items: []service.InventoryRedisReconcileItem{
				{ProductID: 10, MySQLQuantity: 12, RedisQuantity: &redisQuantity, Status: "mismatch"},
			},
		},
	}
	router := gin.New()
	router.GET("/inventory/redis/reconcile", NewInventoryHandler(stub).ReconcileRedisInventoryStock)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/inventory/redis/reconcile", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if stub.reconcileCalls != 1 {
		t.Fatalf("expected one reconcile call, got %d", stub.reconcileCalls)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			CheckedCount int `json:"checked_count"`
			DiffCount    int `json:"diff_count"`
			Items        []struct {
				ProductID     int64  `json:"product_id"`
				MySQLQuantity int64  `json:"mysql_quantity"`
				RedisQuantity *int64 `json:"redis_quantity"`
				Status        string `json:"status"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if body.Code != response.CodeSuccess || body.Data.CheckedCount != 2 || body.Data.DiffCount != 1 {
		t.Fatalf("unexpected response counts: %+v", body)
	}
	if len(body.Data.Items) != 1 || body.Data.Items[0].RedisQuantity == nil ||
		*body.Data.Items[0].RedisQuantity != redisQuantity || body.Data.Items[0].Status != "mismatch" {
		t.Fatalf("unexpected response item: %+v", body.Data.Items)
	}
}

func performRebuildRedisInventoryRequest(service InventoryService) *httptest.ResponseRecorder {
	router := gin.New()
	router.POST("/inventory/redis/rebuild", NewInventoryHandler(service).RebuildRedisInventoryStock)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/inventory/redis/rebuild", nil))
	return recorder
}
