package service

import (
	"errors"
	"testing"
)

// simulatedInventory 模拟数据库产品库存记录内存实体
type simulatedInventory struct {
	ProductID      int64
	StockQuantity  int64
	LockedQuantity int64
}

func (s *simulatedInventory) TotalPhysical() int64 {
	return s.StockQuantity + s.LockedQuantity
}

func (s *simulatedInventory) Reserve(q int64) error {
	if q <= 0 {
		return errors.New("reserve quantity must be positive")
	}
	if s.StockQuantity < q {
		return ErrInsufficientStock
	}
	s.StockQuantity -= q
	s.LockedQuantity += q
	return nil
}

func (s *simulatedInventory) Release(q int64) error {
	if q <= 0 {
		return errors.New("release quantity must be positive")
	}
	if s.LockedQuantity < q {
		return errors.New("insufficient locked quantity to release")
	}
	s.StockQuantity += q
	s.LockedQuantity -= q
	return nil
}

func (s *simulatedInventory) Consume(q int64) error {
	if q <= 0 {
		return errors.New("consume quantity must be positive")
	}
	if s.LockedQuantity < q {
		return errors.New("insufficient locked quantity to consume")
	}
	s.LockedQuantity -= q
	return nil
}

func TestInventoryThreeStateLifecycle_ConservationAndTransitions(t *testing.T) {
	// 初始状态：仓库实物 100 件，可用 100 件，在途锁定 0 件
	inv := &simulatedInventory{
		ProductID:      101,
		StockQuantity:  100,
		LockedQuantity: 0,
	}

	if inv.TotalPhysical() != 100 {
		t.Fatalf("initial total physical must be 100, got: %d", inv.TotalPhysical())
	}

	// 1. 下单预占阶段：用户 A 下单购买 30 件
	if err := inv.Reserve(30); err != nil {
		t.Fatalf("reserve 30 failed: %v", err)
	}
	if inv.StockQuantity != 70 {
		t.Errorf("expected available stock 70, got: %d", inv.StockQuantity)
	}
	if inv.LockedQuantity != 30 {
		t.Errorf("expected locked stock 30, got: %d", inv.LockedQuantity)
	}
	// 守恒定律检验：预占状态下，总物理库存守恒！
	if inv.TotalPhysical() != 100 {
		t.Errorf("conservation violation: expected total 100, got: %d", inv.TotalPhysical())
	}

	// 负向证伪检验：超额预占（当前可用 70，试图预占 71）必定阻断失败
	if err := inv.Reserve(71); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock for over-reservation, got: %v", err)
	}

	// 2. 超时取消/主动关单释放阶段：用户 A 订单取消，释放 30 件
	if err := inv.Release(30); err != nil {
		t.Fatalf("release 30 failed: %v", err)
	}
	if inv.StockQuantity != 100 {
		t.Errorf("expected available stock back to 100, got: %d", inv.StockQuantity)
	}
	if inv.LockedQuantity != 0 {
		t.Errorf("expected locked stock back to 0, got: %d", inv.LockedQuantity)
	}
	// 守恒定律检验：释放后归还可用库存，总物理库存守恒！
	if inv.TotalPhysical() != 100 {
		t.Errorf("conservation violation: expected total 100, got: %d", inv.TotalPhysical())
	}

	// 负向证伪检验：锁定库存归零后，试图释放 1 件必定报错
	if err := inv.Release(1); err == nil {
		t.Fatalf("expected error when releasing from empty locked stock")
	}

	// 3. 再次下单预占：用户 B 下单购买 40 件
	if err := inv.Reserve(40); err != nil {
		t.Fatalf("reserve 40 failed: %v", err)
	}
	if inv.StockQuantity != 60 || inv.LockedQuantity != 40 {
		t.Errorf("expected S=60, L=40, got S=%d, L=%d", inv.StockQuantity, inv.LockedQuantity)
	}
	if inv.TotalPhysical() != 100 {
		t.Errorf("conservation violation: expected total 100, got: %d", inv.TotalPhysical())
	}

	// 4. 支付成功核销阶段：用户 B 付款成功，核销 40 件（实物出库）
	if err := inv.Consume(40); err != nil {
		t.Fatalf("consume 40 failed: %v", err)
	}
	if inv.StockQuantity != 60 {
		t.Errorf("expected available stock unchanged at 60, got: %d", inv.StockQuantity)
	}
	if inv.LockedQuantity != 0 {
		t.Errorf("expected locked stock cleared to 0, got: %d", inv.LockedQuantity)
	}
	// 物理出库检验：总物理库存实打实扣减 40 件，剩余 60 件
	if inv.TotalPhysical() != 60 {
		t.Errorf("expected total physical stock 60 after delivery, got: %d", inv.TotalPhysical())
	}

	// 负向证伪检验：支付核销后锁定库存已为 0，不能被再次核销或释放
	if err := inv.Consume(1); err == nil {
		t.Fatalf("expected error when consuming from empty locked stock")
	}
	if err := inv.Release(1); err == nil {
		t.Fatalf("expected error when releasing from empty locked stock")
	}
}
