package bizcache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"go-order-management-system/internal/model"
	"go-order-management-system/internal/observability"
	"go-order-management-system/internal/request"

	"github.com/redis/go-redis/v9"
)

const inventoryStockGuardTimeout = 500 * time.Millisecond

var ErrInventoryStockRedisUnavailable = errors.New("redis inventory stock guard unavailable")

type InventoryStockGuard struct {
	redisClient *redis.Client
	metrics     *observability.BusinessMetrics
}

func NewInventoryStockGuard(redisClient *redis.Client) *InventoryStockGuard {
	return NewInventoryStockGuardWithMetrics(redisClient, nil)
}

func NewInventoryStockGuardWithMetrics(redisClient *redis.Client, metrics *observability.BusinessMetrics) *InventoryStockGuard {
	return &InventoryStockGuard{
		redisClient: redisClient,
		metrics:     metrics,
	}
}

func InventoryAvailableKey(productID int64) string {
	return fmt.Sprintf("inventory:{stock}:available:%d", productID)
}

func InventoryReservationKey(orderID int64) string {
	return fmt.Sprintf("inventory:{stock}:reservation:%d", orderID)
}

const inventoryAvailableKeyPrefix = "inventory:{stock}:available:"

const preDeductInventoryScript = `
local n = tonumber(ARGV[1])
local reservation = {}

if redis.call("EXISTS", KEYS[n + 1]) == 1 then
    return 1
end

for i = 1, n do
    local quantity = tonumber(ARGV[2 + (i - 1) * 2 + 1])
    local current = redis.call("GET", KEYS[i])
    if not current then
        return 0
    end
    if tonumber(current) < quantity then
        return -1
    end
end

for i = 1, n do
    local product_id = ARGV[2 + (i - 1) * 2]
    local quantity = tonumber(ARGV[2 + (i - 1) * 2 + 1])
    redis.call("DECRBY", KEYS[i], quantity)
    table.insert(reservation, {product_id = product_id, quantity = quantity})
end

redis.call("SET", KEYS[n + 1], cjson.encode(reservation))
return 1
`

const releaseInventoryReservationScript = `
local raw = redis.call("GET", KEYS[1])
if not raw then
    return 0
end

local reservation = cjson.decode(raw)
for _, item in ipairs(reservation) do
    local stock_key = ARGV[1] .. item.product_id
    if redis.call("EXISTS", stock_key) == 1 then
        redis.call("INCRBY", stock_key, item.quantity)
    end
end

redis.call("DEL", KEYS[1])
return 1
`

func (g *InventoryStockGuard) PreDeductInventory(
	ctx context.Context,
	orderID int64,
	items []request.CreateOrderItemRequest,
) (bool, bool) {
	if g == nil || g.redisClient == nil || orderID <= 0 || len(items) == 0 {
		g.recordPreDeduct("disabled")
		return false, false
	}

	keys := make([]string, 0, len(items)+1)
	args := make([]interface{}, 0, 1+len(items)*2)
	args = append(args, len(items))
	for _, item := range items {
		keys = append(keys, InventoryAvailableKey(item.ProductID))
		args = append(args, strconv.FormatInt(item.ProductID, 10), item.Quantity)
	}
	keys = append(keys, InventoryReservationKey(orderID))

	ctx, cancel := context.WithTimeout(ctx, inventoryStockGuardTimeout)
	defer cancel()

	result, err := g.redisClient.Eval(ctx, preDeductInventoryScript, keys, args...).Int64()
	if err != nil {
		log.Printf("redis inventory pre-deduct skipped: order_id=%d err=%v", orderID, err)
		g.recordPreDeduct("error")
		return false, false
	}

	switch result {
	case 1:
		g.recordPreDeduct("applied")
		return true, false
	case -1:
		g.recordPreDeduct("insufficient")
		return false, true
	default:
		g.recordPreDeduct("skipped_missing_key")
		return false, false
	}
}

func (g *InventoryStockGuard) ReleasePreDeductedInventory(ctx context.Context, orderID int64) {
	g.releaseReservation(ctx, orderID)
}

func (g *InventoryStockGuard) RestorePreDeductedInventory(ctx context.Context, orderID int64) {
	g.releaseReservation(ctx, orderID)
}

func (g *InventoryStockGuard) ConfirmPreDeductedInventory(ctx context.Context, orderID int64) {
	if g == nil || g.redisClient == nil || orderID <= 0 {
		g.recordReservation("confirm", "disabled")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, inventoryStockGuardTimeout)
	defer cancel()
	deleted, err := g.redisClient.Del(ctx, InventoryReservationKey(orderID)).Result()
	if err != nil {
		log.Printf("redis inventory reservation cleanup failed: order_id=%d err=%v", orderID, err)
		g.recordReservation("confirm", "error")
		return
	}
	if deleted == 0 {
		g.recordReservation("confirm", "noop")
		return
	}
	g.recordReservation("confirm", "success")
}

