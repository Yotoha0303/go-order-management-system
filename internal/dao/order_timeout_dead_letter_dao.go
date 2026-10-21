package dao

import (
	"context"
	"errors"

	"go-order-management-system/internal/model"

	"gorm.io/gorm"
)

// CreateOrderTimeoutDeadLetter 将超时死信记录持久化到数据库
func CreateOrderTimeoutDeadLetter(ctx context.Context, db *gorm.DB, dl *model.OrderTimeoutDeadLetter) error {
	if dl == nil {
		return errors.New("dead letter record is nil")
	}
	if db == nil {
		return errors.New("database connection is nil")
	}
	return db.WithContext(ctx).Create(dl).Error
}
