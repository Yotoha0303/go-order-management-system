package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"go-order-management-system/internal/dao"
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	orderNoPrefix    = "ORD"
	maxOrderPage     = 1_000_000
	maxOrderPageSize = 100
)

func (p *OrderService) CreateOrder(ctx context.Context, userID int64, req request.CreateOrderRequest) (*model.Order, error) {
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" || len(req.IdempotencyKey) > 128 {
		return nil, ErrInvalidIdempotencyKey
	}

	requestHash, err := buildCreateOrderRequestHash(req)
	if err != nil {
		return nil, err
	}

	var createOrder *model.Order
	var preDeductedOrderID int64
	idempotentReplay := false
	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acquired, err := dao.TryCreateOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey, requestHash)
		if err != nil {
			return err
		}
		if !acquired {
			record, err := dao.GetOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey)
			if err != nil {
				return err
			}
			if record.RequestHash != requestHash {
				return ErrOrderIdempotencyConflict
			}

			switch record.Status {
			case model.OrderAlreadyCreated:
				if record.OrderID == nil || *record.OrderID <= 0 {
					return ErrOrderIdempotencyStateInvalid
				}

				createOrder, err = dao.GetOrderByID(ctx, tx, userID, *record.OrderID)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrOrderIdempotencyStateInvalid
				}
				if err == nil {
					idempotentReplay = true
				}
				return err
			case model.OrderBeingCreated:
				return ErrOrderBeingCreated
			default:
				return ErrOrderIdempotencyStateInvalid
			}
		}

		var totalAmountFen int64
		order := &model.Order{
			UserID:         userID,
			OrderNo:        generateOrderNo(),
			TotalAmountFen: 0,
			Status:         model.OrderStatusPending,
			CreatedAt:      time.Now(),
		}
		if err := dao.CreateOrder(ctx, tx, order); err != nil {
			return err
		}

		if err := validateDuplicateItems(req.Items); err != nil {
			return err
		}

		items := sortedOrderItems(req.Items)
		products := make(map[int64]*model.Product, len(items))
		for _, itemReq := range items {
			product, err := dao.GetProductByID(ctx, tx, itemReq.ProductID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrProductNotFound
				}
				return err
			}
			if product.Status != model.ProductStatusOnSale {
				return ErrProductOffSale
			}
			products[itemReq.ProductID] = product
		}

		applied, insufficient := p.preDeductInventory(ctx, order.ID, items)
		if insufficient {
			return ErrInsufficientStock
		}
		if applied {
			preDeductedOrderID = order.ID
		}

		for _, itemReq := range items {
			product := products[itemReq.ProductID]
			inv, err := dao.GetInventoryByProductIDForUpdate(ctx, tx, itemReq.ProductID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrInventoryNotFound
				}
				return err
			}

			beforeQuantity := inv.StockQuantity
			afterQuantity := beforeQuantity - itemReq.Quantity
			if afterQuantity < 0 {
				return ErrInsufficientStock
			}

			rows, err := dao.DeductInventory(ctx, tx, itemReq.ProductID, itemReq.Quantity)
			if err != nil {
				return err
			}
			if rows == 0 {
				return ErrInsufficientStock
			}

			subtotalFen := product.PriceFen * itemReq.Quantity
			totalAmountFen += subtotalFen

			orderItem := &model.OrderItem{
				OrderID:         order.ID,
				ProductID:       product.ID,
				ProductName:     product.Name,
				ProductPriceFen: product.PriceFen,
				Quantity:        itemReq.Quantity,
				SubtotalFen:     subtotalFen,
			}
			if err := dao.CreateOrderItem(ctx, tx, orderItem); err != nil {
				return err
			}

			stockLog := &model.StockLog{
				ProductID:      product.ID,
				ChangeQuantity: -itemReq.Quantity,
				BeforeQuantity: beforeQuantity,
				AfterQuantity:  afterQuantity,
				BizType:        model.StockBizOrderDeduct,
				BizID:          &order.ID,
				Remark:         "创建订单扣减库存：" + order.OrderNo,
			}
			if err := dao.CreateStockLog(ctx, tx, stockLog); err != nil {
				return ErrCreateStockLogFailed
			}
		}

		if err := dao.PatchOrderTotalPriceFen(ctx, tx, order.ID, totalAmountFen, userID); err != nil {
			return err
		}

		rowsAffected, err := dao.CompleteOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey, order.ID)
		if err != nil || rowsAffected != 1 {
			return ErrOrderIdempotencyStateInvalid
		}

		timeoutAt := order.CreatedAt.Add(p.orderTimeout)
		if err := dao.CreateOrderTimeoutOutbox(ctx, tx, &model.OrderTimeoutOutbox{
			EventID:       uuid.NewString(),
			OrderID:       order.ID,
			UserID:        userID,
			TimeoutAt:     timeoutAt,
			NextAttemptAt: time.Now(),
		}); err != nil {
			return err
		}

		order.TotalAmountFen = totalAmountFen
		createOrder = order
		return nil
	})
	if err != nil {
		if preDeductedOrderID > 0 {
			p.releasePreDeductedInventory(context.WithoutCancel(ctx), preDeductedOrderID)
		}
		p.recordOrderCreate(classifyOrderCreateError(err))
		return nil, err
	}

	if idempotentReplay {
		p.recordOrderCreate("idempotent_replay")
	} else {
		p.recordOrderCreate("success")
	}
	return createOrder, nil
}

