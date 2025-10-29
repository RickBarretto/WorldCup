package main

import (
	"log"
	"net/http"
)

func main() {
	cli := parseCli()
	StartServer(cli.address, cli.peers)
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
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		fs.ServeHTTP(w, r)
	})

	log.Printf("match server listening on %s\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}