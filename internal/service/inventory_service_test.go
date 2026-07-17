package service_test

import (
	"context"
	"errors"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
	"go-order-management-system/internal/service"
	"testing"

	"gorm.io/gorm"
)

func newInventoryService(t *testing.T) (*gorm.DB, *service.InventoryService) {
	t.Helper()
	testDB := setupTestDB(t)
	return testDB, service.NewInventoryService(testDB)
}

type fakeInventoryStockWriter struct {
	productIDs         []int64
	quantities         []int64
	rebuildInventories []*model.Inventory
	rebuildCount       int
	rebuildErr         error
	readStocks         map[int64]*int64
	readErr            error
}

func (f *fakeInventoryStockWriter) SetInventoryStock(_ context.Context, productID int64, quantity int64) {
	f.productIDs = append(f.productIDs, productID)
	f.quantities = append(f.quantities, quantity)
}

func (f *fakeInventoryStockWriter) RebuildInventoryStocks(_ context.Context, inventories []*model.Inventory) (int, error) {
	f.rebuildInventories = append([]*model.Inventory(nil), inventories...)
	if f.rebuildErr != nil {
		return 0, f.rebuildErr
	}
	return f.rebuildCount, nil
}

func (f *fakeInventoryStockWriter) GetInventoryStocks(_ context.Context, productIDs []int64) (map[int64]*int64, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	result := make(map[int64]*int64, len(productIDs))
	for _, productID := range productIDs {
		result[productID] = f.readStocks[productID]
	}
	return result, nil
}

