// Package websocket manages websocket connections and notifications for Abot
// clients. This is used by github.com/itsabot/plugin_training to notify
// trainers of new user messages without requiring page reload.
package websocket

import (
	"runtime"
	"sync"
	"time"

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

// NotifySockets sends listening clients new messages over WebSockets,
// eliminating the need for trainers to constantly reload the page.
func (as AtomicWebSocketSet) NotifySockets(uid uint64, cmd, ret string) error {
	s := as.Get(uid)
	if s == nil {
		// Trainer is not online.
		return nil
	}
	t := time.Now()
	data := []struct {
		Sentence  string
		AvaSent   bool
		CreatedAt *time.Time
	}{
		{
			Sentence:  cmd,
			AvaSent:   false,
			CreatedAt: &t,
		},
	}
	if len(ret) > 0 {
		data = append(data, struct {
			Sentence  string
			AvaSent   bool
			CreatedAt *time.Time
		}{
			Sentence:  ret,
			AvaSent:   true,
			CreatedAt: &t,
		})
	}
	return websocket.JSON.Send(s, &data)
}
