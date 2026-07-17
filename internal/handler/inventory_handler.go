package handler

import (
	"context"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
	"go-order-management-system/internal/response"
	"go-order-management-system/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type InventoryService interface {
	InitInventory(ctx context.Context, req *request.InitInventoryRequest) error
	AddInventory(ctx context.Context, req request.AddInventoryRequest) error
	GetInventoryByProductID(ctx context.Context, productID int64) (*model.Inventory, error)
	RebuildRedisInventoryStock(ctx context.Context) (int, error)
	ReconcileRedisInventoryStock(ctx context.Context) (service.InventoryRedisReconcileReport, error)
}

type InventoryHandler struct {
	inventoryService InventoryService
}

func NewInventoryHandler(inventoryService InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

var _ InventoryService = (*service.InventoryService)(nil)

func (p *InventoryHandler) InitInventory(c *gin.Context) {
	var req request.InitInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeParameterError, "请求参数错误")
		return
	}

	if err := p.inventoryService.InitInventory(c.Request.Context(), &req); err != nil {
		handleError(c, err, response.CodeInitInventoryFailed, "初始化库存错误")
		return
	}

	response.Success(c, nil)
}

func (p *InventoryHandler) AddInventory(c *gin.Context) {
	var req request.AddInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeParameterError, "请求参数错误")
		return
	}

	if err := p.inventoryService.AddInventory(c.Request.Context(), req); err != nil {
		handleError(c, err, response.CodeAddInventoryError, "添加库存失败")
		return
	}

	response.Success(c, nil)
}

func (p *InventoryHandler) GetInventoryByProductID(c *gin.Context) {
	id, ok := parsePositiveID(c, "product_id")
	if !ok {
		return
	}

	inventory, err := p.inventoryService.GetInventoryByProductID(c.Request.Context(), id)
	if err != nil {
		handleError(c, err, response.CodeInventoryNotFound, "查询库存失败")
		return
	}

	response.Success(c, inventory)
}

func (p *InventoryHandler) RebuildRedisInventoryStock(c *gin.Context) {
	count, err := p.inventoryService.RebuildRedisInventoryStock(c.Request.Context())
	if err != nil {
		handleError(c, err, response.CodeRebuildInventoryCacheFailed, "重建 Redis 库存失败")
		return
	}
	response.Success(c, response.InventoryRedisRebuildResponse{RebuildCount: count})
}

func (p *InventoryHandler) ReconcileRedisInventoryStock(c *gin.Context) {
	report, err := p.inventoryService.ReconcileRedisInventoryStock(c.Request.Context())
	if err != nil {
		handleError(c, err, response.CodeReconcileInventoryCacheFailed, "Redis 库存对账失败")
		return
	}
	items := make([]response.InventoryRedisReconcileItem, 0, len(report.Items))
	for _, item := range report.Items {
		items = append(items, response.InventoryRedisReconcileItem{
			ProductID:     item.ProductID,
			MySQLQuantity: item.MySQLQuantity,
			RedisQuantity: item.RedisQuantity,
			Status:        item.Status,
		})
	}
	response.Success(c, response.InventoryRedisReconcileResponse{
		CheckedCount: report.CheckedCount,
		DiffCount:    report.DiffCount,
		Items:        items,
	})
}
