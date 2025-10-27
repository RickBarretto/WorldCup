package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// / Object sent for follower replication of leader operations
type ReplicateRequest struct {
	Op   string `json:"op"`
	Card Card   `json:"card"`
	User string `json:"user,omitempty"`
}

// TradeRequest describes a swap between two users' cards.
type TradeRequest struct {
	UserA   string `json:"user_a"`
	UserB   string `json:"user_b"`
	ACardID int    `json:"a_card_id"`
	BCardID int    `json:"b_card_id"`
}

type PeerID = int
type Address = string
type Peers = map[PeerID]Address

type Node struct {
	id          PeerID
	addr        Address
	peers       Peers
	leaderID    PeerID
	leaderAddr  Address
	deck        *DeckStore
	client      *http.Client
	mu          sync.RWMutex
	trades      map[int]*TradeRequest
	nextTradeID int
}

// / Representation of the Leader state
type Snapshot struct {
	Global      []Card               `json:"global"`
	Users       map[string][]Card    `json:"users"`
	Trades      map[int]TradeRequest `json:"trades"`
	NextTradeID int                  `json:"next_trade_id"`
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
		trades: make(map[int]*TradeRequest),
	}

	node.electLeader()
	return node
}

func (node *Node) isLeader() bool {
	node.mu.RLock()
	defer node.mu.RUnlock()

	return node.leaderID == node.id
}

// / Elect a leader via bully algorithm.
// /
// / The highest available ID is the leader.
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

// / Return the state of the current node for recovery or replication.
func (node *Node) handleSnapshot(writer http.ResponseWriter, request *http.Request) {
	// build snapshot from the in-memory DeckStore
	node.mu.RLock()
	ds := node.deck
	node.mu.RUnlock()

	ds.mu.RLock()
	defer ds.mu.RUnlock()

	snap := Snapshot{
		Global: ds.global.List(),
		Users:  make(map[string][]Card),
	}

	for u, d := range ds.users {
		snap.Users[u] = d.List()
	}

	// copy pending trades and nextTradeID under node lock
	node.mu.RLock()
	snap.Trades = make(map[int]TradeRequest)
	for id, tr := range node.trades {
		if tr == nil {
			continue
		}
		snap.Trades[id] = *tr
	}
	snap.NextTradeID = node.nextTradeID
	node.mu.RUnlock()

	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(snap)
}

// SyncFromLeader attempts to fetch the leader snapshot and replace local state.
// It is safe to call on startup; if the leader is unreachable or returns an error
// the function logs and returns the error without mutating local state.
func (node *Node) SyncFromLeader() error {
	node.mu.RLock()
	leader := node.leaderAddr
	selfAddr := node.addr
	node.mu.RUnlock()

	if leader == "" || leader == selfAddr {
		return nil
	}

	url := strings.TrimRight(leader, "/") + "/snapshot"
	resp, err := node.client.Get(url)
	if err != nil {
		log.Printf("sync: failed to GET snapshot from leader %s: %v", leader, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("sync: leader %s returned non-200: %s", leader, string(body))
		return fmt.Errorf("non-200 from leader: %d", resp.StatusCode)
	}

	var snap Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		log.Printf("sync: failed to decode snapshot from leader %s: %v", leader, err)
		return err
	}

	// build a new DeckStore populated from snapshot
	newStore := NewDeckStore()
	for _, c := range snap.Global {
		newStore.Add("", c)
	}
	for u, cards := range snap.Users {
		for _, c := range cards {
			newStore.Add(u, c)
		}
	}

	node.mu.Lock()
	node.deck = newStore

	// restore trades
	node.trades = make(map[int]*TradeRequest)
	for id, tr := range snap.Trades {
		t := tr
		node.trades[id] = &t
	}
	node.nextTradeID = snap.NextTradeID

	node.mu.Unlock()

	log.Printf("sync: successfully synced state from leader %s (global=%d users=%d)", leader, len(snap.Global), len(snap.Users))
	return nil
}

// / Send commands to other peers to replace the same behavior.
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

// / Forward incoming requests to the leader and proxy the response
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
			if success {
				return
			}
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

// / Move the last card from the global deck to the specific user
// /
// / To have consistent behavior, this reuses the routes
// / DELETE /cards/:id and POST /users/:user/cards
func (node *Node) handleClaim(writer http.ResponseWriter, request *http.Request) {

	if !node.isLeader() {
		node.forwardToLeader(writer, request)
		return
	}

	// extract user from path (/users/:user/claim)
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "users" {
		http.Error(writer, "bad path", http.StatusBadRequest)
		return
	}
	user := parts[1]

	// get global list and pick last card
	list := node.deck.List("")

	if len(list) == 0 {
		node.regenGlobalDeck(20)
		list = node.deck.List("")
	}

	if len(list) == 0 {
		http.Error(writer, "no cards available", http.StatusServiceUnavailable)
		return
	}

	card := list[len(list)-1]

	// DELETE /cards/:id (re-use route)
	node.mu.RLock()
	leader := node.leaderAddr
	node.mu.RUnlock()

	// build URLs against leader so followers that forward will reach the true leader
	deleteURL := strings.TrimRight(leader, "/") + "/cards/" + strconv.Itoa(card.ID)
	reqDel, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		http.Error(writer, "failed to build delete request", http.StatusInternalServerError)
		return
	}

	respDel, err := node.client.Do(reqDel)
	if err != nil || respDel.StatusCode >= 300 {
		if respDel != nil {
			io.Copy(io.Discard, respDel.Body)
			respDel.Body.Close()
		}
		http.Error(writer, "failed to remove from global deck", http.StatusServiceUnavailable)
		return
	}
	io.Copy(io.Discard, respDel.Body)
	respDel.Body.Close()

	// POST /users/:user/cards
	postURL := strings.TrimRight(leader, "/") + "/users/" + user + "/cards"
	body, _ := json.Marshal(card)
	reqPost, err := http.NewRequest("POST", postURL, bytes.NewReader(body))
	if err != nil {
		http.Error(writer, "failed to build post request", http.StatusInternalServerError)
		return
	}
	reqPost.Header.Set("Content-Type", "application/json")

	respPost, err := node.client.Do(reqPost)
	if err != nil {
		if respPost != nil {
			io.Copy(io.Discard, respPost.Body)
			respPost.Body.Close()
		}
		http.Error(writer, "failed to add card to user", http.StatusServiceUnavailable)
		return
	}
	defer respPost.Body.Close()

	// proxy response back to client
	for k, v := range respPost.Header {
		writer.Header()[k] = v
	}
	writer.WriteHeader(respPost.StatusCode)
	io.Copy(writer, respPost.Body)
}