func validateDuplicateItems(items []request.CreateOrderItemRequest) error {
	seen := make(map[int64]struct{}, len(items))

	for _, item := range items {
		if _, ok := seen[item.ProductID]; ok {
			return ErrDuplicateOrderItem
		}
		seen[item.ProductID] = struct{}{}
	}
	return nil
}

func sortedOrderItems(items []request.CreateOrderItemRequest) []request.CreateOrderItemRequest {
	sorted := append([]request.CreateOrderItemRequest(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ProductID == sorted[j].ProductID {
			return sorted[i].Quantity < sorted[j].Quantity
		}
		return sorted[i].ProductID < sorted[j].ProductID
	})
	return sorted
}

func buildCreateOrderRequestHash(req request.CreateOrderRequest) (string, error) {
	items := sortedOrderItems(req.Items)

	payload := struct {
		Version int                              `json:"version"`
		Items   []request.CreateOrderItemRequest `json:"items"`
	}{
		Version: 1,
		Items:   items,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func generateOrderNo() string {
	return orderNoPrefix + uuid.NewString()
}

func classifyOrderCreateError(err error) string {
	switch {
	case errors.Is(err, ErrInsufficientStock):
		return "insufficient_stock"
	case errors.Is(err, ErrOrderIdempotencyConflict):
		return "idempotency_conflict"
	case errors.Is(err, ErrOrderBeingCreated):
		return "idempotency_in_progress"
	case errors.Is(err, ErrProductNotFound):
		return "product_not_found"
	case errors.Is(err, ErrProductOffSale):
		return "product_off_sale"
	case errors.Is(err, ErrInventoryNotFound):
		return "inventory_not_found"
	case errors.Is(err, ErrDuplicateOrderItem):
		return "duplicate_item"
	case errors.Is(err, ErrInvalidIdempotencyKey):
		return "invalid_idempotency_key"
	case errors.Is(err, ErrInvalidUserID):
		return "invalid_user"
	default:
		return "error"
	}
}

func classifyOrderTransitionError(err error) string {
	if err == nil {
		return "success"
	}
	switch {
	case errors.Is(err, ErrOrderNotFound):
		return "order_not_found"
	case errors.Is(err, ErrOrderAlreadyPaid):
		return "already_paid"
	case errors.Is(err, ErrOrderAlreadyFinished):
		return "already_finished"
	case errors.Is(err, ErrOrderAlreadyCanceled):
		return "already_cancelled"
	case errors.Is(err, ErrOrderNotPaid):
		return "not_paid"
	case errors.Is(err, ErrOrderPayFailed), errors.Is(err, ErrOrderCancelFailed), errors.Is(err, ErrOrderFinishFailed):
		return "state_conflict"
	default:
		return "error"
	}
}

func (p *OrderService) recordOrderCreate(result string) {
	if p != nil && p.metrics != nil {
		p.metrics.RecordOrderCreate(result)
	}
}

func (p *OrderService) recordOrderTransition(action, result string) {
	if p != nil && p.metrics != nil {
		p.metrics.RecordOrderStateTransition(action, result)
	}
}

func (p *OrderService) preDeductInventory(
	ctx context.Context,
	orderID int64,
	items []request.CreateOrderItemRequest,
) (bool, bool) {
	if p == nil || p.inventoryPreDeductor == nil {
		return false, false
	}
	return p.inventoryPreDeductor.PreDeductInventory(ctx, orderID, items)
}

func (p *OrderService) releasePreDeductedInventory(ctx context.Context, orderID int64) {
	if p == nil || p.inventoryPreDeductor == nil {
		return
	}
	p.inventoryPreDeductor.ReleasePreDeductedInventory(ctx, orderID)
}

func (p *OrderService) restorePreDeductedInventory(ctx context.Context, orderID int64) {
	if p == nil || p.inventoryPreDeductor == nil {
		return
	}
	p.inventoryPreDeductor.RestorePreDeductedInventory(ctx, orderID)
}

func (p *OrderService) confirmPreDeductedInventory(ctx context.Context, orderID int64) {
	if p == nil || p.inventoryPreDeductor == nil {
		return
	}
	p.inventoryPreDeductor.ConfirmPreDeductedInventory(ctx, orderID)
}

func (p *OrderService) GetOrderByID(ctx context.Context, userID, id int64) (*model.Order, []*model.OrderItem, error) {
	order, err := dao.GetOrderByID(ctx, p.db, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrOrderNotFound
		}
		return nil, nil, err
	}

	items, err := dao.ListOrderItemsByOrderID(ctx, p.db, id)
	if err != nil {
		return nil, nil, err
	}

	return order, items, nil
}

func (p *OrderService) ListOrders(ctx context.Context, userID int64, page, pageSize int) ([]*model.Order, int64, error) {
	if userID <= 0 {
		return nil, 0, ErrInvalidUserID
	}
	if page <= 0 || page > maxOrderPage || pageSize <= 0 || pageSize > maxOrderPageSize {
		return nil, 0, ErrInvalidOrderPagination
	}

	offset := (page - 1) * pageSize
	return dao.ListOrders(ctx, p.db, userID, pageSize, offset)
}

func (p *OrderService) CancelOrder(ctx context.Context, userID, orderID int64) error {
	restored := false
	idempotent := false
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		order, err := dao.GetOrderByID(ctx, tx, userID, orderID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		switch order.Status {
		case model.OrderStatusCancelled:
			idempotent = true
			return nil
		case model.OrderStatusPending:
		case model.OrderStatusPaid:
			return ErrOrderAlreadyPaid
		case model.OrderStatusFinished:
			return ErrOrderAlreadyFinished
		default:
			return ErrOrderCancelFailed
		}

		cancelled, err := cancelPendingOrder(ctx, tx, order, "取消订单回滚库存：")
		if err != nil {
			return err
		}
		if !cancelled {
			return ErrOrderCancelFailed
		}
		restored = true
		return nil
	})
	if err == nil && restored {
		p.restorePreDeductedInventory(context.WithoutCancel(ctx), orderID)
	}
	if err == nil && idempotent {
		p.recordOrderTransition("cancel", "idempotent")
	} else {
		p.recordOrderTransition("cancel", classifyOrderTransitionError(err))
	}
	return err
}

