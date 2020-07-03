package GuavaCache

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Stats struct {
	HitCount         uint64
	MissCount        uint64
	LoadSuccessCount uint64
	LoadErrorCount   uint64
	TotalLoadTime    time.Duration
	EvictionCount    uint64
}

func (s *Stats) RequestCount() uint64 {
	return s.HitCount + s.MissCount
}

func (s *Stats) HitRate() float64 {
	total := s.RequestCount()
	if total == 0 {
		return 1.0
	}
	return float64(s.HitCount) / float64(total)
}

func (s *Stats) MissRate() float64 {
	total := s.RequestCount()
	if total == 0 {
		return 0.0
	}
	return float64(s.MissCount) / float64(total)
}

func (s *Stats) LoadErrorRate() float64 {
	total := s.LoadSuccessCount + s.LoadErrorCount
	if total == 0 {
		return 0.0
	}
	return float64(s.LoadErrorCount) / float64(total)
}

func (s *Stats) AverageLoadPenalty() time.Duration {
	total := s.LoadSuccessCount + s.LoadErrorCount
	if total == 0 {
		return 0.0
	}
	return s.TotalLoadTime / time.Duration(total)
}

func (s *Stats) String() string {
	return fmt.Sprintf("hits: %d, misses: %d, successes: %d, errors: %d, time: %s, evictions: %d",
		s.HitCount, s.MissCount, s.LoadSuccessCount, s.LoadErrorCount, s.TotalLoadTime, s.EvictionCount)
}

type StatsCounter interface {
	RecordHits(count uint64)

	RecordMisses(count uint64)

	RecordLoadSuccess(loadTime time.Duration)

	RecordLoadError(loadTime time.Duration)

	RecordEviction()

	Snapshot(*Stats)
}

type statsCounter struct {
	Stats
}

func (s *statsCounter) RecordHits(count uint64) {
	atomic.AddUint64(&s.Stats.HitCount, count)
}

func (s *statsCounter) RecordMisses(count uint64) {
	atomic.AddUint64(&s.Stats.MissCount, count)
}

func (s *statsCounter) RecordLoadSuccess(loadTime time.Duration) {
	atomic.AddUint64(&s.Stats.LoadSuccessCount, 1)
	atomic.AddInt64((*int64)(&s.Stats.TotalLoadTime), int64(loadTime))
}

func (s *statsCounter) RecordLoadError(loadTime time.Duration) {
	atomic.AddUint64(&s.Stats.LoadErrorCount, 1)
	atomic.AddInt64((*int64)(&s.Stats.TotalLoadTime), int64(loadTime))
}

func (s *statsCounter) RecordEviction() {
	atomic.AddUint64(&s.Stats.EvictionCount, 1)
}

func (s *statsCounter) Snapshot(t *Stats) {
	t.HitCount = atomic.LoadUint64(&s.HitCount)
	t.MissCount = atomic.LoadUint64(&s.MissCount)
	t.LoadSuccessCount = atomic.LoadUint64(&s.LoadSuccessCount)
	t.LoadErrorCount = atomic.LoadUint64(&s.LoadErrorCount)
	t.TotalLoadTime = time.Duration(atomic.LoadInt64((*int64)(&s.TotalLoadTime)))
	t.EvictionCount = atomic.LoadUint64(&s.EvictionCount)
}
