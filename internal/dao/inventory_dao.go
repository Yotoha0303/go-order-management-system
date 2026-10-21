package dao

import (
	"context"
	"errors"
	"go-order-management-system/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrReserveInventoryFailed       = errors.New("reserve inventory failed: insufficient stock or inventory not found")
	ErrReleaseLockedInventoryFailed = errors.New("release locked inventory failed: insufficient locked stock or inventory not found")
	ErrConsumeLockedInventoryFailed = errors.New("consume locked inventory failed: insufficient locked stock or inventory not found")
)

func InitInventory(ctx context.Context, db *gorm.DB, inventory *model.Inventory) error {
	return db.WithContext(ctx).Create(inventory).Error
}

func GetInventoryByProductID(ctx context.Context, db *gorm.DB, productID int64) (*model.Inventory, error) {
	var inventory model.Inventory
	return &inventory, db.WithContext(ctx).Where("product_id = ?", productID).First(&inventory).Error
}

func GetInventoryByProductIDForUpdate(ctx context.Context, db *gorm.DB, productID int64) (*model.Inventory, error) {
	var inventory model.Inventory
	return &inventory, db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_id = ?", productID).First(&inventory).Error
}

func ListAllInventories(ctx context.Context, db *gorm.DB) ([]*model.Inventory, error) {
	var inventories []*model.Inventory
	return inventories, db.WithContext(ctx).Order("product_id asc").Find(&inventories).Error
}

func UpdateInventoryStockQuantity(ctx context.Context, db *gorm.DB, productID int64, stockQuantity int64) error {
	result := db.WithContext(ctx).Model(&model.Inventory{}).Where("product_id = ?", productID).Update("stock_quantity", stockQuantity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func DeductInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) (int64, error) {
	result := db.WithContext(ctx).Model(&model.Inventory{}).Where("product_id = ? AND stock_quantity >= ?", productID, quantity).
		UpdateColumn("stock_quantity", gorm.Expr("stock_quantity - ?", quantity))
	return result.RowsAffected, result.Error
}

// ReserveInventory 原子预占库存：stock_quantity - quantity, locked_quantity + quantity
func ReserveInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) error {
	result := db.WithContext(ctx).Model(&model.Inventory{}).
		Where("product_id = ? AND stock_quantity >= ?", productID, quantity).
		Updates(map[string]interface{}{
			"stock_quantity":  gorm.Expr("stock_quantity - ?", quantity),
			"locked_quantity": gorm.Expr("locked_quantity + ?", quantity),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrReserveInventoryFailed
	}
	return nil
}

// ReleaseLockedInventory 原子释放锁定库存：stock_quantity + quantity, locked_quantity - quantity
func ReleaseLockedInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) error {
	result := db.WithContext(ctx).Model(&model.Inventory{}).
		Where("product_id = ? AND locked_quantity >= ?", productID, quantity).
		Updates(map[string]interface{}{
			"stock_quantity":  gorm.Expr("stock_quantity + ?", quantity),
			"locked_quantity": gorm.Expr("locked_quantity - ?", quantity),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrReleaseLockedInventoryFailed
	}
	return nil
}

// ConsumeLockedInventory 支付核销实物出库：locked_quantity - quantity
func ConsumeLockedInventory(ctx context.Context, db *gorm.DB, productID int64, quantity int64) error {
	result := db.WithContext(ctx).Model(&model.Inventory{}).
		Where("product_id = ? AND locked_quantity >= ?", productID, quantity).
		UpdateColumn("locked_quantity", gorm.Expr("locked_quantity - ?", quantity))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrConsumeLockedInventoryFailed
	}
	return nil
}
