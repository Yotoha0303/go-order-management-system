package model

import "time"

type OrderTimeoutOutbox struct {
	ID            int64      `gorm:"primaryKey;autoIncrement;type:bigint"`
	EventID       string     `gorm:"type:char(36);not null;uniqueIndex:uk_order_timeout_outbox_event_id"`
	OrderID       int64      `gorm:"type:bigint;not null;uniqueIndex:uk_order_timeout_outbox_order_id"`
	UserID        int64      `gorm:"type:bigint;not null"`
	TimeoutAt     time.Time  `gorm:"type:datetime(3);not null"`
	PublishedAt   *time.Time `gorm:"type:datetime(3);index:idx_order_timeout_outbox_pending,priority:1"`
	Attempts      int        `gorm:"type:int;not null;default:0"`
	NextAttemptAt time.Time  `gorm:"type:datetime(3);not null;index:idx_order_timeout_outbox_pending,priority:2"`
	LastError     string     `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt     time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt     time.Time  `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)"`
}

func (OrderTimeoutOutbox) TableName() string {
	return "order_timeout_outbox"
}
