package model

import "time"

type OrderTimeoutDeadLetter struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;type:bigint" json:"id"`
	MessageID   string    `gorm:"type:varchar(64);not null;default:''" json:"message_id"`
	EventID     string    `gorm:"type:varchar(64);not null;default:'';index:idx_ot_dead_letters_event_id" json:"event_id"`
	OrderID     int64     `gorm:"type:bigint;not null;default:0;index:idx_ot_dead_letters_order_id" json:"order_id"`
	UserID      int64     `gorm:"type:bigint;not null;default:0" json:"user_id"`
	RetryCount  int       `gorm:"type:int;not null;default:0" json:"retry_count"`
	Reason      string    `gorm:"type:varchar(64);not null;default:''" json:"reason"`
	Payload     string    `gorm:"type:text;not null" json:"payload"`
	ErrorDetail string    `gorm:"type:text;not null" json:"error_detail"`
	Status      string    `gorm:"type:varchar(32);not null;default:'PENDING';index:idx_ot_dead_letters_status" json:"status"`
	CreatedAt   time.Time `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_ot_dead_letters_created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (OrderTimeoutDeadLetter) TableName() string {
	return "order_timeout_dead_letters"
}
