package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"sync"
)

type Server struct {
	mutex sync.Mutex

	/// Match Related
	players map[string]*PlayerConnection
	waiting []WaitingPlayer

	/// Peer Related
	address Address
	peers   []Address
}

func NewServer(address Address) *Server {
	return &Server{
		peers:   []string{},
		players: make(map[string]*PlayerConnection),
		waiting: make([]WaitingPlayer, 0),
		address: address,
	}
}

func (server *Server) AddPeer(peer Address) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if !slices.Contains(server.peers, peer) {
		server.peers = append(server.peers, peer)
	}
}

func (server *Server) ListPeers() []Address {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	out := make([]Address, len(server.peers))
	copy(out, server.peers)
	return out
}

func (server *Server) LinkPlayer(
	player Username,
	connection *PlayerConnection,
) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.players[player] = connection
}

func (server *Server) UnlinkPlayer(player Username) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	delete(server.players, player)
}

func (server *Server) Connection(playerID string) *PlayerConnection {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	return server.players[playerID]
}

/// Try to match locally
func (server *Server) tryLocalMatch(player Challenger) (*Match, bool) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if len(server.waiting) == 0 {
		return nil, false
	}

	waiter := server.waiting[0]
	server.waiting = server.waiting[1:]
	match := createMatch(waiter, player, server.address)
	return match, true
}

func (server *Server) enqueueWaiter(waiter WaitingPlayer) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.waiting = append(server.waiting, waiter)
}

func (server *Server) popWaiter() *WaitingPlayer {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	if len(server.waiting) == 0 {
		return nil
	}

	waiter := server.waiting[0]
	server.waiting = server.waiting[1:]
	return &waiter
}

func createMatch(
	host WaitingPlayer,
	guest WaitingPlayer,
	local Address,
) *Match {

	hostInfo := PlayerInfo{
		ID:     host.PlayerID,
		Server: local,
		Cards:  host.Cards,
	}

	guestInfo := PlayerInfo{
		ID:     guest.PlayerID,
		Server: local,
		Cards:  guest.Cards,
	}

	match := &Match{
		ID:    newCardID(),
		Host:  hostInfo,
		Guest: guestInfo,
	}

	hostScore := scoreOf(host.Cards)
	guestScore := scoreOf(guest.Cards)

	if hostScore > guestScore {
		match.Winner = host.PlayerID
	} else if guestScore > hostScore {
		match.Winner = guest.PlayerID
	} else {
		match.Winner = "draw"
	}
	return match
}

func scoreOf(cards []Card) int {
	s := 0
	for _, c := range cards {
		s += c.Power
	}
	return s
}

/// Send JSON message to the player's websocket connection if present
func (server *Server) notifyLocal(player Username, payload any) {
	socket := server.Connection(player)

	if socket == nil {
		// Print player with %q so empty player IDs are visible in logs
		log.Printf("no websocket for player %q\n", player)
		return
	}

	socket.sendJSON(payload)
}

/// Invoked when a remote server wants a waiting player.
///
/// The payload includes the challenger info and a callback URL.
/// If there is a waiter, a match is created pairing both players and
/// notify the waiter.
func (server *Server) FindWaiter(
	writer http.ResponseWriter,
	request *http.Request,
) {

	var data struct {
		PlayerID    string `json:"player_id"`
		Cards       []Card `json:"cards"`
		CallbackURL string `json:"callback"`
		Server      string `json:"server"`
	}

	if err := json.NewDecoder(request.Body).Decode(&data); err != nil {
		http.Error(writer, "bad json", http.StatusBadRequest)
		return
	}

	waiter := server.popWaiter()

	if waiter == nil {
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	challenger := WaitingPlayer{PlayerID: data.PlayerID, Cards: data.Cards}
	match := createMatch(*waiter, challenger, server.address)

	go server.notifyLocal(waiter.PlayerID, map[string]any{
		"type":  "match_start",
		"match": match,
	})

	body, _ := json.Marshal(match)
	go func() {
		_, err := http.Post(data.CallbackURL, "application/json", bytes.NewReader(body))
		if err != nil {
			log.Printf("failed to POST match to callback %s: %v", data.CallbackURL, err)
		}
	}()

	writer.Header().Set("content-type", "application/json")
	json.NewEncoder(writer).Encode(match)
}
