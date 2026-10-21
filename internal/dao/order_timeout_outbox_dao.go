package dao

import (
	"context"
	"time"

	"go-order-management-system/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateOrderTimeoutOutbox(ctx context.Context, db *gorm.DB, outbox *model.OrderTimeoutOutbox) error {
	return db.WithContext(ctx).Create(outbox).Error
}

func GetOrderTimeoutOutbox(
	ctx context.Context,
	db *gorm.DB,
	eventID string,
	orderID int64,
) (*model.OrderTimeoutOutbox, error) {
	var outbox model.OrderTimeoutOutbox
	err := db.WithContext(ctx).
		Where("event_id = ? AND order_id = ?", eventID, orderID).
		First(&outbox).Error
	return &outbox, err
}

func ListPendingOrderTimeoutOutbox(
	ctx context.Context,
	db *gorm.DB,
	now time.Time,
	limit int,
) ([]model.OrderTimeoutOutbox, error) {
	var events []model.OrderTimeoutOutbox
	err := db.WithContext(ctx).
		Where("published_at IS NULL AND next_attempt_at <= ?", now).
		Order("id ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ClaimPendingOrderTimeoutOutbox 在短事务内利用 FOR UPDATE SKIP LOCKED 并发安全抢占待发布记录并更新租期
func ClaimPendingOrderTimeoutOutbox(
	ctx context.Context,
	db *gorm.DB,
	now time.Time,
	limit int,
	leaseDuration time.Duration,
	maxAttempts int,
) ([]model.OrderTimeoutOutbox, error) {
	if limit <= 0 {
		return nil, nil
	}
	if leaseDuration <= 0 {
		leaseDuration = 30 * time.Second
	}
	if maxAttempts <= 0 {
		maxAttempts = 10
	}

	var claimed []model.OrderTimeoutOutbox
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []model.OrderTimeoutOutbox
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("published_at IS NULL AND next_attempt_at <= ? AND attempts < ?", now, maxAttempts).
			Order("id ASC").
			Limit(limit).
			Find(&events).Error
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}

		ids := make([]int64, len(events))
		for i, event := range events {
			ids[i] = event.ID
		}

		leaseUntil := now.Add(leaseDuration)
		if err := tx.Model(&model.OrderTimeoutOutbox{}).
			Where("id IN ?", ids).
			Updates(map[string]any{
				"attempts":        gorm.Expr("attempts + 1"),
				"next_attempt_at": leaseUntil,
			}).Error; err != nil {
			return err
		}

		for i := range events {
			events[i].Attempts++
			events[i].NextAttemptAt = leaseUntil
		}
		claimed = events
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func MarkOrderTimeoutOutboxPublished(ctx context.Context, db *gorm.DB, eventID string, now time.Time) error {
	return db.WithContext(ctx).
		Model(&model.OrderTimeoutOutbox{}).
		Where("event_id = ? AND published_at IS NULL", eventID).
		Updates(map[string]any{
			"published_at": now,
			"last_error":   "",
		}).Error
}

func MarkOrderTimeoutOutboxFailed(
	ctx context.Context,
	db *gorm.DB,
	eventID, lastError string,
	nextAttemptAt time.Time,
) error {
	return db.WithContext(ctx).
		Model(&model.OrderTimeoutOutbox{}).
		Where("event_id = ? AND published_at IS NULL", eventID).
		Updates(map[string]any{
			"last_error":      lastError,
			"next_attempt_at": nextAttemptAt,
		}).Error
}
