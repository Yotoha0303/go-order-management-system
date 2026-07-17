package dao

import (
	"context"

	"go-order-management-system/internal/model"

	"gorm.io/gorm"
)

type OperationLogFilter struct {
	UserID *int64
	Action string
	Limit  int
	Offset int
}

func CreateOperationLog(ctx context.Context, db *gorm.DB, log *model.OperationLog) error {
	return db.WithContext(ctx).Create(log).Error
}

func ListOperationLogs(ctx context.Context, db *gorm.DB, filter OperationLogFilter) ([]*model.OperationLog, int64, error) {
	query := db.WithContext(ctx).Model(&model.OperationLog{})
	if filter.UserID != nil && *filter.UserID > 0 {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []*model.OperationLog
	err := query.Order("created_at desc, id desc").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&logs).Error
	return logs, total, err
}
