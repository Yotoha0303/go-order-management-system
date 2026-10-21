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
	"go-order-management-system/pkg/idgen"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	orderNoPrefix    = "ORD"
	maxOrderPage     = 1_000_000
	maxOrderPageSize = 100
)

func (p *OrderService) CreateOrder(ctx context.Context, userID int64, req request.CreateOrderRequest) (*model.Order, error) {
	// 1. 参数防御与商品重复校验（validateDuplicateItems）前置到最外层
	if userID <= 0 {
		return nil, ErrInvalidUserID
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" || len(req.IdempotencyKey) > 128 {
		return nil, ErrInvalidIdempotencyKey
	}
	if err := validateDuplicateItems(req.Items); err != nil {
		return nil, err
	}

	requestHash, err := buildCreateOrderRequestHash(req)
	if err != nil {
		return nil, err
	}

	items := sortedOrderItems(req.Items)

	// 2. 预扣防线前置：通过 idgen.NextID() 预生成 orderID，并在开启 MySQL 事务之前调用 p.preDeductInventory
	//    注意：绝对不在预扣前对 MySQL 发起任何查询！让 Redis 真正成为拦截 99.9% 售罄流量的第一道护城河
	orderID := idgen.NextID()
	applied, insufficient := p.preDeductInventory(ctx, orderID, items)

	// 3. 极速拦截：若 Redis 预扣返回 insufficient，直接返回 ErrInsufficientStock，绝对不触碰 MySQL，零 DB 事务与零连接消耗
	if insufficient {
		p.recordOrderCreate("insufficient_stock")
		return nil, ErrInsufficientStock
	}

	// 4. 极简事务：进入 p.db.Transaction(...)，内部绝对不包含任何外部网络 IO；
	//    model.Order 直接使用预生成的 orderID；计算好 TotalAmountFen 后一次性创建订单（消除多余的 PatchOrderTotalPriceFen 更新）；
	//    完成条件扣减、明细写入、流水写入、Outbox 写入与幂等标记
	var createdOrder *model.Order
	idempotentReplay := false

	if p.db == nil {
		if applied {
			p.releasePreDeductedInventory(context.WithoutCancel(ctx), orderID)
		}
		p.recordOrderCreate("error")
		return nil, errors.New("database connection is nil")
	}

	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acquired, err := dao.TryCreateOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey, requestHash)
		if err != nil {
			return err
		}
		if !acquired {
			record, err := dao.GetOrderIdempotencyKey(tx, ctx, userID, req.IdempotencyKey)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// 在 MySQL RR 隔离级别下，并发先到达的事务已插入记录但尚未提交，后到达的事务快照读不到，必须作为“创建中”处理
					return ErrOrderBeingCreated
				}
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
				replayOrder, err := dao.GetOrderByID(ctx, tx, userID, *record.OrderID)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrOrderIdempotencyStateInvalid
				}
				if err == nil {
					createdOrder = replayOrder
					idempotentReplay = true
				}
				return err
			case model.OrderBeingCreated:
				return ErrOrderBeingCreated
			default:
				return ErrOrderIdempotencyStateInvalid
			}
		}

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

		var totalAmountFen int64
		for _, itemReq := range items {
			product := products[itemReq.ProductID]
			totalAmountFen += product.PriceFen * itemReq.Quantity
		}

		now := time.Now()
		order := &model.Order{
			ID:             orderID,
			UserID:         userID,
			OrderNo:        generateOrderNo(),
			TotalAmountFen: totalAmountFen,
			Status:         model.OrderStatusPending,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := dao.CreateOrder(ctx, tx, order); err != nil {
			return err
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

			if err := dao.ReserveInventory(ctx, tx, itemReq.ProductID, itemReq.Quantity); err != nil {
				if errors.Is(err, dao.ErrReserveInventoryFailed) {
					return ErrInsufficientStock
				}
				return err
			}

			subtotalFen := product.PriceFen * itemReq.Quantity
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

		createdOrder = order
		return nil
	})

	// 5. 补偿闭环：若 MySQL 事务提交失败或异常，在事务外调用 p.releasePreDeductedInventory(context.WithoutCancel(ctx), orderID) 释放预扣凭证
	if err != nil {
		if applied {
			p.releasePreDeductedInventory(context.WithoutCancel(ctx), orderID)
		}
		p.recordOrderCreate(classifyOrderCreateError(err))
		return nil, err
	}

	if idempotentReplay {
		if applied {
			p.releasePreDeductedInventory(context.WithoutCancel(ctx), orderID)
		}
		p.recordOrderCreate("idempotent_replay")
	} else {
		p.recordOrderCreate("success")
	}
	return createdOrder, nil
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
	if err == nil && (restored || idempotent) {
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
	alreadyCancelled := false
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
		if order.Status == model.OrderStatusCancelled {
			alreadyCancelled = true
			return nil
		}
		if order.Status != model.OrderStatusPending {
			ignored = true
			return nil
		}

		restored, err = cancelPendingOrder(ctx, tx, order, "订单超时取消回滚库存：")
		return err
	})
	if err == nil && (restored || alreadyCancelled) {
		p.restorePreDeductedInventory(context.WithoutCancel(ctx), orderID)
	}
	if err == nil && alreadyCancelled {
		p.recordOrderTransition("timeout_cancel", "already_cancelled")
	} else if err == nil && ignored {
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

	items, err := dao.ListOrderItemsByOrderIDForInventory(ctx, tx, order.ID)
	if err != nil {
		return false, err
	}

	sortedItems := aggregateAndSortOrderItems(items)

	for _, item := range sortedItems {
		inventory, err := dao.GetInventoryByProductIDForUpdate(ctx, tx, item.ProductID)
		if err != nil {
			return false, err
		}

		before := inventory.StockQuantity
		after := before + item.Quantity

		if err := dao.ReleaseLockedInventory(ctx, tx, item.ProductID, item.Quantity); err != nil {
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

func aggregateAndSortOrderItems(items []*model.OrderItem) []*model.OrderItem {
	qtyMap := make(map[int64]int64, len(items))
	itemMap := make(map[int64]*model.OrderItem, len(items))
	for _, it := range items {
		if it == nil || it.ProductID <= 0 {
			continue
		}
		qtyMap[it.ProductID] += it.Quantity
		itemMap[it.ProductID] = it
	}
	sorted := make([]*model.OrderItem, 0, len(qtyMap))
	for pID, qty := range qtyMap {
		base := itemMap[pID]
		sorted = append(sorted, &model.OrderItem{
			ID:              base.ID,
			OrderID:         base.OrderID,
			ProductID:       pID,
			ProductName:     base.ProductName,
			ProductPriceFen: base.ProductPriceFen,
			Quantity:        qty,
			SubtotalFen:     base.ProductPriceFen * qty,
		})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ProductID < sorted[j].ProductID
	})
	return sorted
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

	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := dao.PatchOrderStatus(ctx, tx, userID, order.ID, model.OrderStatusPending, model.OrderStatusPaid, "paid_at")
		if err != nil {
			return err
		}

		if row == 0 {
			return ErrOrderPayFailed
		}

		items, err := dao.ListOrderItemsByOrderIDForInventory(ctx, tx, order.ID)
		if err != nil {
			return err
		}

		sortedItems := aggregateAndSortOrderItems(items)
		for _, item := range sortedItems {
			if err := dao.ConsumeLockedInventory(ctx, tx, item.ProductID, item.Quantity); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
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
