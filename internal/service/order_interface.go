package service

import (
	"context"
	"time"

	"go-order-management-system/internal/request"

	"gorm.io/gorm"
)

type OrderService struct {
	db                   *gorm.DB
	orderTimeout         time.Duration
	inventoryPreDeductor InventoryPreDeductor
	metrics              OrderMetrics
}

type InventoryPreDeductor interface {
	PreDeductInventory(ctx context.Context, orderID int64, items []request.CreateOrderItemRequest) (applied bool, insufficient bool)
	ReleasePreDeductedInventory(ctx context.Context, orderID int64)
	RestorePreDeductedInventory(ctx context.Context, orderID int64)
	ConfirmPreDeductedInventory(ctx context.Context, orderID int64)
}

type OrderMetrics interface {
	RecordOrderCreate(result string)
	RecordOrderStateTransition(action, result string)
}

func NewOrderService(db *gorm.DB) *OrderService {
	return NewOrderServiceWithTimeout(db, 30*time.Minute)
}

func NewOrderServiceWithTimeout(db *gorm.DB, orderTimeout time.Duration) *OrderService {
	return NewOrderServiceWithTimeoutAndInventoryPreDeductor(db, orderTimeout, nil)
}

func NewOrderServiceWithTimeoutAndInventoryPreDeductor(
	db *gorm.DB,
	orderTimeout time.Duration,
	inventoryPreDeductor InventoryPreDeductor,
) *OrderService {
	return NewOrderServiceWithTimeoutInventoryPreDeductorAndMetrics(db, orderTimeout, inventoryPreDeductor, nil)
}

func NewOrderServiceWithTimeoutInventoryPreDeductorAndMetrics(
	db *gorm.DB,
	orderTimeout time.Duration,
	inventoryPreDeductor InventoryPreDeductor,
	metrics OrderMetrics,
) *OrderService {
	return &OrderService{
		db:                   db,
		orderTimeout:         orderTimeout,
		inventoryPreDeductor: inventoryPreDeductor,
		metrics:              metrics,
	}
}
