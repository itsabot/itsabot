package main

import (
	"runtime"
	"sync"

	"golang.org/x/net/websocket"
)

// AtomicWebSocketSet maintains open websocket connections with up to one
// available per user. The uint64 key within the map represents the userID to
// whom the connection belongs.
type AtomicWebSocketSet struct {
	sockets map[uint64]*websocket.Conn
	mutex   *sync.Mutex
}

// NewAtomicWebSocketSet returns an AtomicWebSocketSet to maintain open
// WebSocket connections on a per-user basis.
func NewAtomicWebSocketSet() AtomicWebSocketSet {
	return AtomicWebSocketSet{
		sockets: map[uint64]*websocket.Conn{},
		mutex:   &sync.Mutex{},
	}
}

// Get returns a WebSocket connection for a given userID in a thread-safe way.
func (as AtomicWebSocketSet) Get(userID uint64) *websocket.Conn {
	var conn *websocket.Conn
	as.mutex.Lock()
	conn = as.sockets[userID]
	as.mutex.Unlock()
	runtime.Gosched()
	return conn
}

// Set a WebSocket connection for a given userID in a thread-safe way.
func (as AtomicWebSocketSet) Set(userID uint64, conn *websocket.Conn) {
	as.mutex.Lock()
	as.sockets[userID] = conn
	as.mutex.Unlock()
	runtime.Gosched()
}