func (p *OrderService) CancelExpiredOrder(ctx context.Context, eventID string, orderID int64) error {
	restored := false
	ignored := false
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		outbox, err := dao.GetOrderTimeoutOutbox(ctx, tx, eventID, orderID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ignored = true
			return nil
		}
		if err != nil {
			return err
		}
		if time.Now().Before(outbox.TimeoutAt) {
			return errors.New("order timeout deadline has not been reached")
		}

		order, err := dao.GetOrderByIDForSystem(ctx, tx, orderID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ignored = true
			return nil
		}
		if err != nil {
			return err
		}
		if order.Status != model.OrderStatusPending {
			ignored = true
			return nil
		}

		restored, err = cancelPendingOrder(ctx, tx, order, "订单超时取消回滚库存：")
		return err
	})
	if err == nil && restored {
		p.restorePreDeductedInventory(context.WithoutCancel(ctx), orderID)
	}
	if err == nil && ignored {
		p.recordOrderTransition("timeout_cancel", "ignored")
	} else {
		p.recordOrderTransition("timeout_cancel", classifyOrderTransitionError(err))
	}
	return err
}

func cancelPendingOrder(
	ctx context.Context,
	tx *gorm.DB,
	order *model.Order,
	remarkPrefix string,
) (bool, error) {
	rows, err := dao.PatchOrderStatus(
		ctx,
		tx,
		order.UserID,
		order.ID,
		model.OrderStatusPending,
		model.OrderStatusCancelled,
		"cancelled_at",
	)
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, nil
	}

	items, err := dao.ListOrderItemsByOrderID(ctx, tx, order.ID)
	if err != nil {
		return false, err
	}

	for _, item := range items {
		inventory, err := dao.GetInventoryByProductIDForUpdate(ctx, tx, item.ProductID)
		if err != nil {
			return false, err
		}

		before := inventory.StockQuantity
		after := before + item.Quantity

		if err := dao.UpdateInventoryStockQuantity(ctx, tx, item.ProductID, after); err != nil {
			return false, err
		}

		stockLog := &model.StockLog{
			ProductID:      item.ProductID,
			BizID:          &order.ID,
			ChangeQuantity: item.Quantity,
			AfterQuantity:  after,
			BeforeQuantity: before,
			BizType:        model.StockBizOrderRollback,
			Remark:         remarkPrefix + order.OrderNo,
		}

		if err := dao.CreateStockLog(ctx, tx, stockLog); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (p *OrderService) PayOrder(ctx context.Context, userID, orderID int64) error {
	order, err := dao.GetOrderByID(ctx, p.db, userID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrOrderNotFound
		}
		p.recordOrderTransition("pay", classifyOrderTransitionError(err))
		return err
	}

	switch order.Status {
	case model.OrderStatusPaid:
		err = ErrOrderAlreadyPaid
		p.recordOrderTransition("pay", classifyOrderTransitionError(err))
		return err
	case model.OrderStatusFinished:
		err = ErrOrderAlreadyFinished
		p.recordOrderTransition("pay", classifyOrderTransitionError(err))
		return err
	case model.OrderStatusCancelled:
		err = ErrOrderAlreadyCanceled
		p.recordOrderTransition("pay", classifyOrderTransitionError(err))
		return err
	case model.OrderStatusPending:
	default:
		err = ErrOrderPayFailed
		p.recordOrderTransition("pay", classifyOrderTransitionError(err))
		return err
	}

	row, err := dao.PatchOrderStatus(ctx, p.db, userID, order.ID, model.OrderStatusPending, model.OrderStatusPaid, "paid_at")
	if err != nil {
		p.recordOrderTransition("pay", classifyOrderTransitionError(err))
		return err
	}

	if row == 0 {
		err = ErrOrderPayFailed
		p.recordOrderTransition("pay", classifyOrderTransitionError(err))
		return err
	}
	p.confirmPreDeductedInventory(context.WithoutCancel(ctx), orderID)
	p.recordOrderTransition("pay", "success")
	return nil
}

