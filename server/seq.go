package server

import (
	"sync"
	"sync/atomic"
)

type UniqueID struct {
	counter int64
}

func (c *UniqueID) Get() int64 {
	for {
		val := atomic.LoadInt64(&c.counter)
		if atomic.CompareAndSwapInt64(&c.counter, val, val+1) {
			return val
		}
	}
}

var UniqueIdInstance *UniqueID
var once sync.Once

func GetIdInstance() *UniqueID {
	once.Do(func() {
		UniqueIdInstance = &UniqueID{}
	})
	return UniqueIdInstance
}
