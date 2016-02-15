package dt

import (
	"runtime"
	"sync"
)

// atomicSet is a thread-safe map that acts as a Set of strings (no duplicates
// are possible).
type atomicSet struct {
	words map[string]struct{}
	mutex *sync.Mutex
}

// NewAtomicSet returns an atomicSet to track whether string values have been
// set or not in an efficient and thread-safe way.
func NewAtomicSet() atomicSet {
	return atomicSet{
		words: map[string]struct{}{},
		mutex: &sync.Mutex{},
	}
}

// Get returns whether a string exists in a given Set in a thread-safe way.
func (as atomicSet) Get(k string) (exists bool) {
	var b bool
	as.mutex.Lock()
	_, b = as.words[k]
	as.mutex.Unlock()
	runtime.Gosched()
	return b
}

// Set a given string as existing within the atomicSet.
func (as atomicSet) Set(k string) {
	as.mutex.Lock()
	as.words[k] = struct{}{}
	as.mutex.Unlock()
	runtime.Gosched()
}
