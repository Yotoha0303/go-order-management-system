package service

import (
	"time"

	"gorm.io/gorm"
)

type OrderService struct {
	db           *gorm.DB
	orderTimeout time.Duration
}

func NewOrderService(db *gorm.DB) *OrderService {
	return NewOrderServiceWithTimeout(db, 30*time.Minute)
}

func NewOrderServiceWithTimeout(db *gorm.DB, orderTimeout time.Duration) *OrderService {
	return &OrderService{
		db:           db,
		orderTimeout: orderTimeout,
	}
}
