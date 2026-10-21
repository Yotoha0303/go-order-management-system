package idgen

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// Custom epoch: 2024-01-01 00:00:00 UTC (1704067200000 ms)
	defaultEpoch = int64(1704067200000)

	workerIDBits = uint(10)
	sequenceBits = uint(12)

	maxWorkerID = int64(-1 ^ (-1 << workerIDBits)) // 1023 (0..1023)
	maxSequence = int64(-1 ^ (-1 << sequenceBits)) // 4095 (0..4095)

	timeShift   = workerIDBits + sequenceBits // 22
	workerShift = sequenceBits                // 12
)

var (
	ErrInvalidWorkerID = errors.New("idgen: worker ID out of range [0, 1023]")
	ErrInvalidEpoch    = errors.New("idgen: epoch must be positive and not in the future")
	ErrClockMovedBack  = errors.New("idgen: clock moved backwards beyond tolerance (100ms)")
)

// Generator defines the interface for generating unique 64-bit integer IDs.
type Generator interface {
	NextID() int64
}

// Snowflake is a distributed unique ID generator based on Twitter's Snowflake algorithm.
// Total 64 bits:
//   - 1 bit: sign bit (always 0)
//   - 41 bits: timestamp delta in milliseconds (~69 years)
//   - 10 bits: worker / machine ID (0..1023)
//   - 12 bits: sequence number within the same millisecond (0..4095)
type Snowflake struct {
	mu            sync.Mutex
	epoch         int64
	workerID      int64
	lastTimestamp int64
	sequence      int64
	timeFunc      func() int64
}

// NewSnowflake creates a Snowflake generator with default epoch.
func NewSnowflake(workerID int64) (*Snowflake, error) {
	return NewSnowflakeWithEpoch(workerID, defaultEpoch)
}

// NewSnowflakeWithEpoch creates a Snowflake generator with custom epoch in milliseconds.
func NewSnowflakeWithEpoch(workerID int64, epoch int64) (*Snowflake, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("%w: got %d (max %d)", ErrInvalidWorkerID, workerID, maxWorkerID)
	}
	if epoch <= 0 || epoch > time.Now().UnixMilli() {
		return nil, ErrInvalidEpoch
	}
	return &Snowflake{
		epoch:         epoch,
		workerID:      workerID,
		lastTimestamp: -1,
		sequence:      0,
	}, nil
}

// SetTimeFunc sets a custom timestamp provider (primarily for unit tests and clock drift simulation).
func (s *Snowflake) SetTimeFunc(fn func() int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timeFunc = fn
}

func (s *Snowflake) currentTime() int64 {
	if s.timeFunc != nil {
		return s.timeFunc()
	}
	return time.Now().UnixMilli()
}

// NextIDWithError generates a 64-bit unique ID.
// If clock moves backwards by <= 100ms, it spin-waits for the clock to catch up.
// If clock rollback exceeds 100ms, it returns ErrClockMovedBack.
func (s *Snowflake) NextIDWithError() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.currentTime()
	if now < s.lastTimestamp {
		drift := s.lastTimestamp - now
		if drift <= 100 {
			time.Sleep(time.Duration(drift) * time.Millisecond)
			now = s.currentTime()
		}
		if now < s.lastTimestamp {
			return 0, fmt.Errorf("%w: drift=%dms", ErrClockMovedBack, s.lastTimestamp-now)
		}
	}

	if now == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			for now <= s.lastTimestamp {
				time.Sleep(50 * time.Microsecond)
				now = s.currentTime()
			}
		}
	} else {
		s.sequence = 0
	}

	s.lastTimestamp = now

	id := ((now - s.epoch) << timeShift) | (s.workerID << workerShift) | s.sequence
	return id, nil
}

// NextID generates a thread-safe 64-bit unique ID. Panics on severe clock rollback (>100ms).
func (s *Snowflake) NextID() int64 {
	id, err := s.NextIDWithError()
	if err != nil {
		panic(err)
	}
	return id
}

var (
	defaultGeneratorMu sync.RWMutex
	defaultGenerator   *Snowflake
)

func init() {
	var err error
	defaultGenerator, err = NewSnowflake(1)
	if err != nil {
		panic(fmt.Sprintf("init default snowflake generator failed: %v", err))
	}
}

// NextID generates a 64-bit unique ID using the package-level default generator.
func NextID() int64 {
	defaultGeneratorMu.RLock()
	gen := defaultGenerator
	defaultGeneratorMu.RUnlock()
	return gen.NextID()
}

// SetDefaultGenerator sets the default Snowflake generator.
func SetDefaultGenerator(gen *Snowflake) {
	if gen == nil {
		return
	}
	defaultGeneratorMu.Lock()
	defaultGenerator = gen
	defaultGeneratorMu.Unlock()
}
