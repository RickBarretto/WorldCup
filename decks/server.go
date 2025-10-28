package main

import "github.com/gin-gonic/gin"

// / Add Routes to a Node
func (node *Node) AddRoutes(router *gin.Engine) {
	// -- Frontend pages --

	router.Static("/decks/static", "./decks/frontend")
	router.GET("/", func(c *gin.Context) {
		c.File("./decks/frontend/main.html")
	})
	router.GET("/admin", func(c *gin.Context) {
		c.File("./decks/frontend/admin.html")
	})
	router.GET("/user", func(c *gin.Context) {
		c.File("./decks/frontend/user.html")
	})
	router.GET("/trade", func(c *gin.Context) {
		c.File("./decks/frontend/trade.html")
	})

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
