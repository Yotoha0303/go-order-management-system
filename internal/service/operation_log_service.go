package service

import (
	"context"
	"strings"

	"go-order-management-system/internal/dao"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
)

const (
	maxOperationLogPage     = 1_000_000
	maxOperationLogPageSize = 100
)

func (s *OperationLogService) CreateOperationLog(ctx context.Context, log *model.OperationLog) error {
	if s == nil || s.db == nil {
		return ErrDatabaseNotInitialized
	}
	return dao.CreateOperationLog(ctx, s.db, log)
}

func (s *OperationLogService) ListOperationLogs(ctx context.Context, req request.ListOperationLogsRequest) ([]*model.OperationLog, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, ErrDatabaseNotInitialized
	}
	if req.Page <= 0 || req.Page > maxOperationLogPage || req.PageSize <= 0 || req.PageSize > maxOperationLogPageSize {
		return nil, 0, ErrInvalidOperationLogPagination
	}
	if req.UserID != nil && *req.UserID <= 0 {
		return nil, 0, ErrInvalidOperationLogFilter
	}

	filter := dao.OperationLogFilter{
		UserID: req.UserID,
		Action: strings.TrimSpace(req.Action),
		Limit:  req.PageSize,
		Offset: (req.Page - 1) * req.PageSize,
	}
	return dao.ListOperationLogs(ctx, s.db, filter)
}
