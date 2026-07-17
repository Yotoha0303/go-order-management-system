package service

import (
	"context"
	"errors"
	"go-order-management-system/internal/dao"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"

	"gorm.io/gorm"
)

const (
	addInventoryRemarkPrefix  = "手动入库：补充"
	initInventoryRemarkPrefix = "初始化库存："
)

func (p *InventoryService) InitInventory(ctx context.Context, req *request.InitInventoryRequest) error {
	if req.StockQuantity == nil {
		return ErrInvalidStockQuantity
	}

	if *req.StockQuantity < 0 {
		return ErrInvalidStockQuantity
	}

	product, err := dao.GetProductByID(ctx, p.db, req.ProductID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProductNotFound
		}
		return err
	}

	data, err := dao.GetInventoryByProductID(ctx, p.db, req.ProductID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if data.ID != 0 {
		return ErrInitInventoryExists
	}

	if err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inventory := &model.Inventory{
			ProductID:     product.ID,
			StockQuantity: *req.StockQuantity,
		}

		if err := dao.InitInventory(ctx, tx, inventory); err != nil {
			return ErrInitInventoryFailed
		}

		log := &model.StockLog{
			ProductID:      product.ID,
			BeforeQuantity: 0,
			AfterQuantity:  *req.StockQuantity,
			ChangeQuantity: *req.StockQuantity,
			BizType:        model.StockBizInit,
			Remark:         initInventoryRemarkPrefix + product.Name,
		}

		if err := dao.CreateStockLog(ctx, tx, log); err != nil {
			return ErrCreateStockLogFailed
		}
		return nil
	}); err != nil {
		return err
	}

	p.setInventoryStock(ctx, product.ID, *req.StockQuantity)
	return nil
}

func (p *InventoryService) GetInventoryByProductID(ctx context.Context, productID int64) (*model.Inventory, error) {
	inventory, err := dao.GetInventoryByProductID(ctx, p.db, productID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInventoryNotFound
		}
		return nil, err
	}
	return inventory, nil
}

func (p *InventoryService) AddInventory(ctx context.Context, req request.AddInventoryRequest) error {
	if req.Quantity <= 0 {
		return ErrInvalidAddQuantity
	}

	var afterQuantity int64
	if err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		inventory, err := dao.GetInventoryByProductIDForUpdate(ctx, tx, req.ProductID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInventoryNotFound
			}
			return err
		}

		product, err := dao.GetProductByID(ctx, tx, req.ProductID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrProductNotFound
			}
			return err
		}

		beforeQuantity := inventory.StockQuantity
		afterQuantity = beforeQuantity + req.Quantity

		if err := dao.UpdateInventoryStockQuantity(ctx, tx, req.ProductID, afterQuantity); err != nil {
			return err
		}

		log := &model.StockLog{
			ProductID:      req.ProductID,
			BeforeQuantity: beforeQuantity,
			AfterQuantity:  afterQuantity,
			ChangeQuantity: req.Quantity,
			BizType:        model.StockBizManualAdd,
			Remark:         addInventoryRemarkPrefix + product.Name,
		}

		err = dao.CreateStockLog(ctx, tx, log)
		if err != nil {
			return ErrCreateStockLogFailed
		}
		return nil
	}); err != nil {
		return err
	}

	p.setInventoryStock(ctx, req.ProductID, afterQuantity)
	return nil
}

func (p *InventoryService) RebuildRedisInventoryStock(ctx context.Context) (int, error) {
	if p == nil || p.db == nil {
		return 0, ErrDatabaseNotInitialized
	}
	rebuilder, ok := p.stockWriter.(InventoryStockRebuilder)
	if p.stockWriter == nil || !ok {
		return 0, ErrInventoryStockRebuildUnavailable
	}

	inventories, err := dao.ListAllInventories(ctx, p.db)
	if err != nil {
		return 0, err
	}
	count, err := rebuilder.RebuildInventoryStocks(context.WithoutCancel(ctx), inventories)
	if err != nil {
		return 0, ErrInventoryStockRebuildFailed
	}
	return count, nil
}

func (p *InventoryService) ReconcileRedisInventoryStock(ctx context.Context) (InventoryRedisReconcileReport, error) {
	if p == nil || p.db == nil {
		return InventoryRedisReconcileReport{}, ErrDatabaseNotInitialized
	}
	reader, ok := p.stockWriter.(InventoryStockReader)
	if p.stockWriter == nil || !ok {
		return InventoryRedisReconcileReport{}, ErrInventoryStockReconcileUnavailable
	}

	inventories, err := dao.ListAllInventories(ctx, p.db)
	if err != nil {
		return InventoryRedisReconcileReport{}, err
	}
	productIDs := make([]int64, 0, len(inventories))
	for _, inventory := range inventories {
		if inventory != nil && inventory.ProductID > 0 {
			productIDs = append(productIDs, inventory.ProductID)
		}
	}
	redisStocks, err := reader.GetInventoryStocks(ctx, productIDs)
	if err != nil {
		return InventoryRedisReconcileReport{}, ErrInventoryStockReconcileFailed
	}

	report := InventoryRedisReconcileReport{CheckedCount: len(productIDs)}
	for _, inventory := range inventories {
		if inventory == nil || inventory.ProductID <= 0 {
			continue
		}
		redisQuantity, exists := redisStocks[inventory.ProductID]
		if !exists || redisQuantity == nil {
			report.Items = append(report.Items, InventoryRedisReconcileItem{
				ProductID:     inventory.ProductID,
				MySQLQuantity: inventory.StockQuantity,
				Status:        "missing",
			})
			continue
		}
		if *redisQuantity != inventory.StockQuantity {
			report.Items = append(report.Items, InventoryRedisReconcileItem{
				ProductID:     inventory.ProductID,
				MySQLQuantity: inventory.StockQuantity,
				RedisQuantity: redisQuantity,
				Status:        "mismatch",
			})
		}
	}
	report.DiffCount = len(report.Items)
	return report, nil
}

func (p *InventoryService) setInventoryStock(ctx context.Context, productID int64, quantity int64) {
	if p == nil || p.stockWriter == nil {
		return
	}
	p.stockWriter.SetInventoryStock(context.WithoutCancel(ctx), productID, quantity)
}
