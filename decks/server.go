package main

import "github.com/gin-gonic/gin"

// / Add Routes to a Node
func (node *Node) AddRoutes(router *gin.Engine) {
	// -- User endpoints --
	router.GET("/users/:user/claim", gin.WrapF(node.handleClaim))
	router.GET("/users/:user/cards", gin.WrapF(node.handleGetCards))
	
	router.POST("/trade", gin.WrapF(node.handleTrade))
	router.POST("/trade/:id/accept", gin.WrapF(node.handleTradeAccept))

	// -- Admin endpoints --
	router.GET("/cards", gin.WrapF(node.handleGetCards))
	router.POST("/cards", gin.WrapF(node.handlePostCard))
	router.DELETE("/cards/:id", gin.WrapF(node.handleDeleteCard))

	router.POST("/users/:user/cards", gin.WrapF(node.handlePostCard))
	router.DELETE("/users/:user/cards/:id", gin.WrapF(node.handleDeleteCard))


	// -- Peer endpoints --
	router.GET("/status", gin.WrapF(node.handleStatus))
	router.GET("/snapshot", gin.WrapF(node.handleSnapshot))
	router.POST("/replicate", gin.WrapF(node.handleReplicate))
}
