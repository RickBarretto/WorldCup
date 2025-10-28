package main

import "sync"
import "time"

import "github.com/gorilla/websocket"

/// Player Websocket Connection
type PlayerConnection struct {
	connection *websocket.Conn
	mutex      sync.Mutex
}

func newPlayerConnection(connection *websocket.Conn) *PlayerConnection {
	return &PlayerConnection{connection: connection}
}

func (player *PlayerConnection) sendJSON(value any) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.connection == nil {
		return
	}
	timeout := time.Now().Add(5 * time.Second)
	player.connection.SetWriteDeadline(timeout)
	_ = player.connection.WriteJSON(value)
}