// / Generate n random cards and adds to the global deck.
func (node *Node) regenGlobalDeck(n int) {
	node.mu.RLock()
	leader := node.leaderAddr
	node.mu.RUnlock()

	if leader == "" {
		log.Printf("regen: no leader known, aborting regen")
		return
	}

	for i := range n {
		id := int(time.Now().UnixNano()%1e9) + i
		c := Card{ID: id, Name: fmt.Sprintf("Card-%d", id)}
		url := strings.TrimRight(leader, "/") + "/cards"
		body, _ := json.Marshal(c)
		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			log.Printf("regen: failed to build POST request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := node.client.Do(req)
		if err != nil {
			log.Printf("regen: POST /cards failed: %v", err)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			log.Printf("regen: non-2xx from POST /cards: %s", resp.Status)
		}
	}
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

// / Propose trade of two cards
// /
// / Example:
// / POST /trade {"user_a":"alice","user_b":"bob","a_card_id":1,"b_card_id":2}
func (node *Node) handleTrade(writer http.ResponseWriter, request *http.Request) {

	if !node.isLeader() {
		node.forwardToLeader(writer, request)
		return
	}

	var trade TradeRequest
	if err := json.NewDecoder(request.Body).Decode(&trade); err != nil {
		http.Error(writer, "invalid json", http.StatusBadRequest)
		return
	}

	if trade.UserA == "" || trade.UserB == "" || trade.ACardID == 0 || trade.BCardID == 0 {
		http.Error(writer, "missing fields", http.StatusBadRequest)
		return
	}

	// create and store proposal
	node.mu.Lock()
	node.nextTradeID++
	id := node.nextTradeID
	node.trades[id] = &trade
	node.mu.Unlock()

	out := map[string]interface{}{"trade_id": id, "status": "pending"}
	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(out)
}

// / Accept the trade
// /
// / Example:
// / POST /trade/:id/accept with JSON {"user":"bob"}
func (node *Node) handleTradeAccept(writer http.ResponseWriter, request *http.Request) {
	if !node.isLeader() {
		node.forwardToLeader(writer, request)
		return
	}

	// extract id from path
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(writer, "bad path", http.StatusBadRequest)
		return
	}
	idStr := parts[1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(writer, "invalid trade id", http.StatusBadRequest)
		return
	}

	var payload struct {
		User string `json:"user"`
	}
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		http.Error(writer, "invalid json", http.StatusBadRequest)
		return
	}

	node.mu.Lock()
	tr, ok := node.trades[id]
	if !ok {
		node.mu.Unlock()
		http.Error(writer, "trade not found", http.StatusNotFound)
		return
	}
	// ensure acceptor matches UserB
	if payload.User != tr.UserB {
		node.mu.Unlock()
		http.Error(writer, "only the counterparty can accept the trade", http.StatusForbidden)
		return
	}
	// remove from store to prevent double-accept
	delete(node.trades, id)
	node.mu.Unlock()

	// verify both cards still exist
	aCard, okA := findCardInList(node.deck.List(tr.UserA), tr.ACardID)
	bCard, okB := findCardInList(node.deck.List(tr.UserB), tr.BCardID)
	if !okA || !okB {
		http.Error(writer, "one or both cards not found", http.StatusBadRequest)
		return
	}

	// execute swap (leader does the mutating and replicates)
	node.deck.Remove(tr.UserA, tr.ACardID)
	node.replicateToFollowers(ReplicateRequest{Op: "remove", Card: Card{ID: tr.ACardID}, User: tr.UserA})

	node.deck.Remove(tr.UserB, tr.BCardID)
	node.replicateToFollowers(ReplicateRequest{Op: "remove", Card: Card{ID: tr.BCardID}, User: tr.UserB})

	node.deck.Add(tr.UserA, bCard)
	node.replicateToFollowers(ReplicateRequest{Op: "add", Card: bCard, User: tr.UserA})

	node.deck.Add(tr.UserB, aCard)
	node.replicateToFollowers(ReplicateRequest{Op: "add", Card: aCard, User: tr.UserB})

	out := map[string]Card{"user_a_received": bCard, "user_b_received": aCard}
	writer.Header().Set("Content-Type", "application/json")
	json.NewEncoder(writer).Encode(out)
}

// findCardInList returns the card with id from list and a bool indicating presence
func findCardInList(list []Card, id int) (Card, bool) {
	for _, c := range list {
		if c.ID == id {
			return c, true
		}
	}
	return Card{}, false
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
