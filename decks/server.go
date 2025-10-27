package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func (node *Node) Serve(listenAddr Address) {
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