func (p *OrderService) FinishOrder(ctx context.Context, userID, orderID int64) error {
	order, err := dao.GetOrderByID(ctx, p.db, userID, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrOrderNotFound
		}
		p.recordOrderTransition("finish", classifyOrderTransitionError(err))
		return err
	}

	switch order.Status {
	case model.OrderStatusPending:
		err = ErrOrderNotPaid
		p.recordOrderTransition("finish", classifyOrderTransitionError(err))
		return err
	case model.OrderStatusCancelled:
		err = ErrOrderAlreadyCanceled
		p.recordOrderTransition("finish", classifyOrderTransitionError(err))
		return err
	case model.OrderStatusFinished:
		err = ErrOrderAlreadyFinished
		p.recordOrderTransition("finish", classifyOrderTransitionError(err))
		return err
	case model.OrderStatusPaid:
	default:
		err = ErrOrderFinishFailed
		p.recordOrderTransition("finish", classifyOrderTransitionError(err))
		return err
	}

	row, err := dao.PatchOrderStatus(ctx, p.db, userID, order.ID, model.OrderStatusPaid, model.OrderStatusFinished, "completed_at")
	if err != nil {
		p.recordOrderTransition("finish", classifyOrderTransitionError(err))
		return err
	}

	if row == 0 {
		err = ErrOrderFinishFailed
		p.recordOrderTransition("finish", classifyOrderTransitionError(err))
		return err
	}
	p.recordOrderTransition("finish", "success")
	return nil

}
