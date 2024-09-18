package testutils

import (
	"sync"
)

type TableCounter struct {
	counter      map[uint64]uint8
	counterMutex sync.RWMutex
}

func NewTableCounter() *TableCounter {

	tc := new(TableCounter)
	tc.counter = make(map[uint64]uint8)
	return tc
}

func (tc *TableCounter) PutNumber(number uint64) {
	tc.counterMutex.Lock()
	defer tc.counterMutex.Unlock()
	tc.counter[number]++
}

func (tc *TableCounter) ReturnCounter(number uint64) uint8 {
	tc.counterMutex.RLock()
	defer tc.counterMutex.RUnlock()
	return tc.counter[number]

}
