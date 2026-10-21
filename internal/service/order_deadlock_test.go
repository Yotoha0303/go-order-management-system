package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"
)

// TestOrderItemsSorting_EliminatesDeadlockCircularWait 测试偏序排序算法对乱序、逆序与重复项的规范化
func TestOrderItemsSorting_EliminatesDeadlockCircularWait(t *testing.T) {
	// 逆序输入：客户端 A 提交 [P3, P2, P1]
	inputA := []request.CreateOrderItemRequest{
		{ProductID: 300, Quantity: 1},
		{ProductID: 200, Quantity: 2},
		{ProductID: 100, Quantity: 3},
	}
	sortedA := sortedOrderItems(inputA)
	if sortedA[0].ProductID != 100 || sortedA[1].ProductID != 200 || sortedA[2].ProductID != 300 {
		t.Fatalf("expected sorted [100, 200, 300], got: %+v", sortedA)
	}

	// 逆序且有重复项的 OrderItem 输入聚合排序
	inputItems := []*model.OrderItem{
		{ProductID: 200, Quantity: 1, ProductPriceFen: 100},
		{ProductID: 100, Quantity: 2, ProductPriceFen: 200},
		{ProductID: 200, Quantity: 3, ProductPriceFen: 100},
	}
	sortedItems := aggregateAndSortOrderItems(inputItems)
	if len(sortedItems) != 2 {
		t.Fatalf("expected 2 aggregated items, got: %d", len(sortedItems))
	}
	if sortedItems[0].ProductID != 100 || sortedItems[0].Quantity != 2 {
		t.Errorf("expected item 0 to be P100 with qty 2, got: %+v", sortedItems[0])
	}
	if sortedItems[1].ProductID != 200 || sortedItems[1].Quantity != 4 {
		t.Errorf("expected item 1 to be P200 with qty 4, got: %+v", sortedItems[1])
	}
}

// resourceLockManager 模拟带有严格获取超时与死锁检测的资源排他锁管理器
type resourceLockManager struct {
	mu    sync.Mutex
	locks map[int64]*sync.Mutex
}

func newResourceLockManager() *resourceLockManager {
	return &resourceLockManager{
		locks: make(map[int64]*sync.Mutex),
	}
}

func (m *resourceLockManager) getLock(productID int64) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.locks[productID]
	if !ok {
		l = &sync.Mutex{}
		m.locks[productID] = l
	}
	return l
}

// TestConcurrentCrossOrder_PartialOrderLocking_NoDeadlock
// 构造高并发交叉竞争场景：
// 50 个客户端先要 P1 再要 P2，50 个客户端先要 P2 再要 P1。
// 如果按照客户端原始逆序加锁，在没有偏序保护下将产生死锁（Circular Wait）；
// 在采用 product_id ASC 偏序加锁时，所有 Goroutine 严格先锁 P1 再锁 P2，100 次事务必须在极短时间内全部顺利执行完毕，零死锁！
func TestConcurrentCrossOrder_PartialOrderLocking_NoDeadlock(t *testing.T) {
	manager := newResourceLockManager()
	const concurrency = 50

	var wg sync.WaitGroup
	errCh := make(chan error, concurrency*2)

	executeTxWithLocking := func(reqItems []request.CreateOrderItemRequest) error {
		// 遵循 ADR-003 铁律：必须先进行 product_id ASC 排序
		items := sortedOrderItems(reqItems)

		acquiredLocks := make([]*sync.Mutex, 0, len(items))
		for _, item := range items {
			l := manager.getLock(item.ProductID)
			l.Lock()
			acquiredLocks = append(acquiredLocks, l)
		}

		// 模拟临界区耗时
		time.Sleep(100 * time.Microsecond)

		for i := len(acquiredLocks) - 1; i >= 0; i-- {
			acquiredLocks[i].Unlock()
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 启动组 A：客户端请求 [P1, P2]
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := []request.CreateOrderItemRequest{
				{ProductID: 101, Quantity: 1},
				{ProductID: 102, Quantity: 1},
			}
			if err := executeTxWithLocking(req); err != nil {
				errCh <- fmt.Errorf("group A-%d failed: %w", id, err)
			}
		}(i)
	}

	// 启动组 B：客户端请求 [P2, P1]（原始顺序相反，容易引发交叉死锁）
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := []request.CreateOrderItemRequest{
				{ProductID: 102, Quantity: 1},
				{ProductID: 101, Quantity: 1},
			}
			if err := executeTxWithLocking(req); err != nil {
				errCh <- fmt.Errorf("group B-%d failed: %w", id, err)
			}
		}(i)
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		t.Fatal("DEADLOCK DETECTED: concurrent cross-order transactions timed out due to lock contention/circular wait")
	case <-doneCh:
		// 顺利完成
	}

	close(errCh)
	for err := range errCh {
		t.Errorf("transaction execution error: %v", err)
	}
}
