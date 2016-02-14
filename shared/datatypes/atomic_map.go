package dt

import (
	"runtime"
	"sync"

	"github.com/avabot/ava/Godeps/_workspace/src/golang.org/x/net/websocket"
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

type AtomicWebSocketMap struct {
	sockets map[uint64]*websocket.Conn
	mutex   *sync.Mutex
}

func NewAtomicWebSocketMap() AtomicWebSocketMap {
	return AtomicWebSocketMap{
		sockets: map[uint64]*websocket.Conn{},
		mutex:   &sync.Mutex{},
	}
}

func (am AtomicWebSocketMap) Get(k uint64) *websocket.Conn {
	var b *websocket.Conn
	am.mutex.Lock()
	b = am.sockets[k]
	am.mutex.Unlock()
	runtime.Gosched()
	return b
}

func (am AtomicWebSocketMap) Set(k uint64, v *websocket.Conn) {
	am.mutex.Lock()
	am.sockets[k] = v
	am.mutex.Unlock()
	runtime.Gosched()
}
