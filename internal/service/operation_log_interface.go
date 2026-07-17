package service

import "gorm.io/gorm"

type OperationLogService struct {
	db *gorm.DB
}

func NewOperationLogService(db *gorm.DB) *OperationLogService {
	return &OperationLogService{db: db}
}
