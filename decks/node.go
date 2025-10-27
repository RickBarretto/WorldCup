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

/// Object sent for follower replication of leader operations
type ReplicateRequest struct {
	Op   string `json:"op"`
	Card Card   `json:"card"`
	User string `json:"user,omitempty"`
}


type PeerID = int
type Address = string
type Peers = map[PeerID]Address


type Node struct {
	id         PeerID
	addr       Address
	peers      Peers
	leaderID   PeerID
	leaderAddr Address
	deck       *DeckStore
	client     *http.Client
	mu         sync.RWMutex
}


func NewNode(id PeerID, addr Address, peers Peers) *Node {
	node := &Node{
		id:    id,
		addr:  addr,
		peers: peers,
		deck:  NewDeckStore(),
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
	// choose the highest *reachable* ID (bully algorithm variant)
	isAvailable := func(id PeerID, address Address) bool {
		// self is always considered available
		if id == node.id {
			return true
		}

		url := strings.TrimRight(address, "/") + "/status"
		resp, err := node.client.Get(url)
		if err != nil {
			return false
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}

	highestID := -1
	highestAddress := ""

	// consider self
	if isAvailable(node.id, node.addr) {
		highestID = node.id
		highestAddress = node.addr
	}

	for peer_id, peer_address := range node.peers {
		if !isAvailable(peer_id, peer_address) {
			continue
		}

		if peer_id > highestID {
			highestID = peer_id
			highestAddress = peer_address
		}
	}

	// fallback to self if nothing reachable (shouldn't normally happen)
	if highestID == -1 {
		highestID = node.id
		highestAddress = node.addr
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
		// leader failed to respond — trigger immediate re-election and retry once
		newLeader := TriggerReElection(leader, err, node)

		isNewLeader := newLeader != "" && newLeader != leader
		
		if isNewLeader {
			success := forwardRequest(newLeader, request, bodyBytes, node, writer)
			if success { return }
		}

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


func forwardRequest(
	newLeader Address, 
	request *http.Request, 
	bodyBytes []byte, 
	node *Node, 
	writer http.ResponseWriter,
) bool {
	url := strings.TrimRight(newLeader, "/") + request.URL.Path
	retryRequest, error := http.NewRequest(request.Method, url, bytes.NewReader(bodyBytes))
	
	if error == nil {
		retryRequest.Header = request.Header.Clone()
		response, requestError := node.client.Do(retryRequest)
		if requestError == nil {
			defer response.Body.Close()
			for k, v := range response.Header {
				writer.Header()[k] = v
			}
			writer.WriteHeader(response.StatusCode)
			io.Copy(writer, response.Body)
			return true
		}
	}

	return false
}

func TriggerReElection(leader Address, err error, node *Node) Address {
	log.Printf("forward: leader %s unreachable: %v; triggering re-election", leader, err)
	node.electLeader()

	node.mu.RLock()
	newLeader := node.leaderAddr
	node.mu.RUnlock()
	return newLeader
}

// getUserFromRequest extracts the target user for the deck from the request.
// Priority: query parameter `user` -> header `X-User` -> empty string (global deck)
func getUserFromRequest(request *http.Request) string {
	if request == nil {
		return ""
	}
	// First, try to parse path of form /users/:user/...
	path := strings.Trim(request.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) >= 2 && parts[0] == "users" {
		return strings.TrimSpace(parts[1])
	}

	// Query param fallback
	q := request.URL.Query().Get("user")
	if q != "" {
		return strings.TrimSpace(q)
	}

	// Header fallback
	h := request.Header.Get("X-User")
	return strings.TrimSpace(h)
}

func (node *Node) handleGetCards(
	writer http.ResponseWriter,
	request *http.Request,
) {
	user := getUserFromRequest(request)
	cards := node.deck.List(user)
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

	user := getUserFromRequest(request)

	node.deck.Add(user, c)

	// replicate (include user so followers update the same user's deck)
	node.replicateToFollowers(ReplicateRequest{Op: "add", Card: c, User: user})
	writer.WriteHeader(http.StatusCreated)
	json.NewEncoder(writer).Encode(c)
}

func (node *Node) handleDeleteCard(
	writer http.ResponseWriter,
	request *http.Request,
) {
	// Support both:
	//  - /cards/{id}
	//  - /users/{user}/cards/{id}
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")

	var idPart string
	if len(parts) >= 4 && parts[0] == "users" {
		// expect: users/:user/cards/:id
		if parts[2] != "cards" {
			http.Error(writer, "bad path", http.StatusBadRequest)
			return
		}
		idPart = parts[len(parts)-1]
	} else if len(parts) == 2 && parts[0] == "cards" {
		idPart = parts[1]
	} else {
		http.Error(writer, "bad path", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idPart)
	if err != nil {
		http.Error(writer, "invalid id", http.StatusBadRequest)
		return
	}

	if !node.isLeader() {
		node.forwardToLeader(writer, request)
		return
	}

	user := getUserFromRequest(request)

	node.deck.Remove(user, id)
	node.replicateToFollowers(ReplicateRequest{Op: "remove", Card: Card{ID: id}, User: user})
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
		node.deck.Add(req.User, req.Card)
	case "remove":
		node.deck.Remove(req.User, req.Card.ID)
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
