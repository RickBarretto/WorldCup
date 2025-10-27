package main

import "log"

import "github.com/gin-gonic/gin"


func main() {
    router := gin.Default()
	address, node := NodeFromCLI()

	node.StartLeaderLoop()
	node.AddRoutes(router)

    log.Printf("Node%d@%s: Leader%d@%s, peers=[%v]", 
        node.id,
        address,
        node.leaderID,
        node.leaderAddr,
        node.peers,
    )
    router.Run(address)
}
