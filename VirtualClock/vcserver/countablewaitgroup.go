package main

import (
	"sync"
	"sync/atomic"
)

// カウンター付き WaitGroup
type CountableWaitGroup struct {
	sync.WaitGroup
	count int64
}

func (wg *CountableWaitGroup) Add(delta int) {
	atomic.AddInt64(&wg.count, int64(delta))
	wg.WaitGroup.Add(delta)
}

func (wg *CountableWaitGroup) Done() {
	atomic.AddInt64(&wg.count, -1)
	wg.WaitGroup.Done()
}

func (wg *CountableWaitGroup) GetCount() int {
	return int(atomic.LoadInt64(&wg.count))
}
