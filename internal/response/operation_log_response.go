package response

import "go-order-management-system/internal/model"

type OperationLogListResponse struct {
	OperationLogs []*model.OperationLog `json:"operation_logs"`
	Total         int64                 `json:"total"`
	Page          int                   `json:"page"`
	PageSize      int                   `json:"page_size"`
}
