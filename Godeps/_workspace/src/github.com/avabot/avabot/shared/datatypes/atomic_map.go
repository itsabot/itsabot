package dt

import (
	"runtime"
	"sync"
)

type AtomicMap struct {
	words map[string]bool
	mutex *sync.Mutex
}

func NewAtomicMap() AtomicMap {
	return AtomicMap{
		words: map[string]bool{},
		mutex: &sync.Mutex{},
	}
}

func (am AtomicMap) Get(k string) bool {
	var b bool
	am.mutex.Lock()
	b = am.words[k]
	am.mutex.Unlock()
	runtime.Gosched()
	return b
}

func (am AtomicMap) Set(k string, v bool) {
	am.mutex.Lock()
	am.words[k] = v
	am.mutex.Unlock()
	runtime.Gosched()
}
