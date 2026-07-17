package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
	"go-order-management-system/internal/response"
	"go-order-management-system/internal/service"

	"github.com/gin-gonic/gin"
)

type OperationLogService interface {
	ListOperationLogs(ctx context.Context, req request.ListOperationLogsRequest) ([]*model.OperationLog, int64, error)
}

type OperationLogHandler struct {
	operationLogService OperationLogService
}

func NewOperationLogHandler(operationLogService OperationLogService) *OperationLogHandler {
	return &OperationLogHandler{operationLogService: operationLogService}
}

var _ OperationLogService = (*service.OperationLogService)(nil)

func (h *OperationLogHandler) ListOperationLogs(c *gin.Context) {
	req, ok := parseListOperationLogsRequest(c)
	if !ok {
		return
	}

	logs, total, err := h.operationLogService.ListOperationLogs(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, response.CodeQueryOperationLogFailed, "查询操作日志失败")
		return
	}

	response.Success(c, response.OperationLogListResponse{
		OperationLogs: logs,
		Total:         total,
		Page:          req.Page,
		PageSize:      req.PageSize,
	})
}

func parseListOperationLogsRequest(c *gin.Context) (request.ListOperationLogsRequest, bool) {
	page, pageSize, ok := parseOperationLogPagination(c.Query("page"), c.Query("page_size"))
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeOperationLogFilterError, "分页参数错误")
		return request.ListOperationLogsRequest{}, false
	}

	var userID *int64
	if value := c.Query("user_id"); value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			response.Fail(c, http.StatusBadRequest, response.CodeOperationLogFilterError, "user_id 参数错误")
			return request.ListOperationLogsRequest{}, false
		}
		userID = &id
	}

	return request.ListOperationLogsRequest{
		UserID:   userID,
		Action:   strings.TrimSpace(c.Query("action")),
		Page:     page,
		PageSize: pageSize,
	}, true
}

func parseOperationLogPagination(pageValue, pageSizeValue string) (int, int, bool) {
	page, pageSize := 1, 20
	var err error
	if pageValue != "" {
		page, err = strconv.Atoi(pageValue)
		if err != nil {
			return 0, 0, false
		}
	}
	if pageSizeValue != "" {
		pageSize, err = strconv.Atoi(pageSizeValue)
		if err != nil {
			return 0, 0, false
		}
	}
	if page <= 0 || page > 1_000_000 || pageSize <= 0 || pageSize > 100 {
		return 0, 0, false
	}
	return page, pageSize, true
}
