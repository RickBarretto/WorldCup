package main

import (
	"log"

	"github.com/gin-gonic/gin"
)


func main() {
    router := gin.Default()
	address, node := NodeFromCLI()

	node.StartLeaderLoop()
	node.AddRoutes(router)
    // attempt to sync state from the leader on startup (non-blocking errors)
    if err := node.SyncFromLeader(); err != nil {
        log.Printf("warning: could not sync from leader on startup: %v", err)
    }

    log.Printf("Node%d@%s: Leader%d@%s, peers=[%v]", 
        node.id,
        address,
        node.leaderID,
        node.leaderAddr,
        node.peers,
    )
    router.Run(address)
}
