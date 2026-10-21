package bizcache_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"go-order-management-system/internal/bizcache"
	"go-order-management-system/internal/observability"
	"go-order-management-system/internal/request"
)

func TestInventoryStockGuardKeyNames(t *testing.T) {
	if got, want := bizcache.InventoryAvailableKey(1001), "inventory:{stock}:available:1001"; got != want {
		t.Fatalf("InventoryAvailableKey=%q want=%q", got, want)
	}
	if got, want := bizcache.InventoryReservationKey(2002), "inventory:{stock}:reservation:2002"; got != want {
		t.Fatalf("InventoryReservationKey=%q want=%q", got, want)
	}
}

func TestInventoryStockGuard_NoRedis(t *testing.T) {
	guard := bizcache.NewInventoryStockGuard(nil)
	applied, insufficient := guard.PreDeductInventory(context.Background(), 1, []request.CreateOrderItemRequest{
		{ProductID: 1, Quantity: 1},
	})
	if applied || insufficient {
		t.Fatalf("expected no redis guard result, got applied=%v insufficient=%v", applied, insufficient)
	}
	guard.ReleasePreDeductedInventory(context.Background(), 1)
	guard.RestorePreDeductedInventory(context.Background(), 1)
	guard.ConfirmPreDeductedInventory(context.Background(), 1)
	guard.SetInventoryStock(context.Background(), 1, 10)
	guard.IncrInventoryStock(context.Background(), 1, 5, 15)
}

