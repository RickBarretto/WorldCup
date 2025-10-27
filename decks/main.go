package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)


func main() {
    listenAddr, node := initializeNode()

    node.startLeaderLoop()
    node.AddRoutes()
    node.StartAt(listenAddr)
}

func (node *Node) StartAt(listenAddr Address) {
	log.Printf("node %d starting on %s, peers=%v, leader=%d (%s)", node.id, listenAddr, node.peers, node.leaderID, node.leaderAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}

func (node *Node) AddRoutes() {
	http.HandleFunc("/cards", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			node.handleGetCards(w, r)
		case "POST":
			node.handlePostCard(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/cards/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			node.handleDeleteCard(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	http.HandleFunc("/replicate", node.handleReplicate)
	http.HandleFunc("/status", node.handleStatus)
}

/// server expects addr like http://host:port; extract host:port for Listen
func normalizeAddress(addressFlag *string) Address {
	listenAddr := ""
	if strings.HasPrefix(*addressFlag, "http://") {
		listenAddr = strings.TrimPrefix(*addressFlag, "http://")
	} else if strings.HasPrefix(*addressFlag, "https://") {
		listenAddr = strings.TrimPrefix(*addressFlag, "https://")
	} else {
		listenAddr = *addressFlag
	}
	return listenAddr
}

func initializeNode() (Address, *Node) {
	idFlag := flag.Int("id", 1, "numeric id for this node")
	addressFlag := flag.String("addr", "http://localhost:8001", "public address for this node, used by peers (include scheme and port)")
	peersFlag := flag.String("peers", "", "comma-separated list of peers as id=addr,id=addr")
	flag.Parse()

	peers := make(Peers)
	if *peersFlag != "" {
		items := strings.Split(*peersFlag, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			parts := strings.SplitN(item, "=", 2)
			if len(parts) != 2 {
				log.Fatalf("bad peer entry: %s", item)
			}
			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				log.Fatalf("bad peer id: %s", parts[0])
			}
			peers[pid] = parts[1]
		}
	}

	peers[*idFlag] = *addressFlag

	node := NewNode(*idFlag, *addressFlag, peers)
	normalizedAddress := normalizeAddress(addressFlag)
    return normalizedAddress, node
}
