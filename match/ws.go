package main

import "log"
import "net/http"
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

func StartServer(address Address, peers []Address) {
	server := NewServer(address)
	for _, p := range peers {
		if p != "" {
			server.AddPeer(p)
		}
	}

	// -- API --
	http.HandleFunc("/ws", server.upgradeWebsocket())
	http.HandleFunc("/play", server.playMatch())
	http.HandleFunc("/find-waiter", server.FindWaiter)
	http.HandleFunc("/start-remote-match", server.startRemoteMatch())
	http.HandleFunc("/peers", server.managePeers())

	// -- Frontend --
	fs := http.FileServer(http.Dir("./match/frontend"))
	http.Handle("/", fs)

	log.Printf("match server listening on %s\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
