package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ReplicateRequest used by leader to replicate an operation
type ReplicateRequest struct {
	Op   string `json:"op"` // "add" or "remove"
	Card Card   `json:"card"`
}

type PeerID = int
type Address = string
type Peers = map[PeerID]Address


// Node holds node info and state
type Node struct {
	id         PeerID
	addr       Address
	peers      Peers // id -> addr
	leaderID   PeerID
	leaderAddr Address
	deck       *Deck
	client     *http.Client
	mu         sync.RWMutex // protects leaderID/leaderAddr
}

func NewNode(id PeerID, addr Address, peers Peers) *Node {
	node := &Node{
		id:    id,
		addr:  addr,
		peers: peers,
		deck:  NewDeck(),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	node.electLeader()
	return node
}

func (node *Node) isLeader() bool {
	node.mu.RLock()
	defer node.mu.RUnlock()

	return node.leaderID == node.id
}

/// Elect a leader via bully algorithm.
///
/// The highest available ID is the leader.
func (node *Node) electLeader() {
	highestID := node.id
	highestAddress := node.addr

	for peer_id, peer_address := range node.peers {
		if peer_id > highestID {
			highestID = peer_id
			highestAddress = peer_address
		}
	}
	node.mu.Lock()
	node.leaderID = highestID
	node.leaderAddr = highestAddress
	node.mu.Unlock()
}

func (node *Node) StartLeaderLoop() {
	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for range ticker.C {
			node.electLeader()
		}
	}()
}

/// Send commands to other peers to replace the same behavior.
func (node *Node) replicateToFollowers(request ReplicateRequest) {
	data, _ := json.Marshal(request)

	for peerID, peerAddress := range node.peers {
		if peerID == node.id {
			continue
		}

		go func(address string, id int) {
			url := strings.TrimRight(address, "/") + "/replicate"
			httpRequest, err := http.NewRequest("POST", url, bytes.NewReader(data))

			if err != nil {
				log.Printf("replicate: create request to %s: %v", address, err)
				return
			}

			httpRequest.Header.Set("Content-Type", "application/json")

			response, err := node.client.Do(httpRequest)
			if err != nil {
				log.Printf("replicate: POST %s failed: %v", url, err)
				return
			}

			io.Copy(io.Discard, response.Body)
			response.Body.Close()

			if response.StatusCode >= 300 {
				log.Printf("replicate: non-2xx from %s: %s", url, response.Status)
			}
		}(peerAddress, peerID)
	}
}

/// Forward incoming requests to the leader and proxy the response
func (node *Node) forwardToLeader(
	writer http.ResponseWriter,
	request *http.Request,
) {

	node.mu.RLock()
	leader := node.leaderAddr
	node.mu.RUnlock()

	if leader == "" {
		http.Error(writer, "no leader known", http.StatusServiceUnavailable)
		return
	}

	// build URL to leader
	destinationURL := strings.TrimRight(leader, "/") + request.URL.Path

	// read body
	var bodyBytes []byte
	if request.Body != nil {
		data, _ := io.ReadAll(request.Body)
		bodyBytes = data
	}

	req, err := http.NewRequest(request.Method, destinationURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(writer, "failed to create request to leader", http.StatusInternalServerError)
		return
	}

	req.Header = request.Header.Clone()
	resp, err := node.client.Do(req)
	if err != nil {
		http.Error(writer, "leader unreachable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		writer.Header()[k] = v
	}

	writer.WriteHeader(resp.StatusCode)
	io.Copy(writer, resp.Body)
}

func (node *Node) handleGetCards(
	writer http.ResponseWriter,
	request *http.Request,
) {
	cards := node.deck.List()
	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(cards)
}

func (node *Node) handlePostCard(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !node.isLeader() {
		node.forwardToLeader(writer, request)
		return
	}

	var c Card
	if err := json.NewDecoder(request.Body).Decode(&c); err != nil {
		http.Error(writer, "invalid json", http.StatusBadRequest)
		return
	}

	node.deck.Add(c)

	// replicate
	node.replicateToFollowers(ReplicateRequest{Op: "add", Card: c})
	writer.WriteHeader(http.StatusCreated)
	json.NewEncoder(writer).Encode(c)
}

func (node *Node) handleDeleteCard(
	writer http.ResponseWriter,
	request *http.Request,
) {
	// path expected: /cards/{id}
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")

	if len(parts) != 2 {
		http.Error(writer, "bad path", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(parts[1])

	if err != nil {
		http.Error(writer, "invalid id", http.StatusBadRequest)
		return
	}

	if !node.isLeader() {
		node.forwardToLeader(writer, request)
		return
	}

	node.deck.Remove(id)
	node.replicateToFollowers(ReplicateRequest{Op: "remove", Card: Card{ID: id}})
	writer.WriteHeader(http.StatusNoContent)
}

func (node *Node) handleReplicate(
	writer http.ResponseWriter,
	request *http.Request,
) {

	var req ReplicateRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		http.Error(writer, "invalid replicate payload", http.StatusBadRequest)
		return
	}

	switch req.Op {
	case "add":
		node.deck.Add(req.Card)
	case "remove":
		node.deck.Remove(req.Card.ID)
	default:
		http.Error(writer, "unknown op", http.StatusBadRequest)
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (node *Node) handleStatus(
	writer http.ResponseWriter,
	request *http.Request,
) {

	node.mu.RLock()

	leaderID := node.leaderID
	leaderAddr := node.leaderAddr
	node.mu.RUnlock()
	out := map[string]interface{}{
		"node_id":     node.id,
		"node_addr":   node.addr,
		"leader_id":   leaderID,
		"leader_addr": leaderAddr,
	}
	writer.Header().Set("Content-Type", "application/json")

	json.NewEncoder(writer).Encode(out)
}
