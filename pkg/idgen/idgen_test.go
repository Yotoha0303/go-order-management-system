package idgen_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"go-order-management-system/pkg/idgen"
)

func TestSnowflake_Basic(t *testing.T) {
	sf, err := idgen.NewSnowflake(1)
	if err != nil {
		t.Fatalf("failed to create snowflake: %v", err)
	}

	id := sf.NextID()
	if id <= 0 {
		t.Fatalf("expected positive ID, got %d", id)
	}
}

func TestSnowflake_Monotonic(t *testing.T) {
	sf, err := idgen.NewSnowflake(2)
	if err != nil {
		t.Fatalf("failed to create snowflake: %v", err)
	}

	prev := sf.NextID()
	for i := 0; i < 5000; i++ {
		curr := sf.NextID()
		if curr <= prev {
			t.Fatalf("expected strictly increasing ID, got curr=%d <= prev=%d", curr, prev)
		}
		prev = curr
	}
}

func TestSnowflake_Validation(t *testing.T) {
	if _, err := idgen.NewSnowflake(-1); !errors.Is(err, idgen.ErrInvalidWorkerID) {
		t.Fatalf("expected ErrInvalidWorkerID for negative worker ID, got %v", err)
	}
	if _, err := idgen.NewSnowflake(1024); !errors.Is(err, idgen.ErrInvalidWorkerID) {
		t.Fatalf("expected ErrInvalidWorkerID for worker ID 1024, got %v", err)
	}

	futureEpoch := time.Now().Add(24 * time.Hour).UnixMilli()
	if _, err := idgen.NewSnowflakeWithEpoch(1, futureEpoch); !errors.Is(err, idgen.ErrInvalidEpoch) {
		t.Fatalf("expected ErrInvalidEpoch for future epoch, got %v", err)
	}
	if _, err := idgen.NewSnowflakeWithEpoch(1, 0); !errors.Is(err, idgen.ErrInvalidEpoch) {
		t.Fatalf("expected ErrInvalidEpoch for zero epoch, got %v", err)
	}
}

func TestSnowflake_ConcurrentUnique(t *testing.T) {
	sf, err := idgen.NewSnowflake(42)
	if err != nil {
		t.Fatalf("failed to create snowflake: %v", err)
	}

	const goroutines = 50
	const idsPerGoroutine = 2000
	const totalIDs = goroutines * idsPerGoroutine

	var mu sync.Mutex
	generated := make(map[int64]struct{}, totalIDs)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			localIDs := make([]int64, idsPerGoroutine)
			for i := 0; i < idsPerGoroutine; i++ {
				localIDs[i] = sf.NextID()
			}

			mu.Lock()
			for _, id := range localIDs {
				generated[id] = struct{}{}
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(generated) != totalIDs {
		t.Fatalf("expected %d unique IDs, got %d (found %d duplicates)", totalIDs, len(generated), totalIDs-len(generated))
	}
}

func TestSnowflake_PackageNextID(t *testing.T) {
	id1 := idgen.NextID()
	id2 := idgen.NextID()
	if id1 <= 0 || id2 <= 0 {
		t.Fatalf("expected positive IDs from NextID(), got %d, %d", id1, id2)
	}
	if id1 == id2 {
		t.Fatalf("expected unique IDs, got duplicates: %d", id1)
	}
}

func BenchmarkSnowflake_NextID(b *testing.B) {
	sf, err := idgen.NewSnowflake(1)
	if err != nil {
		b.Fatalf("failed to create snowflake: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sf.NextID()
	}
}

func BenchmarkSnowflake_NextID_Parallel(b *testing.B) {
	sf, err := idgen.NewSnowflake(1)
	if err != nil {
		b.Fatalf("failed to create snowflake: %v", err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = sf.NextID()
		}
	})
}

func TestSnowflake_ClockRollback(t *testing.T) {
	sf, err := idgen.NewSnowflake(10)
	if err != nil {
		t.Fatalf("failed to create snowflake: %v", err)
	}

	baseTime := time.Now().UnixMilli()
	currentTime := baseTime

	sf.SetTimeFunc(func() int64 {
		return currentTime
	})

	id1 := sf.NextID()
	if id1 <= 0 {
		t.Fatalf("expected positive ID, got %d", id1)
	}

	// 1. Minor clock rollback (e.g. 50ms <= 100ms tolerance)
	// Clock jumps backward by 50ms, then advances after wait
	rollbackedTime := baseTime - 50
	callCount := 0
	sf.SetTimeFunc(func() int64 {
		callCount++
		if callCount == 1 {
			return rollbackedTime
		}
		// Clock catches up after sleep
		return baseTime + 10
	})

	id2, err := sf.NextIDWithError()
	if err != nil {
		t.Fatalf("expected minor rollback to be tolerated and resolved, got error: %v", err)
	}
	if id2 <= id1 {
		t.Fatalf("expected monotonic increasing ID after minor rollback, got id2=%d <= id1=%d", id2, id1)
	}

	// 2. Severe clock rollback (e.g. 500ms > 100ms tolerance)
	// Freeze clock at baseTime - 500ms
	severeRollbackTime := baseTime - 500
	sf.SetTimeFunc(func() int64 {
		return severeRollbackTime
	})

	_, err = sf.NextIDWithError()
	if !errors.Is(err, idgen.ErrClockMovedBack) {
		t.Fatalf("expected ErrClockMovedBack for severe clock rollback, got %v", err)
	}

	// Verify that NextID() panics on severe rollback
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic on severe clock rollback from NextID()")
		}
		recoveredErr, ok := r.(error)
		if !ok || !errors.Is(recoveredErr, idgen.ErrClockMovedBack) {
			t.Fatalf("expected panic with ErrClockMovedBack, got %v", r)
		}
	}()

	_ = sf.NextID()
}
