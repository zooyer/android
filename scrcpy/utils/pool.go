package utils

import (
	"log"
	"sync"
	"time"
)

type Stats struct {
	counters  map[string]int64
	durations map[string][]time.Duration
	mutex     sync.RWMutex
}

func NewStats() *Stats {
	return &Stats{
		counters:  make(map[string]int64),
		durations: make(map[string][]time.Duration),
	}
}

func (s *Stats) Increment(name string) {
	s.mutex.Lock()
	s.counters[name]++
	s.mutex.Unlock()
}

func (s *Stats) RecordDuration(name string, duration time.Duration) {
	s.mutex.Lock()
	s.durations[name] = append(s.durations[name], duration)
	if len(s.durations[name]) > 100 {
		s.durations[name] = s.durations[name][1:]
	}
	s.mutex.Unlock()
}

func (s *Stats) Print() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for name, count := range s.counters {
		log.Printf("统计 %s: %d", name, count)
	}

	for name, durations := range s.durations {
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := total / time.Duration(len(durations))
			log.Printf("平均 %s: %v", name, avg)
		}
	}
}

// BytePool 字节切片对象池
type BytePool struct {
	pool sync.Pool
	size int
}

func NewBytePool(size int) *BytePool {
	return &BytePool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

func (p *BytePool) Get() []byte {
	return p.pool.Get().([]byte)
}

func (p *BytePool) Put(b []byte) {
	if cap(b) >= p.size {
		p.pool.Put(b[:p.size])
	}
}