func (g *InventoryStockGuard) SetInventoryStock(ctx context.Context, productID int64, quantity int64) {
	if g == nil || g.redisClient == nil || productID <= 0 || quantity < 0 {
		g.recordSync("disabled")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, inventoryStockGuardTimeout)
	defer cancel()
	if err := g.redisClient.Set(ctx, InventoryAvailableKey(productID), quantity, 0).Err(); err != nil {
		log.Printf("redis inventory stock sync failed: product_id=%d err=%v", productID, err)
		g.recordSync("error")
		return
	}
	g.recordSync("success")
}

func (g *InventoryStockGuard) RebuildInventoryStocks(ctx context.Context, inventories []*model.Inventory) (int, error) {
	if g == nil || g.redisClient == nil {
		g.recordSync("rebuild_disabled")
		return 0, ErrInventoryStockRedisUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, inventoryStockGuardTimeout)
	defer cancel()

	pipe := g.redisClient.Pipeline()
	count := 0
	for _, inventory := range inventories {
		if inventory == nil || inventory.ProductID <= 0 || inventory.StockQuantity < 0 {
			continue
		}
		pipe.Set(ctx, InventoryAvailableKey(inventory.ProductID), inventory.StockQuantity, 0)
		count++
	}
	if count == 0 {
		g.recordSync("rebuild_success")
		return 0, nil
	}
	if _, err := pipe.Exec(ctx); err != nil {
		g.recordSync("rebuild_error")
		return 0, err
	}
	g.recordSync("rebuild_success")
	return count, nil
}

func (g *InventoryStockGuard) GetInventoryStocks(ctx context.Context, productIDs []int64) (map[int64]*int64, error) {
	if g == nil || g.redisClient == nil {
		g.recordSync("reconcile_disabled")
		return nil, ErrInventoryStockRedisUnavailable
	}
	stocks := make(map[int64]*int64, len(productIDs))
	if len(productIDs) == 0 {
		g.recordSync("reconcile_success")
		return stocks, nil
	}

	keys := make([]string, 0, len(productIDs))
	validProductIDs := make([]int64, 0, len(productIDs))
	for _, productID := range productIDs {
		if productID <= 0 {
			continue
		}
		validProductIDs = append(validProductIDs, productID)
		keys = append(keys, InventoryAvailableKey(productID))
	}
	if len(keys) == 0 {
		g.recordSync("reconcile_success")
		return stocks, nil
	}

	ctx, cancel := context.WithTimeout(ctx, inventoryStockGuardTimeout)
	defer cancel()
	values, err := g.redisClient.MGet(ctx, keys...).Result()
	if err != nil {
		g.recordSync("reconcile_error")
		return nil, err
	}
	for i, raw := range values {
		productID := validProductIDs[i]
		if raw == nil {
			stocks[productID] = nil
			continue
		}
		text, ok := raw.(string)
		if !ok {
			g.recordSync("reconcile_error")
			return nil, fmt.Errorf("unexpected redis stock value type for product %d", productID)
		}
		quantity, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			g.recordSync("reconcile_error")
			return nil, fmt.Errorf("parse redis stock for product %d: %w", productID, err)
		}
		stocks[productID] = &quantity
	}
	g.recordSync("reconcile_success")
	return stocks, nil
}

func (g *InventoryStockGuard) releaseReservation(ctx context.Context, orderID int64) {
	if g == nil || g.redisClient == nil || orderID <= 0 {
		g.recordReservation("release", "disabled")
		return
	}
	ctx, cancel := context.WithTimeout(ctx, inventoryStockGuardTimeout)
	defer cancel()
	released, err := g.redisClient.Eval(
		ctx,
		releaseInventoryReservationScript,
		[]string{InventoryReservationKey(orderID)},
		inventoryAvailableKeyPrefix,
	).Int64()
	if err != nil {
		log.Printf("redis inventory reservation release failed: order_id=%d err=%v", orderID, err)
		g.recordReservation("release", "error")
		return
	}
	if released == 0 {
		g.recordReservation("release", "noop")
		return
	}
	g.recordReservation("release", "success")
}

func (g *InventoryStockGuard) recordPreDeduct(result string) {
	if g != nil && g.metrics != nil {
		g.metrics.RecordRedisInventoryPreDeduct(result)
	}
}

func (g *InventoryStockGuard) recordReservation(action, result string) {
	if g != nil && g.metrics != nil {
		g.metrics.RecordRedisInventoryReservation(action, result)
	}
}

func (g *InventoryStockGuard) recordSync(result string) {
	if g != nil && g.metrics != nil {
		g.metrics.RecordRedisInventorySync(result)
	}
}
