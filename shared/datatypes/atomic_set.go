package dt

import (
	"runtime"
	"sync"
)

// AtomicSet is a thread-safe map that acts as a Set of strings (no duplicates
// are possible). To initialize this struct, use NewAtomicSet().
type AtomicSet struct {
	words map[string]struct{}
	mutex *sync.Mutex
}

// NewAtomicSet returns an AtomicSet to track whether string values have been
// set or not in an efficient and thread-safe way.
func NewAtomicSet() AtomicSet {
	return AtomicSet{
		words: map[string]struct{}{},
		mutex: &sync.Mutex{},
	}
}

// Get returns whether a string exists in a given Set in a thread-safe way.
func (as AtomicSet) Get(k string) (exists bool) {
	var b bool
	as.mutex.Lock()
	_, b = as.words[k]
	as.mutex.Unlock()
	runtime.Gosched()
	return b
}

// Set a given string as existing within the AtomicSet.
func (as AtomicSet) Set(k string) {
	as.mutex.Lock()
	as.words[k] = struct{}{}
	as.mutex.Unlock()
	runtime.Gosched()
}
