package model

import "time"

type OperationLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;type:bigint" json:"id"`
	UserID     int64     `gorm:"type:bigint;not null;index:idx_operation_logs_user_id_created_at,priority:1" json:"user_id"`
	Username   string    `gorm:"type:varchar(64);not null;default:''" json:"username"`
	Action     string    `gorm:"type:varchar(128);not null;index:idx_operation_logs_action_created_at,priority:1" json:"action"`
	Method     string    `gorm:"type:varchar(16);not null" json:"method"`
	Path       string    `gorm:"type:varchar(255);not null" json:"path"`
	Route      string    `gorm:"type:varchar(255);not null" json:"route"`
	HTTPStatus int       `gorm:"type:int;not null" json:"http_status"`
	RequestID  string    `gorm:"type:char(36);not null;default:''" json:"request_id"`
	ClientIP   string    `gorm:"type:varchar(64);not null;default:''" json:"client_ip"`
	UserAgent  string    `gorm:"type:varchar(255);not null;default:''" json:"user_agent"`
	CreatedAt  time.Time `gorm:"index:idx_operation_logs_user_id_created_at,priority:2;index:idx_operation_logs_action_created_at,priority:2;index:idx_operation_logs_created_at" json:"created_at"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
