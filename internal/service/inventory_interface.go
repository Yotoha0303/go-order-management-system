package service

import (
	"context"

	"go-order-management-system/internal/model"

	"gorm.io/gorm"
)

type InventoryService struct {
	db          *gorm.DB
	stockWriter InventoryStockWriter
}

type InventoryStockWriter interface {
	SetInventoryStock(ctx context.Context, productID int64, quantity int64)
}

type InventoryStockRebuilder interface {
	RebuildInventoryStocks(ctx context.Context, inventories []*model.Inventory) (int, error)
}

type InventoryStockReader interface {
	GetInventoryStocks(ctx context.Context, productIDs []int64) (map[int64]*int64, error)
}

type InventoryRedisReconcileReport struct {
	CheckedCount int
	DiffCount    int
	Items        []InventoryRedisReconcileItem
}

type InventoryRedisReconcileItem struct {
	ProductID     int64
	MySQLQuantity int64
	RedisQuantity *int64
	Status        string
}

func NewInventoryService(db *gorm.DB) *InventoryService {
	return NewInventoryServiceWithStockWriter(db, nil)
}

func NewInventoryServiceWithStockWriter(db *gorm.DB, stockWriter InventoryStockWriter) *InventoryService {
	return &InventoryService{
		db:          db,
		stockWriter: stockWriter,
	}
}