func TestInventoryStockGuard_RecordsDisabledMetrics(t *testing.T) {
	metrics := observability.NewBusinessMetrics()
	guard := bizcache.NewInventoryStockGuardWithMetrics(nil, metrics)

	guard.PreDeductInventory(context.Background(), 1, []request.CreateOrderItemRequest{{ProductID: 1, Quantity: 1}})
	guard.ReleasePreDeductedInventory(context.Background(), 1)
	guard.ConfirmPreDeductedInventory(context.Background(), 1)
	guard.SetInventoryStock(context.Background(), 1, 10)

	output := metrics.RenderPrometheus()
	for _, want := range []string{
		`app_redis_inventory_prededuct_total{result="disabled"} 1`,
		`app_redis_inventory_reservation_total{action="release",result="disabled"} 1`,
		`app_redis_inventory_reservation_total{action="confirm",result="disabled"} 1`,
		`app_redis_inventory_sync_total{result="disabled"} 1`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestInventoryStockGuard_RebuildUnavailableNoRedis(t *testing.T) {
	guard := bizcache.NewInventoryStockGuard(nil)
	_, err := guard.RebuildInventoryStocks(context.Background(), nil)
	if err == nil {
		t.Fatal("expected rebuild to fail when redis is unavailable")
	}
}

func TestInventoryStockGuard_GetInventoryStocksUnavailableNoRedis(t *testing.T) {
	guard := bizcache.NewInventoryStockGuard(nil)
	_, err := guard.GetInventoryStocks(context.Background(), []int64{1})
	if !errors.Is(err, bizcache.ErrInventoryStockRedisUnavailable) {
		t.Fatalf("expected ErrInventoryStockRedisUnavailable, got %v", err)
	}
}

func TestInventoryStockGuard_PreDeductAndRelease_WithRedis(t *testing.T) {
	client := setupTestRedis(t)
	guard := bizcache.NewInventoryStockGuard(client)
	ctx := context.Background()
	const productID int64 = 91001
	const orderID int64 = 92001
	defer client.Del(ctx, bizcache.InventoryAvailableKey(productID), bizcache.InventoryReservationKey(orderID))

	guard.SetInventoryStock(ctx, productID, 5)
	applied, insufficient := guard.PreDeductInventory(ctx, orderID, []request.CreateOrderItemRequest{
		{ProductID: productID, Quantity: 3},
	})
	if !applied || insufficient {
		t.Fatalf("expected redis pre-deduct applied, got applied=%v insufficient=%v", applied, insufficient)
	}
	stock, err := client.Get(ctx, bizcache.InventoryAvailableKey(productID)).Int64()
	if err != nil {
		t.Fatalf("query redis stock failed: %v", err)
	}
	if stock != 2 {
		t.Fatalf("expected redis stock 2 after pre-deduct, got %d", stock)
	}

	guard.ReleasePreDeductedInventory(ctx, orderID)
	stock, err = client.Get(ctx, bizcache.InventoryAvailableKey(productID)).Int64()
	if err != nil {
		t.Fatalf("query redis stock after release failed: %v", err)
	}
	if stock != 5 {
		t.Fatalf("expected redis stock 5 after release, got %d", stock)
	}
}

func TestInventoryStockGuard_GetInventoryStocks_WithRedis(t *testing.T) {
	client := setupTestRedis(t)
	guard := bizcache.NewInventoryStockGuard(client)
	ctx := context.Background()
	const productA int64 = 91005
	const productB int64 = 91006
	defer client.Del(ctx, bizcache.InventoryAvailableKey(productA), bizcache.InventoryAvailableKey(productB))

	guard.SetInventoryStock(ctx, productA, 7)
	_ = client.Del(ctx, bizcache.InventoryAvailableKey(productB)).Err()
	stocks, err := guard.GetInventoryStocks(ctx, []int64{productA, productB})
	if err != nil {
		t.Fatalf("get redis inventory stocks failed: %v", err)
	}
	if stocks[productA] == nil || *stocks[productA] != 7 {
		t.Fatalf("expected product A stock 7, got %v", stocks[productA])
	}
	if _, ok := stocks[productB]; !ok || stocks[productB] != nil {
		t.Fatalf("expected product B missing stock, got %v", stocks[productB])
	}
}

func TestInventoryStockGuard_InsufficientDoesNotDeduct_WithRedis(t *testing.T) {
	client := setupTestRedis(t)
	guard := bizcache.NewInventoryStockGuard(client)
	ctx := context.Background()
	const productID int64 = 91002
	const orderID int64 = 92002
	defer client.Del(ctx, bizcache.InventoryAvailableKey(productID), bizcache.InventoryReservationKey(orderID))

	guard.SetInventoryStock(ctx, productID, 2)
	applied, insufficient := guard.PreDeductInventory(ctx, orderID, []request.CreateOrderItemRequest{
		{ProductID: productID, Quantity: 3},
	})
	if applied || !insufficient {
		t.Fatalf("expected redis insufficient, got applied=%v insufficient=%v", applied, insufficient)
	}
	stock, err := client.Get(ctx, bizcache.InventoryAvailableKey(productID)).Int64()
	if err != nil {
		t.Fatalf("query redis stock failed: %v", err)
	}
	if stock != 2 {
		t.Fatalf("expected redis stock unchanged as 2, got %d", stock)
	}
}

func TestInventoryStockGuard_MissingKeySkips_WithRedis(t *testing.T) {
	client := setupTestRedis(t)
	guard := bizcache.NewInventoryStockGuard(client)
	ctx := context.Background()
	const productID int64 = 91003
	const orderID int64 = 92003
	defer client.Del(ctx, bizcache.InventoryAvailableKey(productID), bizcache.InventoryReservationKey(orderID))
	_ = client.Del(ctx, bizcache.InventoryAvailableKey(productID), bizcache.InventoryReservationKey(orderID)).Err()

	applied, insufficient := guard.PreDeductInventory(ctx, orderID, []request.CreateOrderItemRequest{
		{ProductID: productID, Quantity: 1},
	})
	if applied || insufficient {
		t.Fatalf("expected redis pre-deduct skip on missing key, got applied=%v insufficient=%v", applied, insufficient)
	}
}

func TestInventoryStockGuard_ConfirmDeletesReservation_WithRedis(t *testing.T) {
	client := setupTestRedis(t)
	guard := bizcache.NewInventoryStockGuard(client)
	ctx := context.Background()
	const productID int64 = 91004
	const orderID int64 = 92004
	defer client.Del(ctx, bizcache.InventoryAvailableKey(productID), bizcache.InventoryReservationKey(orderID))

	guard.SetInventoryStock(ctx, productID, 5)
	applied, insufficient := guard.PreDeductInventory(ctx, orderID, []request.CreateOrderItemRequest{
		{ProductID: productID, Quantity: 2},
	})
	if !applied || insufficient {
		t.Fatalf("expected redis pre-deduct applied, got applied=%v insufficient=%v", applied, insufficient)
	}

	guard.ConfirmPreDeductedInventory(ctx, orderID)
	exists, err := client.Exists(ctx, bizcache.InventoryReservationKey(orderID)).Result()
	if err != nil {
		t.Fatalf("query redis reservation failed: %v", err)
	}
	if exists != 0 {
		t.Fatalf("expected redis reservation removed after confirm, got exists=%d", exists)
	}
	stock, err := client.Get(ctx, bizcache.InventoryAvailableKey(productID)).Int64()
	if err != nil {
		t.Fatalf("query redis stock failed: %v", err)
	}
	if stock != 3 {
		t.Fatalf("expected confirmed redis stock to remain 3, got %d", stock)
	}
}

func TestInventoryStockGuard_IncrInventoryStock_DisabledValidation(t *testing.T) {
	metrics := observability.NewBusinessMetrics()
	guard := bizcache.NewInventoryStockGuardWithMetrics(nil, metrics)

	// nil redis
	guard.IncrInventoryStock(context.Background(), 10, 5, 15)
	// invalid productID
	guard.IncrInventoryStock(context.Background(), 0, 5, 15)
	guard.IncrInventoryStock(context.Background(), -1, 5, 15)
	// invalid delta
	guard.IncrInventoryStock(context.Background(), 10, 0, 15)
	guard.IncrInventoryStock(context.Background(), 10, -2, 15)
	// invalid fallbackQuantity
	guard.IncrInventoryStock(context.Background(), 10, 5, -1)

	output := metrics.RenderPrometheus()
	want := `app_redis_inventory_sync_total{result="disabled"} 6`
	if !strings.Contains(output, want) {
		t.Fatalf("expected metrics output to contain %q, got:\n%s", want, output)
	}
}

func TestInventoryStockGuard_IncrInventoryStock_WithRedis(t *testing.T) {
	client := setupTestRedis(t)
	metrics := observability.NewBusinessMetrics()
	guard := bizcache.NewInventoryStockGuardWithMetrics(client, metrics)
	ctx := context.Background()
	const productID int64 = 91010
	defer client.Del(ctx, bizcache.InventoryAvailableKey(productID))

	_ = client.Del(ctx, bizcache.InventoryAvailableKey(productID)).Err()

	// 1. Cold start fallback: key absent -> set to fallbackQuantity
	guard.IncrInventoryStock(ctx, productID, 10, 50)
	stock, err := client.Get(ctx, bizcache.InventoryAvailableKey(productID)).Int64()
	if err != nil {
		t.Fatalf("query redis stock after cold start failed: %v", err)
	}
	if stock != 50 {
		t.Fatalf("expected cold start fallback stock 50, got %d", stock)
	}

	// 2. Key exists: atomic INCRBY delta
	guard.IncrInventoryStock(ctx, productID, 25, 75)
	stock, err = client.Get(ctx, bizcache.InventoryAvailableKey(productID)).Int64()
	if err != nil {
		t.Fatalf("query redis stock after increment failed: %v", err)
	}
	if stock != 75 {
		t.Fatalf("expected incremental stock 75, got %d", stock)
	}

	output := metrics.RenderPrometheus()
	want := `app_redis_inventory_sync_total{result="success"} 2`
	if !strings.Contains(output, want) {
		t.Fatalf("expected metrics output to contain %q, got:\n%s", want, output)
	}
}

func TestInventoryStockGuard_IncrInventoryStock_Concurrent_WithRedis(t *testing.T) {
	client := setupTestRedis(t)
	guard := bizcache.NewInventoryStockGuard(client)
	ctx := context.Background()
	const productID int64 = 91011
	defer client.Del(ctx, bizcache.InventoryAvailableKey(productID))

	// Initial stock 100
	guard.SetInventoryStock(ctx, productID, 100)

	// 20 concurrent increments of +5
	const concurrency = 20
	const delta = 5
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			guard.IncrInventoryStock(ctx, productID, delta, 0)
		}()
	}
	wg.Wait()

	stock, err := client.Get(ctx, bizcache.InventoryAvailableKey(productID)).Int64()
	if err != nil {
		t.Fatalf("query redis stock after concurrent incr failed: %v", err)
	}
	expected := int64(100 + concurrency*delta)
	if stock != expected {
		t.Fatalf("expected final stock %d, got %d (race condition detected!)", expected, stock)
	}
}