func TestInitInventory_ProductNotFound(t *testing.T) {
	_, inventorySvc := newInventoryService(t)
	qty := int64(10)
	err := inventorySvc.InitInventory(context.Background(), &request.InitInventoryRequest{
		ProductID:     99999,
		StockQuantity: &qty,
	})
	if !errors.Is(err, service.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestInitInventory_Success(t *testing.T) {
	testDB, inventorySvc := newInventoryService(t)
	p := seedProduct(t, testDB, "p1", 100, model.ProductStatusOnSale)
	qty := int64(20)

	err := inventorySvc.InitInventory(context.Background(), &request.InitInventoryRequest{
		ProductID:     p.ID,
		StockQuantity: &qty,
	})
	if err != nil {
		t.Fatalf("init inventory failed: %v", err)
	}

	var inv model.Inventory
	if err := testDB.Where("product_id = ?", p.ID).First(&inv).Error; err != nil {
		t.Fatalf("query inventory failed: %v", err)
	}
	if inv.StockQuantity != qty {
		t.Fatalf("expected stock=%d, got %d", qty, inv.StockQuantity)
	}
}

func TestInitInventory_SyncsRedisStock(t *testing.T) {
	testDB := setupTestDB(t)
	writer := &fakeInventoryStockWriter{}
	inventorySvc := service.NewInventoryServiceWithStockWriter(testDB, writer)
	product := seedProduct(t, testDB, "redis-sync-init-product", 100, model.ProductStatusOnSale)
	qty := int64(20)

	err := inventorySvc.InitInventory(context.Background(), &request.InitInventoryRequest{
		ProductID:     product.ID,
		StockQuantity: &qty,
	})
	if err != nil {
		t.Fatalf("init inventory failed: %v", err)
	}

	if len(writer.productIDs) != 1 || writer.productIDs[0] != product.ID || writer.quantities[0] != qty {
		t.Fatalf("expected redis stock sync product=%d qty=%d, got products=%v quantities=%v", product.ID, qty, writer.productIDs, writer.quantities)
	}
}

func TestAddInventory_InvalidQuantity(t *testing.T) {
	_, inventorySvc := newInventoryService(t)
	err := inventorySvc.AddInventory(context.Background(), request.AddInventoryRequest{
		ProductID: 1,
		Quantity:  0,
	})
	if !errors.Is(err, service.ErrInvalidAddQuantity) {
		t.Fatalf("expected ErrInvalidAddQuantity, got %v", err)
	}
}

func TestAddInventory_Success(t *testing.T) {
	testDB, inventorySvc := newInventoryService(t)
	p := seedProduct(t, testDB, "p1", 100, model.ProductStatusOnSale)
	seedInventory(t, testDB, p.ID, 10)

	err := inventorySvc.AddInventory(context.Background(), request.AddInventoryRequest{
		ProductID: p.ID,
		Quantity:  5,
	})
	if err != nil {
		t.Fatalf("add inventory failed: %v", err)
	}

	var inv model.Inventory
	if err := testDB.Where("product_id = ?", p.ID).First(&inv).Error; err != nil {
		t.Fatalf("query inventory failed: %v", err)
	}
	if inv.StockQuantity != 15 {
		t.Fatalf("expected stock=15, got %d", inv.StockQuantity)
	}
}

func TestAddInventory_SyncsRedisStock(t *testing.T) {
	testDB := setupTestDB(t)
	writer := &fakeInventoryStockWriter{}
	inventorySvc := service.NewInventoryServiceWithStockWriter(testDB, writer)
	product := seedProduct(t, testDB, "redis-sync-add-product", 100, model.ProductStatusOnSale)
	seedInventory(t, testDB, product.ID, 10)

	err := inventorySvc.AddInventory(context.Background(), request.AddInventoryRequest{
		ProductID: product.ID,
		Quantity:  5,
	})
	if err != nil {
		t.Fatalf("add inventory failed: %v", err)
	}

	if len(writer.productIDs) != 1 || writer.productIDs[0] != product.ID || writer.quantities[0] != 15 {
		t.Fatalf("expected redis stock sync product=%d qty=%d, got products=%v quantities=%v", product.ID, int64(15), writer.productIDs, writer.quantities)
	}
}

func TestRebuildRedisInventoryStock_Success(t *testing.T) {
	testDB := setupTestDB(t)
	writer := &fakeInventoryStockWriter{rebuildCount: 2}
	inventorySvc := service.NewInventoryServiceWithStockWriter(testDB, writer)
	productB := seedProduct(t, testDB, "redis-rebuild-product-b", 100, model.ProductStatusOnSale)
	productA := seedProduct(t, testDB, "redis-rebuild-product-a", 100, model.ProductStatusOnSale)
	seedInventory(t, testDB, productB.ID, 20)
	seedInventory(t, testDB, productA.ID, 10)

	count, err := inventorySvc.RebuildRedisInventoryStock(context.Background())
	if err != nil {
		t.Fatalf("rebuild redis inventory stock failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected rebuild count 2, got %d", count)
	}
	if len(writer.rebuildInventories) != 2 {
		t.Fatalf("expected 2 inventories passed to rebuilder, got %d", len(writer.rebuildInventories))
	}
	if writer.rebuildInventories[0].ProductID != productA.ID || writer.rebuildInventories[1].ProductID != productB.ID {
		t.Fatalf("expected inventories ordered by product_id asc, got %+v", writer.rebuildInventories)
	}
}

func TestRebuildRedisInventoryStock_Unavailable(t *testing.T) {
	testDB := setupTestDB(t)
	inventorySvc := service.NewInventoryService(testDB)

	_, err := inventorySvc.RebuildRedisInventoryStock(context.Background())
	if !errors.Is(err, service.ErrInventoryStockRebuildUnavailable) {
		t.Fatalf("expected ErrInventoryStockRebuildUnavailable, got %v", err)
	}
}

func TestReconcileRedisInventoryStock_ReportsDifferences(t *testing.T) {
	testDB := setupTestDB(t)
	productA := seedProduct(t, testDB, "redis-reconcile-product-a", 100, model.ProductStatusOnSale)
	productB := seedProduct(t, testDB, "redis-reconcile-product-b", 100, model.ProductStatusOnSale)
	productC := seedProduct(t, testDB, "redis-reconcile-product-c", 100, model.ProductStatusOnSale)
	seedInventory(t, testDB, productA.ID, 10)
	seedInventory(t, testDB, productB.ID, 20)
	seedInventory(t, testDB, productC.ID, 30)
	redisA := int64(10)
	redisB := int64(18)
	writer := &fakeInventoryStockWriter{
		readStocks: map[int64]*int64{
			productA.ID: &redisA,
			productB.ID: &redisB,
			productC.ID: nil,
		},
	}
	inventorySvc := service.NewInventoryServiceWithStockWriter(testDB, writer)

	report, err := inventorySvc.ReconcileRedisInventoryStock(context.Background())
	if err != nil {
		t.Fatalf("reconcile redis inventory stock failed: %v", err)
	}
	if report.CheckedCount != 3 || report.DiffCount != 2 {
		t.Fatalf("unexpected report counts: %+v", report)
	}
	if len(report.Items) != 2 {
		t.Fatalf("expected 2 diff items, got %d", len(report.Items))
	}
	if report.Items[0].ProductID != productB.ID || report.Items[0].Status != "mismatch" ||
		report.Items[0].RedisQuantity == nil || *report.Items[0].RedisQuantity != redisB {
		t.Fatalf("unexpected mismatch item: %+v", report.Items[0])
	}
	if report.Items[1].ProductID != productC.ID || report.Items[1].Status != "missing" ||
		report.Items[1].RedisQuantity != nil {
		t.Fatalf("unexpected missing item: %+v", report.Items[1])
	}
}

func TestReconcileRedisInventoryStock_Unavailable(t *testing.T) {
	testDB := setupTestDB(t)
	inventorySvc := service.NewInventoryService(testDB)

	_, err := inventorySvc.ReconcileRedisInventoryStock(context.Background())
	if !errors.Is(err, service.ErrInventoryStockReconcileUnavailable) {
		t.Fatalf("expected ErrInventoryStockReconcileUnavailable, got %v", err)
	}
}

func TestInitInventory_CreateStockLog(t *testing.T) {
	testDB, inventorySvc := newInventoryService(t)
	p := seedProduct(t, testDB, "p1", 100, model.ProductStatusOnSale)
	qty := int64(20)

	err := inventorySvc.InitInventory(context.Background(), &request.InitInventoryRequest{
		ProductID:     p.ID,
		StockQuantity: &qty,
	})
	if err != nil {
		t.Fatalf("init inventory failed: %v", err)
	}

	var log model.StockLog
	if err := testDB.Where("product_id = ?", p.ID).Order("id ASC").First(&log).Error; err != nil {
		t.Fatalf("query stock log failed: %v", err)
	}

	if log.ChangeQuantity != qty || log.AfterQuantity != qty || log.BizType != model.StockBizInit {
		t.Fatalf("unexpected stock log data: %+v", log)
	}
}
